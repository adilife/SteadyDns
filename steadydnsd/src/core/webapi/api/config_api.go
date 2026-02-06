// core/webapi/config_api.go

package api

import (
	"net/http"
	"strings"

	"SteadyDNS/core/common"

	"github.com/gin-gonic/gin"
)



// ConfigAPIHandlerGin 处理配置管理API请求（Gin版本）
func ConfigAPIHandlerGin(c *gin.Context) {
	// 应用认证中间件
	authHandler := AuthMiddlewareGin(configHandlerGin)
	authHandler(c)
}



// configHandlerGin 配置管理API处理函数（Gin版本）
func configHandlerGin(c *gin.Context) {
	// 获取配置管理器实例
	configManager := common.GetConfigManager()

	// 解析请求路径
	var path string
	if c.Request.URL.Path == "/api/config" {
		path = ""
	} else {
		path = strings.TrimPrefix(c.Request.URL.Path, "/api/config/")
	}
	pathParts := strings.Split(path, "/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "" && c.Request.Method == http.MethodGet:
		// 获取所有配置
		handleGetAllConfigGin(c, configManager)

	case len(pathParts) == 1 && c.Request.Method == http.MethodGet:
		// 获取指定节的配置
		section := pathParts[0]
		handleGetSectionConfigGin(c, configManager, section)

	case len(pathParts) == 2 && c.Request.Method == http.MethodGet:
		// 获取指定配置项
		section := pathParts[0]
		key := pathParts[1]
		handleGetConfigItemGin(c, configManager, section, key)

	case len(pathParts) == 2 && c.Request.Method == http.MethodPut:
		// 更新配置项
		section := pathParts[0]
		key := pathParts[1]
		handleUpdateConfigItemGin(c, configManager, section, key)

	case path == "reload" && c.Request.Method == http.MethodPost:
		// 重载配置
		handleReloadConfigGin(c, configManager)

	case path == "defaults" && c.Request.Method == http.MethodGet:
		// 获取默认配置
		handleGetDefaultConfigGin(c, configManager)

	case path == "reset" && c.Request.Method == http.MethodPost:
		// 重置为默认配置
		handleResetConfigGin(c, configManager)

	case path == "env" && c.Request.Method == http.MethodGet:
		// 获取环境变量
		handleGetEnvVarsGin(c, configManager)

	case path == "env" && c.Request.Method == http.MethodPost:
		// 设置环境变量
		handleSetEnvVarGin(c, configManager)

	case path == "summary" && c.Request.Method == http.MethodGet:
		// 获取配置摘要
		handleGetConfigSummaryGin(c, configManager)

	case path == "validate" && c.Request.Method == http.MethodPost:
		// 验证配置
		handleValidateConfigGin(c, configManager)

	default:
		// 未找到的API端点
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	}
}



// handleGetAllConfigGin 处理获取所有配置的请求（Gin版本）
func handleGetAllConfigGin(c *gin.Context, configManager *common.ConfigManager) {
	// 获取所有配置
	config := common.GetAllConfig()

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}



// handleGetSectionConfigGin 处理获取指定节的配置的请求（Gin版本）
func handleGetSectionConfigGin(c *gin.Context, configManager *common.ConfigManager, section string) {
	// 获取指定节的配置
	config := common.GetSectionConfig(section)

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"section": section,
		"data":    config,
	})
}



// handleGetConfigItemGin 处理获取指定配置项的请求（Gin版本）
func handleGetConfigItemGin(c *gin.Context, configManager *common.ConfigManager, section, key string) {
	// 获取指定配置项
	value := common.GetConfig(section, key)

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"section": section,
		"key":     key,
		"value":   value,
	})
}



// handleUpdateConfigItemGin 处理更新配置项的请求（Gin版本）
func handleUpdateConfigItemGin(c *gin.Context, configManager *common.ConfigManager, section, key string) {
	// 解析请求体
	var request struct {
		Value string `json:"value"`
		User  string `json:"user"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 获取当前值
	oldValue := common.GetConfig(section, key)

	// 更新配置
	if err := common.UpdateConfig(section, key, request.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "配置更新成功",
		"section":   section,
		"key":       key,
		"old_value": oldValue,
		"new_value": request.Value,
	})
}



// handleReloadConfigGin 处理重载配置的请求（Gin版本）
func handleReloadConfigGin(c *gin.Context, configManager *common.ConfigManager) {
	// 重载配置
	if err := common.ReloadConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 直接返回成功响应，移除历史记录添加
	// 原因：reload 操作只是重新读取配置文件，没有实际修改配置，不需要记录历史
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置重载成功",
	})
}





// handleGetDefaultConfigGin 处理获取默认配置的请求（Gin版本）
func handleGetDefaultConfigGin(c *gin.Context, configManager *common.ConfigManager) {
	// 重置为默认配置（但不保存）
	// 这里我们创建一个临时的默认配置
	// 注意：实际实现中，我们应该直接返回默认配置模板，而不是重置当前配置

	// 返回默认配置模板
	defaultConfig := map[string]interface{}{
		"message": "默认配置模板",
		"config":  common.DefaultConfigTemplate,
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    defaultConfig,
	})
}



// handleResetConfigGin 处理重置为默认配置的请求（Gin版本）
func handleResetConfigGin(c *gin.Context, configManager *common.ConfigManager) {
	// 解析请求体
	var request struct {
		User string `json:"user"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		request.User = "system"
	}

	// 重置为默认配置
	if err := common.ResetToDefaults(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已重置为默认值",
	})
}



// handleGetEnvVarsGin 处理获取环境变量的请求（Gin版本）
func handleGetEnvVarsGin(c *gin.Context, configManager *common.ConfigManager) {
	// 获取环境变量
	envVars := configManager.GetEnvVars()

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    envVars,
		"count":   len(envVars),
	})
}



// handleSetEnvVarGin 处理设置环境变量的请求（Gin版本）
func handleSetEnvVarGin(c *gin.Context, configManager *common.ConfigManager) {
	// 解析请求体
	var request struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		User  string `json:"user"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 设置环境变量
	if err := configManager.SetEnvVar(request.Key, request.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "环境变量设置成功",
		"key":     request.Key,
		"value":   request.Value,
	})
}



// handleGetConfigSummaryGin 处理获取配置摘要的请求（Gin版本）
func handleGetConfigSummaryGin(c *gin.Context, configManager *common.ConfigManager) {
	// 获取配置摘要
	summary := configManager.GetConfigSummary()

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}



// handleValidateConfigGin 处理验证配置的请求（Gin版本）
func handleValidateConfigGin(c *gin.Context, configManager *common.ConfigManager) {
	// 验证配置
	if err := configManager.ValidateConfigWithManager(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置验证通过",
	})
}
