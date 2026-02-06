// core/webapi/server_api.go

package api

import (
	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServerAPIHandler 处理服务器管理API请求
func ServerAPIHandler(c *gin.Context) {
	// 应用认证中间件
	authHandler := AuthMiddlewareGin(serverHandlerGin)
	authHandler(c)
}

// serverHandlerGin 服务器管理API处理函数（Gin版本）
func serverHandlerGin(c *gin.Context) {
	// 获取服务器管理器实例
	serverManager := GetServerManager()

	// 获取HTTP服务器实例
	var httpServer = GetHTTPServer()

	// 解析请求路径
	path := c.Request.URL.Path
	// 移除前缀，处理带斜杠和不带斜杠的情况
	path = strings.TrimPrefix(path, "/api/server")
	path = strings.TrimPrefix(path, "/")
	pathParts := strings.Split(path, "/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case (path == "" || path == "status") && c.Request.Method == http.MethodGet:
		// 获取服务器状态
		handleServerStatusGin(c, serverManager)

	case path == "restart" && c.Request.Method == http.MethodPost:
		// 重启HTTP服务器
		err := httpServer.Restart()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "HTTP server restarted successfully",
		})

	case strings.HasPrefix(path, "sdnsd/") && c.Request.Method == http.MethodPost:
		// DNS服务器操作
		action := pathParts[1]
		handleDNSServerActionGin(c, serverManager, action)

	case strings.HasPrefix(path, "httpd/") && c.Request.Method == http.MethodPost:
		// HTTP服务器操作
		action := pathParts[1]
		handleHTTPServerActionGin(c, httpServer, action)

	case path == "reload-forward-groups" && c.Request.Method == http.MethodPost:
		// 重载转发组
		handleReloadForwardGroupsGin(c, serverManager)

	case strings.HasPrefix(path, "logging/level") && c.Request.Method == http.MethodPost:
		// 设置日志级别
		handleSetLogLevelGin(c, serverManager)

	default:
		// 未找到的API端点
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	}
}

// handleServerStatusGin 处理获取服务器状态的请求（Gin版本）
func handleServerStatusGin(c *gin.Context, serverManager *ServerManager) {
	// 获取服务器状态
	status := serverManager.GetServerStatus()

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// handleDNSServerActionGin 处理DNS服务器操作的请求（Gin版本）
func handleDNSServerActionGin(c *gin.Context, serverManager *ServerManager, action string) {
	var err error

	// 根据操作类型执行相应的操作
	switch action {
	case "start":
		err = serverManager.StartDNSServer()
	case "stop":
		err = serverManager.StopDNSServer()
	case "restart":
		err = serverManager.RestartDNSServer()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	// 处理错误
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 获取操作的正确过去式形式
	var actionPastTense string
	switch action {
	case "start":
		actionPastTense = "started"
	case "stop":
		actionPastTense = "stopped"
	case "restart":
		actionPastTense = "restarted"
	default:
		actionPastTense = action + "ed"
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "DNS server " + actionPastTense + " successfully",
	})
}

// handleHTTPServerActionGin 处理HTTP服务器操作的请求（Gin版本）
func handleHTTPServerActionGin(c *gin.Context, httpServer *HTTPServer, action string) {
	var err error

	// 根据操作类型执行相应的操作
	switch action {
	case "restart":
		err = httpServer.Restart()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	// 处理错误
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "HTTP server restarted successfully",
	})
}

// handleReloadForwardGroupsGin 处理重载转发组的请求（Gin版本）
func handleReloadForwardGroupsGin(c *gin.Context, serverManager *ServerManager) {
	// 执行重载转发组操作
	err := serverManager.ReloadForwardGroups()

	// 处理错误
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Forward groups reloaded successfully",
	})
}

// handleSetLogLevelGin 处理设置日志级别的请求（Gin版本）
func handleSetLogLevelGin(c *gin.Context, serverManager *ServerManager) {
	// 解析请求体
	var request struct {
		APILogLevel string `json:"api_log_level"`
		DNSLogLevel string `json:"dns_log_level"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 验证日志级别
	validLevels := map[string]bool{
		"DEBUG":   true,
		"INFO":    true,
		"WARN":    true,
		"WARNING": true,
		"ERROR":   true,
		"FATAL":   true,
	}

	// 验证API日志级别
	if request.APILogLevel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API log level is required"})
		return
	}
	apiLogLevelUpper := strings.ToUpper(request.APILogLevel)
	if !validLevels[apiLogLevelUpper] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API log level"})
		return
	}

	// 验证DNS日志级别
	if request.DNSLogLevel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DNS log level is required"})
		return
	}
	dnsLogLevelUpper := strings.ToUpper(request.DNSLogLevel)
	if !validLevels[dnsLogLevelUpper] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DNS log level"})
		return
	}

	// 更新配置文件中的API日志级别
	err := common.UpdateConfig("API", "LOG_LEVEL", apiLogLevelUpper)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update API log level: " + err.Error(),
		})
		return
	}

	// 更新配置文件中的DNS日志级别
	err = common.UpdateConfig("Logging", "DNS_LOG_LEVEL", dnsLogLevelUpper)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update DNS log level: " + err.Error(),
		})
		return
	}

	// 重载配置
	err = common.ReloadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to reload config: " + err.Error(),
		})
		return
	}

	// 执行设置日志级别操作
	err = serverManager.SetLogLevel(apiLogLevelUpper)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to set API log level: " + err.Error(),
		})
		return
	}

	// 更新DNSForwarder的日志级别
	if sdns.GlobalDNSForwarder != nil {
		sdns.GlobalDNSForwarder.SetLogLevel(dnsLogLevelUpper)
	}

	// 更新数据库包的日志级别
	if database.GetLogManager() != nil {
		database.GetLogManager().SetLogLevel(dnsLogLevelUpper)
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Log levels set successfully",
		"levels": map[string]string{
			"api_log_level": request.APILogLevel,
			"dns_log_level": request.DNSLogLevel,
		},
	})
}
