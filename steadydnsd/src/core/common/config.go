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
const defaultConfigTemplate = `# SteadyDNS Configuration File
# Format: INI/Conf

[Database]
# Database file path (relative to working directory)
DB_PATH=steadydns.db

[Server]
# JWT secret key for authentication
JWT_SECRET_KEY=your_jwt_secret_key_here
# API Server port
PORT=8080

[BIND]
# BIND server address
BIND_ADDRESS=127.0.0.1:5300
# RNDC key file path
RNDC_KEY=/etc/named/rndc.key
# Zone file storage path
ZONE_FILE_PATH=/usr/local/bind9/var/named
# Named configuration path
NAMED_CONF_PATH=/etc/named
# RNDC port
RNDC_PORT=9530
# BIND user
BIND_USER=named
# BIND group
BIND_GROUP=named
# BIND start command
BIND_EXEC_START=/usr/local/bind9/sbin/named -u named
# BIND reload command
BIND_EXEC_RELOAD=/usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT reload
# BIND stop command
BIND_EXEC_STOP=/usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT stop

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
DNS_LOG_LEVEL=DEBUG

[Security]
# DNS query rate limit per IP (queries per minute)
# Default: 60, Recommended: 30-120
# Maximum number of DNS queries allowed per IP address per minute
DNS_RATE_LIMIT_PER_IP=60
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
		if err := os.WriteFile(configPath, []byte(defaultConfigTemplate), 0644); err != nil {
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
	ensureSection("Server")
	ensureSection("JWT")
	ensureSection("API")
	ensureSection("BIND")
	ensureSection("DNS")
	ensureSection("Cache")
	ensureSection("Logging")
	ensureSection("Security")

	// 设置默认值
	setDefault("Database", "DB_PATH", "steadydns.db")
	setDefault("Server", "JWT_SECRET_KEY", "your-default-jwt-secret-key-change-this-in-production")
	setDefault("Server", "GIN_MODE", "debug")
	setDefault("Server", "PORT", "8080")
	setDefault("JWT", "ACCESS_TOKEN_EXPIRATION", "30")
	setDefault("JWT", "REFRESH_TOKEN_EXPIRATION", "7")
	setDefault("JWT", "JWT_ALGORITHM", "HS256")

	// API 安全配置
	setDefault("API", "RATE_LIMIT_ENABLED", "true")
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
	setDefault("Security", "DNS_RATE_LIMIT_PER_IP", "60")
	setDefault("Security", "DNS_RATE_LIMIT_GLOBAL", "10000")
	setDefault("Security", "DNS_BAN_DURATION", "5")
	setDefault("Security", "DNS_MESSAGE_SIZE_LIMIT", "4096")
	setDefault("Security", "DNS_VALIDATION_ENABLED", "true")
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
