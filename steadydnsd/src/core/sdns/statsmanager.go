// core/sdns/statsmanager.go

package sdns

import (
	"sort"
	"sync"
	"time"

	"SteadyDNS/core/common"
)

// StatsManager 统计管理器
type StatsManager struct {
	logger           *common.Logger
	mutex            sync.RWMutex
	networkStats     *NetworkStats
	domainCounters   map[string]int64
	clientCounters   map[string]int64
	qpsHistory       []QPSDataPoint
	latencyData      []int64
	resourceHistory  []ResourceDataPoint
	lastCleanupTime  time.Time
	cleanupInterval  time.Duration
	maxHistoryPoints int

	// 计算结果缓存
	cacheMutex      sync.RWMutex
	topDomainsCache map[int][]DomainStat
	topClientsCache map[int][]ClientStat
	qpsHistoryCache map[string][]QPSDataPoint
	cacheExpiration time.Duration
	lastCacheUpdate time.Time
}

// QPSDataPoint QPS数据点
type QPSDataPoint struct {
	Time time.Time
	QPS  float64
}

// ResourceDataPoint 资源使用数据点
type ResourceDataPoint struct {
	Time   time.Time
	CPU    int
	Memory int
	Disk   int
}

// NewStatsManager 创建统计管理器
func NewStatsManager(logger *common.Logger) *StatsManager {
	return &StatsManager{
		logger:           logger,
		networkStats:     &NetworkStats{LastRequestTime: time.Now()},
		domainCounters:   make(map[string]int64),
		clientCounters:   make(map[string]int64),
		qpsHistory:       make([]QPSDataPoint, 0),
		latencyData:      make([]int64, 0),
		resourceHistory:  make([]ResourceDataPoint, 0),
		lastCleanupTime:  time.Now(),
		cleanupInterval:  5 * time.Minute,
		maxHistoryPoints: 10080, // 7天 * 24小时 * 60分钟

		// 初始化缓存
		topDomainsCache: make(map[int][]DomainStat),
		topClientsCache: make(map[int][]ClientStat),
		qpsHistoryCache: make(map[string][]QPSDataPoint),
		cacheExpiration: 30 * time.Second, // 缓存30秒
		lastCacheUpdate: time.Now(),
	}
}

// UpdateNetworkStats 更新网络统计信息
func (sm *StatsManager) UpdateNetworkStats(bytesIn, bytesOut int, success bool, responseTime time.Duration) {
	// 计算响应时间（在锁外计算）
	latencyMs := int64(responseTime.Milliseconds())

	sm.mutex.Lock()
	{
		sm.networkStats.TotalRequests++
		sm.networkStats.TotalBytesIn += int64(bytesIn)
		sm.networkStats.TotalBytesOut += int64(bytesOut)
		sm.networkStats.LastRequestTime = time.Now()

		if success {
			sm.networkStats.SuccessfulRequests++
		} else {
			sm.networkStats.FailedRequests++
		}

		// 添加延迟数据
		sm.latencyData = append(sm.latencyData, latencyMs)

		// 限制延迟数据点数量
		if len(sm.latencyData) > 10000 {
			sm.latencyData = sm.latencyData[len(sm.latencyData)-10000:]
		}

		// 检查是否需要清理
		now := time.Now()
		if now.Sub(sm.lastCleanupTime) > sm.cleanupInterval {
			// 异步执行清理
			go sm.asyncCleanup()
		}
	}
	sm.mutex.Unlock()

	// 异步计算QPS并更新历史（在锁外执行）
	go sm.asyncUpdateQPSHistory()
}

// asyncCleanup 异步清理过期数据
func (sm *StatsManager) asyncCleanup() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 检查是否已经在清理中，避免重复清理
	if time.Since(sm.lastCleanupTime) < sm.cleanupInterval/2 {
		return
	}

	now := time.Now()
	cutoff := now.Add(-7 * 24 * time.Hour)

	// 清理QPS历史数据
	var filteredQPS []QPSDataPoint
	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) {
			filteredQPS = append(filteredQPS, point)
		}
	}
	sm.qpsHistory = filteredQPS

	// 清理资源使用历史数据
	var filteredResources []ResourceDataPoint
	for _, point := range sm.resourceHistory {
		if point.Time.After(cutoff) {
			filteredResources = append(filteredResources, point)
		}
	}
	sm.resourceHistory = filteredResources

	// 限制延迟数据点数量
	if len(sm.latencyData) > 10000 {
		sm.latencyData = sm.latencyData[len(sm.latencyData)-10000:]
	}

	// 定期重置计数器（每天）
	if now.Sub(sm.lastCleanupTime) > 24*time.Hour {
		sm.domainCounters = make(map[string]int64)
		sm.clientCounters = make(map[string]int64)
	}

	sm.lastCleanupTime = now
}

