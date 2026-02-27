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
// core/sdns/server_probe.go
// 服务器能力探测模块 - 实现动态协议升级，探测DNS服务器的TCP管道化支持能力

package sdns

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	// TCPProbeTimeout TCP探测超时时间
	TCPProbeTimeout = 5 * time.Second

	// ProbeRetryCount 探测重试次数（共3次尝试）
	ProbeRetryCount = 2

	// ProbeRetryInterval 探测重试间隔
	ProbeRetryInterval = 500 * time.Millisecond

	// MinProbeInterval 每服务器最小探测间隔
	MinProbeInterval = 60 * time.Second

	// MaxConcurrentProbes 全局最大并发探测数
	MaxConcurrentProbes = 10

	// ProbeQueueSize 探测队列长度限制
	ProbeQueueSize = 1000

	// DefaultProbeWorkers 默认探测工作协程数
	DefaultProbeWorkers = 5

	// FullRefreshInterval 全量刷新间隔
	FullRefreshInterval = 5 * time.Minute
)

// ProbeResult 探测结果类型
type ProbeResult int

const (
	// ProbeResultUnknown 探测结果未知
	ProbeResultUnknown ProbeResult = iota
	// ProbeResultSuccess 探测成功
	ProbeResultSuccess
	// ProbeResultFailed 探测失败
	ProbeResultFailed
	// ProbeResultTimeout 探测超时
	ProbeResultTimeout
)

// ServerCapability 服务器能力标志
type ServerCapability int

const (
	// CapabilityNone 无特殊能力
	CapabilityNone ServerCapability = 0
	// CapabilityTCP 支持TCP查询
	CapabilityTCP ServerCapability = 1 << iota
	// CapabilityPipeline 支持TCP管道化
	CapabilityPipeline
	// CapabilityEDNS0 支持EDNS0
	CapabilityEDNS0
	// CapabilityDO 支持DNSSEC OK
	CapabilityDO
)

// HasCapability 检查是否具备指定能力
func (sc ServerCapability) HasCapability(cap ServerCapability) bool {
	return sc&cap != 0
}

// AddCapability 添加能力
func (sc *ServerCapability) AddCapability(cap ServerCapability) {
	*sc |= cap
}

// RemoveCapability 移除能力
func (sc *ServerCapability) RemoveCapability(cap ServerCapability) {
	*sc &^= cap
}

// String 返回能力的字符串表示
func (sc ServerCapability) String() string {
	caps := []string{}
	if sc.HasCapability(CapabilityTCP) {
		caps = append(caps, "TCP")
	}
	if sc.HasCapability(CapabilityPipeline) {
		caps = append(caps, "Pipeline")
	}
	if sc.HasCapability(CapabilityEDNS0) {
		caps = append(caps, "EDNS0")
	}
	if sc.HasCapability(CapabilityDO) {
		caps = append(caps, "DO")
	}
	if len(caps) == 0 {
		return "None"
	}
	return fmt.Sprintf("%v", caps)
}

// ServerState 服务器状态信息
type ServerState struct {
	// Address 服务器地址
	Address string
	// Capabilities 服务器能力
	Capabilities ServerCapability
	// LastProbeTime 上次探测时间
	LastProbeTime time.Time
	// LastSuccessTime 上次成功时间
	LastSuccessTime time.Time
	// ProbeCount 探测次数
	ProbeCount int
	// SuccessCount 成功次数
	SuccessCount int
	// FailedCount 失败次数
	FailedCount int
	// AvgResponseTime 平均响应时间
	AvgResponseTime time.Duration
	// IsHealthy 是否健康
	IsHealthy bool
	// mu 保护状态的锁
	mu sync.RWMutex
}

// UpdateProbeResult 更新探测结果
func (s *ServerState) UpdateProbeResult(result ProbeResult, responseTime time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastProbeTime = time.Now()
	s.ProbeCount++

	switch result {
	case ProbeResultSuccess:
		s.SuccessCount++
		s.LastSuccessTime = time.Now()
		s.IsHealthy = true
		// 更新平均响应时间
		if s.AvgResponseTime == 0 {
			s.AvgResponseTime = responseTime
		} else {
			s.AvgResponseTime = (s.AvgResponseTime + responseTime) / 2
		}
	case ProbeResultFailed:
		s.FailedCount++
		// 连续失败超过3次标记为不健康
		if s.FailedCount-s.SuccessCount > 3 {
			s.IsHealthy = false
		}
	case ProbeResultTimeout:
		s.FailedCount++
	}
}

