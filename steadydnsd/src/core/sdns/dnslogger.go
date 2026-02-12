// core/sdns/dnslogger.go
// DNS查询日志管理器 - 支持查询级日志聚合和批量写入

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
	"sync/atomic"
	"time"
)

// QueryLogBuffer 查询日志缓冲区（每个查询独立使用，无锁）
type QueryLogBuffer struct {
	QueryID       string
	StartTime     time.Time
	ClientIP      string
	QueryName     string
	QueryType     string
	Stages        []StageInfo
	ResponseCode  int
	Error         string
}

// StageInfo 阶段信息
type StageInfo struct {
	Name     string
	Duration time.Duration
	Detail   string
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string
	Content   string
}

// DNSLogger DNS查询日志管理器
type DNSLogger struct {
	logDir        string
	logFile       string
	maxFileSize   int64
	maxFiles      int
	file          *os.File
	writer        *bufio.Writer
	batchChan     chan *LogEntry
	wg            sync.WaitGroup
	mu            sync.Mutex
	shutdown      bool
	
	// 对象池用于复用缓冲区
	bufferPool sync.Pool
	
	// 批量写入配置
	batchSize      int
	flushInterval  time.Duration
	writerCount    int
	
	// 统计信息
	writtenCount   int64
	droppedCount   int64
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
		logDir:        logDir,
		logFile:       filepath.Join(logDir, "dns_query.log"),
		maxFileSize:   maxFileSize,
		maxFiles:      maxFiles,
		batchChan:     make(chan *LogEntry, 10000),
		shutdown:      false,
		batchSize:     100,             // 每批100条
		flushInterval: 100 * time.Millisecond, // 100ms刷新
		writerCount:   4,               // 4个writer协程
	}

	// 初始化对象池
	logger.bufferPool = sync.Pool{
		New: func() interface{} {
			return &QueryLogBuffer{
				Stages: make([]StageInfo, 0, 10),
			}
		},
	}

	logger.mu.Lock()
	if err := logger.openLogFile(); err != nil {
		fmt.Printf("打开日志文件失败: %v\n", err)
	}
	logger.mu.Unlock()

	// 启动多个writer协程
	for i := 0; i < logger.writerCount; i++ {
		logger.wg.Add(1)
		go logger.batchWriter()
	}

	return logger
}

// GetBuffer 从对象池获取缓冲区（查询开始时调用）
func (l *DNSLogger) GetBuffer() *QueryLogBuffer {
	buf := l.bufferPool.Get().(*QueryLogBuffer)
	buf.StartTime = time.Now()
	buf.Stages = buf.Stages[:0]
	buf.ResponseCode = 0
	buf.Error = ""
	return buf
}

// PutBuffer 归还缓冲区到对象池
func (l *DNSLogger) PutBuffer(buf *QueryLogBuffer) {
	if buf != nil {
		buf.Stages = buf.Stages[:0]
		l.bufferPool.Put(buf)
	}
}

// StartQuery 开始记录查询
func (l *DNSLogger) StartQuery(queryID, clientIP, queryName, queryType string) *QueryLogBuffer {
	if l.shutdown {
		return nil
	}
	
	buf := l.GetBuffer()
	buf.QueryID = queryID
	buf.ClientIP = clientIP
	buf.QueryName = queryName
	buf.QueryType = queryType
	return buf
}

// RecordStage 记录查询阶段（无锁，只操作本地buffer）
func (l *DNSLogger) RecordStage(buf *QueryLogBuffer, name, detail string) {
	if buf == nil || l.shutdown {
		return
	}
	
	now := time.Now()
	duration := now.Sub(buf.StartTime)
	
	// 如果已有阶段，计算当前阶段耗时
	if len(buf.Stages) > 0 {
		lastStage := buf.Stages[len(buf.Stages)-1]
		duration = now.Sub(buf.StartTime) - lastStage.Duration
	}
	
	buf.Stages = append(buf.Stages, StageInfo{
		Name:     name,
		Duration: duration,
		Detail:   detail,
	})
}

// EndQuery 结束查询并提交汇总日志
func (l *DNSLogger) EndQuery(buf *QueryLogBuffer, responseCode int, err error) {
	if buf == nil || l.shutdown {
		l.PutBuffer(buf)
		return
	}
	
	buf.ResponseCode = responseCode
	if err != nil {
		buf.Error = err.Error()
	}
	
	// 生成汇总日志
	logContent := l.formatLogEntry(buf)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	
	entry := &LogEntry{
		Timestamp: timestamp,
		Content:   logContent,
	}
	
	// 提交到批量channel
	select {
	case l.batchChan <- entry:
		// 成功提交
	default:
		// channel已满，丢弃日志
		atomic.AddInt64(&l.droppedCount, 1)
	}
	
	// 归还缓冲区
	l.PutBuffer(buf)
}