// asyncUpdateQPSHistory 异步更新QPS历史
func (sm *StatsManager) asyncUpdateQPSHistory() {
	// 计算QPS（在锁外计算）
	qps := sm.calculateQPSAsync()

	sm.mutex.Lock()
	{
		now := time.Now()
		sm.qpsHistory = append(sm.qpsHistory, QPSDataPoint{
			Time: now,
			QPS:  qps,
		})

		// 限制历史数据点数量
		if len(sm.qpsHistory) > sm.maxHistoryPoints {
			sm.qpsHistory = sm.qpsHistory[len(sm.qpsHistory)-sm.maxHistoryPoints:]
		}
	}
	sm.mutex.Unlock()
}

// calculateQPSAsync 异步计算QPS
func (sm *StatsManager) calculateQPSAsync() float64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	now := time.Now()
	cutoff := now.Add(-time.Second)
	count := int64(0)

	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) {
			count++
		}
	}

	return float64(count)
}

// RecordQuery 记录DNS查询
func (sm *StatsManager) RecordQuery(domain, clientIP string, responseTime time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 记录域名查询次数
	sm.domainCounters[domain]++

	// 记录客户端查询次数
	sm.clientCounters[clientIP]++

	// 记录响应时间
	latencyMs := int64(responseTime.Milliseconds())
	sm.latencyData = append(sm.latencyData, latencyMs)

	// 限制延迟数据点数量
	if len(sm.latencyData) > 10000 {
		sm.latencyData = sm.latencyData[len(sm.latencyData)-10000:]
	}
}

// CalculateQPS 计算当前QPS
func (sm *StatsManager) CalculateQPS() float64 {
	now := time.Now()
	cutoff := now.Add(-time.Second)
	count := int64(0)

	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) {
			count++
		}
	}

	return float64(count)
}

// GetNetworkStats 获取网络统计信息
func (sm *StatsManager) GetNetworkStats() *NetworkStats {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := *sm.networkStats
	return &stats
}

// GetDomainCounters 获取域名计数器
func (sm *StatsManager) GetDomainCounters() map[string]int64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	counters := make(map[string]int64)
	for domain, count := range sm.domainCounters {
		counters[domain] = count
	}
	return counters
}

// GetClientCounters 获取客户端计数器
func (sm *StatsManager) GetClientCounters() map[string]int64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	counters := make(map[string]int64)
	for client, count := range sm.clientCounters {
		counters[client] = count
	}
	return counters
}

// GetQPSHistory 获取QPS历史数据
func (sm *StatsManager) GetQPSHistory(timeRange string) []QPSDataPoint {
	// 检查缓存
	if history := sm.getCachedQPSHistory(timeRange); history != nil {
		return history
	}

	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var history []QPSDataPoint
	now := time.Now()
	var cutoff time.Time

	switch timeRange {
	case "1h":
		cutoff = now.Add(-1 * time.Hour)
	case "6h":
		cutoff = now.Add(-6 * time.Hour)
	case "24h":
		cutoff = now.Add(-24 * time.Hour)
	case "7d":
		cutoff = now.Add(-7 * 24 * time.Hour)
	default:
		cutoff = now.Add(-1 * time.Hour)
	}

	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) {
			history = append(history, point)
		}
	}

	// 更新缓存
	sm.updateQPSHistoryCache(timeRange, history)

	return history
}

// GetLatencyDistribution 获取延迟分布数据
func (sm *StatsManager) GetLatencyDistribution() map[string]int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	distribution := map[string]int{
		"<10ms":    0,
		"10-20ms":  0,
		"20-50ms":  0,
		"50-100ms": 0,
		">100ms":   0,
	}

	for _, latency := range sm.latencyData {
		switch {
		case latency < 10:
			distribution["<10ms"]++
		case latency < 20:
			distribution["10-20ms"]++
		case latency < 50:
			distribution["20-50ms"]++
		case latency < 100:
			distribution["50-100ms"]++
		default:
			distribution[">100ms"]++
		}
	}

	// 计算百分比
	total := len(sm.latencyData)
	if total > 0 {
		for rangeKey := range distribution {
			distribution[rangeKey] = (distribution[rangeKey] * 100) / total
		}
	}

	return distribution
}

