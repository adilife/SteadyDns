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
// core/webapi/middleWare/timeout.go
// 全局请求超时中间件

package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"SteadyDNS/core/common"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddlewareGin 创建请求超时中间件
func TimeoutMiddlewareGin(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 创建一个通道来接收处理完成的信号
		done := make(chan struct{})

		// 在goroutine中处理请求
		go func() {
			// 使用带超时的上下文
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			close(done)
		}()

		// 等待处理完成或超时
		select {
		case <-done:
			// 处理完成，正常返回
		case <-ctx.Done():
			// 超时，返回504错误
			logger := common.NewLogger()
			logger.Warn("API请求超时: %s %s", c.Request.Method, c.Request.URL.Path)

			// 检查响应是否已经开始写入
			// 注意：如果响应已经开始写入，这里可能会导致错误
			// 但在超时情况下，我们需要返回错误信息
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "请求处理超时"})
			c.Abort()
			return
		}
	}
}

// GetTimeoutByPath 根据请求路径获取对应的超时时间
func GetTimeoutByPath(path string) time.Duration {
	// 为不同的API端点设置不同的超时时间
	switch {
	case path == "/api/login" || path == "/api/refresh-token" || path == "/api/logout":
		// 认证相关API，超时时间较短
		return 10 * time.Second
	case path == "/api/dashboard/summary":
		// 仪表盘摘要，可能涉及多个统计计算
		return 15 * time.Second
	case path == "/api/dashboard/trends":
		// 趋势数据，可能涉及大量历史数据计算
		return 20 * time.Second
	case path == "/api/dashboard/top":
		// 排行榜数据，涉及排序计算
		return 10 * time.Second
	case path == "/api/forward-groups" || path == "/api/forward-servers":
		// 转发组和服务器管理，涉及数据库操作
		return 10 * time.Second
	case strings.HasPrefix(path, "/api/server/"):
		// 服务器管理API，特别是重启操作需要较长时间
		return 30 * time.Second
	default:
		// 默认超时时间
		return 10 * time.Second
	}
}

// TimeoutMiddlewareWithPathGin 根据路径设置不同超时时间的中间件
func TimeoutMiddlewareWithPathGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取路径
		path := c.Request.URL.Path

		// 根据路径获取超时时间
		timeout := GetTimeoutByPath(path)

		// 使用对应超时时间的Gin中间件
		timeoutHandler := TimeoutMiddlewareGin(timeout)
		timeoutHandler(c)
	}
}
