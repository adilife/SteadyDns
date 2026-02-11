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
// core/common/rotatelogger.go

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultMaxSize 默认单个日志文件大小限制 (10MB)
	DefaultMaxSize = 10 * 1024 * 1024
	// DefaultMaxFiles 默认保留的日志文件数量
	DefaultMaxFiles = 10
	// DefaultLogDir 默认日志目录
	DefaultLogDir = "log"
	// DefaultLogFile 默认日志文件名
	DefaultLogFile = "steadydns.log"
)

// RotateLogger 轮转日志记录器
type RotateLogger struct {
	logDir      string
	logFile     string
	maxSize     int64
	maxFiles    int
	currentFile *os.File
	currentSize int64
	mutex       sync.Mutex
	level       LogLevel
	stdout      bool
}

// 确保 RotateLogger 实现了 LoggerInterface 接口
var _ LoggerInterface = (*RotateLogger)(nil)

// NewRotateLogger 创建新的轮转日志记录器
// 参数:
//
//	logDir: 日志目录
//	logFile: 日志文件名
//	maxSize: 单个文件大小限制(字节)
//	maxFiles: 保留文件数量
//
// 返回值:
//
//	*RotateLogger: 轮转日志记录器实例
//	error: 创建过程中的错误
func NewRotateLogger(logDir, logFile string, maxSize int64, maxFiles int) (*RotateLogger, error) {
	if logDir == "" {
		logDir = DefaultLogDir
	}
	if logFile == "" {
		logFile = DefaultLogFile
	}
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	if maxFiles <= 0 {
		maxFiles = DefaultMaxFiles
	}

	logger := &RotateLogger{
		logDir:   logDir,
		logFile:  logFile,
		maxSize:  maxSize,
		maxFiles: maxFiles,
		level:    GetLogLevelFromEnv(),
		stdout:   false,
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开当前日志文件
	if err := logger.openFile(); err != nil {
		return nil, err
	}

	return logger, nil
}

// SetStdout 设置是否同时输出到标准输出
// 参数:
//
//	enable: 是否启用标准输出
func (r *RotateLogger) SetStdout(enable bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.stdout = enable
}

// SetLevel 设置日志级别
// 参数:
//
//	level: 日志级别
func (r *RotateLogger) SetLevel(level LogLevel) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.level = level
}

// GetLevel 获取日志级别
// 返回值:
//
//	LogLevel: 当前日志级别
func (r *RotateLogger) GetLevel() LogLevel {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.level
}

// openFile 打开或创建日志文件
// 返回值:
//
//	error: 打开过程中的错误
func (r *RotateLogger) openFile() error {
	logPath := filepath.Join(r.logDir, r.logFile)

	// 检查文件是否存在
	info, err := os.Stat(logPath)
	if err == nil {
		// 文件存在，获取当前大小
		r.currentSize = info.Size()
		// 如果文件已经超过大小限制，先轮转
		if r.currentSize >= r.maxSize {
			if err := r.rotate(); err != nil {
				return err
			}
			// 重新打开新文件
			logPath = filepath.Join(r.logDir, r.logFile)
			r.currentSize = 0
		}
	}

	// 打开文件(追加模式)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	r.currentFile = file
	if info != nil {
		r.currentSize = info.Size()
	} else {
		r.currentSize = 0
	}

	return nil
}

// rotate 执行日志轮转
// 返回值:
//
//	error: 轮转过程中的错误
func (r *RotateLogger) rotate() error {
	// 关闭当前文件
	if r.currentFile != nil {
		r.currentFile.Close()
		r.currentFile = nil
	}

	// 删除最旧的日志文件
	oldestFile := filepath.Join(r.logDir, fmt.Sprintf("%s.%d", r.logFile, r.maxFiles))
	os.Remove(oldestFile)

	// 重命名现有的日志文件
	for i := r.maxFiles - 1; i >= 1; i-- {
		oldPath := filepath.Join(r.logDir, fmt.Sprintf("%s.%d", r.logFile, i))
		newPath := filepath.Join(r.logDir, fmt.Sprintf("%s.%d", r.logFile, i+1))

		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// 重命名当前日志文件
	currentPath := filepath.Join(r.logDir, r.logFile)
	newPath := filepath.Join(r.logDir, fmt.Sprintf("%s.1", r.logFile))
	if _, err := os.Stat(currentPath); err == nil {
		os.Rename(currentPath, newPath)
	}

	// 重新打开新文件
	return r.openFile()
}

// write 写入日志
// 参数:
//
//	message: 日志消息
//
// 返回值:
//
//	error: 写入过程中的错误
func (r *RotateLogger) write(message string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 检查是否需要轮转
	if r.currentSize >= r.maxSize {
		if err := r.rotate(); err != nil {
			return err
		}
	}

	// 写入消息
	n, err := r.currentFile.WriteString(message)
	if err != nil {
		return err
	}

	r.currentSize += int64(n)

	// 同时输出到标准输出
	if r.stdout {
		fmt.Print(message)
	}

	return nil
}

// log 内部日志记录方法
// 参数:
//
//	level: 日志级别
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) log(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	if err := r.write(logLine); err != nil {
		// 如果写入失败，输出到标准错误
		fmt.Fprintf(os.Stderr, "写入日志失败: %v\n", err)
	}
}

// Debug 记录DEBUG级别日志
// 参数:
//
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) Debug(format string, args ...interface{}) {
	if r.level <= DEBUG {
		r.log("DEBUG", format, args...)
	}
}

