package middleware

import (
	"context"
	"net/http"

	"ai-guardian-challenge/internal/model"
	"ai-guardian-challenge/internal/store"
)

// 上下文键类型
type contextKey string

const userContextKey contextKey = "user"

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	store *store.Store
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(s *store.Store) *AuthMiddleware {
	return &AuthMiddleware{store: s}
}

// RequireAuth 要求登录的中间件
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, `{"error":"未登录"}`, http.StatusUnauthorized)
			return
		}

		user := am.store.GetUserBySession(cookie.Value)
		if user == nil {
			http.Error(w, `{"error":"会话已过期"}`, http.StatusUnauthorized)
			return
		}

		// 将用户信息注入上下文
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser 从上下文中获取当前用户
func GetUser(r *http.Request) *model.User {
	user, _ := r.Context().Value(userContextKey).(*model.User)
	return user
}
