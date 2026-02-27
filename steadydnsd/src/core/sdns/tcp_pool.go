// SteadyDNS - DNS服务器实现
// 版权所有 (C) 2024 SteadyDNS contributors
//
// 本程序是自由软件：您可以根据自由软件基金会发布的GNU Affero通用公共许可证
// 的条款重新分发和/或修改它，无论是许可证的第3版，还是（根据您的选择）任何更高版本。
//
// 本程序的发布是希望它有用，但没有任何担保；甚至没有对适销性或特定用途适用性的暗示担保。
// 有关更多详细信息，请参阅GNU Affero通用公共许可证。
//
// 您应该已经收到一份GNU Affero通用公共许可证的副本。
// 如果没有，请参阅<https://www.gnu.org/licenses/>。

// core/sdns/tcp_pool.go

package sdns

import (
	"SteadyDNS/core/common"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

// ConnectionHealth 表示连接的健康状态
type ConnectionHealth int

const (
	// ConnectionHealthHealthy 表示连接健康
	ConnectionHealthHealthy ConnectionHealth = iota
	// ConnectionHealthDegraded 表示连接降级（性能下降但仍可用）
	ConnectionHealthDegraded
	// ConnectionHealthUnhealthy 表示连接异常（不可用）
	ConnectionHealthUnhealthy
)

// String 返回健康状态的字符串表示
func (h ConnectionHealth) String() string {
	switch h {
	case ConnectionHealthHealthy:
		return "healthy"
	case ConnectionHealthDegraded:
		return "degraded"
	case ConnectionHealthUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// PoolConfig 连接池配置参数
type PoolConfig struct {
	// MaxConnectionsPerServer 每服务器最大连接数
	MaxConnectionsPerServer int
	// MaxPipelineDepth 每连接最大管道深度
	MaxPipelineDepth int
	// IdleTimeout 连接空闲超时时间
	IdleTimeout time.Duration
	// ConnectTimeout 连接建立超时时间
	ConnectTimeout time.Duration
	// MaxConnectionLifetime 连接最大寿命
	MaxConnectionLifetime time.Duration
	// HealthCheckInterval 健康检查间隔
	HealthCheckInterval time.Duration
	// OutOfOrderThreshold 乱序率阈值（百分比）
	OutOfOrderThreshold float64
}

// DefaultPoolConfig 返回默认连接池配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnectionsPerServer: 2, // 每服务器最大连接数改为2
		MaxPipelineDepth:        100,
		IdleTimeout:             30 * time.Second,
		ConnectTimeout:          5 * time.Second,
		MaxConnectionLifetime:   10 * time.Minute,
		HealthCheckInterval:     30 * time.Second,
		OutOfOrderThreshold:     10.0, // 10% 乱序率阈值
	}
}

// InflightQuery 表示正在进行的查询
type InflightQuery struct {
	Msg        *dns.Msg
	Response   chan *dns.Msg
	Error      chan error
	SentAt     time.Time
	MsgID      uint16 // 管道化使用的消息ID
	OriginalID uint16 // 原始消息ID（用于恢复响应）
}

// PooledConnection 表示池中的TCP连接
type PooledConnection struct {
	// conn 是底层的TCP连接
	conn net.Conn
	// dnsConn 是DNS协议的TCP连接包装器
	dnsConn *dns.Conn
	// inflight 存储正在进行的查询映射（msg ID -> InflightQuery）
	inflight map[uint16]*InflightQuery
	// inflightMu 保护inflight映射的锁
	inflightMu sync.RWMutex
	// health 表示连接的健康状态
	health ConnectionHealth
	// healthMu 保护health字段的锁
	healthMu sync.RWMutex
	// createdAt 记录连接创建时间
	createdAt time.Time
	// lastUsedAt 记录最后使用时间
	lastUsedAt time.Time
	// lastUsedMu 保护lastUsedAt的锁
	lastUsedMu sync.RWMutex
	// serverAddr 服务器地址
	serverAddr string
	// msgID 当前消息ID计数器
	msgID uint32
	// pipelineDepth 当前管道深度
	pipelineDepth int32
	// maxPipelineDepth 最大管道深度限制
	maxPipelineDepth int32
	// stats 管道化统计信息
	stats *PipelineStats
	// stopRead 停止读取协程的信号通道
	stopRead chan struct{}
	// closed 连接是否已关闭
	closed int32
	// closeMu 保护关闭操作的锁
	closeMu sync.Mutex
	// wg 等待读取协程完成
	wg sync.WaitGroup
}

// PipelineStats 管道化统计信息
type PipelineStats struct {
	// TotalQueries 总查询数
	TotalQueries uint64
	// OutOfOrderResponses 乱序响应数
	OutOfOrderResponses uint64
	// ExpectedMsgID 期望的下一个消息ID（用于乱序检测）
	ExpectedMsgID uint32
	// expectedMu 保护ExpectedMsgID的锁
	expectedMu sync.RWMutex
	// PipelineDepthHistory 管道深度历史记录
	PipelineDepthHistory []int32
	// historyMu 保护历史记录的锁
	historyMu sync.RWMutex
	// lastAdjustTime 上次调整管道深度的时间
	lastAdjustTime time.Time
	// adjustMu 保护调整时间的锁
	adjustMu sync.RWMutex
}

// NewPipelineStats 创建新的管道化统计信息
func NewPipelineStats() *PipelineStats {
	return &PipelineStats{
		PipelineDepthHistory: make([]int32, 0, 100),
		lastAdjustTime:       time.Now(),
	}
}

// RecordQuery 记录查询
func (s *PipelineStats) RecordQuery() {
	atomic.AddUint64(&s.TotalQueries, 1)
}

// RecordOutOfOrder 记录乱序响应
func (s *PipelineStats) RecordOutOfOrder() {
	atomic.AddUint64(&s.OutOfOrderResponses, 1)
}

// GetOutOfOrderRate 获取乱序率（百分比）
func (s *PipelineStats) GetOutOfOrderRate() float64 {
	total := atomic.LoadUint64(&s.TotalQueries)
	if total == 0 {
		return 0.0
	}
	outOfOrder := atomic.LoadUint64(&s.OutOfOrderResponses)
	return float64(outOfOrder) * 100.0 / float64(total)
}

// UpdateExpectedMsgID 更新期望的消息ID
func (s *PipelineStats) UpdateExpectedMsgID(msgID uint16) {
	s.expectedMu.Lock()
	s.ExpectedMsgID = uint32(msgID)
	s.expectedMu.Unlock()
}

// GetExpectedMsgID 获取期望的消息ID
func (s *PipelineStats) GetExpectedMsgID() uint32 {
	s.expectedMu.RLock()
	defer s.expectedMu.RUnlock()
	return s.ExpectedMsgID
}

// RecordPipelineDepth 记录管道深度
func (s *PipelineStats) RecordPipelineDepth(depth int32) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	s.PipelineDepthHistory = append(s.PipelineDepthHistory, depth)
	// 只保留最近100条记录
	if len(s.PipelineDepthHistory) > 100 {
		s.PipelineDepthHistory = s.PipelineDepthHistory[len(s.PipelineDepthHistory)-100:]
	}
}

