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

// core/common/config.go
package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 默认配置模板
const DefaultConfigTemplate = `# SteadyDNS Configuration File
# Format: INI/Conf
[Database]
# Database file path (relative to working directory)
# Default: steadydns.db
DB_PATH=steadydns.db

[APIServer]
# API Server port
# Default: 8080, Recommended: 8080-8090
API_SERVER_PORT=8080
# API Server IPv4 address
# Default: 0.0.0.0 (listen on all IPv4 addresses)
# Recommended: 127.0.0.1 (localhost only) for production
API_SERVER_IP_ADDR=0.0.0.0
# API Server IPv6 address
# Default: :: (listen on all IPv6 addresses)
# Recommended: ::1 (localhost only) for production
API_SERVER_IPV6_ADDR=::
# GIN running mode (debug/release)
# Default: debug, Recommended: release (production)
GIN_MODE=release

[JWT]
# JWT secret key for authentication
# Default: your-default-jwt-secret-key-change-this-in-production
# Recommended: Use a strong, unique secret key in production
JWT_SECRET_KEY=your-default-jwt-secret-key-change-this-in-production
# Access token expiration (minutes)
# Default: 30, Recommended: 15-60
ACCESS_TOKEN_EXPIRATION=30
# Refresh token expiration (days)
# Default: 7, Recommended: 1-30
REFRESH_TOKEN_EXPIRATION=7
# JWT algorithm
# Default: HS256, Recommended: HS256
JWT_ALGORITHM=HS256

[API]
# API rate limit enabled
# Default: true, Recommended: true
RATE_LIMIT_ENABLED=true
# Rate limit window size (seconds)
# Default: 60, Recommendation: 30-300
RATE_LIMIT_WINDOW_SECONDS=60
# General API limit (requests per minute)
# Default: 300, Recommendation: 100-500
RATE_LIMIT_API=300
# General API maximum failures (trigger ban)
# Default: 10, Recommendation: 5-20
RATE_LIMIT_MAX_FAILURES=10
# General API ban duration (minutes)
# Default: 10, Recommendation: 5-30
RATE_LIMIT_BAN_MINUTES=10
# Login API limit (requests per minute)
# Default: 60, Recommendation: 30-120
RATE_LIMIT_LOGIN=60
# Login API maximum failures (trigger ban)
# Default: 10, Recommendation: 3-10
RATE_LIMIT_LOGIN_MAX_FAILURES=10
# Login API ban duration (minutes)
# Default: 5, Recommendation: 3-15
RATE_LIMIT_LOGIN_BAN_MINUTES=5
# Token refresh API limit (requests per minute)
# Default: 5, Recommendation: 3-10
RATE_LIMIT_REFRESH=5
# Token refresh API maximum failures (trigger ban)
# Default: 3, Recommendation: 2-5
RATE_LIMIT_REFRESH_MAX_FAILURES=3
# Token refresh API ban duration (minutes)
# Default: 3, Recommendation: 2-10
RATE_LIMIT_REFRESH_BAN_MINUTES=3
# Health check API limit (requests per minute)
# Default: 500, Recommendation: 200-1000
RATE_LIMIT_HEALTH=500
# Health check API maximum failures (trigger ban)
# Default: 20, Recommendation: 10-30
RATE_LIMIT_HEALTH_MAX_FAILURES=20
# Health check API ban duration (minutes)
# Default: 10, Recommendation: 5-30
RATE_LIMIT_HEALTH_BAN_MINUTES=10
# User-level API limit (requests per minute)
# Default: 500, Recommendation: 200-1000
RATE_LIMIT_USER=500
# User-level API maximum failures (trigger ban)
# Default: 20, Recommendation: 10-30
RATE_LIMIT_USER_MAX_FAILURES=20
# User-level API ban duration (minutes)
# Default: 15, Recommendation: 5-30
RATE_LIMIT_USER_BAN_MINUTES=15
# API log enabled
# Default: true, Recommended: true
LOG_ENABLED=true
# API log level
# Default: INFO, Recommended: INFO (production)/DEBUG (development)
LOG_LEVEL=INFO
# Log request body
# Default: false, Recommended: false (production)/true (debug)
LOG_REQUEST_BODY=false
# Log response body
# Default: false, Recommended: false (production)/true (debug)
LOG_RESPONSE_BODY=false

[BIND]
# BIND server address
# Default: 127.0.0.1:5300
BIND_ADDRESS=127.0.0.1:5300
# RNDC key file path
# Default: /etc/named/rndc.key
RNDC_KEY=/etc/named/rndc.key
# Zone file storage path
# Default: /usr/local/bind9/var/named
ZONE_FILE_PATH=/usr/local/bind9/var/named
# Named configuration path
# Default: /etc/named
NAMED_CONF_PATH=/etc/named
# RNDC port
# Default: 9530
RNDC_PORT=9530
# BIND user
# Default: named
BIND_USER=named
# BIND group
# Default: named
BIND_GROUP=named
# BIND start command
# Default: /usr/local/bind9/sbin/named -u named
BIND_EXEC_START=/usr/local/bind9/sbin/named -u named
# BIND reload command
# Default: /usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT reload
BIND_EXEC_RELOAD=/usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT reload
# BIND stop command
# Default: /usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT stop
BIND_EXEC_STOP=/usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT stop
# named-checkconf executable path
# Default: /usr/local/bind9/bin/named-checkconf
BIND_CHECKCONF_PATH=/usr/local/bind9/bin/named-checkconf
# named-checkzone executable path
# Default: /usr/local/bind9/bin/named-checkzone
BIND_CHECKZONE_PATH=/usr/local/bind9/bin/named-checkzone

[DNS]
# Client processing worker pool size
# Default: 10000, Recommended: 5000-10000
# Each worker handles one DNS query task. Larger values increase concurrency but also memory usage.
DNS_CLIENT_WORKERS=10000
# Task queue multiplier
# Default: 2, Recommended: 2-5
# Queue size = workers * queue multiplier, used to buffer burst requests
DNS_QUEUE_MULTIPLIER=2
# DNS server priority timeout (milliseconds)
# Default: 50, Recommended: 50-100
# Query timeout for each priority queue, affects DNS query response speed
DNS_PRIORITY_TIMEOUT_MS=50

[Cache]
# Cache size limit (MB)
# Default: 100, Recommended: 50-500
# Maximum memory usage for DNS cache. Larger values improve cache effectiveness but increase memory usage.
DNS_CACHE_SIZE_MB=100
# Cache cleanup interval (seconds)
# Default: 60, Recommended: 30-300
# Interval for periodic cleanup of expired cache, affects cache timeliness and system load
DNS_CACHE_CLEANUP_INTERVAL=60
# Error cache TTL (seconds)
# Default: 3600, Recommended: 1800-7200
# Cache time for DNS query errors, affects error handling efficiency
DNS_CACHE_ERROR_TTL=3600

[Logging]
# Query log storage path (relative to working directory)
# Default: log/, Recommended: relative or absolute path
# Storage location for DNS query logs. Ensure the directory exists and has write permissions.
QUERY_LOG_PATH=log/
# Query log file size limit (MB)
# Default: 10, Recommended: 5-50
# Maximum size of a single log file. A new file will be created when this size is exceeded.
QUERY_LOG_MAX_SIZE=10
# Query log file count limit
# Default: 10, Recommended: 5-20
# Maximum number of log files to retain. The oldest file will be deleted when this limit is exceeded.
QUERY_LOG_MAX_FILES=10
# Log level
# Default: DEBUG, Recommended: INFO (production)/DEBUG (development)
# Log detail level. Possible values: DEBUG, INFO, WARN, ERROR
DNS_LOG_LEVEL=INFO

[Security]
# DNS query rate limit per IP (queries per minute)
# Default: 300, Recommended: 120-600
# Maximum number of DNS queries allowed per IP address per minute
DNS_RATE_LIMIT_PER_IP=300
# Global DNS query rate limit (queries per minute)
# Default: 10000, Recommended: 5000-20000
# Maximum number of DNS queries allowed globally per minute
DNS_RATE_LIMIT_GLOBAL=10000
# DNS query ban duration (minutes)
# Default: 5, Recommended: 1-10
# Duration to ban IP addresses that exceed rate limits
DNS_BAN_DURATION=5
# DNS message size limit (bytes)
# Default: 4096, Recommended: 4096
# Maximum size of DNS messages to prevent amplification attacks
DNS_MESSAGE_SIZE_LIMIT=4096
# DNS query validation enabled
# Default: true, Recommended: true
# Enable DNS message validation to prevent poisoning attacks
DNS_VALIDATION_ENABLED=true

[Plugins]
# BIND Plugin - Authoritative Domain Management, BIND Server Management, Forwarding Queries, Backup
# Restart the service for changes to take effect
BIND_ENABLED=true
# DNS Rules Plugin - DNS query rules management (Reserved - feature not implemented yet)
# Restart the service for changes to take effect
DNS_RULES_ENABLED=false
# Log Analysis Plugin - DNS query log analysis (Reserved - feature not implemented yet)
# Restart the service for changes to take effect
LOG_ANALYSIS_ENABLED=false
`

