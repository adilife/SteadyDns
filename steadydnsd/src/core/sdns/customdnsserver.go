// core/sdns/customdnsserver.go

package sdns

import (
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/miekg/dns"

	"SteadyDNS/core/common"
)

// BufferPool 缓冲区池
type BufferPool struct {
	mu      sync.RWMutex
	buffers chan []byte
	size    int
	count   int
	maxSize int
}

// UDPPacket UDP 数据包任务
type UDPPacket struct {
	buf     []byte
	n       int
	addr    net.Addr
	udpConn *net.UDPConn
}

// NewBufferPool 创建缓冲区池
func NewBufferPool(bufferSize, poolSize int) *BufferPool {
	return &BufferPool{
		buffers: make(chan []byte, poolSize),
		size:    bufferSize,
		maxSize: poolSize,
	}
}

// Get 获取缓冲区
func (p *BufferPool) Get() []byte {
	select {
	case buf := <-p.buffers:
		p.mu.Lock()
		p.count--
		p.mu.Unlock()
		return buf
	default:
		return make([]byte, p.size)
	}
}

// Put 归还缓冲区
func (p *BufferPool) Put(buf []byte) {
	if len(buf) != p.size {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.count < p.maxSize {
		select {
		case p.buffers <- buf:
			p.count++
		default:
			// 缓冲区池已满，丢弃
		}
	}
}

// Size 获取缓冲区大小
func (p *BufferPool) Size() int {
	return p.size
}

// Count 获取当前缓冲区数量
func (p *BufferPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.count
}

// NetworkStats 网络统计信息
type NetworkStats struct {
	TotalRequests       int64         `json:"totalRequests"`
	SuccessfulRequests  int64         `json:"successfulRequests"`
	FailedRequests      int64         `json:"failedRequests"`
	AverageResponseTime time.Duration `json:"averageResponseTime"`
	TotalBytesIn        int64         `json:"totalBytesIn"`
	TotalBytesOut       int64         `json:"totalBytesOut"`
	RequestRate         float64       `json:"requestRate"`
	LastRequestTime     time.Time     `json:"lastRequestTime"`
	// TCP 详细统计
	TCPRequests    int64 `json:"tcpRequests"`
	TCPConnections int64 `json:"tcpConnections"`
	TCPBytesIn     int64 `json:"tcpBytesIn"`
	TCPBytesOut    int64 `json:"tcpBytesOut"`
	// UDP 详细统计
	UDPRequests int64 `json:"udpRequests"`
	UDPBytesIn  int64 `json:"udpBytesIn"`
	UDPBytesOut int64 `json:"udpBytesOut"`
	// 响应时间分布
	ResponseTime10ms   int64 `json:"responseTime10ms"`   // <10ms
	ResponseTime50ms   int64 `json:"responseTime50ms"`   // <50ms
	ResponseTime100ms  int64 `json:"responseTime100ms"`  // <100ms
	ResponseTime500ms  int64 `json:"responseTime500ms"`  // <500ms
	ResponseTime1s     int64 `json:"responseTime1s"`     // <1s
	ResponseTimeOver1s int64 `json:"responseTimeOver1s"` // >1s
}

// CustomDNSServer 自定义DNS服务器
type CustomDNSServer struct {
	addr         string
	net          string
	handler      dns.Handler
	pool         *WorkerPool
	logger       *common.Logger
	wg           sync.WaitGroup
	shutdown     chan struct{}
	bufPool      *BufferPool
	stats        *NetworkStats
	statsMu      sync.RWMutex
	statsManager *StatsManager
	// 新增字段
	listener   net.Listener   // TCP监听器
	packetConn net.PacketConn // UDP数据包连接
	isShutdown bool           // 服务器是否已关闭
	shutdownMu sync.Mutex     // 关闭操作互斥锁
	// 数据包计数管理
	packetCount   int        // 并发数据包计数
	packetCountMu sync.Mutex // 数据包计数互斥锁
}

// NewCustomDNSServer 创建自定义DNS服务器
func NewCustomDNSServer(addr, net string, handler dns.Handler, pool *WorkerPool, logger *common.Logger) *CustomDNSServer {
	server := &CustomDNSServer{
		addr:     addr,
		net:      net,
		handler:  handler,
		pool:     pool,
		logger:   logger,
		shutdown: make(chan struct{}),
		stats: &NetworkStats{
			LastRequestTime: time.Now(),
		},
		statsManager: NewStatsManager(logger),
	}

	// 为UDP服务器创建缓冲区池
	if net == "udp" {
		// 从配置文件获取参数，与任务队列大小匹配
		workers := common.GetConfigInt("DNS", "DNS_CLIENT_WORKERS", 10000)
		multiplier := common.GetConfigInt("DNS", "DNS_QUEUE_MULTIPLIER", 2)
		poolSize := workers * multiplier

		server.bufPool = NewBufferPool(
			4096,     // DNS消息最大长度
			poolSize, // 与任务队列大小匹配
		)
	}

	return server
}

// ListenAndServe 启动DNS服务器
func (s *CustomDNSServer) ListenAndServe() error {
	s.logger.Info("启动%s DNS服务器，监听地址 %s", s.net, s.addr)

	if s.net == "udp" {
		return s.listenAndServeUDP()
	} else if s.net == "tcp" {
		return s.listenAndServeTCP()
	}

	return nil
}

// listenAndServeUDP 启动UDP DNS服务器
func (s *CustomDNSServer) listenAndServeUDP() error {
	// 尝试创建UDP监听器
	pc, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		return err
	}

	// 保存监听器
	s.packetConn = pc

	// 记录监听成功日志
	s.logger.Info("UDP DNS服务器成功监听: %s", s.addr)

	// 启动处理协程
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handleUDPPackets(pc)
	}()

	// 等待关闭信号
	<-s.shutdown

	// 收到关闭信号后，关闭监听器
	if s.packetConn != nil {
		if err := s.packetConn.Close(); err != nil {
			s.logger.Error("关闭UDP监听器失败: %v", err)
		} else {
			s.logger.Info("UDP监听器已关闭")
		}
		s.packetConn = nil
	}

	return nil
}

