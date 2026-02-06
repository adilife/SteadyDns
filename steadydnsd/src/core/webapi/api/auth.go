// core/webapi/auth.go
package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddlewareGin 认证中间件（Gin版本）
func AuthMiddlewareGin(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := GetTokenFromRequest(c.Request)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供访问令牌"})
			c.Abort()
			return
		}

		claims, err := jwtManager.GetUserFromToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// 将用户信息添加到请求上下文（如果需要）
		ctx := context.WithValue(c.Request.Context(), "user", claims)
		c.Request = c.Request.WithContext(ctx)
		next(c)
	}
}

// GetTokenFromRequest 从请求中获取令牌
func GetTokenFromRequest(r *http.Request) string {
	// 从Authorization头获取令牌
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// 检查Authorization头格式
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
	}

	// 从查询参数获取令牌
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}
