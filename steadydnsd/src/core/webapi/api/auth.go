// core/webapi/auth.go
package api

import (
	"SteadyDNS/core/webapi/middleware"
	"context"
	"net/http"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := middleware.GetTokenFromRequest(r)
		if token == "" {
			middleware.SendErrorResponse(w, "未提供访问令牌", http.StatusUnauthorized)
			return
		}

		claims, err := jwtManager.GetUserFromToken(token)
		if err != nil {
			middleware.SendErrorResponse(w, "无效的访问令牌", http.StatusUnauthorized)
			return
		}

		// 将用户信息添加到请求上下文（如果需要）
		ctx := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