// listenAndServeTCP 启动TCP DNS服务器
func (s *CustomDNSServer) listenAndServeTCP() error {
	// 尝试创建TCP监听器
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	// 保存监听器
	s.listener = l

	// 记录监听成功日志
	s.logger.Info("TCP DNS服务器成功监听: %s", s.addr)

	// 启动处理协程
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handleTCPConnections(l)
	}()

	// 等待关闭信号
	<-s.shutdown

	// 收到关闭信号后，关闭监听器
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Error("关闭TCP监听器失败: %v", err)
		} else {
			s.logger.Info("TCP监听器已关闭")
		}
		s.listener = nil
	}

	return nil
}

// handleUDPPackets 处理UDP数据包
func (s *CustomDNSServer) handleUDPPackets(pc net.PacketConn) {
	// 尝试转换为UDPConn以提高效率
	udpConn, ok := pc.(*net.UDPConn)
	if !ok {
		s.logger.Warn("无法转换为UDPConn，使用PacketConn模式")
	}

	// 流量控制参数
	const maxConcurrentPackets = 10000 // 最大并发数据包数

	// 从配置文件获取参数
	workers := common.GetConfigInt("DNS", "DNS_CLIENT_WORKERS", 10000)
	multiplier := common.GetConfigInt("DNS", "DNS_QUEUE_MULTIPLIER", 2)
	channelSize := workers * multiplier

	// 创建任务通道
	taskChan := make(chan *UDPPacket, channelSize)

	// 启动工作协程池
	wg := s.startUDPWorkerPool(pc, udpConn, taskChan)

	for {
		// 检查是否收到关闭信号
		select {
		case <-s.shutdown:
			s.logger.Info("收到关闭信号，退出UDP处理协程")
			// 关闭任务通道
			close(taskChan)
			// 等待所有工作协程退出
			wg.Wait()
			return
		default:
			// 继续处理
		}

		// 流量控制
		if s.getPacketCount() > maxConcurrentPackets {
			// 短暂休眠，降低处理速度
			time.Sleep(time.Millisecond * 1)
			continue
		}

		// 从缓冲区池获取缓冲区
		var buf []byte
		if s.bufPool != nil {
			buf = s.bufPool.Get()
		} else {
			buf = make([]byte, 512) // DNS消息最大长度
		}

		var n int
		var addr net.Addr
		var err error

		// 设置读取超时，以便及时检查shutdown通道
		if udpConn != nil {
			udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			udpAddr := &net.UDPAddr{}
			n, udpAddr, err = udpConn.ReadFromUDP(buf)
			addr = udpAddr
		} else {
			pc.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, addr, err = pc.ReadFrom(buf)
		}

		if err != nil {
			// 归还缓冲区
			if s.bufPool != nil {
				s.bufPool.Put(buf)
			}

			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时错误，继续尝试
				continue
			}

			// 检查是否是监听器关闭导致的错误
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// 临时错误，继续尝试
				s.logger.Error("读取UDP数据包失败: %v", err)
				continue
			}

			// 监听器关闭或其他严重错误，退出
			s.logger.Info("UDP监听器已关闭，退出处理协程")
			// 关闭任务通道
			close(taskChan)
			// 等待所有工作协程退出
			wg.Wait()
			return
		}

		// 增加数据包计数
		s.incrementPacketCount()

		// 创建任务并发送到通道
		task := &UDPPacket{
			buf:     buf,
			n:       n,
			addr:    addr,
			udpConn: udpConn,
		}

		select {
		case <-s.shutdown:
			// 收到关闭信号，退出
			s.logger.Info("收到关闭信号，退出UDP处理协程")
			// 归还缓冲区
			if s.bufPool != nil {
				s.bufPool.Put(buf)
			}
			// 减少数据包计数
			s.decrementPacketCount()
			// 关闭任务通道
			close(taskChan)
			// 等待所有工作协程退出
			wg.Wait()
			return
		case taskChan <- task:
			// 任务发送成功
		default:
			// 通道已满，丢弃任务
			s.logger.Warn("UDP任务通道已满，丢弃数据包")
			// 归还缓冲区
			if s.bufPool != nil {
				s.bufPool.Put(buf)
			}
			// 减少数据包计数
			s.decrementPacketCount()
		}
	}
}