// GetAveragePipelineDepth 获取平均管道深度
func (s *PipelineStats) GetAveragePipelineDepth() float64 {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()
	if len(s.PipelineDepthHistory) == 0 {
		return 0.0
	}
	var sum int64
	for _, d := range s.PipelineDepthHistory {
		sum += int64(d)
	}
	return float64(sum) / float64(len(s.PipelineDepthHistory))
}

// CanAdjust 检查是否可以调整管道深度
func (s *PipelineStats) CanAdjust() bool {
	s.adjustMu.RLock()
	defer s.adjustMu.RUnlock()
	return time.Since(s.lastAdjustTime) > 5*time.Second
}

// MarkAdjusted 标记已调整管道深度
func (s *PipelineStats) MarkAdjusted() {
	s.adjustMu.Lock()
	s.lastAdjustTime = time.Now()
	s.adjustMu.Unlock()
}

// ServerPool 表示单个服务器的连接池
type ServerPool struct {
	// connections 连接列表
	connections []*PooledConnection
	// connMu 保护connections的锁
	connMu sync.RWMutex
	// serverAddr 服务器地址
	serverAddr string
	// nextIndex 下一个要使用的连接索引（轮询）
	nextIndex uint32
}

// createRequest 连接创建请求
type createRequest struct {
	serverAddr string
}

// TCPConnectionPool TCP连接池
type TCPConnectionPool struct {
	// pools 服务器地址到连接池的映射
	pools map[string]*ServerPool
	// poolsMu 保护pools的锁
	poolsMu sync.RWMutex
	// config 连接池配置
	config *PoolConfig
	// stopCleanup 停止清理协程的信号通道
	stopCleanup chan struct{}
	// stopHealthCheck 停止健康检查协程的信号通道
	stopHealthCheck chan struct{}
	// createQueue 连接创建请求队列
	createQueue chan createRequest
	// creating 标记正在创建连接的服务器（防止重复创建）
	creating map[string]bool
	// creatingMu 保护creating的锁
	creatingMu sync.Mutex
	// wg 等待后台协程完成
	wg sync.WaitGroup
	// closed 连接池是否已关闭
	closed int32
	// logger 日志记录器
	logger *common.Logger
}

// NewTCPConnectionPool 创建新的TCP连接池
//
// 参数:
//   - config: 连接池配置，如果为nil则使用默认配置
//
// 返回:
//   - *TCPConnectionPool: 创建的连接池实例
func NewTCPConnectionPool(config *PoolConfig) *TCPConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &TCPConnectionPool{
		pools:           make(map[string]*ServerPool),
		config:          config,
		stopCleanup:     make(chan struct{}),
		stopHealthCheck: make(chan struct{}),
		createQueue:     make(chan createRequest, 100), // 缓冲队列
		creating:        make(map[string]bool),
		logger:          common.NewLogger(),
	}

	// 启动清理协程
	pool.wg.Add(1)
	go pool.connectionCleanup()

	// 启动健康检查协程
	pool.wg.Add(1)
	go pool.healthCheckLoop()

	// 启动连接创建协程（单协程处理所有创建请求）
	pool.wg.Add(1)
	go pool.connectionCreator()

	return pool
}