// UpdateCapabilities 更新服务器能力
func (s *ServerState) UpdateCapabilities(caps ServerCapability) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Capabilities = caps
}

// GetCapabilities 获取服务器能力
func (s *ServerState) GetCapabilities() ServerCapability {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Capabilities
}

// GetStats 获取统计信息
func (s *ServerState) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"address":           s.Address,
		"capabilities":      s.Capabilities.String(),
		"last_probe_time":   s.LastProbeTime,
		"last_success_time": s.LastSuccessTime,
		"probe_count":       s.ProbeCount,
		"success_count":     s.SuccessCount,
		"failed_count":      s.FailedCount,
		"avg_response_time": s.AvgResponseTime,
		"is_healthy":        s.IsHealthy,
	}
}

// ServerCapabilityProber 服务器能力探测器
type ServerCapabilityProber struct {
	// states 服务器状态映射
	states map[string]*ServerState
	// statesMu 保护states的锁
	statesMu sync.RWMutex
	// probeQueue 探测任务队列
	probeQueue chan string
	// stopChan 停止信号
	stopChan chan struct{}
	// wg 等待组
	wg sync.WaitGroup
	// workers 工作协程数
	workers int
	// logger 日志记录器
	logger Logger
	// rateLimiter 速率限制器
	rateLimiter *ProbeRateLimiter
	// tcpPool TCP连接池，用于保存探测成功的连接
	tcpPool *TCPConnectionPool
}

// ProbeRateLimiter 探测速率限制器
type ProbeRateLimiter struct {
	// lastProbe 上次探测时间映射
	lastProbe map[string]time.Time
	// mu 保护lastProbe的锁
	mu sync.RWMutex
	// minInterval 最小探测间隔
	minInterval time.Duration
}

// CanProbe 检查是否可以探测指定服务器
func (r *ProbeRateLimiter) CanProbe(serverAddr string) bool {
	r.mu.RLock()
	lastTime, exists := r.lastProbe[serverAddr]
	r.mu.RUnlock()

	if !exists {
		return true
	}

	return time.Since(lastTime) >= r.minInterval
}

// MarkProbed 标记服务器已被探测
func (r *ProbeRateLimiter) MarkProbed(serverAddr string) {
	r.mu.Lock()
	r.lastProbe[serverAddr] = time.Now()
	r.mu.Unlock()
}

// Logger 日志接口
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct{}

// Debug 调试日志
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	// 默认不输出调试日志
}

// Info 信息日志
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

// Warn 警告日志
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

// Error 错误日志
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// NewServerCapabilityProber 创建新的服务器能力探测器
//
// 参数:
//   - workers: 工作协程数，如果为0则使用默认值
//   - logger: 日志记录器，如果为nil则使用默认日志
//   - tcpPool: TCP连接池，用于保存探测成功的连接，可为nil
//
// 返回:
//   - *ServerCapabilityProber: 创建的探测器实例
func NewServerCapabilityProber(workers int, logger Logger, tcpPool *TCPConnectionPool) *ServerCapabilityProber {
	if workers <= 0 {
		workers = DefaultProbeWorkers
	}
	if logger == nil {
		logger = &DefaultLogger{}
	}

	prober := &ServerCapabilityProber{
		states:      make(map[string]*ServerState),
		probeQueue:  make(chan string, ProbeQueueSize),
		stopChan:    make(chan struct{}),
		workers:     workers,
		logger:      logger,
		rateLimiter: &ProbeRateLimiter{lastProbe: make(map[string]time.Time), minInterval: MinProbeInterval},
		tcpPool:     tcpPool,
	}

	// 启动工作协程
	for i := 0; i < workers; i++ {
		prober.wg.Add(1)
		go prober.probeWorker(i)
	}

	// 启动定期全量刷新协程
	prober.wg.Add(1)
	go prober.refreshLoop()

	return prober
}

