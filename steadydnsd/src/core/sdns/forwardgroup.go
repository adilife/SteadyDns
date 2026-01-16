// core/sdns/forwardgroup.go

package sdns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"

	"github.com/miekg/dns"
)

// ServerStats 服务器统计信息
type ServerStats struct {
	Mu                      sync.RWMutex
	Address                 string
	Queries                 int64
	SuccessfulQueries       int64
	FailedQueries           int64
	TotalResponseTime       time.Duration
	LastQueryTime           time.Time
	LastSuccessfulQueryTime time.Time
	HealthCheckTime         time.Time
	Status                  string
	QPS                     float64
	Latency                 float64
	WindowStartTime         time.Time
	WindowQueries           int64
}

// DNSForwarder 主转发器结构
type DNSForwarder struct {
	groups          map[string]*ForwardGroup
	domainIndex     []string      // 域名索引，按长度降序排列，用于最长匹配
	defaultGroup    *ForwardGroup // 默认转发组
	serverStats     map[string]*ServerStats
	mu              sync.RWMutex // 保护 groups 映射的锁
	statsMu         sync.RWMutex // 保护 serverStats 映射的锁
	cacheTTL        time.Duration
	priorityTimeout time.Duration      // 优先级队列超时时间
	logger          *common.Logger     // 日志函数
	forwardPool     *ForwardWorkerPool // 专用的DNS转发协程池

	// 域名匹配缓存
	matchCache   map[string]*cacheEntry // 域名匹配结果缓存
	matchCacheMu sync.RWMutex           // 保护matchCache的锁
}

// cacheEntry 域名匹配缓存项
type cacheEntry struct {
	group     *ForwardGroup // 匹配到的转发组
	expiresAt time.Time     // 过期时间
}

// ForwardWorkerPool 专用的DNS转发协程池
type ForwardWorkerPool struct {
	taskChan    chan *DNSForwardTask
	workerCount int
	wg          sync.WaitGroup
	shutdown    bool
}

// NewForwardWorkerPool 创建新的DNS转发协程池
func NewForwardWorkerPool(workerCount int) *ForwardWorkerPool {
	if workerCount <= 0 {
		workerCount = 50000 // 默认值
	}

	pool := &ForwardWorkerPool{
		taskChan:    make(chan *DNSForwardTask, workerCount*2), // 队列大小为协程数的2倍
		workerCount: workerCount,
		shutdown:    false,
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker 工作协程
func (p *ForwardWorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case task, ok := <-p.taskChan:
			if !ok {
				return
			}

			// 处理任务
			task.Process()
		}
	}
}

// SubmitTask 提交任务到协程池，处理协程池被占满的情况
func (p *ForwardWorkerPool) SubmitTask(task *DNSForwardTask) {
	select {
	case p.taskChan <- task:
		// 任务已加入队列
	default:
		// 队列已满，直接在当前协程中执行
		task.Process()
	}
}

// Close 关闭协程池
func (p *ForwardWorkerPool) Close() {
	p.shutdown = true
	close(p.taskChan)
	p.wg.Wait()
}

// ForwardGroup 表示一个转发服务器组
type ForwardGroup struct {
	Name           string               `json:"name"`            // 组名，长度0-63
	Description    string               `json:"description"`     // 描述，长度0-65535
	PriorityQueues map[int][]*DNSServer `json:"priority_queues"` // 按优先级分组的DNS服务器列表
}

// DNSServer 表示单个DNS服务器
type DNSServer struct {
	Address     string `json:"address"`     // DNS服务器地址，支持IPv4/IPv6
	Port        int    `json:"port"`        // 端口，默认53
	Description string `json:"description"` // 描述，长度0-65535
	QueueIndex  int    `json:"queue_index"` // 队列序号
	Priority    int    `json:"priority"`    // 优先级 (1-3)
}