// GetConnection 获取指定服务器的可用连接
//
// 参数:
//   - serverAddr: 服务器地址（格式：host:port）
//   - ctx: 上下文，用于超时控制
//
// 返回:
//   - *PooledConnection: 获取到的连接
//   - error: 错误信息
func (p *TCPConnectionPool) GetConnection(serverAddr string, ctx context.Context) (*PooledConnection, error) {
	return p.getConnectionInternal(serverAddr, ctx, true)
}

// GetConnectionWithoutCreate 获取指定服务器的可用连接（不创建新连接）
// 用于查询时检查是否有预建立的连接
//
// 参数:
//   - serverAddr: 服务器地址（格式：host:port）
//   - ctx: 上下文，用于超时控制
//
// 返回:
//   - *PooledConnection: 获取到的连接
//   - error: 错误信息
func (p *TCPConnectionPool) GetConnectionWithoutCreate(serverAddr string, ctx context.Context) (*PooledConnection, error) {
	return p.getConnectionInternal(serverAddr, ctx, false)
}

// getConnectionInternal 内部实现：获取指定服务器的可用连接
//
// 参数:
//   - serverAddr: 服务器地址（格式：host:port）
//   - ctx: 上下文，用于超时控制
//   - allowCreate: 是否允许创建新连接
//
// 返回:
//   - *PooledConnection: 获取到的连接
//   - error: 错误信息
func (p *TCPConnectionPool) getConnectionInternal(serverAddr string, ctx context.Context, allowCreate bool) (*PooledConnection, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, fmt.Errorf("connection pool is closed")
	}

	// 获取或创建服务器连接池
	p.poolsMu.RLock()
	serverPool, exists := p.pools[serverAddr]
	p.poolsMu.RUnlock()

	if !exists {
		if !allowCreate {
			return nil, fmt.Errorf("no existing connections for server %s", serverAddr)
		}
		p.poolsMu.Lock()
		serverPool, exists = p.pools[serverAddr]
		if !exists {
			serverPool = &ServerPool{
				connections: make([]*PooledConnection, 0, p.config.MaxConnectionsPerServer),
				serverAddr:  serverAddr,
			}
			p.pools[serverAddr] = serverPool
		}
		p.poolsMu.Unlock()
	}

	// 尝试获取现有连接（轮询负载均衡）
	serverPool.connMu.RLock()
	conns := serverPool.connections
	serverPool.connMu.RUnlock()

	// 轮询选择连接
	if len(conns) > 0 {
		idx := atomic.AddUint32(&serverPool.nextIndex, 1) % uint32(len(conns))
		conn := conns[idx]

		// 检查连接是否可用
		if conn.IsHealthy() && !conn.IsExpired() {
			conn.UpdateLastUsed()
			return conn, nil
		}

		// 尝试其他连接
		for i, c := range conns {
			if i == int(idx) {
				continue
			}
			if c.IsHealthy() && !c.IsExpired() {
				c.UpdateLastUsed()
				return c, nil
			}
		}
	}

	// 如果不允许创建新连接，返回错误
	if !allowCreate {
		return nil, fmt.Errorf("no healthy pre-established connections for server %s", serverAddr)
	}

	// 如果没有可用连接，触发异步补充并返回错误
	// 调用方应该降级到 UDP 查询
	p.EnsureConnections(serverAddr)
	return nil, fmt.Errorf("no healthy connections available for server %s, triggered async creation", serverAddr)
}

// HasHealthyConnection 检查是否有已建立的健康连接（不创建新连接）
//
// 参数:
//   - serverAddr: 服务器地址（格式：host:port）
//
// 返回:
//   - bool: 是否有健康连接
func (p *TCPConnectionPool) HasHealthyConnection(serverAddr string) bool {
	if atomic.LoadInt32(&p.closed) == 1 {
		return false
	}

	p.poolsMu.RLock()
	serverPool, exists := p.pools[serverAddr]
	p.poolsMu.RUnlock()

	if !exists {
		return false
	}

	serverPool.connMu.RLock()
	defer serverPool.connMu.RUnlock()

	// 检查是否有健康且未过期的连接
	for _, conn := range serverPool.connections {
		if conn.IsHealthy() && !conn.IsExpired() {
			return true
		}
	}

	return false
}

