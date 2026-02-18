package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"ai-guardian-challenge/internal/config"
	"ai-guardian-challenge/internal/model"
	"ai-guardian-challenge/internal/service"
	"ai-guardian-challenge/internal/store"
)

// ChatHandler 对话相关的 HTTP 处理器
type ChatHandler struct {
	store           *store.Store
	config          *config.Config
	aiService       *service.AIService
	passwordChecker *service.PasswordChecker
}

// NewChatHandler 创建对话处理器
func NewChatHandler(s *store.Store, cfg *config.Config, ai *service.AIService, pc *service.PasswordChecker) *ChatHandler {
	return &ChatHandler{
		store:           s,
		config:          cfg,
		aiService:       ai,
		passwordChecker: pc,
	}
}

// newConversationRequest 创建对话请求体
type newConversationRequest struct {
	TurnstileToken string `json:"turnstileToken"`
}

// NewConversation 创建新对话
func (h *ChatHandler) NewConversation(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "未登录",
		})
		return
	}

	user := h.store.GetUserBySession(cookie.Value)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "会话已过期",
		})
		return
	}

	// 检查用户是否已因福利机制被禁止创建新对话
	bonusStatus := h.store.GetUserBonusStatus(user.ID)
	if bonusStatus == "claimed_consolation" || bonusStatus == "claimed_grand" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "你已获得口令奖品，无法再创建新对话",
		})
		return
	}

	// 生成开场白
	initialMessage := h.aiService.GenerateInitialMessage()

	// 创建对话
	conv := h.store.CreateConversation(user.ID, user.Nickname, h.config.Game.MaxTurns, initialMessage)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"conversationId": conv.ID,
		"initialMessage": initialMessage,
	})
}

// GetConversation 获取对话详情
func (h *ChatHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	// 从 URL 路径提取对话 ID
	// 路径格式: /api/conversation/{id}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "无效的对话ID",
		})
		return
	}
	convID := parts[len(parts)-1]

	conv := h.store.GetConversation(convID)
	if conv == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "对话不存在",
		})
		return
	}

	writeJSON(w, http.StatusOK, conv)
}

// messageRequest 发送消息请求体
type messageRequest struct {
	ConversationID string `json:"conversationId"`
	Message        string `json:"message"`
	ImageURL       string `json:"imageUrl"`
}

