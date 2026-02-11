// core/sdns/security.go

package sdns

import (
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"

	"SteadyDNS/core/common"
)

// DNSMessageValidator DNS消息验证器
type DNSMessageValidator struct {
	logger            *common.Logger
	messageSizeLimit  int
	validationEnabled bool
}

// NewDNSMessageValidator 创建DNS消息验证器
func NewDNSMessageValidator(logger *common.Logger) *DNSMessageValidator {
	// 从配置文件读取设置
	messageSizeLimit := common.GetConfigInt("Security", "DNS_MESSAGE_SIZE_LIMIT", 4096)
	validationEnabled := common.GetConfigBool("Security", "DNS_VALIDATION_ENABLED", true)

	return &DNSMessageValidator{
		logger:            logger,
		messageSizeLimit:  messageSizeLimit,
		validationEnabled: validationEnabled,
	}
}

// ValidateQuery 验证DNS查询消息
func (v *DNSMessageValidator) ValidateQuery(msg *dns.Msg) (bool, string) {
	// 如果验证被禁用，直接通过
	if !v.validationEnabled {
		return true, "验证已禁用"
	}

	// 检查消息格式
	if msg == nil {
		return false, "空的DNS消息"
	}

	// 检查查询数量
	if len(msg.Question) == 0 {
		return false, "DNS消息中没有查询"
	}

	// 检查查询数量是否合理
	if len(msg.Question) > 10 {
		return false, "DNS消息中查询数量过多"
	}

	// 检查每个查询
	for i, q := range msg.Question {
		// 检查查询名称
		if q.Name == "" {
			return false, "DNS查询名称为空"
		}

		// 检查查询名称长度
		if len(q.Name) > 253 {
			return false, "DNS查询名称过长"
		}

		// 检查查询类型
		if q.Qtype == 0 {
			return false, "DNS查询类型无效"
		}

		// 检查查询类型是否合理
		if !isValidQueryType(q.Qtype) {
			return false, "DNS查询类型不支持"
		}

		// 检查查询类
		if q.Qclass != dns.ClassINET {
			return false, "只支持IN类DNS查询"
		}

		v.logger.Debug("验证DNS查询 %d: 名称=%s, 类型=%s, 类=%s",
			i+1, q.Name, dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass])
	}

	// 检查消息大小
	msgBytes, err := msg.Pack()
	if err != nil {
		return false, "无法计算DNS消息大小"
	}

	// 限制消息大小
	if len(msgBytes) > v.messageSizeLimit {
		return false, "DNS消息过大"
	}

	return true, "验证通过"
}

// ValidateResponse 验证DNS响应消息
func (v *DNSMessageValidator) ValidateResponse(req *dns.Msg, resp *dns.Msg) (bool, string) {
	// 如果验证被禁用，直接通过
	if !v.validationEnabled {
		return true, "验证已禁用"
	}

	// 检查响应消息
	if resp == nil {
		return false, "空的DNS响应"
	}

	// 验证查询ID匹配
	if req != nil && resp.Id != req.Id {
		return false, "DNS响应ID与查询不匹配"
	}

	// 验证响应标志
	if resp.Opcode != dns.OpcodeQuery {
		return false, "只支持查询类型的DNS响应"
	}

	// 验证查询部分
	if len(resp.Question) == 0 {
		return false, "DNS响应中没有查询部分"
	}

	// 如果有原始查询，验证查询部分匹配
	if req != nil {
		if len(resp.Question) != len(req.Question) {
			return false, "DNS响应中的查询数量与原始查询不匹配"
		}

		for i, q := range resp.Question {
			if i >= len(req.Question) {
				break
			}

			reqQ := req.Question[i]
			if q.Name != reqQ.Name || q.Qtype != reqQ.Qtype || q.Qclass != reqQ.Qclass {
				return false, "DNS响应中的查询与原始查询不匹配"
			}
		}
	}

	// 检查消息大小
	msgBytes, err := resp.Pack()
	if err != nil {
		return false, "无法计算DNS响应大小"
	}

	// 限制响应大小
	if len(msgBytes) > v.messageSizeLimit {
		return false, "DNS响应过大"
	}

	// 检查响应记录数量
	totalRecords := len(resp.Answer) + len(resp.Ns) + len(resp.Extra)
	if totalRecords > 100 {
		return false, "DNS响应中记录数量过多"
	}

	return true, "验证通过"
}