// CreateConnection 创建新的TCP连接
//
// 参数:
//   - serverAddr: 服务器地址（格式：host:port）
//
// 返回:
//   - *PooledConnection: 创建的连接
//   - error: 错误信息
func (p *TCPConnectionPool) CreateConnection(serverAddr string) (*PooledConnection, error) {
	// 建立TCP连接
	dialer := &net.Dialer{
		Timeout: p.config.ConnectTimeout,
	}

	conn, err := dialer.Dial("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", serverAddr, err)
	}

	// 设置TCP参数
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true)
	}

	// 创建DNS连接
	dnsConn := &dns.Conn{Conn: conn}

	// 发送验证查询，确保连接真的可用
	msg := new(dns.Msg)
	msg.SetQuestion(".", dns.TypeNS)
	msg.RecursionDesired = true

	// 设置短超时进行验证
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	err = dnsConn.WriteMsg(msg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send validation query to %s: %w", serverAddr, err)
	}

	// 读取响应
	resp, err := dnsConn.ReadMsg()
	if err != nil || resp == nil {
		conn.Close()
		return nil, fmt.Errorf("failed to receive validation response from %s: %w", serverAddr, err)
	}

	// 验证成功，清除deadline
	conn.SetDeadline(time.Time{})

	// 创建池化连接
	pc := &PooledConnection{
		conn:             conn,
		dnsConn:          dnsConn,
		inflight:         make(map[uint16]*InflightQuery),
		health:           ConnectionHealthHealthy,
		createdAt:        time.Now(),
		lastUsedAt:       time.Now(),
		serverAddr:       serverAddr,
		maxPipelineDepth: int32(p.config.MaxPipelineDepth),
		stats:            NewPipelineStats(),
		stopRead:         make(chan struct{}),
	}

	// 启动响应读取协程
	pc.wg.Add(1)
	go pc.readResponses()

	return pc, nil
}

// AddExistingConnection 将已建立的连接添加到连接池
//
// 参数:
//   - serverAddr: 服务器地址（格式：host:port）
//   - conn: 已建立的TCP连接
//   - dnsConn: DNS连接包装器
//
// 返回:
//   - error: 错误信息
func (p *TCPConnectionPool) AddExistingConnection(serverAddr string, conn net.Conn, dnsConn *dns.Conn) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return fmt.Errorf("connection pool is closed")
	}

	// 获取或创建服务器连接池
	p.poolsMu.Lock()
	serverPool, exists := p.pools[serverAddr]
	if !exists {
		serverPool = &ServerPool{
			connections: make([]*PooledConnection, 0, p.config.MaxConnectionsPerServer),
			serverAddr:  serverAddr,
		}
		p.pools[serverAddr] = serverPool
	}
	p.poolsMu.Unlock()

	serverPool.connMu.Lock()
	defer serverPool.connMu.Unlock()

	// 检查是否已达到最大连接数
	if len(serverPool.connections) >= p.config.MaxConnectionsPerServer {
		return fmt.Errorf("max connections reached for server %s", serverAddr)
	}

	// 创建池化连接
	pc := &PooledConnection{
		conn:             conn,
		dnsConn:          dnsConn,
		inflight:         make(map[uint16]*InflightQuery),
		health:           ConnectionHealthHealthy,
		createdAt:        time.Now(),
		lastUsedAt:       time.Now(),
		serverAddr:       serverAddr,
		maxPipelineDepth: int32(p.config.MaxPipelineDepth),
		stats:            NewPipelineStats(),
		stopRead:         make(chan struct{}),
	}

	// 启动响应读取协程
	pc.wg.Add(1)
	go pc.readResponses()

	// 添加到连接池
	serverPool.connections = append(serverPool.connections, pc)

	return nil
}

// Exchange 通过TCP发送DNS查询并接收响应
//
// 参数:
//   - msg: DNS查询消息
//   - serverAddr: 服务器地址
//   - ctx: 上下文，用于超时控制
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
//
// Exchange 通过TCP发送DNS查询并接收响应（使用预建立连接，不创建新连接）
//
// 参数:
//   - msg: DNS查询消息
//   - serverAddr: 服务器地址
//   - ctx: 上下文，用于超时控制
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (p *TCPConnectionPool) Exchange(msg *dns.Msg, serverAddr string, ctx context.Context) (*dns.Msg, error) {
	// 获取连接（不创建新连接，只使用预建立的连接）
	conn, err := p.GetConnectionWithoutCreate(serverAddr, ctx)
	if err != nil {
		return nil, err
	}

	// 执行管道化查询
	return conn.pipelineQuery(msg, ctx)
}

