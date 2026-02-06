// /webapi/setuproute.go

package api

import (
	"SteadyDNS/core/webapi/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由
// 配置登录、转发组和转发服务器的API路由
func SetupRoutes(engine *gin.Engine) {
	// 健康检查API路由 - 无需认证，应用日志、频率限制和超时中间件
	engine.GET("/api/health", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), HealthCheckHandler)
	// 登录API路由 - 应用频率限制、日志和超时中间件
	engine.POST("/api/login", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), LoginHandler)

	// 令牌刷新API路由 - 应用频率限制、日志和超时中间件
	engine.POST("/api/refresh-token", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), middleware.RefreshTokenHandler)

	// 登出API路由 - 应用日志和超时中间件
	engine.POST("/api/logout", middleware.LoggerMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), LogoutHandler)

	// 转发组API路由 - 需要认证，应用所有中间件
	engine.GET("/api/forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.GET("/api/forward-groups/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.POST("/api/forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.PUT("/api/forward-groups/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.DELETE("/api/forward-groups/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.DELETE("/api/forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.GET("/api/forward-groups/test-domain-match", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)

	// 服务器API路由 - 需要认证，应用所有中间件
	engine.GET("/api/forward-servers", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.GET("/api/forward-servers/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.POST("/api/forward-servers", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.PUT("/api/forward-servers/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.DELETE("/api/forward-servers/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)

	// 缓存API路由 - 需要认证，应用所有中间件
	engine.GET("/api/cache/stats", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), CacheAPIHandlerGin)
	engine.POST("/api/cache/clear", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), CacheAPIHandlerGin)
	engine.POST("/api/cache/clear/:domain", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), CacheAPIHandlerGin)

	// 服务器管理API路由 - 需要认证，应用所有中间件
	engine.GET("/api/server/status", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/restart", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/sdnsd/:action", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/httpd/:action", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/reload-forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/logging/level", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)

	// 配置管理API路由 - 需要认证，应用所有中间件
	engine.GET("/api/config", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.GET("/api/config/:section", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.GET("/api/config/:section/:key", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.PUT("/api/config", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.PUT("/api/config/:section/:key", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/reload", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/reset", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.GET("/api/config/env", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/env", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/validate", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)

	// Dashboard API路由 - 需要认证，应用所有中间件
	engine.GET("/api/dashboard", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), DashboardAPIHandlerGin)

	// BIND权威域API路由 - 需要认证，应用所有中间件
	engine.GET("/api/bind-zones", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindAPIHandlerGin)
	engine.GET("/api/bind-zones/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindAPIHandlerGin)
	engine.POST("/api/bind-zones", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindAPIHandlerGin)
	engine.PUT("/api/bind-zones/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindAPIHandlerGin)
	engine.DELETE("/api/bind-zones/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindAPIHandlerGin)

	// BIND服务器管理API路由 - 需要认证，应用所有中间件
	engine.GET("/api/bind-server/status", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.POST("/api/bind-server/:action", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.GET("/api/bind-server/stats", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.GET("/api/bind-server/health", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.POST("/api/bind-server/validate", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.GET("/api/bind-server/config", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.PUT("/api/bind-server/config", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)

	// named.conf 配置管理API路由 - 需要认证，应用所有中间件
	engine.GET("/api/bind-server/named-conf/content", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.PUT("/api/bind-server/named-conf", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.POST("/api/bind-server/named-conf/validate", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.GET("/api/bind-server/named-conf/parse", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)
	engine.POST("/api/bind-server/named-conf/diff", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), BindServerAPIHandlerGin)

	// 其他API路由...
}