// Config 存储配置信息
type Config struct {
	sections map[string]map[string]string
}

// 全局配置实例
var globalConfig *Config

// getConfigFilePath 获取配置文件路径
func getConfigFilePath() string {
	// 获取工作目录
	workingDir, err := os.Getwd()
	if err != nil {
		// 如果获取工作目录失败，使用当前目录
		workingDir = "."
	}
	// 返回相对于工作目录的配置文件路径
	return filepath.Join(workingDir, "config", "steadydns.conf")
}

// LoadConfig 加载配置文件
func LoadConfig() {
	// 获取配置文件路径
	configPath := getConfigFilePath()

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		NewLogger().Info("配置文件不存在: %s，正在创建默认配置", configPath)

		// 确保配置目录存在
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			NewLogger().Error("创建配置目录失败: %v", err)
		}

		// 写入默认配置
		if err := os.WriteFile(configPath, []byte(DefaultConfigTemplate), 0644); err != nil {
			NewLogger().Error("创建默认配置文件失败: %v", err)
			// 继续使用内存中的默认配置
			globalConfig = &Config{
				sections: make(map[string]map[string]string),
			}
			setDefaultConfig()
			return
		}
		NewLogger().Info("默认配置文件创建成功")
	}

	// 解析配置文件
	config, err := parseINI()
	if err != nil {
		NewLogger().Error("解析配置文件失败: %v", err)
		// 使用默认配置
		globalConfig = &Config{
			sections: make(map[string]map[string]string),
		}
		setDefaultConfig()
		return
	}

	globalConfig = config

	// 应用默认值
	setDefaultConfig()
}