// ValidateForwardGroup 验证转发组配置
func ValidateForwardGroup(group *ForwardGroup) error {
	// 验证域名格式
	if err := validateDomain(group.Name); err != nil {
		return fmt.Errorf("域名格式错误: %v", err)
	}

	if len(group.Description) > 65535 {
		return fmt.Errorf("描述长度不能超过65535")
	}

	for priority, servers := range group.PriorityQueues {
		if priority < 1 || priority > 3 {
			return fmt.Errorf("优先级必须在1-3之间")
		}

		for _, server := range servers {
			if err := ValidateDNSServer(server); err != nil {
				return fmt.Errorf("队列 %d 中的服务器配置错误: %v", priority, err)
			}
		}
	}

	return nil
}

// validateDomain 验证域名格式是否符合DNS规范
func validateDomain(domain string) error {
	// 检查域名是否为空
	if len(domain) == 0 {
		return fmt.Errorf("域名不能为空")
	}

	// 检查域名总长度（包括点）是否超过255个字符
	if len(domain) > 255 {
		return fmt.Errorf("域名长度不能超过255个字符")
	}

	// 检查是否为默认转发组
	if domain == "Default" {
		return nil
	}

	// 分割域名标签
	tags := strings.Split(domain, ".")

	// 检查每个标签
	for _, tag := range tags {
		// 检查标签长度
		if len(tag) == 0 || len(tag) > 63 {
			return fmt.Errorf("域名标签长度必须在1-63之间")
		}

		// 检查标签是否以连字符开头或结尾
		if strings.HasPrefix(tag, "-") || strings.HasSuffix(tag, "-") {
			return fmt.Errorf("域名标签不能以连字符开头或结尾")
		}

		// 检查标签是否只包含字母、数字和连字符
		for _, c := range tag {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return fmt.Errorf("域名标签只能包含字母、数字和连字符")
			}
		}
	}

	return nil
}

// ValidateDNSServer 验证DNS服务器配置
func ValidateDNSServer(server *DNSServer) error {
	if server.Address == "" {
		return fmt.Errorf("DNS服务器地址不能为空")
	}

	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}

	if len(server.Description) > 65535 {
		return fmt.Errorf("描述长度不能超过65535")
	}

	// 验证地址格式
	ip := net.ParseIP(server.Address)
	if ip == nil {
		return fmt.Errorf("无效的IP地址: %s", server.Address)
	}

	return nil
}