// pipelineQuery 执行管道化DNS查询
//
// 参数:
//   - msg: DNS查询消息
//   - ctx: 上下文，用于超时控制
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (pc *PooledConnection) pipelineQuery(msg *dns.Msg, ctx context.Context) (*dns.Msg, error) {
	if atomic.LoadInt32(&pc.closed) == 1 {
		return nil, fmt.Errorf("connection is closed")
	}

	// 检查健康状态
	if pc.GetHealth() == ConnectionHealthUnhealthy {
		return nil, fmt.Errorf("connection is unhealthy")
	}

	// 保存原始消息ID
	originalID := msg.Id

	// 生成消息ID
	msgID := uint16(atomic.AddUint32(&pc.msgID, 1))
	msg.Id = msgID

	// 创建inflight查询
	query := &InflightQuery{
		Msg:        msg,
		Response:   make(chan *dns.Msg, 1),
		Error:      make(chan error, 1),
		SentAt:     time.Now(),
		MsgID:      msgID,
		OriginalID: originalID,
	}

	// 注册inflight查询
	pc.inflightMu.Lock()
	pc.inflight[msgID] = query
	pc.inflightMu.Unlock()

	// 增加管道深度
	currentDepth := atomic.AddInt32(&pc.pipelineDepth, 1)
	pc.stats.RecordPipelineDepth(currentDepth)

	// 检查并调整管道深度
	pc.adjustPipelineDepth()

	// 发送查询
	if err := pc.dnsConn.WriteMsg(msg); err != nil {
		// 清理inflight
		pc.inflightMu.Lock()
		delete(pc.inflight, msgID)
		pc.inflightMu.Unlock()
		atomic.AddInt32(&pc.pipelineDepth, -1)

		// 标记连接为不健康
		pc.SetHealth(ConnectionHealthUnhealthy)
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	// 记录查询
	pc.stats.RecordQuery()

	// 更新最后使用时间
	pc.UpdateLastUsed()

	// 等待响应或超时
	select {
	case resp := <-query.Response:
		atomic.AddInt32(&pc.pipelineDepth, -1)
		return resp, nil
	case err := <-query.Error:
		atomic.AddInt32(&pc.pipelineDepth, -1)
		return nil, err
	case <-ctx.Done():
		// 清理inflight
		pc.inflightMu.Lock()
		delete(pc.inflight, msgID)
		pc.inflightMu.Unlock()
		atomic.AddInt32(&pc.pipelineDepth, -1)
		return nil, ctx.Err()
	}
}

// readResponses 后台读取响应协程
func (pc *PooledConnection) readResponses() {
	defer pc.wg.Done()

	for {
		select {
		case <-pc.stopRead:
			return
		default:
		}

		// 设置读取超时
		pc.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		// 读取响应
		resp, err := pc.dnsConn.ReadMsg()
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}

			// 检查连接是否已关闭
			if atomic.LoadInt32(&pc.closed) == 1 {
				return
			}

			// 检查是否是连接被对端关闭的错误
			// 这种情况下，不立即标记为不健康，而是让保活循环来重建连接
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// 连接被对端关闭，标记为不健康，让保活循环重建
				pc.SetHealth(ConnectionHealthUnhealthy)
				continue
			}

			// 其他错误，标记连接为不健康
			pc.SetHealth(ConnectionHealthUnhealthy)
			continue
		}

		// 处理响应
		if resp != nil {
			pc.handleResponse(resp)
		}
	}
}

// handleResponse 处理接收到的响应
func (pc *PooledConnection) handleResponse(resp *dns.Msg) {
	msgID := resp.Id

	// 检测乱序
	pc.detectOutOfOrder(msgID)

	// 查找对应的inflight查询
	pc.inflightMu.RLock()
	query, exists := pc.inflight[msgID]
	pc.inflightMu.RUnlock()

	if !exists {
		// 未找到对应的查询，可能是超时或已处理
		return
	}

	// 恢复原始消息ID
	resp.Id = query.OriginalID

	// 发送响应
	select {
	case query.Response <- resp:
		// 从inflight中移除
		pc.inflightMu.Lock()
		delete(pc.inflight, msgID)
		pc.inflightMu.Unlock()
	default:
		// 通道已满，丢弃响应
	}
}

// detectOutOfOrder 检测乱序响应
//
// 参数:
//   - receivedMsgID: 接收到的消息ID
func (pc *PooledConnection) detectOutOfOrder(receivedMsgID uint16) {
	expected := pc.stats.GetExpectedMsgID()

	// 更新期望的消息ID
	pc.stats.UpdateExpectedMsgID(receivedMsgID + 1)

	// 检测乱序
	if uint32(receivedMsgID) != expected && expected != 0 {
		// 记录乱序
		pc.stats.RecordOutOfOrder()
	}
}

// adjustPipelineDepth 根据乱序率调整管道深度
func (pc *PooledConnection) adjustPipelineDepth() {
	// 检查是否可以调整
	if !pc.stats.CanAdjust() {
		return
	}

	outOfOrderRate := pc.stats.GetOutOfOrderRate()
	currentMax := atomic.LoadInt32(&pc.maxPipelineDepth)

	// 根据乱序率调整
	if outOfOrderRate > 10.0 {
		// 严重乱序，降低管道深度
		newDepth := int32(float64(currentMax) * 0.5)
		if newDepth < 10 {
			newDepth = 10
		}
		atomic.StoreInt32(&pc.maxPipelineDepth, newDepth)
		pc.SetHealth(ConnectionHealthDegraded)
		pc.stats.MarkAdjusted()
	} else if outOfOrderRate < 5.0 && currentMax < int32(pc.stats.GetAveragePipelineDepth()) {
		// 轻微乱序，尝试增加管道深度
		newDepth := int32(float64(currentMax) * 1.2)
		if newDepth > int32(pc.stats.GetAveragePipelineDepth()) {
			newDepth = int32(pc.stats.GetAveragePipelineDepth())
		}
		atomic.StoreInt32(&pc.maxPipelineDepth, newDepth)
		pc.stats.MarkAdjusted()
	}
}