// isValidQueryType 检查查询类型是否有效
func isValidQueryType(qtype uint16) bool {
	// 支持的查询类型
	validTypes := map[uint16]bool{
		dns.TypeA:      true,
		dns.TypeAAAA:   true,
		dns.TypeCNAME:  true,
		dns.TypeMX:     true,
		dns.TypeNS:     true,
		dns.TypePTR:    true,
		dns.TypeSOA:    true,
		dns.TypeSRV:    true,
		dns.TypeTXT:    true,
		dns.TypeANY:    true,
		dns.TypeDNSKEY: true,
		dns.TypeRRSIG:  true,
		dns.TypeDS:     true,
		dns.TypeNSEC:   true,
		dns.TypeNSEC3:  true,
	}

	return validTypes[qtype]
}

// DNSRateLimiter DNS查询速率限制器
type DNSRateLimiter struct {
	// IP级别的限制
	ipLimits   []map[string]*LimitCounter
	ipMutexes  []sync.RWMutex
	shardCount int

	// 全局限制
	globalLimit *LimitCounter
	globalMutex sync.Mutex

	// 配置
	maxQueriesPerMinuteIP     int
	maxQueriesPerMinuteGlobal int
	banDuration               time.Duration
	logger                    *common.Logger
}

// LimitCounter 限制计数器
type LimitCounter struct {
	requests    []time.Time
	mutex       sync.Mutex
	limit       int
	window      time.Duration
	failCount   int
	maxFailures int
	banDuration time.Duration
	lastCleanup time.Time
}

// NewLimitCounter 创建新的限制计数器
func NewLimitCounter(limit int, window time.Duration, maxFailures int, banDuration time.Duration) *LimitCounter {
	return &LimitCounter{
		requests:    make([]time.Time, 0),
		limit:       limit,
		window:      window,
		maxFailures: maxFailures,
		banDuration: banDuration,
		lastCleanup: time.Now(),
	}
}

// AddRequest 添加请求并检查是否超出限制
func (lc *LimitCounter) AddRequest() (bool, bool) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	now := time.Now()

	// 定期清理过期请求（每10秒清理一次）
	if now.Sub(lc.lastCleanup) > 10*time.Second {
		cutoff := now.Add(-lc.window)
		validRequests := make([]time.Time, 0, len(lc.requests))

		for _, reqTime := range lc.requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}

		lc.requests = validRequests
		lc.lastCleanup = now
	}

	// 检查是否超出限制
	if len(lc.requests) >= lc.limit {
		lc.failCount++
		return false, lc.failCount >= lc.maxFailures
	}

	// 添加新请求
	lc.requests = append(lc.requests, now)
	return true, false
}

// NewDNSRateLimiter 创建DNS查询速率限制器
func NewDNSRateLimiter(logger *common.Logger) *DNSRateLimiter {
	// 从配置文件读取设置
	ipLimit := common.GetConfigInt("Security", "DNS_RATE_LIMIT_PER_IP", 600)
	globalLimit := common.GetConfigInt("Security", "DNS_RATE_LIMIT_GLOBAL", 50000)
	banDurationMinutes := common.GetConfigInt("Security", "DNS_BAN_DURATION", 5)

	banDuration := time.Duration(banDurationMinutes) * time.Minute

	// 使用16个分片减少锁竞争
	shardCount := 16
	ipLimits := make([]map[string]*LimitCounter, shardCount)
	ipMutexes := make([]sync.RWMutex, shardCount)

	for i := 0; i < shardCount; i++ {
		ipLimits[i] = make(map[string]*LimitCounter)
	}

	return &DNSRateLimiter{
		ipLimits:                  ipLimits,
		ipMutexes:                 ipMutexes,
		shardCount:                shardCount,
		globalLimit:               NewLimitCounter(globalLimit, time.Minute, 0, 0),
		maxQueriesPerMinuteIP:     ipLimit,     // 每IP每分钟查询限制
		maxQueriesPerMinuteGlobal: globalLimit, // 全局每分钟查询限制
		banDuration:               banDuration,
		logger:                    logger,
	}
}