// LoadForwardGroups 从数据库加载转发组配置
func (f *DNSForwarder) LoadForwardGroups() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 清空现有转发组
	f.groups = make(map[string]*ForwardGroup)
	f.defaultGroup = nil

	// 从数据库获取所有转发组
	allGroups, err := database.GetForwardGroups()
	if err != nil {
		return fmt.Errorf("获取所有转发组失败: %v", err)
	}

	// 检查是否有ID=1的默认转发组
	var defaultGroup *database.ForwardGroup
	var defaultDNSServers []database.DNSServer
	var hasDefaultGroup bool

	for _, group := range allGroups {
		// 跳过禁用的转发组
		if !group.Enable {
			continue
		}

		// 处理默认转发组（ID=1）
		if group.ID == 1 {
			defaultGroup = &group
			defaultDNSServers = group.Servers
			hasDefaultGroup = true
		}

		// 创建DNS包中的ForwardGroup结构体
		dnsGroup := &ForwardGroup{
			Name:           group.Domain,
			Description:    group.Description,
			PriorityQueues: make(map[int][]*DNSServer),
		}

		// 按照优先级分组DNS服务器
		for _, server := range group.Servers {
			if _, exists := dnsGroup.PriorityQueues[server.Priority]; !exists {
				dnsGroup.PriorityQueues[server.Priority] = []*DNSServer{}
			}
			dnsServer := &DNSServer{
				Address:     server.Address,
				Port:        server.Port,
				Description: server.Description,
				QueueIndex:  server.QueueIndex,
				Priority:    server.Priority,
			}
			dnsGroup.PriorityQueues[server.Priority] = append(dnsGroup.PriorityQueues[server.Priority], dnsServer)
		}

		// 更新groups映射
		f.groups[group.Domain] = dnsGroup
		f.logger.Info("转发组配置加载成功，域名: %s, 服务器数量: %d", group.Domain, len(group.Servers))
	}

	// 如果没有ID=1的默认转发组，创建一个
	if !hasDefaultGroup {
		f.logger.Info("未找到ID=1的转发组，正在创建默认转发组")
		defaultGroup = &database.ForwardGroup{
			ID:          1,
			Domain:      "Default",
			Description: "Default Domain",
			Enable:      true,
			Servers:     []database.DNSServer{},
		}
		if err := database.CreateForwardGroup(defaultGroup); err != nil {
			return fmt.Errorf("创建默认转发组失败: %v", err)
		}
		f.logger.Info("默认转发组创建成功")

		// 创建DNS包中的ForwardGroup结构体
		dnsGroup := &ForwardGroup{
			Name:           defaultGroup.Domain,
			Description:    defaultGroup.Description,
			PriorityQueues: make(map[int][]*DNSServer),
		}

		// 更新groups映射
		f.groups[defaultGroup.Domain] = dnsGroup
		defaultDNSServers = defaultGroup.Servers
	}

	// 设置默认转发组
	f.defaultGroup = f.groups[defaultGroup.Domain]

	// 初始化域名索引
	f.initDomainIndex()

	// 清除域名匹配缓存，因为转发组配置已更新
	f.clearMatchCache()

	// 清理服务器统计信息，只保留当前活跃的服务器
	f.CleanupServerStats(defaultDNSServers)

	return nil
}

// UpdateServerStats 更新服务器统计信息
func (f *DNSForwarder) UpdateServerStats() {
	f.statsMu.RLock()
	serverCount := len(f.serverStats)
	if serverCount == 0 {
		f.statsMu.RUnlock()
		return
	}

	// 预分配足够的容量
	statsList := make([]*ServerStats, 0, serverCount)
	for _, stats := range f.serverStats {
		statsList = append(statsList, stats)
	}
	f.statsMu.RUnlock() // 提前释放读锁

	now := time.Now()

	for _, stats := range statsList {
		stats.Mu.Lock()

		// 计算QPS
		windowDuration := now.Sub(stats.WindowStartTime)
		if windowDuration > time.Second {
			stats.QPS = float64(stats.WindowQueries) / windowDuration.Seconds()
			stats.WindowStartTime = now
			stats.WindowQueries = 0
		}

		// 计算平均延迟
		if stats.SuccessfulQueries > 0 {
			stats.Latency = float64(stats.TotalResponseTime.Milliseconds()) / float64(stats.SuccessfulQueries)
		}

		// 检查服务器健康状态
		if now.Sub(stats.LastSuccessfulQueryTime) > 30*time.Second {
			stats.Status = "unhealthy"
		}

		stats.Mu.Unlock()
	}
}

// GetServerStats 获取服务器统计信息
func (f *DNSForwarder) GetServerStats(address string) *ServerStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	if stats, exists := f.serverStats[address]; exists {
		return stats
	}
	return nil
}

// GetAllServerStats 获取所有服务器统计信息
func (f *DNSForwarder) GetAllServerStats() map[string]*ServerStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	// 创建副本以避免并发访问问题
	statsCopy := make(map[string]*ServerStats)
	for addr, stats := range f.serverStats {
		statsCopy[addr] = stats
	}
	return statsCopy
}