// handleTCPConnections 处理TCP连接
func (s *CustomDNSServer) handleTCPConnections(l net.Listener) {
	for {
		// 检查是否收到关闭信号
		select {
		case <-s.shutdown:
			s.logger.Info("收到关闭信号，退出TCP处理协程")
			return
		default:
			// 继续处理
		}

		// 设置接受连接超时，以便及时检查shutdown通道
		if tcpListener, ok := l.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
		}

		conn, err := l.Accept()
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时错误，继续尝试
				continue
			}

			// 检查是否是监听器关闭导致的错误
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// 临时错误，继续尝试
				s.logger.Error("接受TCP连接失败: %v", err)
				continue
			}

			// 监听器关闭或其他严重错误，退出
			s.logger.Info("TCP监听器已关闭，退出处理协程")
			return
		}

		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			defer c.Close()
			s.handleTCPConnection(c)
		}(conn)
	}
}

// handleTCPConnection 处理单个TCP连接
func (s *CustomDNSServer) handleTCPConnection(conn net.Conn) {
	// 提取客户端IP地址
	clientIP := conn.RemoteAddr().String()
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		clientIP = tcpAddr.IP.String()
	}

	// 设置TCP连接参数
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true) // 禁用Nagle算法，提高实时性
	}

	// 创建TCP响应 writer
	writer := &TCPResponseWriter{
		conn:     conn,
		clientIP: clientIP,
	}

	// 连接状态跟踪
	startTime := time.Now()
	messageCount := 0

	// 处理多个DNS消息（长连接）
	for {
		// 检查是否收到关闭信号
		select {
		case <-s.shutdown:
			s.logger.Info("收到关闭信号，关闭TCP连接: %v", clientIP)
			return
		default:
			// 继续处理
		}

		// 设置读取超时，以便及时检查shutdown通道
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

		// 读取DNS消息长度
		lengthBuf := make([]byte, 2)
		n, err := conn.Read(lengthBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 连接超时，正常关闭
				s.logger.Debug("TCP连接超时: %v, 处理消息数: %d, 连接时长: %v", clientIP, messageCount, time.Since(startTime))
			} else if err.Error() != "EOF" {
				s.logger.Error("读取DNS消息长度失败: %v", err)
			} else {
				// s.logger.Debug("TCP连接正常关闭: %v, 处理消息数: %d, 连接时长: %v", clientIP, messageCount, time.Since(startTime))
			}
			return
		}

		if n != 2 {
			s.logger.Error("读取DNS消息长度不完整")
			return
		}

		// 解析消息长度
		length := uint16(lengthBuf[0])<<8 | uint16(lengthBuf[1])
		if length > 4096 {
			s.logger.Error("DNS消息长度超过限制: %d", length)
			return
		}

		// 读取DNS消息
		buf := make([]byte, length)
		n, err = conn.Read(buf)
		if err != nil {
			s.logger.Error("读取DNS消息失败: %v", err)
			return
		}

		if n != int(length) {
			s.logger.Error("读取DNS消息不完整")
			return
		}

		// 解析DNS消息
		var msg dns.Msg
		if err := msg.Unpack(buf); err != nil {
			s.logger.Error("解析DNS消息失败: %v", err)
			continue
		}

		// 增加消息计数
		messageCount++

		// 记录开始时间
		startTime := time.Now()
		// 提交到协程池处理
		s.pool.SubmitWithClientIP(s.handler, writer, &msg, clientIP)
		// 计算响应时间
		responseTime := time.Since(startTime)
		// 更新统计信息
		s.updateStats(n, 0, true, responseTime)
	}

	// 关闭连接（实际上会由defer语句处理）

}