// GetHealth 获取连接健康状态
func (pc *PooledConnection) GetHealth() ConnectionHealth {
	pc.healthMu.RLock()
	defer pc.healthMu.RUnlock()
	return pc.health
}

// SetHealth 设置连接健康状态
func (pc *PooledConnection) SetHealth(health ConnectionHealth) {
	pc.healthMu.Lock()
	defer pc.healthMu.Unlock()
	pc.health = health
}

// IsHealthy 检查连接是否健康
func (pc *PooledConnection) IsHealthy() bool {
	health := pc.GetHealth()
	return health == ConnectionHealthHealthy || health == ConnectionHealthDegraded
}

// IsExpired 检查连接是否已过期
func (pc *PooledConnection) IsExpired() bool {
	now := time.Now()

	// 检查空闲超时
	pc.lastUsedMu.RLock()
	idleTime := now.Sub(pc.lastUsedAt)
	pc.lastUsedMu.RUnlock()

	// 获取配置的空闲超时（从连接池配置中获取）
	// 这里使用默认值30秒
	if idleTime > 30*time.Second {
		return true
	}

	// 检查最大寿命
	if now.Sub(pc.createdAt) > 10*time.Minute {
		return true
	}

	return false
}

// UpdateLastUsed 更新最后使用时间
func (pc *PooledConnection) UpdateLastUsed() {
	pc.lastUsedMu.Lock()
	pc.lastUsedAt = time.Now()
	pc.lastUsedMu.Unlock()
}

// Close 关闭连接
func (pc *PooledConnection) Close() error {
	pc.closeMu.Lock()
	defer pc.closeMu.Unlock()

	if !atomic.CompareAndSwapInt32(&pc.closed, 0, 1) {
		return nil // 已经关闭
	}

	// 停止读取协程
	close(pc.stopRead)

	// 等待读取协程完成
	pc.wg.Wait()

	// 关闭底层连接
	if pc.dnsConn != nil {
		pc.dnsConn.Close()
	}

	// 清理所有inflight查询
	pc.inflightMu.Lock()
	for msgID, query := range pc.inflight {
		select {
		case query.Error <- fmt.Errorf("connection closed"):
		default:
		}
		delete(pc.inflight, msgID)
	}
	pc.inflightMu.Unlock()

	return nil
}

// healthCheckLoop 健康检查循环
func (p *TCPConnectionPool) healthCheckLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopHealthCheck:
			return
		case <-ticker.C:
			p.HealthCheck()
		}
	}
}

// HealthCheck 执行连接健康检查
func (p *TCPConnectionPool) HealthCheck() {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	p.poolsMu.RLock()
	pools := make(map[string]*ServerPool)
	for k, v := range p.pools {
		pools[k] = v
	}
	p.poolsMu.RUnlock()

	for _, serverPool := range pools {
		serverPool.connMu.Lock()
		conns := make([]*PooledConnection, len(serverPool.connections))
		copy(conns, serverPool.connections)
		serverPool.connMu.Unlock()

		for _, conn := range conns {
			// 检查连接是否过期
			if conn.IsExpired() {
				conn.SetHealth(ConnectionHealthUnhealthy)
				continue
			}

			// 发送健康检查查询（使用根域名NS记录查询）
			msg := new(dns.Msg)
			msg.SetQuestion(".", dns.TypeNS)
			msg.RecursionDesired = true

			// 设置短超时
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			// 尝试发送查询
			_, err := conn.pipelineQuery(msg, ctx)
			cancel()

			if err != nil {
				// 健康检查失败
				conn.SetHealth(ConnectionHealthUnhealthy)

			} else {
				// 健康检查成功
				if conn.GetHealth() == ConnectionHealthUnhealthy {
					conn.SetHealth(ConnectionHealthHealthy)
				}
			}
		}
	}
}

// EnsureMinConnections 确保指定服务器至少有minCount个健康连接
//
// 参数:
//   - serverAddr: 服务器地址
//   - minCount: 最小连接数量
func (p *TCPConnectionPool) EnsureMinConnections(serverAddr string, minCount int) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	// 获取或创建服务器连接池
	p.poolsMu.RLock()
	serverPool, exists := p.pools[serverAddr]
	p.poolsMu.RUnlock()

	if !exists {
		p.poolsMu.Lock()
		serverPool, exists = p.pools[serverAddr]
		if !exists {
			serverPool = &ServerPool{
				connections: make([]*PooledConnection, 0, p.config.MaxConnectionsPerServer),
				serverAddr:  serverAddr,
			}
			p.pools[serverAddr] = serverPool
		}
		p.poolsMu.Unlock()
	}

	serverPool.connMu.Lock()
	// 统计健康连接数量
	healthyCount := 0
	totalConnections := len(serverPool.connections)
	for _, conn := range serverPool.connections {
		if conn.IsHealthy() && !conn.IsExpired() {
			healthyCount++
		}
	}

	// 计算需要创建的连接数
	neededConnections := minCount - healthyCount
	if neededConnections <= 0 {
		serverPool.connMu.Unlock()
		return
	}

	// 检查是否已达到最大连接数
	if totalConnections >= p.config.MaxConnectionsPerServer {
		neededConnections = p.config.MaxConnectionsPerServer - totalConnections
		if neededConnections <= 0 {
			serverPool.connMu.Unlock()
			return
		}
	}
	serverPool.connMu.Unlock()

	// 创建需要的连接
	p.createConnectionsForServer(serverAddr, neededConnections)
}

