// core/webapi/server_api.go

package api

import (
	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
	"encoding/json"
	"net/http"
	"strings"
)

// ServerAPIHandler 处理服务器管理API请求
func ServerAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(serverHandler)
	authHandler(w, r)
}

// serverHandler 服务器管理API处理函数
func serverHandler(w http.ResponseWriter, r *http.Request) {
	// 获取服务器管理器实例
	serverManager := GetServerManager()

	// 获取HTTP服务器实例
	var httpServer = GetHTTPServer()

	// 解析请求路径
	path := strings.TrimPrefix(r.URL.Path, "/api/server/")
	pathParts := strings.Split(path, "/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "status" && r.Method == http.MethodGet:
		// 获取服务器状态
		handleServerStatus(w, r, serverManager)

	case strings.HasPrefix(path, "sdnsd/") && r.Method == http.MethodPost:
		// DNS服务器操作
		action := pathParts[1]
		handleDNSServerAction(w, r, serverManager, action)

	case strings.HasPrefix(path, "httpd/") && r.Method == http.MethodPost:
		// HTTP服务器操作
		action := pathParts[1]
		handleHTTPServerAction(w, r, httpServer, action)

	case path == "reload-forward-groups" && r.Method == http.MethodPost:
		// 重载转发组
		handleReloadForwardGroups(w, r, serverManager)

	case strings.HasPrefix(path, "logging/level") && r.Method == http.MethodPost:
		// 设置日志级别
		handleSetLogLevel(w, r, serverManager)

	default:
		// 未找到的API端点
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// handleServerStatus 处理获取服务器状态的请求
func handleServerStatus(w http.ResponseWriter, r *http.Request, serverManager *ServerManager) {
	// 获取服务器状态
	status := serverManager.GetServerStatus()

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// handleDNSServerAction 处理DNS服务器操作的请求
func handleDNSServerAction(w http.ResponseWriter, r *http.Request, serverManager *ServerManager, action string) {
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
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "DNS server " + actionPastTense + " successfully",
	})
}

// handleHTTPServerAction 处理HTTP服务器操作的请求
func handleHTTPServerAction(w http.ResponseWriter, r *http.Request, httpServer *HTTPServer, action string) {
	var err error

	// 根据操作类型执行相应的操作
	switch action {
	case "restart":
		err = httpServer.Restart()
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "HTTP server restarted successfully",
	})
}

// handleReloadForwardGroups 处理重载转发组的请求
func handleReloadForwardGroups(w http.ResponseWriter, r *http.Request, serverManager *ServerManager) {
	// 执行重载转发组操作
	err := serverManager.ReloadForwardGroups()

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Forward groups reloaded successfully",
	})
}

// handleSetLogLevel 处理设置日志级别的请求
func handleSetLogLevel(w http.ResponseWriter, r *http.Request, serverManager *ServerManager) {
	// 解析请求体
	var request struct {
		APILogLevel string `json:"api_log_level"`
		DNSLogLevel string `json:"dns_log_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		http.Error(w, "API log level is required", http.StatusBadRequest)
		return
	}
	apiLogLevelUpper := strings.ToUpper(request.APILogLevel)
	if !validLevels[apiLogLevelUpper] {
		http.Error(w, "Invalid API log level", http.StatusBadRequest)
		return
	}

	// 验证DNS日志级别
	if request.DNSLogLevel == "" {
		http.Error(w, "DNS log level is required", http.StatusBadRequest)
		return
	}
	dnsLogLevelUpper := strings.ToUpper(request.DNSLogLevel)
	if !validLevels[dnsLogLevelUpper] {
		http.Error(w, "Invalid DNS log level", http.StatusBadRequest)
		return
	}

	// 更新配置文件中的API日志级别
	err := common.UpdateConfig("API", "LOG_LEVEL", apiLogLevelUpper)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to update API log level: " + err.Error(),
		})
		return
	}

	// 更新配置文件中的DNS日志级别
	err = common.UpdateConfig("Logging", "DNS_LOG_LEVEL", dnsLogLevelUpper)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to update DNS log level: " + err.Error(),
		})
		return
	}

	// 重载配置
	err = common.ReloadConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to reload config: " + err.Error(),
		})
		return
	}

	// 执行设置日志级别操作
	err = serverManager.SetLogLevel(apiLogLevelUpper)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Log levels set successfully",
		"levels": map[string]string{
			"api_log_level": request.APILogLevel,
			"dns_log_level": request.DNSLogLevel,
		},
	})
}