// probeWorker 探测工作协程
func (p *ServerCapabilityProber) probeWorker(id int) {
	defer p.wg.Done()

	p.logger.Debug("探测工作协程 %d 已启动", id)

	for {
		select {
		case serverAddr, ok := <-p.probeQueue:
			if !ok {
				p.logger.Debug("探测工作协程 %d: 队列已关闭，退出", id)
				return
			}
			p.performProbe(serverAddr)
		case <-p.stopChan:
			p.logger.Debug("探测工作协程 %d: 收到停止信号，退出", id)
			return
		}
	}
}

// performProbe 执行探测
func (p *ServerCapabilityProber) performProbe(serverAddr string) {
	// 检查速率限制
	if !p.rateLimiter.CanProbe(serverAddr) {
		p.logger.Debug("跳过探测 %s: 速率限制", serverAddr)
		return
	}

	p.rateLimiter.MarkProbed(serverAddr)

	p.logger.Debug("开始探测服务器: %s", serverAddr)

	// 获取或创建服务器状态
	state := p.GetOrCreateServerState(serverAddr)

	// 执行TCP能力探测
	caps, result, responseTime := p.probeTCPCapability(serverAddr)

	// 更新探测结果
	state.UpdateProbeResult(result, responseTime)
	state.UpdateCapabilities(caps)

	if result == ProbeResultSuccess {
		p.logger.Debug("探测成功 %s: 能力=%s, 响应时间=%v", serverAddr, caps.String(), responseTime)
	} else {
		p.logger.Debug("探测失败 %s: 结果=%d", serverAddr, result)
	}
}

// probeTCPCapability 探测TCP能力
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - ServerCapability: 服务器能力
//   - ProbeResult: 探测结果
//   - time.Duration: 响应时间
func (p *ServerCapabilityProber) probeTCPCapability(serverAddr string) (ServerCapability, ProbeResult, time.Duration) {
	caps := CapabilityNone

	// 解析地址
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		// 如果没有端口，使用默认DNS端口
		host = serverAddr
		port = "53"
	}

	addr := net.JoinHostPort(host, port)

	// 尝试建立TCP连接
	startTime := time.Now()

	dialer := &net.Dialer{
		Timeout: TCPProbeTimeout,
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		p.logger.Debug("TCP连接失败 %s: %v", addr, err)
		return caps, ProbeResultFailed, time.Since(startTime)
	}
	// 注意：连接可能在探测成功后交给连接池管理
	// 使用一个标志来跟踪连接是否被转移
	connectionTransferred := false
	defer func() {
		// 如果连接未被转移给连接池，则在这里关闭
		if !connectionTransferred && conn != nil {
			conn.Close()
		}
	}()

	// 设置TCP参数
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true)
	}

	// 创建DNS连接
	dnsConn := &dns.Conn{Conn: conn}

	// 生成随机查询ID
	queryID := generateRandomID()

	// 创建探测查询（使用根域名NS记录）
	msg := new(dns.Msg)
	msg.Id = queryID
	msg.SetQuestion(".", dns.TypeNS)
	msg.RecursionDesired = true

	// 添加EDNS0选项
	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	opt.SetUDPSize(4096)
	opt.SetDo(true)
	msg.Extra = append(msg.Extra, opt)

	// 发送查询
	if err := dnsConn.WriteMsg(msg); err != nil {
		p.logger.Debug("发送探测查询失败 %s: %v", addr, err)
		return caps, ProbeResultFailed, time.Since(startTime)
	}

	// 读取响应
	resp, err := dnsConn.ReadMsg()
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			p.logger.Debug("探测超时 %s", addr)
			return caps, ProbeResultTimeout, time.Since(startTime)
		}
		p.logger.Debug("读取探测响应失败 %s: %v", addr, err)
		return caps, ProbeResultFailed, time.Since(startTime)
	}

	responseTime := time.Since(startTime)

	// 验证响应
	if resp == nil {
		p.logger.Debug("收到空响应 %s", addr)
		return caps, ProbeResultFailed, responseTime
	}

	// 检查响应ID（某些服务器可能返回不同的ID，记录警告但继续）
	if resp.Id != queryID {
		p.logger.Debug("响应ID不匹配 %s: 期望=%d, 实际=%d", addr, queryID, resp.Id)
		// 不立即返回失败，继续检查响应内容
		// 某些DNS服务器出于安全考虑会修改响应ID
	}

	// 服务器支持TCP
	caps.AddCapability(CapabilityTCP)

	// 检查EDNS0支持
	if resp.IsEdns0() != nil {
		caps.AddCapability(CapabilityEDNS0)
	}

	// 检查DO位
	if opt := resp.IsEdns0(); opt != nil && opt.Do() {
		caps.AddCapability(CapabilityDO)
	}

	// 尝试管道化探测
	if p.probePipelineCapability(dnsConn) {
		caps.AddCapability(CapabilityPipeline)
	}

	// 探测成功，将连接添加到TCP连接池（而不是关闭）
	// 这样后续查询可以直接使用已建立的连接
	// 注意：使用原始的serverAddr作为键，确保与查询时使用的地址格式一致
	if p.tcpPool != nil {
		err := p.tcpPool.AddExistingConnection(serverAddr, conn, dnsConn)
		if err != nil {
			p.logger.Debug("将探测连接添加到连接池失败 %s: %v", serverAddr, err)
			// 添加失败，连接将在defer中关闭
		} else {
			p.logger.Debug("探测连接已添加到连接池 %s", serverAddr)
			// 连接已交给连接池管理，设置标志防止defer关闭
			connectionTransferred = true

			// 异步创建额外的连接，确保连接池达到最大连接数
			// 这样首次查询时就有多个可用连接
			go p.tcpPool.EnsureMinConnections(serverAddr, p.tcpPool.GetConfig().MaxConnectionsPerServer)
		}
	}

	return caps, ProbeResultSuccess, responseTime
}