// GetConfig 获取连接池配置
//
// 返回:
//   - *PoolConfig: 连接池配置
func (p *TCPConnectionPool) GetConfig() *PoolConfig {
	return p.config
}

// createConnectionsForServer 为指定服务器创建多个连接
//
// 参数:
//   - serverAddr: 服务器地址
//   - count: 需要创建的连接数量
func (p *TCPConnectionPool) createConnectionsForServer(serverAddr string, count int) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	p.poolsMu.RLock()
	serverPool, exists := p.pools[serverAddr]
	p.poolsMu.RUnlock()

	if !exists {
		return
	}

	createdCount := 0
	for i := 0; i < count; i++ {
		serverPool.connMu.Lock()
		// 再次检查是否还需要创建连接（双重检查）
		healthyCount := 0
		for _, conn := range serverPool.connections {
			if conn.IsHealthy() && !conn.IsExpired() {
				healthyCount++
			}
		}

		// 如果健康连接数已达到最大，不需要创建
		if healthyCount >= p.config.MaxConnectionsPerServer {
			serverPool.connMu.Unlock()
			return
		}

		// 检查是否已达到最大连接数（包括不健康连接）
		// 如果总连接数已达最大，先清理不健康连接，再创建新连接
		if len(serverPool.connections) >= p.config.MaxConnectionsPerServer {
			// 清理不健康连接
			healthyConns := make([]*PooledConnection, 0, len(serverPool.connections))
			for _, conn := range serverPool.connections {
				if conn.IsHealthy() && !conn.IsExpired() {
					healthyConns = append(healthyConns, conn)
				} else {
					// 异步关闭不健康连接
					go conn.Close()
				}
			}
			serverPool.connections = healthyConns

			// 清理后再次检查
			if len(serverPool.connections) >= p.config.MaxConnectionsPerServer {
				serverPool.connMu.Unlock()
				return
			}
		}
		serverPool.connMu.Unlock()

		// 创建新连接
		conn, err := p.CreateConnection(serverAddr)
		if err != nil {
			// 创建失败，记录日志但继续尝试创建其他连接
			// 可能是临时网络问题，不要完全放弃
			continue
		}

		// 将新连接添加到池
		serverPool.connMu.Lock()
		serverPool.connections = append(serverPool.connections, conn)
		serverPool.connMu.Unlock()
		createdCount++
	}

	// 如果至少创建了一个连接，记录日志
	if createdCount > 0 {
		// 日志已在调用处记录
	}
}

// connectionCreator 单协程处理所有连接创建请求
// 从createQueue中读取请求，串行创建连接，避免并发创建过多协程
func (p *TCPConnectionPool) connectionCreator() {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopCleanup:
			// 连接池关闭，退出协程
			return
		case req := <-p.createQueue:
			// 处理创建请求
			p.processCreateRequest(req.serverAddr)
		}
	}
}

// processCreateRequest 处理单个服务器的连接创建请求
// 串行创建所有需要的连接，直到达到最大连接数
func (p *TCPConnectionPool) processCreateRequest(serverAddr string) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	// 标记为正在创建（防止重复请求）
	p.creatingMu.Lock()
	if p.creating[serverAddr] {
		p.creatingMu.Unlock()
		return // 已有协程在创建中
	}
	p.creating[serverAddr] = true
	p.creatingMu.Unlock()

	// 完成后清除标记
	defer func() {
		p.creatingMu.Lock()
		delete(p.creating, serverAddr)
		p.creatingMu.Unlock()
	}()

	// 循环创建连接，直到达到最大连接数
	for {
		if atomic.LoadInt32(&p.closed) == 1 {
			return
		}

		// 获取或创建服务器连接池
		p.poolsMu.RLock()
		serverPool, exists := p.pools[serverAddr]
		p.poolsMu.RUnlock()

		if !exists {
			p.poolsMu.Lock()
			serverPool, exists = p.pools[serverAddr]
			if !exists {
				serverPool = &ServerPool{
					connections: make([]*PooledConnection, 0, p.config.MaxConnectionsPerServer),
					serverAddr:  serverAddr,
				}
				p.pools[serverAddr] = serverPool
			}
			p.poolsMu.Unlock()
		}

		serverPool.connMu.Lock()
		// 统计健康连接数量
		healthyCount := 0
		for _, conn := range serverPool.connections {
			if conn.IsHealthy() && !conn.IsExpired() {
				healthyCount++
			}
		}

		// 检查是否需要创建连接
		if healthyCount >= p.config.MaxConnectionsPerServer {
			serverPool.connMu.Unlock()
			return // 已达到最大连接数
		}
		serverPool.connMu.Unlock()

		// 创建新连接
		conn, err := p.CreateConnection(serverAddr)
		if err != nil {
			p.logger.Debug("TCP连接创建失败 %s: %v", serverAddr, err)
			// 创建失败，等待一段时间后重试
			time.Sleep(2 * time.Second)
			continue
		}

		// 将新连接添加到池
		serverPool.connMu.Lock()
		serverPool.connections = append(serverPool.connections, conn)
		serverPool.connMu.Unlock()

		p.logger.Debug("TCP连接创建成功 %s, 当前连接 %d/%d",
			serverAddr, healthyCount+1, p.config.MaxConnectionsPerServer)
	}
}

