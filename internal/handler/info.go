package handler

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"ai-guardian-challenge/internal/config"
	"ai-guardian-challenge/internal/model"
	"ai-guardian-challenge/internal/store"
)

// InfoHandler 站点信息相关的 HTTP 处理器
type InfoHandler struct {
	store  *store.Store
	config *config.Config
}

// NewInfoHandler 创建信息处理器
func NewInfoHandler(s *store.Store, cfg *config.Config) *InfoHandler {
	return &InfoHandler{store: s, config: cfg}
}

// GetSiteInfo 返回站点配置信息
func (h *InfoHandler) GetSiteInfo(w http.ResponseWriter, r *http.Request) {
	deadline := h.config.DeadlineTime()
	isExpired := time.Now().After(deadline)

	info := model.SiteInfo{
		Deadline:    h.config.Game.Deadline,
		IsExpired:   isExpired,
		CaptchaType: "simple", // 使用简化验证
		AdminQQ:     h.config.Admin.Contact,
		AdminEmail:  h.config.Admin.Email,
		AdminWechat: h.config.Admin.Wechat,
	}

	writeJSON(w, http.StatusOK, info)
}

// GetWinners 获取获奖者列表（分页）
func (h *InfoHandler) GetWinners(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 5
	}

	winners, total := h.store.GetWinners(page, pageSize)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	writeJSON(w, http.StatusOK, model.PaginatedResponse{
		Data:       winners,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetPublicConversations 获取公开对话列表（分页）
func (h *InfoHandler) GetPublicConversations(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 15
	}

	convs, total := h.store.GetPublicConversations(page, pageSize)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	writeJSON(w, http.StatusOK, model.PaginatedResponse{
		Data:       convs,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetUserConversations 获取当前用户的对话列表（分页，需登录）
func (h *InfoHandler) GetUserConversations(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, `{"error":"未登录"}`, http.StatusUnauthorized)
		return
	}

	user := h.store.GetUserBySession(cookie.Value)
	if user == nil {
		http.Error(w, `{"error":"会话已过期"}`, http.StatusUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 15
	}

	convs, total := h.store.GetUserConversations(user.ID, page, pageSize)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	writeJSON(w, http.StatusOK, model.PaginatedResponse{
		Data:       convs,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}
