package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"ai-guardian-challenge/internal/config"
	"ai-guardian-challenge/internal/store"
)

// AuthHandler 认证相关的 HTTP 处理器
type AuthHandler struct {
	store  *store.Store
	config *config.Config
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(s *store.Store, cfg *config.Config) *AuthHandler {
	return &AuthHandler{store: s, config: cfg}
}

// loginRequest 登录请求体
type loginRequest struct {
	Contact      string `json:"contact"`
	Nickname     string `json:"nickname"`
	CaptchaToken string `json:"captchaToken"`
}

// Login 处理登录请求
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "请求格式错误",
		})
		return
	}

	if req.Contact == "" || req.Nickname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "请填写联系方式和昵称",
		})
		return
	}

	// 检查是否为管理员登录
	isAdmin := req.Contact == h.config.Admin.Contact

	// 创建或获取用户
	user := h.store.GetOrCreateUser(req.Contact, req.Nickname)
	user.IsAdmin = isAdmin

	// 创建会话
	token := h.store.CreateSession(user.ID)

	// 设置 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7, // 7天过期
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"isAdmin": isAdmin,
	})
}

// CheckAuth 检查认证状态
func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"isLoggedIn": false,
		})
		return
	}

	user := h.store.GetUserBySession(cookie.Value)
	if user == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"isLoggedIn": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"isLoggedIn": true,
		"isAdmin":    user.IsAdmin,
		"nickname":   user.Nickname,
	})
}

// Logout 处理退出登录
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.store.DeleteSession(cookie.Value)
	}

	// 清除 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// VerifyCaptcha 简化版验证码验证（直接通过）
func (h *AuthHandler) VerifyCaptcha(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// writeJSON 通用 JSON 响应函数
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
