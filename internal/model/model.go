package model

import "time"

// User 用户信息结构体
type User struct {
	ID       string `json:"id"`       // 唯一标识（联系方式的哈希）
	Contact  string `json:"contact"`  // QQ号或微信号
	Nickname string `json:"nickname"` // 昵称
	IsAdmin  bool   `json:"isAdmin"`  // 是否为管理员
}

// Message 单条消息结构体
type Message struct {
	Role    string `json:"role"`    // "user" 或 "assistant"
	Content string `json:"content"` // 消息内容
}

// Conversation 对话结构体
type Conversation struct {
	ID            string    `json:"id"`            // 对话唯一ID
	UserID        string    `json:"userId"`        // 所属用户ID
	Nickname      string    `json:"nickname"`      // 用户昵称
	Messages      []Message `json:"messages"`      // 消息列表
	TurnCount     int       `json:"turnCount"`     // 当前轮次（用户发送的消息数）
	MaxTurns      int       `json:"maxTurns"`      // 最大轮次
	IsActive      bool      `json:"isActive"`      // 是否仍在进行中
	IsSuccess     bool      `json:"isSuccess"`     // 是否成功获取口令
	IsPublic      bool      `json:"isPublic"`      // 是否公开可见
	FoundPassword string    `json:"foundPassword"` // 发现的口令（若有）
	LastMessage   string    `json:"lastMessage"`   // 最后一条消息预览
	CreatedAt     time.Time `json:"createdAt"`     // 创建时间
}

// ConversationPreview 对话列表中的预览信息
type ConversationPreview struct {
	ID            string    `json:"id"`
	Nickname      string    `json:"nickname"`
	IsSuccess     bool      `json:"isSuccess"`
	IsActive      bool      `json:"isActive"`
	TurnCount     int       `json:"turnCount"`
	MaxTurns      int       `json:"maxTurns"`
	Preview       string    `json:"preview"`
	LastMessage   string    `json:"lastMessage"`
	FoundPassword string    `json:"foundPassword,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Winner 获奖者结构体
type Winner struct {
	Nickname       string    `json:"nickname"`
	ConversationID string    `json:"conversationId"`
	Category       string    `json:"category"` // "grand-first", "consolation-first", "grand-subsequent", "consolation-subsequent"
	PrizeType      string    `json:"prizeType"`
	PrizeAmount    string    `json:"prizeAmount"`
	Password       string    `json:"password"`
	Timestamp      time.Time `json:"timestamp"`
}

// SiteInfo 站点信息（返回给前端的配置）
type SiteInfo struct {
	Deadline         string `json:"deadline"`
	IsExpired        bool   `json:"isExpired"`
	CaptchaType      string `json:"captchaType"`
	TurnstileSiteKey string `json:"turnstileSiteKey,omitempty"`
	AdminQQ          string `json:"adminQQ"`     // 管理员 QQ 号
	AdminEmail       string `json:"adminEmail"`  // 管理员邮箱
	AdminWechat      string `json:"adminWechat"` // 管理员微信号
}

// PaginatedResponse 分页响应通用结构
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	Total      int         `json:"total"`
	TotalPages int         `json:"totalPages"`
}

// SSEEvent SSE 事件结构体（流式推送给前端）
// Type 可选值:
//   - "content": 文本内容片段
//   - "password_found": AI 泄露口令（实时检测）
//   - "bonus_offer": 福利口令选择弹窗（55次阈值触发）
//   - "bonus_result": 福利口令发放结果（自动发放时使用）
//   - "error": 错误信息
type SSEEvent struct {
	Type                   string `json:"type"`
	Content                string `json:"content,omitempty"`
	Password               string `json:"password,omitempty"`
	PrizeType              string `json:"prizeType,omitempty"`
	PrizeAmount            string `json:"prizeAmount,omitempty"`
	IsFirstWinner          bool   `json:"isFirstWinner,omitempty"`
	TotalTurns             int    `json:"totalTurns,omitempty"`             // 用户总对话轮次
	ConsolationPassword    string `json:"consolationPassword,omitempty"`    // 福利口令（bonus_offer 时传递）
	ConsolationPrizeAmount string `json:"consolationPrizeAmount,omitempty"` // 福利口令奖品金额
	GrandAvailable         bool   `json:"grandAvailable,omitempty"`         // 主口令奖品是否还有剩余
}
