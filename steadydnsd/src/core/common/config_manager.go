// core/common/config_manager.go

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	logger        *Logger
	envVars       map[string]string
	mu            sync.RWMutex
}

// 全局配置管理器实例
var globalConfigManager *ConfigManager
var configManagerOnce sync.Once

// GetConfigManager 获取配置管理器实例（单例模式）
func GetConfigManager() *ConfigManager {
	configManagerOnce.Do(func() {
		globalConfigManager = &ConfigManager{
			logger:  NewLogger(),
			envVars: make(map[string]string),
		}
		// 加载环境变量
		globalConfigManager.loadEnvVars()
	})
	return globalConfigManager
}





// loadEnvVars 加载环境变量
func (cm *ConfigManager) loadEnvVars() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 读取所有环境变量
	envVars := os.Environ()
	for _, envVar := range envVars {
		if idx := strings.Index(envVar, "="); idx != -1 {
			key := envVar[:idx]
			value := envVar[idx+1:]
			cm.envVars[key] = value
		}
	}

	cm.logger.Info("加载环境变量成功，共 %d 个环境变量", len(cm.envVars))
}

// GetEnvVars 获取环境变量
func (cm *ConfigManager) GetEnvVars() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 创建环境变量副本
	envVarsCopy := make(map[string]string)
	for key, value := range cm.envVars {
		envVarsCopy[key] = value
	}

	return envVarsCopy
}

// SetEnvVar 设置环境变量
func (cm *ConfigManager) SetEnvVar(key, value string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 设置环境变量
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("设置环境变量失败: %v", err)
	}
	
	// 更新内部缓存
	cm.envVars[key] = value
	
	cm.logger.Info("设置环境变量成功: %s=%s", key, value)
	return nil
}

// UnsetEnvVar 取消设置环境变量
func (cm *ConfigManager) UnsetEnvVar(key string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 检查环境变量是否存在
	_, exists := cm.envVars[key]
	if !exists {
		return fmt.Errorf("环境变量不存在: %s", key)
	}
	
	// 取消设置环境变量
	if err := os.Unsetenv(key); err != nil {
		return fmt.Errorf("取消设置环境变量失败: %v", err)
	}
	
	// 从内部缓存中删除
	delete(cm.envVars, key)
	
	cm.logger.Info("取消设置环境变量成功: %s", key)
	return nil
}



// ValidateConfigWithManager 使用配置管理器验证配置
func (cm *ConfigManager) ValidateConfigWithManager() error {
	// 首先调用基础验证
	if err := ValidateConfig(); err != nil {
		return err
	}

	// 执行更详细的验证
	// 1. 验证路径存在性
	pathsToCheck := []struct {
		section string
		key     string
	}{
		{"Logging", "QUERY_LOG_PATH"},
		{"Database", "DB_PATH"},
	}

	for _, pathConfig := range pathsToCheck {
		path := GetConfig(pathConfig.section, pathConfig.key)
		if path != "" {
			// 确保路径存在或可以创建
			if !filepath.IsAbs(path) {
				workingDir, _ := os.Getwd()
				path = filepath.Join(workingDir, path)
			}

			// 确保目录存在
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("无法创建目录: %s, 错误: %v", dir, err)
			}
		}
	}

	// 2. 验证端口号范围
	port := GetConfig("HTTP", "PORT")
	if port != "" {
		portInt := 0
		if _, err := fmt.Sscanf(port, "%d", &portInt); err == nil {
			if portInt < 1 || portInt > 65535 {
				return fmt.Errorf("HTTP端口号超出范围: %d, 有效范围: 1-65535", portInt)
			}
		}
	}

	// 3. 验证JWT密钥强度
	jwtSecret := GetConfig("JWT", "JWT_SECRET_KEY")
	if len(jwtSecret) < 8 {
		cm.logger.Warn("JWT密钥强度较弱，建议至少8个字符")
	}

	cm.logger.Info("配置验证通过")
	return nil
}

// GetConfigSummary 获取配置摘要
func (cm *ConfigManager) GetConfigSummary() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// 获取当前配置
	currentConfig := GetAllConfig()
	
	// 统计配置信息
	sectionCount := len(currentConfig)
	keyCount := 0
	for _, keyValues := range currentConfig {
		keyCount += len(keyValues)
	}
	
	// 获取环境变量统计
	envVarCount := len(cm.envVars)
	
	summary := map[string]interface{}{
		"config": map[string]interface{}{
			"sections": sectionCount,
			"keys":     keyCount,
		},
		"env_vars": map[string]interface{}{
			"count": envVarCount,
		},
		"last_updated": time.Now(),
	}
	
	return summary
}
