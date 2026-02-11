// core/common/logger.go

package common

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// 日志级别常量
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
)

// LogLevel 日志级别类型
type LogLevel int

// 日志级别字符串映射
var logLevelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// 字符串到日志级别的映射
var logLevelValues = map[string]LogLevel{
	"DEBUG": DEBUG,
	"INFO":  INFO,
	"WARN":  WARN,
	"ERROR": ERROR,
	"FATAL": FATAL,
}

// String 返回日志级别的字符串表示
func (level LogLevel) String() string {
	if name, ok := logLevelNames[level]; ok {
		return name
	}
	return "UNKNOWN"
}

// ParseLogLevel 从字符串解析日志级别
func ParseLogLevel(levelStr string) LogLevel {
	levelStr = strings.ToUpper(levelStr)
	if level, ok := logLevelValues[levelStr]; ok {
		return level
	}
	return INFO // 默认INFO级别
}

// GetLogLevelFromEnv 从环境变量获取日志级别
func GetLogLevelFromEnv() LogLevel {
	levelStr := GetConfig("Logging", "DNS_LOG_LEVEL")
	if levelStr == "" {
		return INFO // 默认INFO级别
	}
	return ParseLogLevel(levelStr)
}

// 全局日志输出器
var globalLogger LoggerInterface

// SetGlobalLogger 设置全局日志输出器
func SetGlobalLogger(logger LoggerInterface) {
	globalLogger = logger
}

// GetGlobalLogger 获取全局日志输出器
func GetGlobalLogger() LoggerInterface {
	if globalLogger == nil {
		// 如果没有设置全局日志器，创建一个默认的终端日志器
		return NewDefaultLogger()
	}
	return globalLogger
}

// LoggerInterface 日志接口
type LoggerInterface interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	LogError(format string, err error, args ...interface{})
	Printf(format string, args ...interface{})
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

// Logger 日志管理器 - 现在作为 LoggerInterface 的适配器
type Logger struct {
	level LogLevel
}

// NewLogger 创建新的日志管理器
// 现在返回的 Logger 会使用全局日志输出器
func NewLogger() *Logger {
	return &Logger{
		level: GetLogLevelFromEnv(),
	}
}

// NewLoggerWithLevel 创建指定级别的日志管理器
func NewLoggerWithLevel(level LogLevel) *Logger {
	return &Logger{
		level: level,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
	// 同时设置全局日志器的级别
	if globalLogger != nil {
		globalLogger.SetLevel(level)
	}
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// Debug 打印DEBUG级别日志
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		GetGlobalLogger().Debug(format, args...)
	}
}

// Info 打印INFO级别日志
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		GetGlobalLogger().Info(format, args...)
	}
}

// Warn 打印WARN级别日志
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		GetGlobalLogger().Warn(format, args...)
	}
}

// Error 打印ERROR级别日志
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		GetGlobalLogger().Error(format, args...)
	}
}

// Fatal 打印FATAL级别日志并退出程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	if l.level <= FATAL {
		GetGlobalLogger().Fatal(format, args...)
	}
}

// LogError 记录错误日志，包含错误详情
func (l *Logger) LogError(format string, err error, args ...interface{}) {
	if l.level <= ERROR {
		GetGlobalLogger().LogError(format, err, args...)
	}
}

// Printf 兼容旧的日志打印方法
func (l *Logger) Printf(format string, args ...interface{}) {
	l.Info(format, args...)
}

// DefaultLogger 默认的终端日志实现
type DefaultLogger struct {
	level LogLevel
}

// NewDefaultLogger 创建默认的终端日志器
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		level: INFO,
	}
}

// SetLevel 设置日志级别
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel 获取日志级别
func (l *DefaultLogger) GetLevel() LogLevel {
	return l.level
}

// Debug 打印DEBUG级别日志
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

// Info 打印INFO级别日志
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

// Warn 打印WARN级别日志
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

// Error 打印ERROR级别日志
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		l.log("ERROR", format, args...)
	}
}

// Fatal 打印FATAL级别日志并退出程序
func (l *DefaultLogger) Fatal(format string, args ...interface{}) {
	if l.level <= FATAL {
		l.log("FATAL", format, args...)
	}
	os.Exit(1)
}

// LogError 记录错误日志，包含错误详情
func (l *DefaultLogger) LogError(format string, err error, args ...interface{}) {
	if l.level <= ERROR {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		message := fmt.Sprintf(format, args...)
		errorDetails := "nil"
		if err != nil {
			errorDetails = err.Error()
		}
		fmt.Printf("[%s] [ERROR] %s - Error: %s\n", timestamp, message, errorDetails)
	}
}

// Printf 兼容旧的日志打印方法
func (l *DefaultLogger) Printf(format string, args ...interface{}) {
	l.Info(format, args...)
}

// log 内部日志打印方法
func (l *DefaultLogger) log(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [%s] %s\n", timestamp, level, message)
}