// updateStats 更新网络统计信息
func (s *CustomDNSServer) updateStats(bytesIn, bytesOut int, success bool, responseTime time.Duration) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.stats.TotalRequests++
	s.stats.TotalBytesIn += int64(bytesIn)
	s.stats.TotalBytesOut += int64(bytesOut)
	s.stats.LastRequestTime = time.Now()

	if success {
		s.stats.SuccessfulRequests++
	} else {
		s.stats.FailedRequests++
	}

	// 根据网络类型更新详细统计
	if s.net == "tcp" {
		s.stats.TCPRequests++
		s.stats.TCPBytesIn += int64(bytesIn)
		s.stats.TCPBytesOut += int64(bytesOut)
	} else if s.net == "udp" {
		s.stats.UDPRequests++
		s.stats.UDPBytesIn += int64(bytesIn)
		s.stats.UDPBytesOut += int64(bytesOut)
	}

	// 简单计算请求率（最近10秒的请求数）
	elapsed := time.Since(s.stats.LastRequestTime)
	if elapsed > 0 {
		s.stats.RequestRate = float64(s.stats.TotalRequests) / elapsed.Seconds()
	}

	// 使用统计管理器更新统计信息
	if s.statsManager != nil {
		s.statsManager.UpdateNetworkStats(bytesIn, bytesOut, success, responseTime)
	}
}

// GetStats 获取网络统计信息
func (s *CustomDNSServer) GetStats() *NetworkStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	// 创建统计信息副本
	stats := *s.stats
	return &stats
}

// GetStatsManager 获取统计管理器
func (s *CustomDNSServer) GetStatsManager() *StatsManager {
	return s.statsManager
}