// GetTopDomains 获取热门域名
func (sm *StatsManager) GetTopDomains(limit int) []DomainStat {
	// 检查缓存
	if stats := sm.getCachedTopDomains(limit); stats != nil {
		return stats
	}

	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	type domainStat struct {
		domain string
		count  int64
	}

	var stats []domainStat
	for domain, count := range sm.domainCounters {
		stats = append(stats, domainStat{domain, count})
	}

	// 按查询次数排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].count > stats[j].count
	})

	// 限制返回数量
	if len(stats) > limit {
		stats = stats[:limit]
	}

	// 转换为返回格式
	totalQueries := sm.networkStats.TotalRequests
	result := make([]DomainStat, len(stats))
	for i, stat := range stats {
		percentage := 0.0
		if totalQueries > 0 {
			percentage = float64(stat.count) / float64(totalQueries) * 100
		}
		result[i] = DomainStat{
			Rank:       i + 1,
			Domain:     stat.domain,
			Queries:    int(stat.count),
			Percentage: percentage,
		}
	}

	// 更新缓存
	sm.updateTopDomainsCache(limit, result)

	return result
}

// GetTopClients 获取热门客户端
func (sm *StatsManager) GetTopClients(limit int) []ClientStat {
	// 检查缓存
	if stats := sm.getCachedTopClients(limit); stats != nil {
		return stats
	}

	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	type clientStat struct {
		client string
		count  int64
	}

	var stats []clientStat
	for client, count := range sm.clientCounters {
		stats = append(stats, clientStat{client, count})
	}

	// 按查询次数排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].count > stats[j].count
	})

	// 限制返回数量
	if len(stats) > limit {
		stats = stats[:limit]
	}

	// 转换为返回格式
	totalQueries := sm.networkStats.TotalRequests
	result := make([]ClientStat, len(stats))
	for i, stat := range stats {
		percentage := 0.0
		if totalQueries > 0 {
			percentage = float64(stat.count) / float64(totalQueries) * 100
		}
		result[i] = ClientStat{
			Rank:       i + 1,
			IP:         stat.client,
			Queries:    int(stat.count),
			Percentage: percentage,
		}
	}

	// 更新缓存
	sm.updateTopClientsCache(limit, result)

	return result
}

// getCachedTopDomains 获取缓存的热门域名
func (sm *StatsManager) getCachedTopDomains(limit int) []DomainStat {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	// 检查缓存是否过期
	if time.Since(sm.lastCacheUpdate) > sm.cacheExpiration {
		return nil
	}

	// 检查缓存是否存在
	if stats, exists := sm.topDomainsCache[limit]; exists {
		return stats
	}

	return nil
}

// updateTopDomainsCache 更新热门域名缓存
func (sm *StatsManager) updateTopDomainsCache(limit int, stats []DomainStat) {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	sm.topDomainsCache[limit] = stats
	sm.lastCacheUpdate = time.Now()
}

// getCachedTopClients 获取缓存的热门客户端
func (sm *StatsManager) getCachedTopClients(limit int) []ClientStat {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	// 检查缓存是否过期
	if time.Since(sm.lastCacheUpdate) > sm.cacheExpiration {
		return nil
	}

	// 检查缓存是否存在
	if stats, exists := sm.topClientsCache[limit]; exists {
		return stats
	}

	return nil
}

// updateTopClientsCache 更新热门客户端缓存
func (sm *StatsManager) updateTopClientsCache(limit int, stats []ClientStat) {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	sm.topClientsCache[limit] = stats
	sm.lastCacheUpdate = time.Now()
}

// clearCache 清理缓存
func (sm *StatsManager) clearCache() {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	sm.topDomainsCache = make(map[int][]DomainStat)
	sm.topClientsCache = make(map[int][]ClientStat)
	sm.qpsHistoryCache = make(map[string][]QPSDataPoint)
	sm.lastCacheUpdate = time.Now()
}