// probePipelineCapability 探测管道化能力
//
// 参数:
//   - dnsConn: DNS连接
//
// 返回:
//   - bool: 是否支持管道化
func (p *ServerCapabilityProber) probePipelineCapability(dnsConn *dns.Conn) bool {
	// 发送多个查询而不等待响应
	queryCount := 3
	queryIDs := make([]uint16, queryCount)

	for i := 0; i < queryCount; i++ {
		queryIDs[i] = generateRandomID()
		msg := new(dns.Msg)
		msg.Id = queryIDs[i]
		msg.SetQuestion(".", dns.TypeNS)
		msg.RecursionDesired = true

		if err := dnsConn.WriteMsg(msg); err != nil {
			p.logger.Debug("管道化探测发送失败: %v", err)
			return false
		}
	}

	// 等待所有响应
	receivedCount := 0
	for i := 0; i < queryCount; i++ {
		// 设置读取超时
		dnsConn.Conn.SetReadDeadline(time.Now().Add(2 * time.Second))

		resp, err := dnsConn.ReadMsg()
		if err != nil {
			p.logger.Debug("管道化探测读取失败: %v", err)
			return false
		}

		// 验证响应ID（某些服务器可能返回不同的ID）
		found := false
		for _, id := range queryIDs {
			if resp.Id == id {
				found = true
				break
			}
		}
		if !found {
			p.logger.Debug("管道化探测收到未知响应ID: %d, 期望的IDs=%v", resp.Id, queryIDs)
			// 不立即返回失败，继续接收其他响应
			// 某些DNS服务器出于安全考虑会修改响应ID
		}
		receivedCount++
	}

	// 清除读取超时
	dnsConn.Conn.SetReadDeadline(time.Time{})

	p.logger.Debug("管道化探测成功: 发送=%d, 接收=%d", queryCount, receivedCount)
	return receivedCount == queryCount
}

// generateRandomID 生成随机查询ID
func generateRandomID() uint16 {
	b := make([]byte, 2)
	rand.Read(b)
	return binary.BigEndian.Uint16(b)
}

// GetOrCreateServerState 获取或创建服务器状态
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - *ServerState: 服务器状态
func (p *ServerCapabilityProber) GetOrCreateServerState(serverAddr string) *ServerState {
	p.statesMu.RLock()
	state, exists := p.states[serverAddr]
	p.statesMu.RUnlock()

	if exists {
		return state
	}

	p.statesMu.Lock()
	defer p.statesMu.Unlock()

	// 双重检查
	if state, exists = p.states[serverAddr]; exists {
		return state
	}

	state = &ServerState{
		Address:   serverAddr,
		IsHealthy: true,
		mu:        sync.RWMutex{},
	}
	p.states[serverAddr] = state
	return state
}

