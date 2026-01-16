// core/sdns/dnslogger.go

package sdns

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
)

// DNSLogger DNS查询日志管理器
type DNSLogger struct {
	logDir      string
	logFile     string
	maxFileSize int64
	maxFiles    int
	file        *os.File
	writer      *bufio.Writer
	logChan     chan string
	wg          sync.WaitGroup
	mu          sync.Mutex
	shutdown    bool
}

// NewDNSLogger 创建DNS日志管理器
func NewDNSLogger(logDir string, maxFileSize int64, maxFiles int) *DNSLogger {
	if logDir == "" {
		logDir = "log"
	}

	if maxFileSize <= 0 {
		maxFileSize = 10 * 1024 * 1024 // 默认10MB
	}

	if maxFiles <= 0 {
		maxFiles = 10 // 默认10个文件
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		logDir = "."
	}

	logger := &DNSLogger{
		logDir:      logDir,
		logFile:     filepath.Join(logDir, "dns_query.log"),
		maxFileSize: maxFileSize,
		maxFiles:    maxFiles,
		logChan:     make(chan string, 1000),
		shutdown:    false,
	}

	if err := logger.openLogFile(); err != nil {
		fmt.Printf("打开日志文件失败: %v\n", err)
	}

	logger.wg.Add(1)
	go logger.logWriter()

	return logger
}

// openLogFile 打开日志文件
func (l *DNSLogger) openLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查当前日志文件大小
	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
	}

	// 检查是否需要滚动日志
	if fi, err := os.Stat(l.logFile); err == nil && fi.Size() >= l.maxFileSize {
		l.rollLogFile()
	}

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(l.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.writer = bufio.NewWriter(file)
	return nil
}

// rollLogFile 滚动日志文件
func (l *DNSLogger) rollLogFile() {
	// 关闭当前文件
	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
		l.file = nil
		l.writer = nil
	}

	// 获取所有日志文件
	logFiles, err := filepath.Glob(l.logFile + ".*")
	if err != nil {
		return
	}

	// 按日期排序并删除多余的文件
	if len(logFiles) >= l.maxFiles {
		// 简单实现：删除最旧的文件
		for i := 0; i < len(logFiles)-l.maxFiles+1; i++ {
			os.Remove(logFiles[i])
		}
	}

	// 重命名当前文件
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	newLogFile := l.logFile + "." + timestamp
	os.Rename(l.logFile, newLogFile)
}

// logWriter 异步写入日志
func (l *DNSLogger) logWriter() {
	defer l.wg.Done()

	for {
		select {
		case logMsg, ok := <-l.logChan:
			if !ok {
				return
			}

			l.mu.Lock()
			if l.file == nil {
				if err := l.openLogFile(); err != nil {
					l.mu.Unlock()
					continue
				}
			}

			// 检查文件大小
			if fi, err := l.file.Stat(); err == nil && fi.Size() >= l.maxFileSize {
				l.rollLogFile()
				l.openLogFile()
			}

			// 写入日志
			fmt.Fprintln(l.writer, logMsg)
			l.writer.Flush()
			l.mu.Unlock()
		case <-time.After(1 * time.Second):
			l.mu.Lock()
			if l.shutdown {
				l.mu.Unlock()
				return
			}
			l.mu.Unlock()
		}
	}
}

// Log 记录DNS查询日志
func (l *DNSLogger) Log(format string, args ...interface{}) {
	if l.shutdown {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[%s] %s", timestamp, fmt.Sprintf(format, args...))

	select {
	case l.logChan <- logMsg:
	default:
		// 通道已满，丢弃日志
		fmt.Println("日志通道已满，丢弃日志")
	}
}

// Close 关闭日志管理器
func (l *DNSLogger) Close() {
	l.shutdown = true
	close(l.logChan)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
	}
}

// getClientIP 从ResponseWriter中获取客户端IP
func getClientIP(w interface{}) string {
	// 尝试从不同类型的ResponseWriter中获取客户端IP
	switch v := w.(type) {
	case interface{ RemoteAddr() string }:
		addr := v.RemoteAddr()
		if addr == "" {
			return "unknown"
		}
		// 提取IP地址部分
		if strings.Contains(addr, ":") {
			// IPv4或IPv6地址
			parts := strings.Split(addr, ":")
			if len(parts) >= 2 {
				// 对于IPv4，格式为 "192.168.1.1:12345"
				// 对于IPv6，格式为 "[2001:db8::1]:12345"
				ip := parts[0]
				ip = strings.TrimPrefix(ip, "[")
				if ip == "" {
					return "unknown"
				}
				return ip
			}
		}
		return addr
	case interface{ LocalAddr() net.Addr }:
		// 尝试获取本地地址，虽然不是客户端地址，但比unknown好
		addr := v.LocalAddr()
		if addr != nil {
			addrStr := addr.String()
			if addrStr == "" {
				return "unknown"
			}
			// 提取IP地址部分
			if strings.Contains(addrStr, ":") {
				parts := strings.Split(addrStr, ":")
				if len(parts) >= 2 {
					ip := parts[0]
					ip = strings.TrimPrefix(ip, "[")
					if ip == "" {
						return "unknown"
					}
					return ip
				}
			}
			return addrStr
		}
	default:
		// 尝试使用反射获取客户端地址信息
		// 注意：由于Go语言的限制，我们不能直接访问非导出字段
		// 所以这里只是一个尝试，如果失败则返回unknown
		val := reflect.ValueOf(w)

		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// 尝试获取pcSession字段（这是net.Addr类型，可能包含客户端地址）
		// 注意：这可能会失败，因为pcSession是一个非导出字段
		pcSessionField := val.FieldByName("pcSession")
		if pcSessionField.IsValid() && !pcSessionField.IsNil() {
			defer func() {
				if r := recover(); r != nil {
					// 捕获反射错误
				}
			}()

			addr := pcSessionField.Interface()
			if addrNet, ok := addr.(net.Addr); ok {
				addrStr := addrNet.String()
				if addrStr == "" {
					return "unknown"
				}
				// 提取IP地址部分
				if strings.Contains(addrStr, ":") {
					parts := strings.Split(addrStr, ":")
					if len(parts) >= 2 {
						ip := parts[0]
						ip = strings.TrimPrefix(ip, "[")
						if ip == "" {
							return "unknown"
						}
						return ip
					}
				}
				return addrStr
			}
		}
	}

	return "unknown"
}
