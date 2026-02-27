/*
SteadyDNS - DNS服务器实现

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
// core/webapi/middleware/auth.go

package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddlewareGin 认证中间件（Gin版本）
// 验证请求中的JWT令牌，确保用户已登录
func AuthMiddlewareGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := GetTokenFromRequest(c.Request)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供访问令牌"})
			c.Abort()
			return
		}

		claims, err := GetJWTManager().GetUserFromToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// 将用户信息添加到请求上下文
		ctx := context.WithValue(c.Request.Context(), "user", claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
