// core/sdns/customdnsserver.go

package sdns

import (
	"net"
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

// TCPConnectionPool TCP连接池
type TCPConnectionPool struct {
	mu           sync.RWMutex
	idleConns    chan *tcpConnWrapper
	maxConns     int
	maxIdleConns int
	connTimeout  time.Duration
	idleTimeout  time.Duration
	totalConns   int
	stats        *TCPConnectionStats
}

// tcpConnWrapper TCP连接包装器
type tcpConnWrapper struct {
	conn     net.Conn
	lastUsed time.Time
	pool     *TCPConnectionPool
	inUse    bool
}

// TCPConnectionStats TCP连接池统计信息
type TCPConnectionStats struct {
	TotalConns      int           `json:"totalConns"`
	IdleConns       int           `json:"idleConns"`
	ActiveConns     int           `json:"activeConns"`
	PoolHits        int64         `json:"poolHits"`
	PoolMisses      int64         `json:"poolMisses"`
	ConnCreated     int64         `json:"connCreated"`
	ConnClosed      int64         `json:"connClosed"`
	ConnReused      int64         `json:"connReused"`
	AverageIdleTime time.Duration `json:"averageIdleTime"`
}

// NewTCPConnectionPool 创建TCP连接池
func NewTCPConnectionPool(maxConns, maxIdleConns int, connTimeout, idleTimeout time.Duration) *TCPConnectionPool {
	pool := &TCPConnectionPool{
		idleConns:    make(chan *tcpConnWrapper, maxIdleConns),
		maxConns:     maxConns,
		maxIdleConns: maxIdleConns,
		connTimeout:  connTimeout,
		idleTimeout:  idleTimeout,
		stats:        &TCPConnectionStats{},
	}

	// 启动连接清理 goroutine
	go pool.cleanupIdleConns()

	return pool
}

// Get 获取TCP连接
func (p *TCPConnectionPool) Get() (*tcpConnWrapper, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 尝试从空闲连接池获取
	select {
	case conn := <-p.idleConns:
		// 检查连接是否超时
		if time.Since(conn.lastUsed) > p.idleTimeout {
			p.closeConn(conn)
			p.stats.PoolMisses++
			return p.createNewConn()
		}

		conn.inUse = true
		conn.lastUsed = time.Now()
		p.stats.PoolHits++
		p.stats.ConnReused++
		p.stats.ActiveConns++
		p.stats.IdleConns--
		return conn, nil
	default:
		// 没有空闲连接，创建新连接
		p.stats.PoolMisses++
		return p.createNewConn()
	}
}

// createNewConn 创建新的TCP连接
func (p *TCPConnectionPool) createNewConn() (*tcpConnWrapper, error) {
	if p.totalConns >= p.maxConns {
		return nil, nil
	}

	p.totalConns++
	p.stats.ConnCreated++
	p.stats.ActiveConns++

	return &tcpConnWrapper{
		pool:     p,
		lastUsed: time.Now(),
		inUse:    true,
	}, nil
}

// Put 归还TCP连接到池
func (p *TCPConnectionPool) Put(conn *tcpConnWrapper) {
	if conn == nil || conn.conn == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	conn.inUse = false
	conn.lastUsed = time.Now()

	// 检查连接是否有效
	if p.isConnValid(conn) {
		// 尝试放入空闲连接池
		select {
		case p.idleConns <- conn:
			p.stats.ActiveConns--
			p.stats.IdleConns++
			return
		default:
			// 空闲连接池已满，关闭连接
			p.closeConn(conn)
			return
		}
	}

	// 连接无效，关闭连接
	p.closeConn(conn)
}

// isConnValid 检查连接是否有效
func (p *TCPConnectionPool) isConnValid(conn *tcpConnWrapper) bool {
	// 检查连接是否关闭
	if conn == nil || conn.conn == nil {
		return false
	}

	// 检查连接是否超时
	if time.Since(conn.lastUsed) > p.idleTimeout {
		return false
	}

	return true
}

// closeConn 关闭TCP连接
func (p *TCPConnectionPool) closeConn(conn *tcpConnWrapper) {
	if conn == nil || conn.conn == nil {
		return
	}

	conn.conn.Close()
	p.totalConns--
	p.stats.ConnClosed++
	p.stats.ActiveConns--
}

// cleanupIdleConns 清理空闲连接
func (p *TCPConnectionPool) cleanupIdleConns() {
	ticker := time.NewTicker(p.idleTimeout / 2)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanupExpiredConns()
	}
}

// cleanupExpiredConns 清理过期的空闲连接
func (p *TCPConnectionPool) cleanupExpiredConns() {
	p.mu.Lock()
	defer p.mu.Unlock()

	expired := make([]*tcpConnWrapper, 0)

	// 检查空闲连接是否过期
	for {
		select {
		case conn := <-p.idleConns:
			if time.Since(conn.lastUsed) > p.idleTimeout {
				expired = append(expired, conn)
			} else {
				// 连接未过期，放回池
				p.idleConns <- conn
			}
		default:
			break
		}
	}

	// 关闭过期连接
	for _, conn := range expired {
		p.closeConn(conn)
		p.stats.IdleConns--
	}
}

// GetStats 获取TCP连接池统计信息
func (p *TCPConnectionPool) GetStats() *TCPConnectionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 创建统计信息副本
	stats := *p.stats
	stats.TotalConns = p.totalConns
	stats.IdleConns = len(p.idleConns)
	stats.ActiveConns = p.totalConns - len(p.idleConns)

	return &stats
}

