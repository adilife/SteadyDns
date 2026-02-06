// core/webapi/config_api.go

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"SteadyDNS/core/common"
)

// ConfigAPIHandler 处理配置管理API请求
func ConfigAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(configHandler)
	authHandler(w, r)
}

// configHandler 配置管理API处理函数
func configHandler(w http.ResponseWriter, r *http.Request) {
	// 获取配置管理器实例
	configManager := common.GetConfigManager()

	// 解析请求路径
	var path string
	if r.URL.Path == "/api/config" {
		path = ""
	} else {
		path = strings.TrimPrefix(r.URL.Path, "/api/config/")
	}
	pathParts := strings.Split(path, "/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "" && r.Method == http.MethodGet:
		// 获取所有配置
		handleGetAllConfig(w, r, configManager)

	case len(pathParts) == 1 && r.Method == http.MethodGet:
		// 获取指定节的配置
		section := pathParts[0]
		handleGetSectionConfig(w, r, configManager, section)

	case len(pathParts) == 2 && r.Method == http.MethodGet:
		// 获取指定配置项
		section := pathParts[0]
		key := pathParts[1]
		handleGetConfigItem(w, r, configManager, section, key)

	case len(pathParts) == 2 && r.Method == http.MethodPut:
		// 更新配置项
		section := pathParts[0]
		key := pathParts[1]
		handleUpdateConfigItem(w, r, configManager, section, key)

	case path == "reload" && r.Method == http.MethodPost:
		// 重载配置
		handleReloadConfig(w, r, configManager)

	case path == "defaults" && r.Method == http.MethodGet:
		// 获取默认配置
		handleGetDefaultConfig(w, r, configManager)

	case path == "reset" && r.Method == http.MethodPost:
		// 重置为默认配置
		handleResetConfig(w, r, configManager)

	case path == "env" && r.Method == http.MethodGet:
		// 获取环境变量
		handleGetEnvVars(w, r, configManager)

	case path == "env" && r.Method == http.MethodPost:
		// 设置环境变量
		handleSetEnvVar(w, r, configManager)

	case path == "summary" && r.Method == http.MethodGet:
		// 获取配置摘要
		handleGetConfigSummary(w, r, configManager)

	case path == "validate" && r.Method == http.MethodPost:
		// 验证配置
		handleValidateConfig(w, r, configManager)

	default:
		// 未找到的API端点
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// handleGetAllConfig 处理获取所有配置的请求
func handleGetAllConfig(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 获取所有配置
	config := common.GetAllConfig()

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    config,
	})
}

// handleGetSectionConfig 处理获取指定节的配置的请求
func handleGetSectionConfig(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager, section string) {
	// 获取指定节的配置
	config := common.GetSectionConfig(section)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"section": section,
		"data":    config,
	})
}

// handleGetConfigItem 处理获取指定配置项的请求
func handleGetConfigItem(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager, section, key string) {
	// 获取指定配置项
	value := common.GetConfig(section, key)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"section": section,
		"key":     key,
		"value":   value,
	})
}

// handleUpdateConfigItem 处理更新配置项的请求
func handleUpdateConfigItem(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager, section, key string) {
	// 解析请求体
	var request struct {
		Value string `json:"value"`
		User  string `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取当前值
	oldValue := common.GetConfig(section, key)

	// 更新配置
	if err := common.UpdateConfig(section, key, request.Value); err != nil {
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
		"success":   true,
		"message":   "配置更新成功",
		"section":   section,
		"key":       key,
		"old_value": oldValue,
		"new_value": request.Value,
	})
}

// handleReloadConfig 处理重载配置的请求
func handleReloadConfig(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 重载配置
	if err := common.ReloadConfig(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 直接返回成功响应，移除历史记录添加
	// 原因：reload 操作只是重新读取配置文件，没有实际修改配置，不需要记录历史
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置重载成功",
	})
}



// handleGetDefaultConfig 处理获取默认配置的请求
func handleGetDefaultConfig(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 重置为默认配置（但不保存）
	// 这里我们创建一个临时的默认配置
	// 注意：实际实现中，我们应该直接返回默认配置模板，而不是重置当前配置

	// 返回默认配置模板
	defaultConfig := map[string]interface{}{
		"message": "默认配置模板",
		"config":  common.DefaultConfigTemplate,
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    defaultConfig,
	})
}

// handleResetConfig 处理重置为默认配置的请求
func handleResetConfig(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 解析请求体
	var request struct {
		User string `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		request.User = "system"
	}

	// 重置为默认配置
	if err := common.ResetToDefaults(); err != nil {
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
		"message": "配置已重置为默认值",
	})
}

// handleGetEnvVars 处理获取环境变量的请求
func handleGetEnvVars(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 获取环境变量
	envVars := configManager.GetEnvVars()

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    envVars,
		"count":   len(envVars),
	})
}

// handleSetEnvVar 处理设置环境变量的请求
func handleSetEnvVar(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 解析请求体
	var request struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		User  string `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置环境变量
	if err := configManager.SetEnvVar(request.Key, request.Value); err != nil {
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
		"message": "环境变量设置成功",
		"key":     request.Key,
		"value":   request.Value,
	})
}

// handleGetConfigSummary 处理获取配置摘要的请求
func handleGetConfigSummary(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 获取配置摘要
	summary := configManager.GetConfigSummary()

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    summary,
	})
}

// handleValidateConfig 处理验证配置的请求
func handleValidateConfig(w http.ResponseWriter, r *http.Request, configManager *common.ConfigManager) {
	// 验证配置
	if err := configManager.ValidateConfigWithManager(); err != nil {
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
		"message": "配置验证通过",
	})
}