// CleanupServerStats 清理不再使用的服务器统计信息
func (f *DNSForwarder) CleanupServerStats(activeServers []database.DNSServer) {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()

	// 创建活跃服务器地址集合
	activeServerAddrs := make(map[string]bool)
	for _, server := range activeServers {
		addr := fmt.Sprintf("%s:%d", server.Address, server.Port)
		activeServerAddrs[addr] = true
	}

	// 清理不再活跃的服务器统计信息
	for addr := range f.serverStats {
		if !activeServerAddrs[addr] {
			delete(f.serverStats, addr)
			f.logger.Info("清理不再使用的服务器统计信息: %s", addr)
		}
	}
}

// initDomainIndex 初始化域名索引，按域名长度降序排序
func (f *DNSForwarder) initDomainIndex() {

	// 提取所有非默认转发组的域名
	domains := make([]string, 0, len(f.groups))
	for domain := range f.groups {
		// 跳过默认转发组（使用"Default"作为key）
		if domain != "Default" {
			domains = append(domains, domain)
		}
	}

	// 按域名长度降序排序，用于最长匹配
	for i := 0; i < len(domains); i++ {
		for j := i + 1; j < len(domains); j++ {
			if len(domains[i]) < len(domains[j]) {
				domains[i], domains[j] = domains[j], domains[i]
			}
		}
	}

	// 更新域名索引
	f.domainIndex = domains
}

// matchDomain 根据查询域名匹配最合适的转发组
// 实现最长匹配机制：完全匹配或前缀+当前域名
func (f *DNSForwarder) matchDomain(queryDomain string) *ForwardGroup {
	// 移除末尾的点
	queryDomain = strings.TrimSuffix(queryDomain, ".")

	// 检查缓存
	f.matchCacheMu.RLock()
	entry, found := f.matchCache[queryDomain]
	f.matchCacheMu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.group
	}

	// 尝试最长匹配
	var matchedGroup *ForwardGroup
	for _, domain := range f.domainIndex {
		// 完全匹配（如jcgov.gov.cn）
		if queryDomain == domain {
			f.mu.RLock()
			matchedGroup = f.groups[domain]
			f.mu.RUnlock()
			break
		}
		// 前缀+当前域名（如www.jcgov.gov.cn）
		if strings.HasSuffix(queryDomain, "."+domain) {
			f.mu.RLock()
			matchedGroup = f.groups[domain]
			f.mu.RUnlock()
			break
		}
	}

	// 没有匹配到，返回默认转发组
	if matchedGroup == nil {
		f.mu.RLock()
		matchedGroup = f.defaultGroup
		f.mu.RUnlock()
	}

	// 更新缓存
	f.matchCacheMu.Lock()
	f.matchCache[queryDomain] = &cacheEntry{
		group:     matchedGroup,
		expiresAt: time.Now().Add(f.cacheTTL),
	}
	f.matchCacheMu.Unlock()

	return matchedGroup
}

// CheckServerHealth 检查服务器健康状态
func (f *DNSForwarder) CheckServerHealth(addr string) bool {
	// 创建一个简单的DNS查询
	query := new(dns.Msg)
	query.SetQuestion("example.com.", dns.TypeA)
	query.RecursionDesired = true

	// 设置超时
	c := new(dns.Client)
	c.Timeout = 2 * time.Second

	// 执行查询
	result, _, err := c.Exchange(query, addr)
	if err != nil {
		f.logger.Debug("健康检查 - 服务器 %s 失败: %v", addr, err)
		return false
	}

	if result == nil || result.Rcode != dns.RcodeSuccess {
		f.logger.Debug("健康检查 - 服务器 %s 失败，返回码: %d", addr, result.Rcode)
		return false
	}

	f.logger.Debug("健康检查 - 服务器 %s 成功", addr)
	return true
}

// IsServerHealthy 检查服务器是否健康
func (f *DNSForwarder) IsServerHealthy(addr string) bool {
	stats := f.GetServerStats(addr)
	if stats == nil {
		return false
	}

	stats.Mu.RLock()
	defer stats.Mu.RUnlock()

	// 检查服务器状态
	if stats.Status != "healthy" {
		return false
	}

	// 检查最后一次成功查询时间
	if time.Since(stats.LastSuccessfulQueryTime) > 30*time.Second {
		return false
	}

	return true
}