// Close 关闭TCP连接池
func (p *TCPConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭所有空闲连接
	for {
		select {
		case conn := <-p.idleConns:
			p.closeConn(conn)
		default:
			break
		}
	}
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
	tcpPool      *TCPConnectionPool
	bufPool      *BufferPool
	stats        *NetworkStats
	statsMu      sync.RWMutex
	statsManager *StatsManager
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

	// 为TCP服务器创建连接池
	if net == "tcp" {
		server.tcpPool = NewTCPConnectionPool(
			3000,           // 最大连接数，支持高并发
			800,            // 最大空闲连接数，支持高并发
			30*time.Second, // 连接超时
			60*time.Second, // 空闲超时
		)
	} else if net == "udp" {
		// 为UDP服务器创建缓冲区池
		server.bufPool = NewBufferPool(
			512,   // DNS消息最大长度
			12000, // 缓冲区池大小，支持高并发
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
	defer pc.Close()

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
	return nil
}

// listenAndServeTCP 启动TCP DNS服务器
func (s *CustomDNSServer) listenAndServeTCP() error {
	// 尝试创建TCP监听器
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer l.Close()

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
	const batchSize = 100              // 批量处理大小
	var packetCount int
	var packetCountMu sync.Mutex

	for {
		// 流量控制
		packetCountMu.Lock()
		currentCount := packetCount
		packetCountMu.Unlock()

		if currentCount > maxConcurrentPackets {
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

		// 使用UDPConn的ReadFromUDP方法提高效率
		if udpConn != nil {
			udpAddr := &net.UDPAddr{}
			n, udpAddr, err = udpConn.ReadFromUDP(buf)
			addr = udpAddr
		} else {
			n, addr, err = pc.ReadFrom(buf)
		}

		if err != nil {
			// 归还缓冲区
			if s.bufPool != nil {
				s.bufPool.Put(buf)
			}

			select {
			case <-s.shutdown:
				return
			default:
				s.logger.Error("读取UDP数据包失败: %v", err)
				continue
			}
		}

		// 提取客户端IP地址
		clientIP := addr.String()
		if udpAddr, ok := addr.(*net.UDPAddr); ok {
			clientIP = udpAddr.IP.String()
		} else if tcpAddr, ok := addr.(*net.TCPAddr); ok {
			clientIP = tcpAddr.IP.String()
		}

		// 解析DNS消息
		var msg dns.Msg
		if err := msg.Unpack(buf[:n]); err != nil {
			s.logger.Error("解析DNS消息失败: %v", err)
			// 归还缓冲区
			if s.bufPool != nil {
				s.bufPool.Put(buf)
			}
			continue
		}

		// 创建UDP响应 writer
		writer := &UDPResponseWriter{
			pc:       pc,
			udpConn:  udpConn,
			addr:     addr,
			clientIP: clientIP,
		}

		// 增加数据包计数
		packetCountMu.Lock()
		packetCount++
		packetCountMu.Unlock()

		// 提交到协程池处理（使用闭包保存缓冲区引用）
		go func(buf []byte, n int) {
			defer func() {
				// 处理完成后归还缓冲区
				if s.bufPool != nil {
					s.bufPool.Put(buf)
				}
				// 减少数据包计数
				packetCountMu.Lock()
				packetCount--
				packetCountMu.Unlock()
			}()
			// 记录开始时间
			startTime := time.Now()
			// 提交到协程池处理
			s.pool.SubmitWithClientIP(s.handler, writer, &msg, clientIP)
			// 计算响应时间
			responseTime := time.Since(startTime)
			// 更新统计信息
			s.updateStats(n, 0, true, responseTime)
		}(buf, n)
	}
}

// handleTCPConnections 处理TCP连接
func (s *CustomDNSServer) handleTCPConnections(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return
			default:
				s.logger.Error("接受TCP连接失败: %v", err)
				continue
			}
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
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

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
		// 读取DNS消息长度
		lengthBuf := make([]byte, 2)
		n, err := conn.Read(lengthBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 连接超时，正常关闭
				s.logger.Info("TCP连接超时: %v, 处理消息数: %d, 连接时长: %v", clientIP, messageCount, time.Since(startTime))
			} else if err.Error() != "EOF" {
				s.logger.Error("读取DNS消息长度失败: %v", err)
			} else {
				s.logger.Info("TCP连接正常关闭: %v, 处理消息数: %d, 连接时长: %v", clientIP, messageCount, time.Since(startTime))
			}
			break
		}

		if n != 2 {
			s.logger.Error("读取DNS消息长度不完整")
			break
		}

		// 解析消息长度
		length := uint16(lengthBuf[0])<<8 | uint16(lengthBuf[1])
		if length > 4096 {
			s.logger.Error("DNS消息长度超过限制: %d", length)
			break
		}

		// 读取DNS消息
		buf := make([]byte, length)
		n, err = conn.Read(buf)
		if err != nil {
			s.logger.Error("读取DNS消息失败: %v", err)
			break
		}

		if n != int(length) {
			s.logger.Error("读取DNS消息不完整")
			break
		}

		// 解析DNS消息
		var msg dns.Msg
		if err := msg.Unpack(buf); err != nil {
			s.logger.Error("解析DNS消息失败: %v", err)
			continue
		}

		// 更新读取超时
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

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

	// 关闭连接
	conn.Close()
	s.logger.Info("TCP连接已关闭: %v, 处理消息数: %d, 连接时长: %v", clientIP, messageCount, time.Since(startTime))
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

// Close 关闭DNS服务器
func (s *CustomDNSServer) Close() {
	close(s.shutdown)
	s.wg.Wait()

	// 关闭TCP连接池
	if s.tcpPool != nil {
		s.tcpPool.Close()
	}
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
