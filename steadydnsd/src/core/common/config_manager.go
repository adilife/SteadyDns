// core/common/config_manager.go

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ConfigHistory 配置变更历史记录
type ConfigHistory struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	User      string                 `json:"user"`
	Action    string                 `json:"action"`
	Changes   map[string]interface{} `json:"changes"`
	Config    map[string]map[string]string `json:"config"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	logger        *Logger
	history       []ConfigHistory
	historyFile   string
	backupDir     string
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
			logger:      NewLogger(),
			history:     make([]ConfigHistory, 0),
			historyFile: getHistoryFilePath(),
			backupDir:   getBackupDirPath(),
			envVars:     make(map[string]string),
		}
		// 加载历史记录
		globalConfigManager.loadHistory()
		// 加载环境变量
		globalConfigManager.loadEnvVars()
		// 确保备份目录存在
		globalConfigManager.ensureBackupDir()
	})
	return globalConfigManager
}

// getHistoryFilePath 获取历史记录文件路径
func getHistoryFilePath() string {
	configDir := filepath.Join(getConfigDir(), "history")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		NewLogger().Error("创建历史记录目录失败: %v", err)
	}
	return filepath.Join(configDir, "config_history.json")
}

// getBackupDirPath 获取备份目录路径
func getBackupDirPath() string {
	backupDir := filepath.Join(getConfigDir(), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		NewLogger().Error("创建备份目录失败: %v", err)
	}
	return backupDir
}

// getConfigDir 获取配置目录路径
func getConfigDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "."
	}
	return filepath.Join(workingDir, "config")
}

// loadHistory 加载历史记录
func (cm *ConfigManager) loadHistory() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 检查历史记录文件是否存在
	if _, err := os.Stat(cm.historyFile); os.IsNotExist(err) {
		// 文件不存在，初始化空历史
		cm.history = make([]ConfigHistory, 0)
		return
	}
	
	// 读取历史记录文件
	content, err := os.ReadFile(cm.historyFile)
	if err != nil {
		cm.logger.Error("读取历史记录文件失败: %v", err)
		cm.history = make([]ConfigHistory, 0)
		return
	}
	
	// 解析历史记录
	if err := json.Unmarshal(content, &cm.history); err != nil {
		cm.logger.Error("解析历史记录失败: %v", err)
		cm.history = make([]ConfigHistory, 0)
		return
	}
	
	cm.logger.Info("加载历史记录成功，共 %d 条记录", len(cm.history))
}

// saveHistory 保存历史记录
func (cm *ConfigManager) saveHistory() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 限制历史记录数量（最多100条）
	if len(cm.history) > 100 {
		cm.history = cm.history[len(cm.history)-100:]
	}
	
	// 序列化历史记录
	content, err := json.MarshalIndent(cm.history, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化历史记录失败: %v", err)
	}
	
	// 写入历史记录文件
	if err := os.WriteFile(cm.historyFile, content, 0644); err != nil {
		return fmt.Errorf("写入历史记录文件失败: %v", err)
	}
	
	return nil
}

// AddHistory 添加历史记录
func (cm *ConfigManager) AddHistory(user, action string, changes map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 创建历史记录
	history := ConfigHistory{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		User:      user,
		Action:    action,
		Changes:   changes,
		Config:    GetAllConfig(),
	}
	
	// 添加到历史记录
	cm.history = append(cm.history, history)
	
	// 保存历史记录
	return cm.saveHistory()
}

// GetHistory 获取历史记录
func (cm *ConfigManager) GetHistory(limit int) []ConfigHistory {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// 复制历史记录
	historyCopy := make([]ConfigHistory, len(cm.history))
	copy(historyCopy, cm.history)
	
	// 按时间倒序排序
	sort.Slice(historyCopy, func(i, j int) bool {
		return historyCopy[i].Timestamp.After(historyCopy[j].Timestamp)
	})
	
	// 限制返回数量
	if limit > 0 && len(historyCopy) > limit {
		historyCopy = historyCopy[:limit]
	}
	
	return historyCopy
}

// RollbackToHistory 回滚到指定历史版本
func (cm *ConfigManager) RollbackToHistory(historyID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 查找指定的历史记录
	var targetHistory *ConfigHistory
	for i := range cm.history {
		if cm.history[i].ID == historyID {
			targetHistory = &cm.history[i]
			break
		}
	}
	
	if targetHistory == nil {
		return fmt.Errorf("未找到指定的历史记录: %s", historyID)
	}
	
	// 保存当前配置到历史记录
	currentChanges := map[string]interface{}{
		"rollback_to": historyID,
		"rollback_time": time.Now(),
	}
	cm.AddHistory("system", "rollback", currentChanges)
	
	// 恢复配置
	// 首先加载默认配置
	ResetToDefaults()
	
	// 然后应用历史配置
	for section, keyValues := range targetHistory.Config {
		for key, value := range keyValues {
			UpdateConfig(section, key, value)
		}
	}
	
	// 重载配置
	ReloadConfig()
	
	cm.logger.Info("配置回滚成功，回滚到历史记录: %s", historyID)
	return nil
}

// BackupConfig 备份配置
func (cm *ConfigManager) BackupConfig(comment string) (string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("config_backup_%s.json", timestamp)
	if comment != "" {
		// 清理注释中的特殊字符
		comment = strings.ReplaceAll(comment, " ", "_")
		comment = strings.ReplaceAll(comment, "/", "_")
		comment = strings.ReplaceAll(comment, "\\", "_")
		backupFileName = fmt.Sprintf("config_backup_%s_%s.json", timestamp, comment)
	}
	backupPath := filepath.Join(cm.backupDir, backupFileName)
	
	// 获取当前配置
	currentConfig := GetAllConfig()
	
	// 创建备份数据
	backupData := map[string]interface{}{
		"timestamp": time.Now(),
		"comment":   comment,
		"config":    currentConfig,
		"env_vars":  cm.envVars,
	}
	
	// 序列化备份数据
	content, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化备份数据失败: %v", err)
	}
	
	// 写入备份文件
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("写入备份文件失败: %v", err)
	}
	
	cm.logger.Info("配置备份成功: %s", backupFileName)
	return backupFileName, nil
}

// RestoreConfig 从备份恢复配置
func (cm *ConfigManager) RestoreConfig(backupFileName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 构建备份文件路径
	backupPath := filepath.Join(cm.backupDir, backupFileName)
	
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupFileName)
	}
	
	// 读取备份文件
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %v", err)
	}
	
	// 解析备份数据
	var backupData map[string]interface{}
	if err := json.Unmarshal(content, &backupData); err != nil {
		return fmt.Errorf("解析备份数据失败: %v", err)
	}
	
	// 提取配置数据
	configData, ok := backupData["config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("备份数据中缺少配置信息")
	}
	
	// 保存当前配置到历史记录
	currentChanges := map[string]interface{}{
		"restore_from": backupFileName,
		"restore_time": time.Now(),
	}
	cm.AddHistory("system", "restore", currentChanges)
	
	// 恢复配置
	// 首先加载默认配置
	ResetToDefaults()
	
	// 然后应用备份配置
	for section, keyValues := range configData {
		if sectionMap, ok := keyValues.(map[string]interface{}); ok {
			for key, value := range sectionMap {
				if valueStr, ok := value.(string); ok {
					UpdateConfig(section, key, valueStr)
				}
			}
		}
	}
	
	// 重载配置
	ReloadConfig()
	
	cm.logger.Info("配置恢复成功，从备份文件: %s", backupFileName)
	return nil
}

// ListBackups 列出所有备份
func (cm *ConfigManager) ListBackups() ([]string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// 读取备份目录
	files, err := os.ReadDir(cm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("读取备份目录失败: %v", err)
	}
	
	// 过滤备份文件
	var backups []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "config_backup_") && strings.HasSuffix(file.Name(), ".json") {
			backups = append(backups, file.Name())
		}
	}
	
	// 按时间倒序排序
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	
	return backups, nil
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
	
	// 添加到历史记录
	changes := map[string]interface{}{
		"env_var": key,
		"old_value": cm.envVars[key],
		"new_value": value,
	}
	cm.AddHistory("system", "set_env_var", changes)
	
	cm.logger.Info("设置环境变量成功: %s=%s", key, value)
	return nil
}

// UnsetEnvVar 取消设置环境变量
func (cm *ConfigManager) UnsetEnvVar(key string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 检查环境变量是否存在
	oldValue, exists := cm.envVars[key]
	if !exists {
		return fmt.Errorf("环境变量不存在: %s", key)
	}
	
	// 取消设置环境变量
	if err := os.Unsetenv(key); err != nil {
		return fmt.Errorf("取消设置环境变量失败: %v", err)
	}
	
	// 从内部缓存中删除
	delete(cm.envVars, key)
	
	// 添加到历史记录
	changes := map[string]interface{}{
		"env_var": key,
		"old_value": oldValue,
		"new_value": "",
	}
	cm.AddHistory("system", "unset_env_var", changes)
	
	cm.logger.Info("取消设置环境变量成功: %s", key)
	return nil
}

// ensureBackupDir 确保备份目录存在
func (cm *ConfigManager) ensureBackupDir() {
	if err := os.MkdirAll(cm.backupDir, 0755); err != nil {
		cm.logger.Error("创建备份目录失败: %v", err)
	}
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
	
	// 获取历史记录统计
	historyCount := len(cm.history)
	
	// 获取备份统计
	backups, _ := cm.ListBackups()
	backupCount := len(backups)
	
	// 获取环境变量统计
	envVarCount := len(cm.envVars)
	
	summary := map[string]interface{}{
		"config": map[string]interface{}{
			"sections": sectionCount,
			"keys":     keyCount,
		},
		"history": map[string]interface{}{
			"count": historyCount,
		},
		"backups": map[string]interface{}{
			"count": backupCount,
		},
		"env_vars": map[string]interface{}{
			"count": envVarCount,
		},
		"last_updated": time.Now(),
	}
	
	return summary
}