// incrementPacketCount 增加数据包计数
func (s *CustomDNSServer) incrementPacketCount() {
	s.packetCountMu.Lock()
	s.packetCount++
	s.packetCountMu.Unlock()
}

// decrementPacketCount 减少数据包计数
func (s *CustomDNSServer) decrementPacketCount() {
	s.packetCountMu.Lock()
	s.packetCount--
	s.packetCountMu.Unlock()
}

// getPacketCount 获取当前数据包计数
func (s *CustomDNSServer) getPacketCount() int {
	s.packetCountMu.Lock()
	defer s.packetCountMu.Unlock()
	return s.packetCount
}

// startUDPWorkerPool 启动UDP工作协程池
func (s *CustomDNSServer) startUDPWorkerPool(pc net.PacketConn, udpConn *net.UDPConn, taskChan chan *UDPPacket) *sync.WaitGroup {
	// 从配置文件获取参数
	workers := common.GetConfigInt("DNS", "DNS_CLIENT_WORKERS", 10000)
	multiplier := common.GetConfigInt("DNS", "DNS_QUEUE_MULTIPLIER", 2)
	channelSize := workers * multiplier

	// 计算工作协程数量
	cpuCount := runtime.NumCPU()
	workerCount := cpuCount * 4
	if workerCount > workers {
		workerCount = workers
	}
	if workerCount < 100 {
		workerCount = 1000
	}

	s.logger.Info("启动UDP处理工作协程，数量: %d，任务通道大小: %d", workerCount, channelSize)

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go s.udpWorker(pc, udpConn, taskChan, &wg)
	}

	return &wg
}

// udpWorker UDP工作协程
func (s *CustomDNSServer) udpWorker(pc net.PacketConn, udpConn *net.UDPConn, taskChan chan *UDPPacket, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		case task, ok := <-taskChan:
			if !ok {
				return
			}

			// 处理UDP数据包
			s.processUDPPacket(pc, udpConn, task)
		}
	}
}

// processUDPPacket 处理UDP数据包
func (s *CustomDNSServer) processUDPPacket(pc net.PacketConn, udpConn *net.UDPConn, task *UDPPacket) {
	// 提取客户端IP地址
	clientIP := task.addr.String()
	if udpAddr, ok := task.addr.(*net.UDPAddr); ok {
		clientIP = udpAddr.IP.String()
	} else if tcpAddr, ok := task.addr.(*net.TCPAddr); ok {
		clientIP = tcpAddr.IP.String()
	}

	// 解析DNS消息
	var msg dns.Msg
	if err := msg.Unpack(task.buf[:task.n]); err != nil {
		s.logger.Error("解析DNS消息失败: %v", err)
		// 归还缓冲区
		if s.bufPool != nil {
			s.bufPool.Put(task.buf)
		}
		// 减少数据包计数
		s.decrementPacketCount()
		return
	}

	// 创建UDP响应 writer
	writer := &UDPResponseWriter{
		pc:       pc,
		udpConn:  udpConn,
		addr:     task.addr,
		clientIP: clientIP,
	}

	// 记录开始时间
	startTime := time.Now()
	// 提交到协程池处理
	s.pool.SubmitWithClientIP(s.handler, writer, &msg, clientIP)
	// 计算响应时间
	responseTime := time.Since(startTime)
	// 更新统计信息
	s.updateStats(task.n, 0, true, responseTime)

	// 处理完成后归还缓冲区
	if s.bufPool != nil {
		s.bufPool.Put(task.buf)
	}
	// 减少数据包计数
	s.decrementPacketCount()
}

// Close 关闭DNS服务器
func (s *CustomDNSServer) Close() {
	s.Shutdown()
}