// formatLogEntry 格式化日志条目
func (l *DNSLogger) formatLogEntry(buf *QueryLogBuffer) string {
	var sb strings.Builder
	
	totalTime := time.Since(buf.StartTime)
	
	sb.WriteString(fmt.Sprintf("QueryID=%s ", buf.QueryID))
	sb.WriteString(fmt.Sprintf("ClientIP=%s ", buf.ClientIP))
	sb.WriteString(fmt.Sprintf("Query=%s/%s ", buf.QueryName, buf.QueryType))
	sb.WriteString(fmt.Sprintf("TotalTime=%.2fms ", float64(totalTime)/float64(time.Millisecond)))
	
	// 各阶段详情
	for _, stage := range buf.Stages {
		sb.WriteString(fmt.Sprintf("%s=%.2fms[%s] ", 
			stage.Name,
			float64(stage.Duration)/float64(time.Millisecond),
			stage.Detail))
	}
	
	if buf.Error != "" {
		sb.WriteString(fmt.Sprintf("Error=%s ", buf.Error))
	}
	
	sb.WriteString(fmt.Sprintf("ResponseCode=%d", buf.ResponseCode))
	
	return sb.String()
}

// batchWriter 批量写入协程
func (l *DNSLogger) batchWriter() {
	defer l.wg.Done()
	
	batch := make([]*LogEntry, 0, l.batchSize)
	timer := time.NewTimer(l.flushInterval)
	defer timer.Stop()
	
	for {
		select {
		case entry, ok := <-l.batchChan:
			if !ok {
				// channel关闭，刷新剩余日志
				if len(batch) > 0 {
					l.flushBatch(batch)
				}
				return
			}
			
			batch = append(batch, entry)
			
			// 达到批量大小，立即刷新
			if len(batch) >= l.batchSize {
				l.flushBatch(batch)
				batch = batch[:0]
				timer.Reset(l.flushInterval)
			}
			
		case <-timer.C:
			// 定时刷新
			if len(batch) > 0 {
				l.flushBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(l.flushInterval)
		}
	}
}

// flushBatch 批量刷新日志到文件
func (l *DNSLogger) flushBatch(batch []*LogEntry) {
	if len(batch) == 0 {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.shutdown || l.writer == nil {
		return
	}
	
	// 检查文件大小，需要时滚动
	if fi, err := l.file.Stat(); err == nil && fi.Size() >= l.maxFileSize {
		l.rollLogFile()
		l.openLogFile()
	}
	
	// 批量写入
	for _, entry := range batch {
		fmt.Fprintf(l.writer, "[%s] %s\n", entry.Timestamp, entry.Content)
	}
	
	// 一次刷新
	l.writer.Flush()
	
	atomic.AddInt64(&l.writtenCount, int64(len(batch)))
}

// openLogFile 打开日志文件
func (l *DNSLogger) openLogFile() error {
	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
	}

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
	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
		l.file = nil
		l.writer = nil
	}

	logFiles, err := filepath.Glob(l.logFile + ".*")
	if err != nil {
		return
	}

	if len(logFiles) >= l.maxFiles {
		for i := 0; i < len(logFiles)-l.maxFiles+1; i++ {
			os.Remove(logFiles[i])
		}
	}

	timestamp := time.Now().Format("2006-01-02-15-04-05")
	newLogFile := l.logFile + "." + timestamp
	os.Rename(l.logFile, newLogFile)
}

// Log 记录普通日志（兼容旧接口）
func (l *DNSLogger) Log(format string, args ...interface{}) {
	if l.shutdown {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	content := fmt.Sprintf(format, args...)
	
	entry := &LogEntry{
		Timestamp: timestamp,
		Content:   content,
	}

	select {
	case l.batchChan <- entry:
	default:
		atomic.AddInt64(&l.droppedCount, 1)
	}
}

// Close 关闭日志管理器
func (l *DNSLogger) Close() {
	l.shutdown = true
	close(l.batchChan)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
	}
}

// GetStats 获取统计信息
func (l *DNSLogger) GetStats() (written, dropped int64) {
	return atomic.LoadInt64(&l.writtenCount), atomic.LoadInt64(&l.droppedCount)
}

// getClientIP 从ResponseWriter中获取客户端IP
func getClientIP(w interface{}) string {
	switch v := w.(type) {
	case interface{ RemoteAddr() string }:
		addr := v.RemoteAddr()
		if addr == "" {
			return "unknown"
		}
		if strings.Contains(addr, ":") {
			parts := strings.Split(addr, ":")
			if len(parts) >= 2 {
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
		addr := v.LocalAddr()
		if addr != nil {
			addrStr := addr.String()
			if addrStr == "" {
				return "unknown"
			}
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
		val := reflect.ValueOf(w)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		pcSessionField := val.FieldByName("pcSession")
		if pcSessionField.IsValid() && !pcSessionField.IsNil() {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			addr := pcSessionField.Interface()
			if addrNet, ok := addr.(net.Addr); ok {
				addrStr := addrNet.String()
				if addrStr == "" {
					return "unknown"
				}
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
