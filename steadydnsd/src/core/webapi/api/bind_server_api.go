// core/webapi/bind_server_api.go

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"SteadyDNS/core/bind"
)

// BindServerAPIHandler 处理BIND服务器管理API请求
func BindServerAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(bindServerHandler)
	authHandler(w, r)
}

// bindServerHandler BIND服务器管理API处理函数
func bindServerHandler(w http.ResponseWriter, r *http.Request) {
	// 获取BIND管理器实例
	bindManager := bind.NewBindManager()

	// 解析请求路径
	path := strings.TrimPrefix(r.URL.Path, "/api/bind-server/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "status" && r.Method == http.MethodGet:
		// 获取BIND服务器状态
		handleBindServerStatus(w, r, bindManager)

	case path == "start" && r.Method == http.MethodPost:
		// 启动BIND服务器
		handleBindServerAction(w, r, bindManager, "start")

	case path == "stop" && r.Method == http.MethodPost:
		// 停止BIND服务器
		handleBindServerAction(w, r, bindManager, "stop")

	case path == "restart" && r.Method == http.MethodPost:
		// 重启BIND服务器
		handleBindServerAction(w, r, bindManager, "restart")

	case path == "reload" && r.Method == http.MethodPost:
		// 重载BIND服务器
		handleBindServerAction(w, r, bindManager, "reload")

	case path == "stats" && r.Method == http.MethodGet:
		// 获取BIND服务器统计信息
		handleBindServerStats(w, r, bindManager)

	case path == "health" && r.Method == http.MethodGet:
		// 检查BIND服务健康状态
		handleBindServerHealth(w, r, bindManager)

	case path == "validate" && r.Method == http.MethodPost:
		// 验证BIND配置
		handleBindServerValidate(w, r, bindManager)

	case path == "config" && r.Method == http.MethodGet:
		// 获取BIND配置
		handleBindServerConfig(w, r, bindManager)

	case path == "config" && r.Method == http.MethodPut:
		// 更新BIND配置
		handleBindServerUpdateConfig(w, r, bindManager)

	default:
		// 未找到的API端点
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// handleBindServerStatus 处理获取BIND服务器状态的请求
func handleBindServerStatus(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager) {
	// 获取BIND服务器状态
	status, err := bindManager.GetBindStatus()

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"status": status,
		},
	})
}

// handleBindServerAction 处理BIND服务器操作的请求
func handleBindServerAction(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager, action string) {
	var err error

	// 根据操作类型执行相应的操作
	switch action {
	case "start":
		err = bindManager.StartBind()
	case "stop":
		err = bindManager.StopBind()
	case "restart":
		err = bindManager.RestartBind()
	case "reload":
		err = bindManager.ReloadBind()
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
		"message": "BIND server " + action + "ed successfully",
	})
}

// handleBindServerStats 处理获取BIND服务器统计信息的请求
func handleBindServerStats(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager) {
	// 获取BIND服务器统计信息
	stats, err := bindManager.GetBindStats()

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// handleBindServerHealth 处理检查BIND服务健康状态的请求
func handleBindServerHealth(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager) {
	// 检查BIND服务健康状态
	health, err := bindManager.CheckBindHealth()

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    health,
	})
}

// handleBindServerValidate 处理验证BIND配置的请求
func handleBindServerValidate(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager) {
	// 验证BIND配置
	err := bindManager.ValidateConfig()

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
		"message": "BIND configuration validated successfully",
	})
}

// handleBindServerConfig 处理获取BIND配置的请求
func handleBindServerConfig(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager) {
	// 获取BIND配置
	config, err := bindManager.GetBindConfig()

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    config,
	})
}

// handleBindServerUpdateConfig 处理更新BIND配置的请求
func handleBindServerUpdateConfig(w http.ResponseWriter, r *http.Request, bindManager *bind.BindManager) {
	// 解析请求体
	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 执行更新BIND配置操作
	err := bindManager.UpdateBindConfig(request)

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
		"message": "BIND configuration updated successfully",
	})
}