// StartHealthChecks 启动健康检查协程
func (f *DNSForwarder) StartHealthChecks() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// 从当前活跃的groups中获取服务器地址
			f.mu.RLock()
			var servers []string
			for _, group := range f.groups {
				for _, priorityServers := range group.PriorityQueues {
					for _, server := range priorityServers {
						addr := fmt.Sprintf("%s:%d", server.Address, server.Port)
						servers = append(servers, addr)
					}
				}
			}
			f.mu.RUnlock()

			// 对每个活跃服务器进行健康检查
			for _, addr := range servers {
				go func(address string) {
					healthy := f.CheckServerHealth(address)
					stats := f.getOrCreateServerStats(address)
					stats.Mu.Lock()
					stats.HealthCheckTime = time.Now()
					if healthy {
						stats.Status = "healthy"
						stats.LastSuccessfulQueryTime = time.Now()
					} else {
						stats.Status = "unhealthy"
					}
					stats.Mu.Unlock()
				}(addr)
			}
		}
	}()
}

// LoadConfig 从配置加载
func (f *DNSForwarder) LoadConfig() {
	// 从配置读取优先级超时时间
	priorityTimeoutMsStr := common.GetConfig("DNS", "DNS_PRIORITY_TIMEOUT_MS")

	f.logger.Debug("加载配置 - DNS_PRIORITY_TIMEOUT_MS配置值: '%s'", priorityTimeoutMsStr)

	var priorityTimeoutMs int
	if priorityTimeoutMsStr != "" {
		if t, err := strconv.Atoi(priorityTimeoutMsStr); err == nil && t > 0 {
			priorityTimeoutMs = t
			f.logger.Debug("加载配置 - 从配置读取到超时时间: %dms", priorityTimeoutMs)
		} else {
			priorityTimeoutMs = 50 // 默认50ms
			f.logger.Info("加载配置 - 配置值无效，使用默认值: %dms", priorityTimeoutMs)
		}
	} else {
		priorityTimeoutMs = 50 // 默认50ms
		f.logger.Info("加载配置 - 配置未设置，使用默认值: %dms", priorityTimeoutMs)
	}

	f.priorityTimeout = time.Duration(priorityTimeoutMs) * time.Millisecond
	f.logger.Debug("加载配置 - 最终优先级超时时间: %v", f.priorityTimeout)
}

// NewDNSForwarder 创建新的DNS转发器
func NewDNSForwarder(forwardAddr string) *DNSForwarder {
	forwarder := &DNSForwarder{
		groups:      make(map[string]*ForwardGroup),
		domainIndex: make([]string, 0),
		serverStats: make(map[string]*ServerStats),
		cacheTTL:    30 * time.Second,
		logger:      common.NewLogger(),
		matchCache:  make(map[string]*cacheEntry), // 初始化域名匹配缓存
	}

	// 加载配置
	forwarder.LoadConfig()

	// 从配置获取客户端协程池配置
	clientWorkersStr := common.GetConfig("DNS", "DNS_CLIENT_WORKERS")
	var clientWorkers int
	if clientWorkersStr != "" {
		if cw, err := strconv.Atoi(clientWorkersStr); err == nil && cw > 0 {
			clientWorkers = cw
		} else {
			clientWorkers = 10000 // 默认值
		}
	} else {
		clientWorkers = 10000 // 默认值
	}

	// 创建专用的DNS转发协程池（客户端协程池的5倍）
	forwardPoolSize := clientWorkers * 5
	forwarder.forwardPool = NewForwardWorkerPool(forwardPoolSize)
	forwarder.logger.Debug("创建专用DNS转发协程池，大小: %d", forwardPoolSize)

	// 加载转发组配置
	if err := forwarder.LoadForwardGroups(); err != nil {
		forwarder.logger.Error("加载转发组配置失败: %v", err)
	}

	// 启动健康检查协程
	forwarder.StartHealthChecks()

	// 启动统计更新协程
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			forwarder.UpdateServerStats()
		}
	}()

	// 启动缓存清理协程，每5分钟清理一次过期缓存
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			forwarder.cleanupMatchCache()
		}
	}()

	return forwarder
}