// parseINI 解析INI配置文件
func parseINI() (*Config, error) {
	configPath := getConfigFilePath()
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{
		sections: make(map[string]map[string]string),
	}

	var currentSection string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释和空行
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// 处理节 [Section]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(line[1 : len(line)-1])
			if _, exists := config.sections[currentSection]; !exists {
				config.sections[currentSection] = make(map[string]string)
			}
			continue
		}

		// 处理键值对 key=value
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// 如果还没有当前节，使用默认节
			if currentSection == "" {
				currentSection = "Default"
				if _, exists := config.sections[currentSection]; !exists {
					config.sections[currentSection] = make(map[string]string)
				}
			}

			config.sections[currentSection][key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// setDefaultConfig 设置默认配置值
func setDefaultConfig() {
	// 确保所有必要的节存在
	ensureSection("Database")
	ensureSection("APIServer")
	ensureSection("JWT")
	ensureSection("API")
	ensureSection("BIND")
	ensureSection("DNS")
	ensureSection("Cache")
	ensureSection("Logging")
	ensureSection("Security")
	ensureSection("Plugins")

	// 设置默认值
	setDefault("Database", "DB_PATH", "steadydns.db")
	setDefault("APIServer", "API_SERVER_PORT", "8080")
	setDefault("APIServer", "API_SERVER_IP_ADDR", "0.0.0.0")
	setDefault("APIServer", "API_SERVER_IPV6_ADDR", "::")
	setDefault("APIServer", "GIN_MODE", "debug")
	setDefault("JWT", "JWT_SECRET_KEY", "your-default-jwt-secret-key-change-this-in-production")
	setDefault("JWT", "ACCESS_TOKEN_EXPIRATION", "30")
	setDefault("JWT", "REFRESH_TOKEN_EXPIRATION", "7")
	setDefault("JWT", "JWT_ALGORITHM", "HS256")

	// API 速率限制配置
	setDefault("API", "RATE_LIMIT_ENABLED", "true")
	setDefault("API", "RATE_LIMIT_WINDOW_SECONDS", "60")
	setDefault("API", "RATE_LIMIT_API", "300")
	setDefault("API", "RATE_LIMIT_MAX_FAILURES", "10")
	setDefault("API", "RATE_LIMIT_BAN_MINUTES", "10")
	setDefault("API", "RATE_LIMIT_LOGIN", "60")
	setDefault("API", "RATE_LIMIT_LOGIN_MAX_FAILURES", "10")
	setDefault("API", "RATE_LIMIT_LOGIN_BAN_MINUTES", "5")
	setDefault("API", "RATE_LIMIT_REFRESH", "5")
	setDefault("API", "RATE_LIMIT_REFRESH_MAX_FAILURES", "3")
	setDefault("API", "RATE_LIMIT_REFRESH_BAN_MINUTES", "3")
	setDefault("API", "RATE_LIMIT_HEALTH", "500")
	setDefault("API", "RATE_LIMIT_HEALTH_MAX_FAILURES", "20")
	setDefault("API", "RATE_LIMIT_HEALTH_BAN_MINUTES", "10")
	setDefault("API", "RATE_LIMIT_USER", "500")
	setDefault("API", "RATE_LIMIT_USER_MAX_FAILURES", "20")
	setDefault("API", "RATE_LIMIT_USER_BAN_MINUTES", "15")
	// API 日志配置
	setDefault("API", "LOG_ENABLED", "true")
	setDefault("API", "LOG_LEVEL", "INFO")
	setDefault("API", "LOG_REQUEST_BODY", "false")
	setDefault("API", "LOG_RESPONSE_BODY", "false")
	setDefault("BIND", "BIND_ADDRESS", "127.0.0.1:5300")
	setDefault("BIND", "RNDC_KEY", "/etc/named/rndc.key")
	setDefault("BIND", "ZONE_FILE_PATH", "/usr/local/bind9/var/named")
	setDefault("BIND", "NAMED_CONF_PATH", "/etc/named")
	setDefault("BIND", "RNDC_PORT", "9530")
	setDefault("BIND", "BIND_USER", "named")
	setDefault("BIND", "BIND_GROUP", "named")
	setDefault("BIND", "BIND_EXEC_START", "/usr/local/bind9/sbin/named -u named")
	setDefault("BIND", "BIND_EXEC_RELOAD", "/usr/local/bind9/sbin/rndc -k /etc/named/rndc.key -s 127.0.0.1 -p 9530 reload")
	setDefault("BIND", "BIND_EXEC_STOP", "/usr/local/bind9/sbin/rndc -k /etc/named/rndc.key -s 127.0.0.1 -p 9530 stop")
	setDefault("BIND", "BIND_CHECKCONF_PATH", "/usr/local/bind9/bin/named-checkconf")
	setDefault("BIND", "BIND_CHECKZONE_PATH", "/usr/local/bind9/bin/named-checkzone")
	setDefault("DNS", "DNS_CLIENT_WORKERS", "10000")
	setDefault("DNS", "DNS_QUEUE_MULTIPLIER", "2")
	setDefault("DNS", "DNS_PRIORITY_TIMEOUT_MS", "50")
	setDefault("Cache", "DNS_CACHE_SIZE_MB", "100")
	setDefault("Cache", "DNS_CACHE_CLEANUP_INTERVAL", "60")
	setDefault("Cache", "DNS_CACHE_ERROR_TTL", "3600")
	setDefault("Logging", "QUERY_LOG_PATH", "log/")
	setDefault("Logging", "QUERY_LOG_MAX_SIZE", "10")
	setDefault("Logging", "QUERY_LOG_MAX_FILES", "10")
	setDefault("Logging", "DNS_LOG_LEVEL", "DEBUG")
	setDefault("Security", "DNS_RATE_LIMIT_PER_IP", "300")
	setDefault("Security", "DNS_RATE_LIMIT_GLOBAL", "10000")
	setDefault("Security", "DNS_BAN_DURATION", "5")
	setDefault("Security", "DNS_MESSAGE_SIZE_LIMIT", "4096")
	setDefault("Security", "DNS_VALIDATION_ENABLED", "true")
	// 插件配置
	setDefault("Plugins", "BIND_ENABLED", "true")
	// 预留插件配置（功能暂未实现）
	setDefault("Plugins", "DNS_RULES_ENABLED", "false")
	setDefault("Plugins", "LOG_ANALYSIS_ENABLED", "false")
}

// ensureSection 确保节存在
func ensureSection(section string) {
	if globalConfig == nil {
		globalConfig = &Config{
			sections: make(map[string]map[string]string),
		}
	}

	if _, exists := globalConfig.sections[section]; !exists {
		globalConfig.sections[section] = make(map[string]string)
	}
}

// setDefault 设置默认值（如果不存在）
func setDefault(section, key, defaultValue string) {
	// 首先检查环境变量
	envKey := strings.ToUpper(key)
	if envValue := os.Getenv(envKey); envValue != "" {
		globalConfig.sections[section][key] = envValue
		return
	}

	// 检查配置文件中是否存在
	if _, exists := globalConfig.sections[section][key]; !exists {
		globalConfig.sections[section][key] = defaultValue
	}
}

// GetConfig 获取配置值
func GetConfig(section, key string) string {
	// 首先检查环境变量
	envKey := strings.ToUpper(key)
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}

	// 检查配置文件
	if globalConfig != nil {
		if sectionMap, exists := globalConfig.sections[section]; exists {
			if value, exists := sectionMap[key]; exists {
				return value
			}
		}
	}

	// 返回空字符串
	return ""
}