// Info 记录INFO级别日志
// 参数:
//
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) Info(format string, args ...interface{}) {
	if r.level <= INFO {
		r.log("INFO", format, args...)
	}
}

// Warn 记录WARN级别日志
// 参数:
//
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) Warn(format string, args ...interface{}) {
	if r.level <= WARN {
		r.log("WARN", format, args...)
	}
}

// Error 记录ERROR级别日志
// 参数:
//
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) Error(format string, args ...interface{}) {
	if r.level <= ERROR {
		r.log("ERROR", format, args...)
	}
}

// Fatal 记录FATAL级别日志并退出程序
// 参数:
//
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) Fatal(format string, args ...interface{}) {
	if r.level <= FATAL {
		r.log("FATAL", format, args...)
	}
	os.Exit(1)
}

// LogError 记录错误日志，包含错误详情
// 参数:
//
//	format: 格式字符串
//	err: 错误对象
//	args: 格式化参数
func (r *RotateLogger) LogError(format string, err error, args ...interface{}) {
	if r.level <= ERROR {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		message := fmt.Sprintf(format, args...)
		errorDetails := "nil"
		if err != nil {
			errorDetails = err.Error()
		}
		logLine := fmt.Sprintf("[%s] [ERROR] %s - Error: %s\n", timestamp, message, errorDetails)

		if writeErr := r.write(logLine); writeErr != nil {
			fmt.Fprintf(os.Stderr, "写入日志失败: %v\n", writeErr)
		}
	}
}

// Printf 兼容旧的日志打印方法
// 参数:
//
//	format: 格式字符串
//	args: 格式化参数
func (r *RotateLogger) Printf(format string, args ...interface{}) {
	r.Info(format, args...)
}

// Write 实现 io.Writer 接口，用于 GIN 等框架的日志输出
// 参数:
//
//	p: 要写入的字节切片
//
// 返回值:
//
//	n: 写入的字节数
//	err: 写入过程中的错误
func (r *RotateLogger) Write(p []byte) (n int, err error) {
	// 去除末尾的换行符，因为 log 方法会自动添加
	message := strings.TrimSuffix(string(p), "\n")
	message = strings.TrimSuffix(message, "\r")
	if message != "" {
		r.Info(message)
	}
	return len(p), nil
}

// Close 关闭日志文件
func (r *RotateLogger) Close() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.currentFile != nil {
		r.currentFile.Close()
		r.currentFile = nil
	}
}

// GetLogFiles 获取所有日志文件列表
// 返回值:
//
//	[]string: 日志文件路径列表
func (r *RotateLogger) GetLogFiles() []string {
	var files []string

	// 添加当前日志文件
	currentPath := filepath.Join(r.logDir, r.logFile)
	if _, err := os.Stat(currentPath); err == nil {
		files = append(files, currentPath)
	}

	// 添加轮转的历史文件
	for i := 1; i <= r.maxFiles; i++ {
		historyPath := filepath.Join(r.logDir, fmt.Sprintf("%s.%d", r.logFile, i))
		if _, err := os.Stat(historyPath); err == nil {
			files = append(files, historyPath)
		}
	}

	return files
}

// CleanupOldLogs 清理旧的日志文件
// 返回值:
//
//	int: 删除的文件数量
//	error: 清理过程中的错误
func (r *RotateLogger) CleanupOldLogs() (int, error) {
	files, err := filepath.Glob(filepath.Join(r.logDir, r.logFile+"*"))
	if err != nil {
		return 0, err
	}

	// 按修改时间排序
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{path: file, modTime: info.ModTime()})
	}

	// 按修改时间降序排序
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].modTime.After(fileInfos[j].modTime)
	})

	// 删除超出限制的文件
	deleted := 0
	for i, fi := range fileInfos {
		// 保留当前文件和最近的maxFiles个历史文件
		if i > r.maxFiles {
			if err := os.Remove(fi.path); err == nil {
				deleted++
			}
		}
	}

	return deleted, nil
}

// GetLogStats 获取日志统计信息
// 返回值:
//
//	map[string]interface{}: 统计信息
func (r *RotateLogger) GetLogStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["logDir"] = r.logDir
	stats["logFile"] = r.logFile
	stats["maxSize"] = r.maxSize
	stats["maxFiles"] = r.maxFiles
	stats["currentSize"] = r.currentSize
	stats["level"] = r.level.String()
	stats["stdout"] = r.stdout

	// 计算总大小
	var totalSize int64
	files := r.GetLogFiles()
	for _, file := range files {
		info, err := os.Stat(file)
		if err == nil {
			totalSize += info.Size()
		}
	}
	stats["totalSize"] = totalSize
	stats["fileCount"] = len(files)

	return stats
}

// ParseSize 解析大小字符串
// 参数:
//
//	sizeStr: 大小字符串，如 "10MB", "100KB", "1GB"
//
// 返回值:
//
//	int64: 字节数
//	error: 解析错误
func ParseSize(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))

	var multiplier int64 = 1
	switch {
	case strings.HasSuffix(sizeStr, "GB"):
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	case strings.HasSuffix(sizeStr, "MB"):
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	case strings.HasSuffix(sizeStr, "KB"):
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	case strings.HasSuffix(sizeStr, "B"):
		sizeStr = strings.TrimSuffix(sizeStr, "B")
	}

	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("无法解析大小: %v", err)
	}

	return size * multiplier, nil
}