// AddForwardGroup 添加转发组
func (f *DNSForwarder) AddForwardGroup(group *ForwardGroup) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := ValidateForwardGroup(group); err != nil {
		return err
	}

	f.groups[group.Name] = group
	return nil
}

// ForwardQuery 转发DNS查询
func (f *DNSForwarder) ForwardQuery(query *dns.Msg) (*dns.Msg, error) {
	startTime := time.Now()

	var queryDomain, queryType string
	if len(query.Question) > 0 {
		queryDomain = query.Question[0].Name
		queryType = dns.TypeToString[query.Question[0].Qtype]
	}

	// 使用最长匹配算法选择合适的转发组
	matchedGroup := f.matchDomain(queryDomain)
	if matchedGroup == nil {
		f.logger.Error("转发查询 - 没有可用的转发组")
		return nil, fmt.Errorf("没有可用的转发组")
	}

	// 无锁执行转发操作
	f.logger.Debug("转发查询 - 开始使用组: %s, 域名: %s, 类型: %s", matchedGroup.Name, queryDomain, queryType)
	result, err := f.tryForwardWithPriority(matchedGroup, query)
	f.logger.Debug("转发查询 - 完成, 耗时: %v, 错误: %v", time.Since(startTime), err)

	if err == nil && result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("所有转发服务器都不可用")
}

// TestDomainMatch 测试域名匹配，返回匹配到的转发组名称
func (f *DNSForwarder) TestDomainMatch(domain string) string {
	matchedGroup := f.matchDomain(domain)
	if matchedGroup != nil {
		return matchedGroup.Name
	}
	return ""
}

// cleanupMatchCache 清理过期的域名匹配缓存
func (f *DNSForwarder) cleanupMatchCache() {
	f.matchCacheMu.Lock()
	defer f.matchCacheMu.Unlock()

	now := time.Now()
	removed := 0

	// 遍历缓存，删除过期条目
	for domain, entry := range f.matchCache {
		if now.After(entry.expiresAt) {
			delete(f.matchCache, domain)
			removed++
		}
	}

	if removed > 0 {
		f.logger.Debug("清理域名匹配缓存，删除了 %d 个过期条目", removed)
	}
}

// clearMatchCache 清除所有域名匹配缓存
func (f *DNSForwarder) clearMatchCache() {
	f.matchCacheMu.Lock()
	defer f.matchCacheMu.Unlock()

	f.matchCache = make(map[string]*cacheEntry)
	f.logger.Debug("清除了所有域名匹配缓存")
}

// DNSForwardTask DNS转发任务
type DNSForwardTask struct {
	address    string
	query      *dns.Msg
	resultChan chan *dns.Msg
	errorChan  chan error
	forwarder  *DNSForwarder
	cancelChan chan struct{} // 取消信号通道
}

// Process 处理DNS转发任务
func (t *DNSForwardTask) Process() {
	// 执行DNS查询
	result, err := t.forwarder.forwardToServer(t.address, t.query, t.cancelChan)

	// 检查是否收到取消信号
	select {
	case <-t.cancelChan:
		// 收到取消信号，但是如果查询已经成功完成，仍然发送结果
		if err == nil && result != nil {
			t.forwarder.logger.Debug("转发查询 - 任务被取消，但查询已成功完成，服务器: %s", t.address)
			// 尝试发送结果，即使取消通道已关闭
			select {
			case t.resultChan <- result:
				// 结果发送成功
			default:
				// 结果发送失败，可能是因为结果通道已满或已关闭
				t.forwarder.logger.Debug("转发查询 - 结果发送失败，服务器: %s", t.address)
			}
		} else {
			t.forwarder.logger.Debug("转发查询 - 任务被取消，服务器: %s", t.address)
		}
		return
	default:
		// 没有取消信号，处理查询结果
		if err == nil && result != nil {
			t.forwarder.logger.Debug("转发查询 - 服务器 %s 成功", t.address)
			t.resultChan <- result
		} else {
			t.forwarder.logger.Debug("转发查询 - 服务器 %s 失败: %v", t.address, err)
			t.errorChan <- err
		}
	}
}