// Shutdown 关闭DNS服务器（与Close方法功能相同，用于兼容server_manager.go中的调用）
func (s *CustomDNSServer) Shutdown() error {
	s.shutdownMu.Lock()

	// 检查是否已经关闭
	if s.isShutdown {
		s.shutdownMu.Unlock()
		s.logger.Info("DNS服务器已经关闭")
		return nil
	}

	s.logger.Info("开始关闭DNS服务器...")

	// 标记服务器为关闭状态
	s.isShutdown = true

	// 关闭shutdown通道，通知所有goroutine退出
	close(s.shutdown)

	// 解锁，允许其他操作执行
	s.shutdownMu.Unlock()

	// 等待所有goroutine退出
	s.wg.Wait()
	s.logger.Info("所有goroutine已退出")

	// 关闭协程池

	if s.pool != nil {
		s.pool.Close()
		s.logger.Info("协程池已关闭")
		// 将 pool 设置为 nil，避免重复关闭
		s.pool = nil
	}

	s.logger.Info("DNS服务器已成功关闭")
	return nil
}

// UDPResponseWriter UDP响应 writer
type UDPResponseWriter struct {
	pc       net.PacketConn
	udpConn  *net.UDPConn
	addr     net.Addr
	clientIP string
}

// WriteMsg 写入DNS响应
func (w *UDPResponseWriter) WriteMsg(m *dns.Msg) error {
	buf, err := m.Pack()
	if err != nil {
		return err
	}

	// 使用UDPConn的WriteToUDP方法提高效率
	if w.udpConn != nil {
		if udpAddr, ok := w.addr.(*net.UDPAddr); ok {
			_, err = w.udpConn.WriteToUDP(buf, udpAddr)
			return err
		}
	}

	_, err = w.pc.WriteTo(buf, w.addr)
	return err
}

// RemoteAddr 返回远程地址
func (w *UDPResponseWriter) RemoteAddr() net.Addr {
	return w.addr
}

// LocalAddr 返回本地地址
func (w *UDPResponseWriter) LocalAddr() net.Addr {
	return w.pc.LocalAddr()
}

// Write 写入数据
func (w *UDPResponseWriter) Write(b []byte) (int, error) {
	// 使用UDPConn的WriteToUDP方法提高效率
	if w.udpConn != nil {
		if udpAddr, ok := w.addr.(*net.UDPAddr); ok {
			n, err := w.udpConn.WriteToUDP(b, udpAddr)
			return n, err
		}
	}

	return w.pc.WriteTo(b, w.addr)
}

// Close 关闭连接
func (w *UDPResponseWriter) Close() error {
	return nil
}

// TsigStatus 返回TSIG状态
func (w *UDPResponseWriter) TsigStatus() error {
	return nil
}

// TsigTimersOnly 返回TSIG计时器状态
func (w *UDPResponseWriter) TsigTimersOnly(bool) {
}

// Hijack 劫持连接
func (w *UDPResponseWriter) Hijack() {
}

// TCPResponseWriter TCP响应 writer
type TCPResponseWriter struct {
	conn     net.Conn
	clientIP string
}

// WriteMsg 写入DNS响应
func (w *TCPResponseWriter) WriteMsg(m *dns.Msg) error {
	buf, err := m.Pack()
	if err != nil {
		return err
	}

	// 写入消息长度
	length := uint16(len(buf))
	if _, err := w.conn.Write([]byte{byte(length >> 8), byte(length)}); err != nil {
		return err
	}

	_, err = w.conn.Write(buf)
	return err
}

// RemoteAddr 返回远程地址
func (w *TCPResponseWriter) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

// LocalAddr 返回本地地址
func (w *TCPResponseWriter) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

// Write 写入数据
func (w *TCPResponseWriter) Write(b []byte) (int, error) {
	return w.conn.Write(b)
}

// Close 关闭连接
func (w *TCPResponseWriter) Close() error {
	return w.conn.Close()
}

// TsigStatus 返回TSIG状态
func (w *TCPResponseWriter) TsigStatus() error {
	return nil
}

// TsigTimersOnly 返回TSIG计时器状态
func (w *TCPResponseWriter) TsigTimersOnly(bool) {
}

// Hijack 劫持连接
func (w *TCPResponseWriter) Hijack() {
}
