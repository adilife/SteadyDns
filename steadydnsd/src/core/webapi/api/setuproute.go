// /webapi/setuproute.go

package api

import (
	"SteadyDNS/core/webapi/middleware"
	"net/http"
)

// SetupRoutes 设置API路由
// 配置登录、转发组和转发服务器的API路由
func SetupRoutes(mux *http.ServeMux) {
	// 健康检查API路由 - 无需认证，应用日志、频率限制和超时中间件
	mux.HandleFunc("/api/health", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(HealthCheckHandler))))
	// 登录API路由 - 应用频率限制、日志和超时中间件
	mux.HandleFunc("/api/login", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(LoginHandler))))

	// 令牌刷新API路由 - 应用频率限制、日志和超时中间件
	mux.HandleFunc("/api/refresh-token", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(middleware.RefreshTokenHandler))))

	// 登出API路由 - 应用日志和超时中间件
	mux.HandleFunc("/api/logout", middleware.LoggerMiddleware(middleware.TimeoutMiddlewareWithPath(LogoutHandler)))

	// 转发组API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/forward-groups/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ForwardGroupAPIHandler))))
	mux.HandleFunc("/api/forward-groups", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ForwardGroupAPIHandler))))

	// 服务器API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/forward-servers/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ForwardServerAPIHandler))))
	mux.HandleFunc("/api/forward-servers", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ForwardServerAPIHandler))))

	// 缓存API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/cache/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(CacheAPIHandler))))
	mux.HandleFunc("/api/cache", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(CacheAPIHandler))))

	// 服务器管理API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/server/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ServerAPIHandler))))
	mux.HandleFunc("/api/server", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ServerAPIHandler))))

	// 配置管理API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/config/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ConfigAPIHandler))))
	mux.HandleFunc("/api/config", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(ConfigAPIHandler))))

	// Dashboard API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/dashboard/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(DashboardAPIHandler))))
	mux.HandleFunc("/api/dashboard", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(DashboardAPIHandler))))

	// BIND权威域API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/bind-zones/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(BindAPIHandler))))
	mux.HandleFunc("/api/bind-zones", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(BindAPIHandler))))

	// BIND服务器管理API路由 - 需要认证，应用所有中间件
	mux.HandleFunc("/api/bind-server/", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(BindServerAPIHandler))))
	mux.HandleFunc("/api/bind-server", middleware.LoggerMiddleware(middleware.RateLimitMiddleware(middleware.TimeoutMiddlewareWithPath(BindServerAPIHandler))))

	// 其他API路由...
}