// tryForwardWithPriority 尝试按优先级转发查询
func (f *DNSForwarder) tryForwardWithPriority(group *ForwardGroup, query *dns.Msg) (*dns.Msg, error) {
	// 整体查询超时时间（5秒）
	overallTimeout := 5 * time.Second
	// 优先级队列启动间隔（从配置读取）
	priorityInterval := f.priorityTimeout

	// 创建通道来接收查询结果（容量设置为服务器总数的估计值）
	resultChan := make(chan *dns.Msg, 10)
	errorChan := make(chan error, 10)
	// 创建整体取消通道
	cancelChan := make(chan struct{})

	// 按优先级顺序（1 -> 2 -> 3）启动查询
	for priority := 1; priority <= 3; priority++ {
		servers, exists := group.PriorityQueues[priority]
		if !exists || len(servers) == 0 {
			// 当某个优先级队列中无服务器时，跳过该队列
			f.logger.Debug("转发查询 - 优先级队列 %d 为空，跳过", priority)
			continue
		}

		f.logger.Debug("转发查询 - 启动优先级队列 %d, 服务器数量: %d", priority, len(servers))

		// 为当前优先级队列中的所有服务器创建任务
		for i, server := range servers {
			addr := fmt.Sprintf("%s:%d", server.Address, server.Port)
			f.logger.Debug("转发查询 - 尝试服务器 %d: %s (优先级 %d)", i+1, addr, priority)

			// 创建转发任务
			task := &DNSForwardTask{
				address:    addr,
				query:      query,
				resultChan: resultChan,
				errorChan:  errorChan,
				forwarder:  f,
				cancelChan: cancelChan, // 传递取消通道
			}

			// 使用专用的DNS转发协程池处理任务
			f.forwardPool.SubmitTask(task)
		}

		// 如果不是最后一个优先级队列，等待指定间隔后再启动下一个队列
		if priority < 3 {
			f.logger.Debug("转发查询 - 等待 %v 后启动下一优先级队列", priorityInterval)
			select {
			case <-time.After(priorityInterval):
				// 等待完成，继续启动下一优先级队列
			case result := <-resultChan:
				// 已收到结果，直接返回
				f.logger.Debug("转发查询 - 已收到结果，停止启动更多优先级队列")
				close(cancelChan)
				return result, nil
			case <-cancelChan:
				// 已收到取消信号，直接返回
				f.logger.Debug("转发查询 - 已收到取消信号，停止启动更多优先级队列")
				return nil, fmt.Errorf("查询被取消")
			}
		}
	}

	// 等待整体超时或结果
	select {
	case result := <-resultChan:
		f.logger.Debug("转发查询 - 成功收到结果")
		// 收到有效结果，取消其他任务
		close(cancelChan)
		return result, nil
	case <-time.After(overallTimeout):
		f.logger.Debug("转发查询 - 整体超时，等待结果返回...")
		// 整体超时，但是等待一小段时间，看看是否有查询结果返回
		select {
		case result := <-resultChan:
			f.logger.Debug("转发查询 - 超时后收到结果")
			close(cancelChan)
			return result, nil
		case <-time.After(100 * time.Millisecond):
			f.logger.Debug("转发查询 - 最终超时")
			// 检查是否有结果已经在通道中
			select {
			case result := <-resultChan:
				f.logger.Debug("转发查询 - 最终超时前收到结果")
				close(cancelChan)
				return result, nil
			default:
				// 没有结果，关闭取消通道并返回超时错误
				close(cancelChan)
				return nil, fmt.Errorf("整体查询超时")
			}
		}
	}
}