// GetServerState 获取服务器状态
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - *ServerState: 服务器状态
//   - bool: 是否找到
func (p *ServerCapabilityProber) GetServerState(serverAddr string) (*ServerState, bool) {
	p.statesMu.RLock()
	defer p.statesMu.RUnlock()

	state, exists := p.states[serverAddr]
	return state, exists
}

// SubmitProbe 提交探测任务
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - bool: 是否成功提交
func (p *ServerCapabilityProber) SubmitProbe(serverAddr string) bool {
	select {
	case p.probeQueue <- serverAddr:
		p.logger.Debug("成功提交探测任务: %s", serverAddr)
		return true
	default:
		p.logger.Warn("探测队列已满，无法提交任务: %s", serverAddr)
		return false
	}
}

// submitProbeTask 提交探测任务（内部使用）
func (p *ServerCapabilityProber) submitProbeTask(serverAddr string) bool {
	return p.SubmitProbe(serverAddr)
}

// Stop 停止探测器
func (p *ServerCapabilityProber) Stop() {
	p.logger.Info("停止服务器能力探测器")

	close(p.stopChan)

	// 等待所有工作协程退出
	p.wg.Wait()

	// 关闭探测队列
	close(p.probeQueue)

	p.logger.Info("服务器能力探测器已停止")
}

// refreshLoop 定期刷新循环
func (p *ServerCapabilityProber) refreshLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(FullRefreshInterval)
	defer ticker.Stop()

	// 立即执行一次全量刷新
	p.RefreshAllServers()

	for {
		select {
		case <-ticker.C:
			p.RefreshAllServers()
		case <-p.stopChan:
			return
		}
	}
}

// RefreshAllServers 全量刷新所有服务器能力
// 对所有已知服务器进行能力探测
func (p *ServerCapabilityProber) RefreshAllServers() {
	p.logger.Info("开始全量刷新服务器能力表")

	p.statesMu.RLock()
	servers := make([]string, 0, len(p.states))
	for addr := range p.states {
		servers = append(servers, addr)
	}
	p.statesMu.RUnlock()

	if len(servers) == 0 {
		p.logger.Debug("没有服务器需要刷新")
		return
	}

	p.logger.Debug("需要刷新 %d 个服务器", len(servers))

	// 为每个服务器提交探测任务
	successCount := 0
	for _, addr := range servers {
		if p.submitProbeTask(addr) {
			successCount++
		}
	}

	p.logger.Info("全量刷新完成，成功提交 %d/%d 个探测任务", successCount, len(servers))
}

// AddServer 添加新服务器到探测器
//
// 参数:
//   - serverAddr: 服务器地址
func (p *ServerCapabilityProber) AddServer(serverAddr string) {
	p.GetOrCreateServerState(serverAddr)
	p.logger.Debug("添加服务器到探测器: %s", serverAddr)
}

// RemoveServer 从探测器移除服务器
//
// 参数:
//   - serverAddr: 服务器地址
func (p *ServerCapabilityProber) RemoveServer(serverAddr string) {
	p.statesMu.Lock()
	delete(p.states, serverAddr)
	p.statesMu.Unlock()

	p.rateLimiter.mu.Lock()
	delete(p.rateLimiter.lastProbe, serverAddr)
	p.rateLimiter.mu.Unlock()

	p.logger.Debug("从探测器移除服务器: %s", serverAddr)
}

// GetAllServerStates 获取所有服务器状态
//
// 返回值:
//   - map[string]*ServerState: 服务器状态映射的副本
func (p *ServerCapabilityProber) GetAllServerStates() map[string]*ServerState {
	p.statesMu.RLock()
	defer p.statesMu.RUnlock()

	states := make(map[string]*ServerState)
	for addr, state := range p.states {
		states[addr] = state
	}
	return states
}

// GetServerCount 获取服务器数量
//
// 返回值:
//   - int: 服务器数量
func (p *ServerCapabilityProber) GetServerCount() int {
	p.statesMu.RLock()
	defer p.statesMu.RUnlock()
	return len(p.states)
}

// GetQueueSize 获取当前探测队列大小
//
// 返回值:
//   - int: 当前队列中的任务数
func (p *ServerCapabilityProber) GetQueueSize() int {
	return len(p.probeQueue)
}