// DomainStat 域名统计信息
type DomainStat struct {
	Rank       int     `json:"rank"`
	Domain     string  `json:"domain"`
	Queries    int     `json:"queries"`
	Percentage float64 `json:"percentage"`
}

// ClientStat 客户端统计信息
type ClientStat struct {
	Rank       int     `json:"rank"`
	IP         string  `json:"ip"`
	Queries    int     `json:"queries"`
	Percentage float64 `json:"percentage"`
}

// cleanup 清理过期数据
func (sm *StatsManager) cleanup() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-7 * 24 * time.Hour)

	// 清理QPS历史数据
	var filteredQPS []QPSDataPoint
	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) {
			filteredQPS = append(filteredQPS, point)
		}
	}
	sm.qpsHistory = filteredQPS

	// 清理资源使用历史数据
	var filteredResources []ResourceDataPoint
	for _, point := range sm.resourceHistory {
		if point.Time.After(cutoff) {
			filteredResources = append(filteredResources, point)
		}
	}
	sm.resourceHistory = filteredResources

	// 限制延迟数据点数量
	if len(sm.latencyData) > 10000 {
		sm.latencyData = sm.latencyData[len(sm.latencyData)-10000:]
	}

	// 定期重置计数器（每天）
	if now.Sub(sm.lastCleanupTime) > 24*time.Hour {
		sm.domainCounters = make(map[string]int64)
		sm.clientCounters = make(map[string]int64)
		sm.lastCleanupTime = now
	}
}

// UpdateResourceUsage 更新资源使用情况
func (sm *StatsManager) UpdateResourceUsage(cpu, memory, disk int) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	sm.resourceHistory = append(sm.resourceHistory, ResourceDataPoint{
		Time:   now,
		CPU:    cpu,
		Memory: memory,
		Disk:   disk,
	})

	// 限制历史数据点数量
	if len(sm.resourceHistory) > sm.maxHistoryPoints {
		sm.resourceHistory = sm.resourceHistory[len(sm.resourceHistory)-sm.maxHistoryPoints:]
	}
}

// getCachedQPSHistory 获取缓存的QPS历史数据
func (sm *StatsManager) getCachedQPSHistory(timeRange string) []QPSDataPoint {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	// 检查缓存是否过期
	if time.Since(sm.lastCacheUpdate) > sm.cacheExpiration {
		return nil
	}

	// 检查缓存是否存在
	if history, exists := sm.qpsHistoryCache[timeRange]; exists {
		return history
	}

	return nil
}

// updateQPSHistoryCache 更新QPS历史数据缓存
func (sm *StatsManager) updateQPSHistoryCache(timeRange string, history []QPSDataPoint) {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	sm.qpsHistoryCache[timeRange] = history
	sm.lastCacheUpdate = time.Now()
}

// GetResourceHistory 获取资源使用历史数据
func (sm *StatsManager) GetResourceHistory(timeRange string) []ResourceDataPoint {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var history []ResourceDataPoint
	now := time.Now()
	var cutoff time.Time

	switch timeRange {
	case "1h":
		cutoff = now.Add(-1 * time.Hour)
	case "6h":
		cutoff = now.Add(-6 * time.Hour)
	case "24h":
		cutoff = now.Add(-24 * time.Hour)
	case "7d":
		cutoff = now.Add(-7 * 24 * time.Hour)
	default:
		cutoff = now.Add(-1 * time.Hour)
	}

	for _, point := range sm.resourceHistory {
		if point.Time.After(cutoff) {
			history = append(history, point)
		}
	}

	return history
}

// GetSystemHealth 获取系统健康度
func (sm *StatsManager) GetSystemHealth() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// 简单的健康度计算，综合考虑各种指标
	health := 100

	// 检查最近是否有请求
	if time.Since(sm.networkStats.LastRequestTime) > 5*time.Minute {
		health -= 20
	}

	// 检查失败率
	failureRate := 0.0
	totalRequests := sm.networkStats.TotalRequests
	if totalRequests > 0 {
		failureRate = float64(sm.networkStats.FailedRequests) / float64(totalRequests) * 100
	}
	if failureRate > 10 {
		health -= int(failureRate)
	}

	// 确保健康度在0-100之间
	if health < 0 {
		health = 0
	}

	return health
}

// ExtractDomainFromQuery 从DNS查询中提取域名
func ExtractDomainFromQuery(query string) string {
	// 简单实现，实际应该解析DNS消息
	return query
}