// getOrCreateServerStats 获取或创建服务器统计信息
func (f *DNSForwarder) getOrCreateServerStats(addr string) *ServerStats {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()

	if stats, exists := f.serverStats[addr]; exists {
		return stats
	}

	stats := &ServerStats{
		Address:         addr,
		Status:          "healthy",
		WindowStartTime: time.Now(),
		LastQueryTime:   time.Now(),
	}
	f.serverStats[addr] = stats
	return stats
}

// forwardToServer 向单个DNS服务器转发查询
func (f *DNSForwarder) forwardToServer(addr string, query *dns.Msg, cancelChan chan struct{}) (*dns.Msg, error) {
	startTime := time.Now()

	// 获取或创建服务器统计信息
	stats := f.getOrCreateServerStats(addr)

	// 创建结果通道
	resultChan := make(chan *dns.Msg, 1)
	errorChan := make(chan error, 1)

	// 在goroutine中执行DNS查询
	go func() {
		f.logger.Debug("转发查询 - 开始执行DNS查询，服务器: %s", addr)
		c := new(dns.Client)
		c.Timeout = 5 * time.Second

		result, rtt, err := c.Exchange(query, addr)
		f.logger.Debug("转发查询 - DNS查询完成，服务器: %s, 耗时: %v, 错误: %v", addr, rtt, err)

		if err != nil {
			f.logger.Debug("转发查询 - DNS查询失败，服务器: %s, 错误: %v", addr, err)
			errorChan <- err
			return
		}

		if result == nil {
			f.logger.Debug("转发查询 - DNS查询返回空结果，服务器: %s", addr)
			errorChan <- fmt.Errorf("DNS查询返回空结果")
			return
		}

		if result.Rcode != dns.RcodeSuccess {
			f.logger.Debug("转发查询 - DNS查询返回错误码，服务器: %s, 返回码: %d", addr, result.Rcode)
			errorChan <- fmt.Errorf("DNS查询失败，返回码: %d", result.Rcode)
			return
		}

		f.logger.Debug("转发查询 - DNS查询成功，服务器: %s, 耗时: %v", addr, rtt)
		resultChan <- result
	}()

	// 等待取消信号或查询结果
	select {
	case <-cancelChan:
		// 收到取消信号，立即返回
		f.logger.Debug("转发查询 - 查询被取消，服务器: %s", addr)
		return nil, fmt.Errorf("查询被取消")
	case result := <-resultChan:
		// 收到查询结果
		duration := time.Since(startTime)
		f.logger.Debug("转发查询 - 服务器 %s 响应成功, 耗时: %v", addr, duration)

		// 更新服务器统计信息
		stats.Mu.Lock()
		stats.Queries++
		stats.SuccessfulQueries++
		stats.TotalResponseTime += duration
		stats.LastQueryTime = time.Now()
		stats.LastSuccessfulQueryTime = time.Now()
		stats.WindowQueries++
		stats.Status = "healthy"
		stats.Mu.Unlock()

		return result, nil
	case err := <-errorChan:
		// 收到错误
		duration := time.Since(startTime)
		f.logger.Debug("转发查询 - 服务器 %s 响应失败, 耗时: %v, 错误: %v", addr, duration, err)

		// 更新服务器统计信息
		stats.Mu.Lock()
		stats.Queries++
		stats.FailedQueries++
		stats.LastQueryTime = time.Now()
		// 检查失败率，超过50%标记为不健康
		if stats.Queries > 10 && float64(stats.FailedQueries)/float64(stats.Queries) > 0.5 {
			stats.Status = "unhealthy"
		}
		stats.Mu.Unlock()

		return nil, err
	}
}