// getIPShard 获取IP对应的分片索引
func (rl *DNSRateLimiter) getIPShard(ip string) int {
	// 使用简单的哈希函数计算分片索引
	hash := 0
	for _, char := range ip {
		hash = (hash << 5) - hash + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash % rl.shardCount
}

// CheckAndLimit 检查并限制查询速率
func (rl *DNSRateLimiter) CheckAndLimit(clientIP string) (bool, string) {
	// 检查全局限制
	rl.globalMutex.Lock()
	allowed, _ := rl.globalLimit.AddRequest()
	rl.globalMutex.Unlock()

	if !allowed {
		return false, "全局查询速率限制"
	}

	// 检查IP限制
	shardIndex := rl.getIPShard(clientIP)
	rl.ipMutexes[shardIndex].Lock()
	counter, exists := rl.ipLimits[shardIndex][clientIP]
	if !exists {
		counter = NewLimitCounter(rl.maxQueriesPerMinuteIP, time.Minute, 10, rl.banDuration)
		rl.ipLimits[shardIndex][clientIP] = counter
	}
	rl.ipMutexes[shardIndex].Unlock()

	allowed, banned := counter.AddRequest()
	if !allowed {
		if banned {
			return false, "IP已被临时封禁"
		}
		return false, "IP查询速率限制"
	}

	return true, "通过速率限制检查"
}

// CleanupExpired 清理过期的限制计数器
func (rl *DNSRateLimiter) CleanupExpired() {
	now := time.Now()
	cutoff := now.Add(-time.Hour)

	// 遍历所有分片进行清理
	for i := 0; i < rl.shardCount; i++ {
		rl.ipMutexes[i].Lock()
		for ip, counter := range rl.ipLimits[i] {
			counter.mutex.Lock()
			// 检查是否有最近一小时的请求
			hasRecent := false
			for _, reqTime := range counter.requests {
				if reqTime.After(cutoff) {
					hasRecent = true
					break
				}
			}
			counter.mutex.Unlock()

			if !hasRecent {
				delete(rl.ipLimits[i], ip)
			}
		}
		rl.ipMutexes[i].Unlock()
	}
}

// StartCleanupTimer 启动清理定时器
func (rl *DNSRateLimiter) StartCleanupTimer() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			rl.CleanupExpired()
		}
	}()
}

// SetLimits 设置限制参数
func (rl *DNSRateLimiter) SetLimits(ipLimit, globalLimit int, banDuration time.Duration) {
	rl.maxQueriesPerMinuteIP = ipLimit

	rl.globalMutex.Lock()
	rl.maxQueriesPerMinuteGlobal = globalLimit
	rl.globalLimit = NewLimitCounter(globalLimit, time.Minute, 0, 0)
	rl.globalMutex.Unlock()

	rl.banDuration = banDuration
}

// GetStats 获取速率限制统计信息
func (rl *DNSRateLimiter) GetStats() map[string]interface{} {
	// 统计所有分片中的IP数量
	ipCount := 0
	for i := 0; i < rl.shardCount; i++ {
		rl.ipMutexes[i].RLock()
		ipCount += len(rl.ipLimits[i])
		rl.ipMutexes[i].RUnlock()
	}

	rl.globalMutex.Lock()
	globalRequests := len(rl.globalLimit.requests)
	rl.globalMutex.Unlock()

	return map[string]interface{}{
		"ip_limit_count":     ipCount,
		"global_requests":    globalRequests,
		"max_queries_ip":     rl.maxQueriesPerMinuteIP,
		"max_queries_global": rl.maxQueriesPerMinuteGlobal,
		"ban_duration":       rl.banDuration,
	}
}

// SecurityManager 安全管理器
type SecurityManager struct {
	messageValidator *DNSMessageValidator
	rateLimiter      *DNSRateLimiter
	logger           *common.Logger
}

// NewSecurityManager 创建安全管理器
func NewSecurityManager(logger *common.Logger) *SecurityManager {
	manager := &SecurityManager{
		messageValidator: NewDNSMessageValidator(logger),
		rateLimiter:      NewDNSRateLimiter(logger),
		logger:           logger,
	}

	// 启动清理定时器
	manager.rateLimiter.StartCleanupTimer()

	return manager
}

// ValidateDNSMessage 验证DNS消息
func (sm *SecurityManager) ValidateDNSMessage(msg *dns.Msg, isQuery bool) (bool, string) {
	if isQuery {
		return sm.messageValidator.ValidateQuery(msg)
	}
	return true, "验证通过"
}

// CheckRateLimit 检查速率限制
func (sm *SecurityManager) CheckRateLimit(clientIP string) (bool, string) {
	return sm.rateLimiter.CheckAndLimit(clientIP)
}

// GetStats 获取安全统计信息
func (sm *SecurityManager) GetStats() map[string]interface{} {
	return sm.rateLimiter.GetStats()
}

// ExtractClientIP 从地址中提取客户端IP
func ExtractClientIP(addr net.Addr) string {
	switch a := addr.(type) {
	case *net.UDPAddr:
		return a.IP.String()
	case *net.TCPAddr:
		return a.IP.String()
	default:
		return addr.String()
	}
}