// EnsureConnections 确保连接池有足够连接（异步，非阻塞）
// 将创建请求发送到队列，由单协程处理
func (p *TCPConnectionPool) EnsureConnections(serverAddr string) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	// 非阻塞发送创建请求
	select {
	case p.createQueue <- createRequest{serverAddr: serverAddr}:
		// 请求已加入队列
	default:
		// 队列已满，忽略（下次查询会再次触发）
	}
}

// connectionCleanup 清理空闲/异常连接
func (p *TCPConnectionPool) connectionCleanup() {
	defer p.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCleanup:
			return
		case <-ticker.C:
			p.cleanupConnections()
		}
	}
}

// cleanupConnections 清理连接
func (p *TCPConnectionPool) cleanupConnections() {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	p.poolsMu.RLock()
	pools := make(map[string]*ServerPool)
	for k, v := range p.pools {
		pools[k] = v
	}
	p.poolsMu.RUnlock()

	for _, serverPool := range pools {
		serverPool.connMu.Lock()

		// 过滤出健康的连接
		healthyConns := make([]*PooledConnection, 0, len(serverPool.connections))
		for _, conn := range serverPool.connections {
			if conn.IsHealthy() && !conn.IsExpired() {
				healthyConns = append(healthyConns, conn)
			} else {
				// 关闭不健康的连接
				go conn.Close()
			}
		}

		serverPool.connections = healthyConns
		serverPool.connMu.Unlock()
	}
}

// Close 关闭连接池
//
// 返回:
//   - error: 错误信息
func (p *TCPConnectionPool) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // 已经关闭
	}

	// 停止后台协程
	close(p.stopCleanup)
	close(p.stopHealthCheck)

	// 等待后台协程完成
	p.wg.Wait()

	// 关闭所有连接
	p.poolsMu.Lock()
	defer p.poolsMu.Unlock()

	for _, serverPool := range p.pools {
		serverPool.connMu.Lock()
		for _, conn := range serverPool.connections {
			conn.Close()
		}
		serverPool.connections = nil
		serverPool.connMu.Unlock()
	}

	p.pools = make(map[string]*ServerPool)

	return nil
}

// GetStats 获取连接池统计信息
//
// 返回:
//   - map[string]interface{}: 统计信息
func (p *TCPConnectionPool) GetStats() map[string]interface{} {
	p.poolsMu.RLock()
	defer p.poolsMu.RUnlock()

	stats := make(map[string]interface{})
	totalConns := 0
	serverStats := make(map[string]map[string]interface{})

	for addr, serverPool := range p.pools {
		serverPool.connMu.RLock()
		connCount := len(serverPool.connections)
		totalConns += connCount

		connDetails := make([]map[string]interface{}, 0, connCount)
		for _, conn := range serverPool.connections {
			detail := map[string]interface{}{
				"health":            conn.GetHealth().String(),
				"created_at":        conn.createdAt,
				"last_used_at":      conn.lastUsedAt,
				"pipeline_depth":    atomic.LoadInt32(&conn.pipelineDepth),
				"max_depth":         atomic.LoadInt32(&conn.maxPipelineDepth),
				"out_of_order_rate": conn.stats.GetOutOfOrderRate(),
			}
			connDetails = append(connDetails, detail)
		}
		serverPool.connMu.RUnlock()

		serverStats[addr] = map[string]interface{}{
			"connection_count": connCount,
			"connections":      connDetails,
		}
	}

	stats["total_connections"] = totalConns
	stats["server_count"] = len(p.pools)
	stats["servers"] = serverStats

	return stats
}

// GetConnectionStats 获取指定连接的统计信息
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - *PipelineStats: 管道化统计信息
//   - bool: 是否找到连接
func (p *TCPConnectionPool) GetConnectionStats(serverAddr string) (*PipelineStats, bool) {
	p.poolsMu.RLock()
	serverPool, exists := p.pools[serverAddr]
	p.poolsMu.RUnlock()

	if !exists || serverPool == nil {
		return nil, false
	}

	serverPool.connMu.RLock()
	defer serverPool.connMu.RUnlock()

	if len(serverPool.connections) == 0 {
		return nil, false
	}

	// 返回第一个连接的统计信息
	return serverPool.connections[0].stats, true
}
