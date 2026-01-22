// cmd/main.go

package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
	"SteadyDNS/core/webapi"
)

func main() {
	// 加载环境变量
	common.LoadEnv()

	// 检查数据库文件是否存在
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "steadydns.db"
	}

	// 检查数据库文件是否存在
	dbExists := checkDBFileExists(dbPath)

	// 初始化数据库
	database.InitDB()

	// 创建日志记录器
	logger := common.NewLogger()

	// 根据数据库文件是否存在执行相应操作
	if !dbExists {
		logger.Warn("数据库文件不存在，开始初始化...")
		// 执行初始化操作
		if err := database.InitializeDatabase(); err != nil {
			log.Fatalf("数据库初始化失败: %v", err)
		}
		logger.Warn("数据库初始化完成")
	} else {
		logger.Info("数据库文件已存在，使用现有数据库")
	}

	// 启动DNS服务器
	logger.Info("启动DNS服务器...")
	if err := sdns.StartDNSServer(logger); err != nil {
		log.Fatalf("DNS服务器启动失败: %v", err)
	} else {
		// 检查DNS服务器是否正在运行
		if sdns.IsDNSServerRunning() {
			logger.Info("DNS服务器启动成功")
		} else {
			logger.Warn("DNS服务器已启动，但状态检查显示未运行，可能存在启动问题")
		}
	}

	// 设置路由
	setupRoutes()

	// 获取端口，优先从配置文件读取，其次是环境变量，最后使用默认值8080
	port := common.GetConfig("Server", "PORT")
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
	}

	logger.Info("API服务器启动...")
	logger.Info("API服务器启动成功，监听端口: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Error("API服务器启动失败: %v", err)
	}
}

// checkDBFileExists 检查数据库文件是否存在
// 参数:
//
//	dbPath: 数据库文件路径
//
// 返回值:
//
//	bool: 数据库文件是否存在
func checkDBFileExists(dbPath string) bool {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		common.NewLogger().Error("获取数据库路径失败: %v", err)
		absPath = dbPath
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		common.NewLogger().Error("数据库文件 %s 不存在", absPath)
		return false
	}

	common.NewLogger().Info("数据库文件 %s 存在", absPath)
	return true
}

// setupRoutes 设置API路由
// 配置登录、转发组和转发服务器的API路由
func setupRoutes() {
	// 健康检查API路由 - 无需认证，应用日志、频率限制和超时中间件
	http.HandleFunc("/api/health", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.HealthCheckHandler))))
	// 登录API路由 - 应用频率限制、日志和超时中间件
	http.HandleFunc("/api/login", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.LoginHandler))))

	// 令牌刷新API路由 - 应用频率限制、日志和超时中间件
	http.HandleFunc("/api/refresh-token", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.RefreshTokenHandler))))

	// 登出API路由 - 应用日志和超时中间件
	http.HandleFunc("/api/logout", webapi.LoggerMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.LogoutHandler)))

	// 转发组API路由 - 需要认证，应用所有中间件
	http.HandleFunc("/api/forward-groups/", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.ForwardGroupAPIHandler))))
	http.HandleFunc("/api/forward-groups", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.ForwardGroupAPIHandler))))

	// 服务器API路由 - 需要认证，应用所有中间件
	http.HandleFunc("/api/forward-servers/", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.ForwardServerAPIHandler))))
	http.HandleFunc("/api/forward-servers", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.ForwardServerAPIHandler))))

	// 缓存API路由 - 需要认证，应用所有中间件
	http.HandleFunc("/api/cache/", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.CacheAPIHandler))))
	http.HandleFunc("/api/cache", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.CacheAPIHandler))))

	// Dashboard API路由 - 需要认证，应用所有中间件
	http.HandleFunc("/api/dashboard/", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.DashboardAPIHandler))))
	http.HandleFunc("/api/dashboard", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.DashboardAPIHandler))))

	// BIND权威域API路由 - 需要认证，应用所有中间件
	http.HandleFunc("/api/bind-zones/", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.BindAPIHandler))))
	http.HandleFunc("/api/bind-zones", webapi.LoggerMiddleware(webapi.RateLimitMiddleware(webapi.TimeoutMiddlewareWithPath(webapi.BindAPIHandler))))

	// 其他API路由...
}
