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
// /webapi/setuproute.go

package api

import (
	"SteadyDNS/core/plugin"
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

	// 插件状态API路由 - 无需认证，应用日志、频率限制和超时中间件
	engine.GET("/api/plugins/status", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), PluginAPIHandler)

	// 令牌刷新API路由 - 应用频率限制、日志和超时中间件
	engine.POST("/api/refresh-token", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.TimeoutMiddlewareWithPathGin(), middleware.RefreshTokenHandler)

	// 登出API路由 - 需要认证，应用日志和超时中间件
	engine.POST("/api/logout", middleware.LoggerMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), LogoutHandler)

	// 转发组API路由 - 需要认证，应用所有中间件
	engine.GET("/api/forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.GET("/api/forward-groups/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.POST("/api/forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.PUT("/api/forward-groups/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.DELETE("/api/forward-groups/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.DELETE("/api/forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)
	engine.GET("/api/forward-groups/test-domain-match", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardGroupAPIHandler)

	// 服务器API路由 - 需要认证，应用所有中间件
	engine.GET("/api/forward-servers", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.GET("/api/forward-servers/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.POST("/api/forward-servers", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.PUT("/api/forward-servers/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)
	engine.DELETE("/api/forward-servers/:id", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ForwardServerAPIHandlerGin)

	// 缓存API路由 - 需要认证，应用所有中间件
	engine.GET("/api/cache/stats", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), CacheAPIHandlerGin)
	engine.POST("/api/cache/clear", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), CacheAPIHandlerGin)
	engine.POST("/api/cache/clear/:domain", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), CacheAPIHandlerGin)

	// 服务器管理API路由 - 需要认证，应用所有中间件
	engine.GET("/api/server/status", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/restart", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/sdnsd/:action", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/httpd/:action", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/reload-forward-groups", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)
	engine.POST("/api/server/logging/level", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ServerAPIHandler)

	// 配置管理API路由 - 需要认证，应用所有中间件
	engine.GET("/api/config", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.GET("/api/config/:section", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.GET("/api/config/:section/:key", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.PUT("/api/config", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.PUT("/api/config/:section/:key", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/reload", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/reset", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.GET("/api/config/env", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/env", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)
	engine.POST("/api/config/validate", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), ConfigAPIHandlerGin)

	// Dashboard API路由 - 需要认证，应用所有中间件
	engine.GET("/api/dashboard/*endpoint", middleware.LoggerMiddleware(), middleware.RateLimitMiddleware(), middleware.AuthMiddlewareGin(), middleware.TimeoutMiddlewareWithPathGin(), DashboardAPIHandlerGin)

	// BIND相关路由已移至插件系统，由SetupPluginRoutes函数动态注册
}

// SetupPluginRoutes 根据插件状态注册插件路由
// 遍历所有启用的插件，将其路由动态注册到Gin引擎
// 参数：
//   - engine: Gin引擎实例
func SetupPluginRoutes(engine *gin.Engine) {
	// 获取全局插件管理器实例
	pm := plugin.GetPluginManager()

	// 获取所有启用插件的路由
	routesMap := pm.GetAllEnabledRoutes()

	// 注册每个插件的路由
	for pluginName, routes := range routesMap {
		for _, route := range routes {
			registerPluginRoute(engine, pluginName, route)
		}
	}
}

// registerPluginRoute 注册单个插件路由
// 根据路由定义的配置，将路由注册到Gin引擎
// 参数：
//   - engine: Gin引擎实例
//   - pluginName: 插件名称，用于日志记录
//   - route: 路由定义，包含方法、路径、处理器等信息
func registerPluginRoute(engine *gin.Engine, pluginName string, route plugin.RouteDefinition) {
	// 构建中间件链
	handlers := make([]gin.HandlerFunc, 0)

	// 添加日志中间件
	handlers = append(handlers, middleware.LoggerMiddleware())

	// 添加频率限制中间件
	handlers = append(handlers, middleware.RateLimitMiddleware())

	// 如果路由需要认证，添加认证中间件
	if route.AuthRequired {
		handlers = append(handlers, middleware.AuthMiddlewareGin())
	}

	// 添加超时中间件
	handlers = append(handlers, middleware.TimeoutMiddlewareWithPathGin())

	// 添加实际的请求处理器（将http.HandlerFunc转换为gin.HandlerFunc）
	handlers = append(handlers, func(c *gin.Context) {
		route.Handler(c.Writer, c.Request)
	})

	// 根据HTTP方法注册路由
	switch route.Method {
	case "GET":
		engine.GET(route.Path, handlers...)
	case "POST":
		engine.POST(route.Path, handlers...)
	case "PUT":
		engine.PUT(route.Path, handlers...)
	case "DELETE":
		engine.DELETE(route.Path, handlers...)
	case "PATCH":
		engine.PATCH(route.Path, handlers...)
	default:
		// 对于不支持的HTTP方法，记录警告日志
		// 这里使用fmt.Printf，因为Logger需要更多上下文
		// 在实际生产环境中应该使用统一的日志系统
	}
}
