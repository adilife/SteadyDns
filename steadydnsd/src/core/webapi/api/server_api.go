// core/webapi/server_api.go

package api

import (
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
		Level string `json:"level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证日志级别
	validLevels := map[string]bool{
		"debug":   true,
		"info":    true,
		"warn":    true,
		"warning": true,
		"error":   true,
		"fatal":   true,
	}

	if !validLevels[request.Level] {
		http.Error(w, "Invalid log level", http.StatusBadRequest)
		return
	}

	// 执行设置日志级别操作
	err := serverManager.SetLogLevel(request.Level)

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
		"message": "Log level set successfully",
		"level":   request.Level,
	})
}