// GetConfigInt 获取整数类型的配置值
func GetConfigInt(section, key string, defaultVal int) int {
	value := GetConfig(section, key)
	if value == "" {
		return defaultVal
	}

	intVal := 0
	fmt.Sscanf(value, "%d", &intVal)
	return intVal
}

// GetConfigBool 获取布尔类型的配置值
func GetConfigBool(section, key string, defaultVal bool) bool {
	value := GetConfig(section, key)
	if value == "" {
		return defaultVal
	}

	boolVal := defaultVal
	fmt.Sscanf(value, "%t", &boolVal)
	return boolVal
}

// GetConfigFloat 获取浮点数类型的配置值
func GetConfigFloat(section, key string, defaultVal float64) float64 {
	value := GetConfig(section, key)
	if value == "" {
		return defaultVal
	}

	floatVal := defaultVal
	fmt.Sscanf(value, "%f", &floatVal)
	return floatVal
}

// GetConfigPath 获取路径类型的配置值（确保路径存在）
func GetConfigPath(section, key string, defaultPath string) string {
	path := GetConfig(section, key)
	if path == "" {
		path = defaultPath
	}

	// 确保路径存在
	if !filepath.IsAbs(path) {
		// 使用当前工作目录作为基准
		workingDir, err := os.Getwd()
		if err != nil {
			NewLogger().Error("获取工作目录失败: %v", err)
			// 如果获取工作目录失败，使用当前目录作为备选
			workingDir = "."
		}
		path = filepath.Join(workingDir, path)
	}

	// 创建目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		NewLogger().Error("创建目录失败: %v", err)
	}

	return path
}