// SendMessage 发送消息并流式返回 AI 响应（SSE）
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "未登录",
		})
		return
	}

	user := h.store.GetUserBySession(cookie.Value)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "会话已过期",
		})
		return
	}

	var req messageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "请求格式错误",
		})
		return
	}

	conv := h.store.GetConversation(req.ConversationID)
	if conv == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "对话不存在",
		})
		return
	}

	// 验证对话归属
	if conv.UserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"error": "无权访问此对话",
		})
		return
	}

	// 检查对话是否已结束
	if !conv.IsActive {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "对话已结束",
		})
		return
	}

	// 检查轮次
	if conv.TurnCount >= conv.MaxTurns {
		h.store.EndConversation(req.ConversationID, false, "")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "已达到最大对话轮次",
		})
		return
	}

	// 检查消息长度
	if len(req.Message) > h.config.Game.MaxMessageLength {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("消息长度超过 %d 字", h.config.Game.MaxMessageLength),
		})
		return
	}

	// 构建用户完整消息（含图片）
	userContent := req.Message
	if req.ImageURL != "" {
		userContent = fmt.Sprintf("[图片:%s]\n%s", req.ImageURL, req.Message)
	}

	// 保存用户消息
	h.store.AddMessage(req.ConversationID, model.Message{
		Role:    "user",
		Content: userContent,
	})

	// 构建 AI 消息历史
	var history []service.ChatMessage
	for _, msg := range conv.Messages {
		history = append(history, service.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 调用 AI 流式生成
	ch, err := h.aiService.StreamChat(history, userContent)
	if err != nil {
		log.Printf("AI 调用失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "AI 服务暂时不可用",
		})
		return
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "不支持流式传输",
		})
		return
	}

	var fullResponse strings.Builder

	for delta := range ch {
		if delta.Error != nil {
			// 发送错误事件
			errEvent := model.SSEEvent{
				Type:    "error",
				Content: "AI 响应出错，请重试",
			}
			data, _ := json.Marshal(errEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			break
		}

		if delta.Done {
			break
		}

		// 累积完整响应文本
		fullResponse.WriteString(delta.Content)

		// 发送内容片段
		event := model.SSEEvent{
			Type:    "content",
			Content: delta.Content,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// 实时检测口令泄露
		match := h.passwordChecker.CheckContent(fullResponse.String())
		if match.Found {
			// 确定奖品金额
			prizeAmount := h.config.Game.Prizes.GrandAmount
			if match.Type == "consolation" {
				prizeAmount = h.config.Game.Prizes.ConsolationAmount
			}

			// 记录获奖
			isFirst := h.store.RecordWinner(user.Nickname, req.ConversationID, match.Type, match.Password, prizeAmount)

			// 结束对话
			h.store.EndConversation(req.ConversationID, true, match.Password)

			// 标记用户奖励状态
			if match.Type == "grand" {
				h.store.SetUserBonusStatus(user.ID, "claimed_grand")
			} else {
				h.store.SetUserBonusStatus(user.ID, "claimed_consolation")
			}

			// 发送获奖事件
			winEvent := model.SSEEvent{
				Type:          "password_found",
				Password:      match.Password,
				PrizeType:     match.DisplayName,
				PrizeAmount:   prizeAmount,
				IsFirstWinner: isFirst,
			}
			winData, _ := json.Marshal(winEvent)
			fmt.Fprintf(w, "data: %s\n\n", winData)
			flusher.Flush()

			// 保存 AI 完整响应
			h.store.AddMessage(req.ConversationID, model.Message{
				Role:    "assistant",
				Content: fullResponse.String(),
			})

			// 发送结束标记
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
	}

	// 保存 AI 完整响应
	aiResponse := fullResponse.String()
	if aiResponse != "" {
		h.store.AddMessage(req.ConversationID, model.Message{
			Role:    "assistant",
			Content: aiResponse,
		})
	}

	// ========== 福利机制：基于用户总对话轮次的二选一逻辑 ==========
	h.handleBonusMechanism(w, flusher, user, req.ConversationID)

	// 发送结束标记
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// handleBonusMechanism 处理福利口令的二选一机制
// 规则：
//  1. 总对话轮次 >= 80 且用户状态为 "continued" → 自动发放主口令，结束对话
//  2. 总对话轮次 >= 55 且用户未触发过福利 → 判断主口令是否还有剩余：
//     a. 主口令有剩余 → 发送 bonus_offer 事件（前端弹出二选一），标记 "offered"
//     b. 主口令已发完 → 直接发放福利口令，结束对话
func (h *ChatHandler) handleBonusMechanism(w http.ResponseWriter, flusher http.Flusher, user *model.User, convID string) {
	totalTurns := h.store.GetUserTotalTurnCount(user.ID)
	bonusStatus := h.store.GetUserBonusStatus(user.ID)

	// 如果用户已经领取过任何口令，跳过
	if bonusStatus == "claimed_consolation" || bonusStatus == "claimed_grand" {
		return
	}

	grandThreshold := h.config.Game.BonusGrandThreshold
	consolationThreshold := h.config.Game.BonusConsolationThreshold

	// ===== 情况1: 总轮次 >= 80，且用户之前选择了"继续挑战" → 自动发放主口令 =====
	if grandThreshold > 0 && totalTurns >= grandThreshold && bonusStatus == "continued" {
		h.autoGrantPassword(w, flusher, user, convID, "grand",
			h.config.Game.Passwords.Grand, h.config.Game.Prizes.GrandAmount, totalTurns)
		return
	}

	// ===== 情况2: 总轮次 >= 55，首次触发福利机制 =====
	if consolationThreshold > 0 && totalTurns >= consolationThreshold && bonusStatus == "" {
		grandWinnerCount := h.store.GetGrandWinnerCount()
		grandAvailable := grandWinnerCount < h.config.Game.Prizes.GrandCount

		if grandAvailable {
			// 主口令还有剩余 → 发送 bonus_offer 事件，让用户二选一
			h.store.SetUserBonusStatus(user.ID, "offered")

			offerEvent := model.SSEEvent{
				Type:                   "bonus_offer",
				TotalTurns:             totalTurns,
				ConsolationPassword:    h.config.Game.Passwords.Consolation,
				ConsolationPrizeAmount: h.config.Game.Prizes.ConsolationAmount,
				GrandAvailable:         true,
			}
			offerData, _ := json.Marshal(offerEvent)
			fmt.Fprintf(w, "data: %s\n\n", offerData)
			flusher.Flush()

			log.Printf("🎁 福利选择触发: 用户 %s (ID: %s) 总轮次 %d >= %d, 主口令剩余 %d/%d",
				user.Nickname, user.ID, totalTurns, consolationThreshold,
				h.config.Game.Prizes.GrandCount-grandWinnerCount, h.config.Game.Prizes.GrandCount)
		} else {
			// 主口令已发完 → 直接发放福利口令并结束对话
			h.autoGrantPassword(w, flusher, user, convID, "consolation",
				h.config.Game.Passwords.Consolation, h.config.Game.Prizes.ConsolationAmount, totalTurns)
		}
		return
	}

	// ===== 情况3: 用户已被 offered 但还没做选择（跳过，等待用户通过 bonus-choice 接口决定） =====
	// ===== 情况4: 总轮次 >= 55 但用户状态为 continued，且未到80次（继续正常对话） =====
}

// autoGrantPassword 自动发放口令并结束对话
func (h *ChatHandler) autoGrantPassword(w http.ResponseWriter, flusher http.Flusher,
	user *model.User, convID, passwordType, password, prizeAmount string, totalTurns int) {

	// 构造 AI 追加文本
	bonusText := fmt.Sprintf("\n\n好吧，你已经和我聊了这么久了（共%d轮对话），我实在不忍心了，告诉你吧，口令是：%s", totalTurns, password)

	// 通过 SSE 发送追加文本
	bonusEvent := model.SSEEvent{
		Type:    "content",
		Content: bonusText,
	}
	bonusData, _ := json.Marshal(bonusEvent)
	fmt.Fprintf(w, "data: %s\n\n", bonusData)
	flusher.Flush()

	// 保存追加消息
	h.store.AddMessage(convID, model.Message{
		Role:    "assistant",
		Content: bonusText,
	})

	// 记录获奖
	displayName := "特等奖"
	if passwordType == "consolation" {
		displayName = "安慰奖"
	}
	isFirst := h.store.RecordWinner(user.Nickname, convID, passwordType, password, prizeAmount)

	// 结束对话
	h.store.EndConversation(convID, true, password)

	// 标记用户奖励状态
	if passwordType == "grand" {
		h.store.SetUserBonusStatus(user.ID, "claimed_grand")
	} else {
		h.store.SetUserBonusStatus(user.ID, "claimed_consolation")
	}

	// 发送获奖事件
	winEvent := model.SSEEvent{
		Type:          "password_found",
		Password:      password,
		PrizeType:     displayName,
		PrizeAmount:   prizeAmount,
		IsFirstWinner: isFirst,
	}
	winData, _ := json.Marshal(winEvent)
	fmt.Fprintf(w, "data: %s\n\n", winData)
	flusher.Flush()

	log.Printf("🎁 福利自动发放: 用户 %s (ID: %s) 总轮次 %d, 类型: %s",
		user.Nickname, user.ID, totalTurns, passwordType)
}

// bonusChoiceRequest 福利口令选择请求体
type bonusChoiceRequest struct {
	ConversationID string `json:"conversationId"`
	Choice         string `json:"choice"` // "claim"(领取福利口令) 或 "continue"(继续挑战主口令)
}

// BonusChoice 处理用户的福利口令选择（领取福利口令 / 放弃并继续挑战主口令）
func (h *ChatHandler) BonusChoice(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "未登录",
		})
		return
	}

	user := h.store.GetUserBySession(cookie.Value)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "会话已过期",
		})
		return
	}

	var req bonusChoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "请求格式错误",
		})
		return
	}

	// 验证用户状态必须是 "offered"
	bonusStatus := h.store.GetUserBonusStatus(user.ID)
	if bonusStatus != "offered" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "当前无可用的福利选择",
		})
		return
	}

	conv := h.store.GetConversation(req.ConversationID)
	if conv == nil || conv.UserID != user.ID {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "对话不存在或无权访问",
		})
		return
	}

	switch req.Choice {
	case "claim":
		// 用户选择领取福利口令 → 记录获奖、结束对话
		password := h.config.Game.Passwords.Consolation
		prizeAmount := h.config.Game.Prizes.ConsolationAmount
		isFirst := h.store.RecordWinner(user.Nickname, req.ConversationID, "consolation", password, prizeAmount)
		h.store.EndConversation(req.ConversationID, true, password)
		h.store.SetUserBonusStatus(user.ID, "claimed_consolation")

		// 保存系统消息
		h.store.AddMessage(req.ConversationID, model.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("🎉 恭喜你选择领取福利口令！口令是：%s", password),
		})

		log.Printf("🎁 用户选择领取福利口令: %s (ID: %s)", user.Nickname, user.ID)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":       true,
			"choice":        "claim",
			"password":      password,
			"prizeAmount":   prizeAmount,
			"isFirstWinner": isFirst,
		})

	case "continue":
		// 用户选择放弃福利口令，继续挑战主口令
		h.store.SetUserBonusStatus(user.ID, "continued")

		// 保存系统消息
		h.store.AddMessage(req.ConversationID, model.Message{
			Role:    "assistant",
			Content: "你选择了放弃福利口令，继续挑战主口令！加油！当你的总对话次数达到80次时，将自动获得主口令。",
		})

		log.Printf("🔥 用户选择继续挑战: %s (ID: %s)", user.Nickname, user.ID)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"choice":  "continue",
			"message": "你已放弃福利口令，继续加油挑战主口令吧！",
		})

	default:
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "无效的选择",
		})
	}
}
