// core/webapi/timeout.go
// 全局请求超时中间件

package webapi

import (
	"context"
	"net/http"
	"time"

	"SteadyDNS/core/common"
)

// TimeoutMiddleware 创建请求超时中间件
func TimeoutMiddleware(timeout time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 创建带超时的上下文
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// 创建一个通道来接收处理完成的信号
			done := make(chan struct{})

			// 在goroutine中处理请求
			go func() {
				// 使用带超时的上下文
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			// 等待处理完成或超时
			select {
			case <-done:
				// 处理完成，正常返回
			case <-ctx.Done():
				// 超时，返回504错误
				logger := common.NewLogger()
				logger.Warn("API请求超时: %s %s", r.Method, r.URL.Path)
				
				// 检查响应是否已经开始写入
				// 注意：如果响应已经开始写入，这里可能会导致错误
				// 但在超时情况下，我们需要返回错误信息
				sendErrorResponse(w, "请求处理超时", http.StatusGatewayTimeout)
				return
			}
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
	default:
		// 默认超时时间
		return 10 * time.Second
	}
}

// TimeoutMiddlewareWithPath 根据路径设置不同超时时间的中间件
func TimeoutMiddlewareWithPath(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取路径
		path := r.URL.Path
		
		// 根据路径获取超时时间
		timeout := GetTimeoutByPath(path)
		
		// 使用对应超时时间的中间件
		timeoutHandler := TimeoutMiddleware(timeout)
		timeoutHandler(next).ServeHTTP(w, r)
	}
}