// ConfigItem 配置项，包含键、值和注释
type ConfigItem struct {
	Key      string
	Value    string
	Comments []string
}

// ConfigSection 配置节，包含节名和配置项
type ConfigSection struct {
	Name     string
	Items    []ConfigItem
	Comments []string
}

// parseConfigTemplate 解析配置模板，提取节、注释和默认值
// 返回按顺序排列的节名切片和节映射
func parseConfigTemplate(template string) ([]string, map[string]*ConfigSection) {
	sections := make(map[string]*ConfigSection)
	var sectionOrder []string
	var currentSection *ConfigSection
	var currentComments []string

	// 按行解析模板
	lines := strings.Split(template, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 处理注释行
		if strings.HasPrefix(trimmedLine, "#") {
			currentComments = append(currentComments, trimmedLine)
			continue
		}

		// 处理空行
		if trimmedLine == "" {
			continue
		}

		// 处理节标题
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			// 保存当前节
			if currentSection != nil {
				sections[currentSection.Name] = currentSection
			}

			// 创建新节
			sectionName := strings.TrimSpace(trimmedLine[1 : len(trimmedLine)-1])
			currentSection = &ConfigSection{
				Name:     sectionName,
				Items:    make([]ConfigItem, 0),
				Comments: currentComments,
			}
			sectionOrder = append(sectionOrder, sectionName)
			currentComments = make([]string, 0)
			continue
		}

		// 处理键值对
		if idx := strings.Index(trimmedLine, "="); idx != -1 {
			key := strings.TrimSpace(trimmedLine[:idx])
			value := strings.TrimSpace(trimmedLine[idx+1:])

			// 创建配置项
			item := ConfigItem{
				Key:      key,
				Value:    value,
				Comments: currentComments,
			}

			// 添加到当前节
			if currentSection != nil {
				currentSection.Items = append(currentSection.Items, item)
			}
			currentComments = make([]string, 0)
		}
	}

	// 保存最后一个节
	if currentSection != nil {
		sections[currentSection.Name] = currentSection
	}

	return sectionOrder, sections
}

// SaveConfig 保存配置到文件
func SaveConfig() error {
	configPath := getConfigFilePath()

	// 解析默认配置模板，获取按顺序排列的节名和节映射
	sectionOrder, templateSections := parseConfigTemplate(DefaultConfigTemplate)

	// 生成配置文件内容
	var configContent strings.Builder

	// 写入各个节的配置（按照模板中的顺序）
	for _, sectionName := range sectionOrder {
		section := templateSections[sectionName]
		// 写入节的注释
		for _, comment := range section.Comments {
			configContent.WriteString(comment + "\n")
		}

		// 写入节标题
		configContent.WriteString(fmt.Sprintf("[%s]\n", sectionName))

		// 写入节内的配置项
		for _, item := range section.Items {
			// 写入配置项的注释
			for _, comment := range item.Comments {
				configContent.WriteString(comment + "\n")
			}

			// 获取当前配置值，如果没有则使用默认值
			value := item.Value
			if globalConfig != nil {
				if sectionMap, exists := globalConfig.sections[sectionName]; exists {
					if v, exists := sectionMap[item.Key]; exists {
						value = v
					}
				}
			}

			// 写入配置项
			configContent.WriteString(fmt.Sprintf("%s=%s\n", item.Key, value))
		}

		// 节之间空一行
		configContent.WriteString("\n")
	}

	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configPath, []byte(configContent.String()), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	NewLogger().Info("配置保存成功: %s", configPath)
	return nil
}

// UpdateConfig 更新配置值
func UpdateConfig(section, key, value string) error {
	// 确保配置实例存在
	if globalConfig == nil {
		LoadConfig()
	}

	// 确保节存在
	ensureSection(section)

	// 更新配置值
	globalConfig.sections[section][key] = value

	// 保存配置到文件
	return SaveConfig()
}

// ValidateConfig 验证配置有效性
func ValidateConfig() error {
	// 确保配置实例存在
	if globalConfig == nil {
		LoadConfig()
	}

	// 验证必要的配置项
	requiredSections := []string{"Database", "HTTP", "JWT", "API", "BIND", "DNS", "Cache", "Logging", "Security"}
	for _, section := range requiredSections {
		if _, exists := globalConfig.sections[section]; !exists {
			return fmt.Errorf("缺少必要的配置节: %s", section)
		}
	}

	// 验证特定配置项
	// HTTP端口验证
	port := GetConfig("HTTP", "PORT")
	if port == "" {
		return fmt.Errorf("缺少HTTP端口配置")
	}

	// JWT密钥验证
	jwtSecret := GetConfig("JWT", "JWT_SECRET_KEY")
	if jwtSecret == "" {
		return fmt.Errorf("缺少JWT密钥配置")
	}

	// 验证配置值的有效性
	// 例如：端口号范围检查、路径存在性检查等

	NewLogger().Info("配置验证成功")
	return nil
}

// GetAllConfig 获取所有配置
func GetAllConfig() map[string]map[string]string {
	// 确保配置实例存在
	if globalConfig == nil {
		LoadConfig()
	}

	// 创建配置副本
	configCopy := make(map[string]map[string]string)
	for section, keyValues := range globalConfig.sections {
		configCopy[section] = make(map[string]string)
		for key, value := range keyValues {
			configCopy[section][key] = value
		}
	}

	return configCopy
}

// GetSectionConfig 获取指定节的配置
func GetSectionConfig(section string) map[string]string {
	// 确保配置实例存在
	if globalConfig == nil {
		LoadConfig()
	}

	// 检查节是否存在
	if sectionMap, exists := globalConfig.sections[section]; exists {
		// 创建副本
		sectionCopy := make(map[string]string)
		for key, value := range sectionMap {
			sectionCopy[key] = value
		}
		return sectionCopy
	}

	return make(map[string]string)
}

// ResetToDefaults 重置为默认配置
func ResetToDefaults() error {
	// 重新加载默认配置模板
	configPath := getConfigFilePath()

	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 写入默认配置
	if err := os.WriteFile(configPath, []byte(DefaultConfigTemplate), 0644); err != nil {
		return fmt.Errorf("写入默认配置失败: %v", err)
	}

	// 重新加载配置
	LoadConfig()

	NewLogger().Info("配置已重置为默认值")
	return nil
}

// ReloadConfig 重载配置
func ReloadConfig() error {
	LoadConfig()
	NewLogger().Info("配置重载成功")
	return nil
}
