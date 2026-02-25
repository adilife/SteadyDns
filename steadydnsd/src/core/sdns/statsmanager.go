// core/sdns/statsmanager.go

package sdns

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
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
	networkHistory   []NetworkDataPoint
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

	// 持久化相关
	persistEnabled  bool
	persistInterval time.Duration
	lastPersistTime time.Time
	persistMutex    sync.Mutex
	stopPersist     chan struct{}
	retentionDays   int

	// 资源监控相关
	stopResourceMonitor chan struct{}

	// 网络流量监控相关
	networkMonitorMutex sync.RWMutex
	lastNetworkStats    *NetworkInterfaceStats
	currentNetworkSpeed *NetworkSpeed
}

// NetworkInterfaceStats 网络接口统计信息
type NetworkInterfaceStats struct {
	BytesRecv   uint64
	BytesSent   uint64
	PacketsRecv uint64
	PacketsSent uint64
	Timestamp   time.Time
}

// NetworkSpeed 网络速度
type NetworkSpeed struct {
	InboundBps  uint64
	OutboundBps uint64
	Timestamp   time.Time
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

// NetworkDataPoint 网络流量数据点
type NetworkDataPoint struct {
	Time        time.Time
	InboundBps  uint64
	OutboundBps uint64
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
		networkHistory:   make([]NetworkDataPoint, 0),
		lastCleanupTime:  time.Now(),
		cleanupInterval:  5 * time.Minute,
		maxHistoryPoints: 10080, // 7天 * 24小时 * 60分钟

		// 初始化缓存
		topDomainsCache: make(map[int][]DomainStat),
		topClientsCache: make(map[int][]ClientStat),
		qpsHistoryCache: make(map[string][]QPSDataPoint),
		cacheExpiration: 30 * time.Second, // 缓存30秒
		lastCacheUpdate: time.Now(),

		// 初始化持久化
		persistEnabled:  true,
		persistInterval: 1 * time.Minute,
		lastPersistTime: time.Now(),
		stopPersist:     make(chan struct{}),
		retentionDays:   7,

		// 初始化资源监控
		stopResourceMonitor: make(chan struct{}),

		// 初始化网络流量监控
		currentNetworkSpeed: &NetworkSpeed{Timestamp: time.Now()},
	}
}

// UpdateNetworkStats 更新网络统计信息
// 注意：延迟数据已在 RecordQuery 中记录，此方法仅更新网络流量统计
func (sm *StatsManager) UpdateNetworkStats(bytesIn, bytesOut int, success bool) {
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

	// 清理网络流量历史数据
	var filteredNetwork []NetworkDataPoint
	for _, point := range sm.networkHistory {
		if point.Time.After(cutoff) {
			filteredNetwork = append(filteredNetwork, point)
		}
	}
	sm.networkHistory = filteredNetwork

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

// GetPeakQPS 获取指定时间范围内的峰值QPS
func (sm *StatsManager) GetPeakQPS(duration time.Duration) float64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	maxQPS := 0.0

	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) && point.QPS > maxQPS {
			maxQPS = point.QPS
		}
	}

	return maxQPS
}

// GetAverageQPS 获取指定时间范围内的平均QPS
func (sm *StatsManager) GetAverageQPS(duration time.Duration) float64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	sum := 0.0
	count := 0

	for _, point := range sm.qpsHistory {
		if point.Time.After(cutoff) {
			sum += point.QPS
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}

// GetCurrentQPS 获取当前QPS（最近1秒）
func (sm *StatsManager) GetCurrentQPS() float64 {
	return sm.CalculateQPS()
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

// StartPersistence 启动后台持久化任务
func (sm *StatsManager) StartPersistence() {
	if !sm.persistEnabled {
		return
	}

	go func() {
		ticker := time.NewTicker(sm.persistInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.persistToDatabase()
				sm.cleanOldDatabaseRecords()
			case <-sm.stopPersist:
				sm.logger.Info("QPS历史数据持久化任务已停止")
				return
			}
		}
	}()

	sm.logger.Info("QPS历史数据持久化任务已启动，间隔: %v", sm.persistInterval)
}

// StopPersistence 停止后台持久化任务
func (sm *StatsManager) StopPersistence() {
	if sm.stopPersist != nil {
		close(sm.stopPersist)
	}
}

// StartResourceMonitor 启动系统资源监控
func (sm *StatsManager) StartResourceMonitor() {
	sm.collectNetworkStats()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.collectResourceUsage()
			case <-sm.stopResourceMonitor:
				sm.logger.Info("系统资源监控任务已停止")
				return
			}
		}
	}()

	sm.logger.Info("系统资源监控任务已启动，采集间隔: 10s")
}

// StopResourceMonitor 停止系统资源监控
func (sm *StatsManager) StopResourceMonitor() {
	if sm.stopResourceMonitor != nil {
		close(sm.stopResourceMonitor)
	}
}

// collectResourceUsage 采集系统资源使用情况
func (sm *StatsManager) collectResourceUsage() {
	cpu, memory, disk := sm.getSystemResourceUsage()
	sm.UpdateResourceUsage(cpu, memory, disk)
	sm.collectNetworkStats()

	// 保存网络流量数据到历史记录
	sm.networkMonitorMutex.RLock()
	if sm.currentNetworkSpeed != nil {
		sm.UpdateNetworkHistory(sm.currentNetworkSpeed.InboundBps, sm.currentNetworkSpeed.OutboundBps)
	}
	sm.networkMonitorMutex.RUnlock()
}

// getSystemResourceUsage 获取系统资源使用率
func (sm *StatsManager) getSystemResourceUsage() (cpu int, memory int, disk int) {
	cpu = sm.getCPUUsage()
	memory = sm.getMemoryUsage()
	disk = sm.getDiskUsage()
	return
}

// GetSystemResourceUsageForAPI 获取系统资源使用率（供API调用）
func (sm *StatsManager) GetSystemResourceUsageForAPI() (cpu int, memory int, disk int) {
	return sm.getSystemResourceUsage()
}

// getCPUUsage 获取CPU使用率
func (sm *StatsManager) getCPUUsage() int {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 1 {
		return 0
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 5 {
		return 0
	}

	if fields[0] != "cpu" {
		return 0
	}

	user, _ := strconv.ParseInt(fields[1], 10, 64)
	nice, _ := strconv.ParseInt(fields[2], 10, 64)
	system, _ := strconv.ParseInt(fields[3], 10, 64)
	idle, _ := strconv.ParseInt(fields[4], 10, 64)

	total := user + nice + system + idle
	if total == 0 {
		return 0
	}

	used := user + nice + system
	usage := int((used * 100) / total)

	if usage > 100 {
		usage = 100
	}

	return usage
}

// getMemoryUsage 获取内存使用率
func (sm *StatsManager) getMemoryUsage() int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	var total, available int64

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, _ := strconv.ParseInt(fields[1], 10, 64)

		switch fields[0] {
		case "MemTotal:":
			total = value
		case "MemAvailable:":
			available = value
		}
	}

	if total == 0 {
		return 0
	}

	used := total - available
	usage := int((used * 100) / total)

	if usage > 100 {
		usage = 100
	}

	return usage
}

// getDiskUsage 获取磁盘使用率
func (sm *StatsManager) getDiskUsage() int {
	var stat syscall.Statfs_t

	wd, err := os.Getwd()
	if err != nil {
		return 0
	}

	err = syscall.Statfs(wd, &stat)
	if err != nil {
		return 0
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)

	if total == 0 {
		return 0
	}

	used := total - free
	usage := int((used * 100) / total)

	if usage > 100 {
		usage = 100
	}

	return usage
}

// persistToDatabase 将内存中的QPS历史数据持久化到数据库
func (sm *StatsManager) persistToDatabase() {
	sm.persistMutex.Lock()
	defer sm.persistMutex.Unlock()

	sm.mutex.RLock()
	if len(sm.qpsHistory) == 0 && len(sm.resourceHistory) == 0 && len(sm.networkHistory) == 0 {
		sm.mutex.RUnlock()
		return
	}

	lastPersist := sm.lastPersistTime
	var toPersistQPS []QPSDataPoint
	for _, point := range sm.qpsHistory {
		if point.Time.After(lastPersist) {
			toPersistQPS = append(toPersistQPS, point)
		}
	}

	var toPersistResource []ResourceDataPoint
	for _, point := range sm.resourceHistory {
		if point.Time.After(lastPersist) {
			toPersistResource = append(toPersistResource, point)
		}
	}

	var toPersistNetwork []NetworkDataPoint
	for _, point := range sm.networkHistory {
		if point.Time.After(lastPersist) {
			toPersistNetwork = append(toPersistNetwork, point)
		}
	}
	sm.mutex.RUnlock()

	if len(toPersistQPS) > 0 {
		records := make([]database.QPSHistory, len(toPersistQPS))
		for i, point := range toPersistQPS {
			records[i] = database.QPSHistory{
				Timestamp: point.Time,
				QPS:       point.QPS,
			}
		}

		if err := database.SaveQPSHistoryBatch(records); err != nil {
			sm.logger.Error("持久化QPS历史数据失败: %v", err)
		} else {
			sm.logger.Debug("持久化了 %d 条QPS历史记录", len(toPersistQPS))
		}
	}

	if len(toPersistResource) > 0 {
		records := make([]database.ResourceHistory, len(toPersistResource))
		for i, point := range toPersistResource {
			records[i] = database.ResourceHistory{
				Timestamp: point.Time,
				CPU:       point.CPU,
				Memory:    point.Memory,
				Disk:      point.Disk,
			}
		}

		if err := database.SaveResourceHistoryBatch(records); err != nil {
			sm.logger.Error("持久化资源使用历史数据失败: %v", err)
		} else {
			sm.logger.Debug("持久化了 %d 条资源使用历史记录", len(toPersistResource))
		}
	}

	if len(toPersistNetwork) > 0 {
		records := make([]database.NetworkHistory, len(toPersistNetwork))
		for i, point := range toPersistNetwork {
			records[i] = database.NetworkHistory{
				Timestamp:   point.Time,
				InboundBps:  point.InboundBps,
				OutboundBps: point.OutboundBps,
			}
		}

		if err := database.SaveNetworkHistoryBatch(records); err != nil {
			sm.logger.Error("持久化网络流量历史数据失败: %v", err)
		} else {
			sm.logger.Debug("持久化了 %d 条网络流量历史记录", len(toPersistNetwork))
		}
	}

	if len(toPersistQPS) > 0 || len(toPersistResource) > 0 || len(toPersistNetwork) > 0 {
		sm.mutex.Lock()
		if len(toPersistQPS) > 0 {
			sm.lastPersistTime = toPersistQPS[len(toPersistQPS)-1].Time
		} else if len(toPersistResource) > 0 {
			sm.lastPersistTime = toPersistResource[len(toPersistResource)-1].Time
		} else if len(toPersistNetwork) > 0 {
			sm.lastPersistTime = toPersistNetwork[len(toPersistNetwork)-1].Time
		}
		sm.mutex.Unlock()
	}
}

// cleanOldDatabaseRecords 清理数据库中的过期记录
func (sm *StatsManager) cleanOldDatabaseRecords() {
	if err := database.CleanOldQPSHistory(sm.retentionDays); err != nil {
		sm.logger.Error("清理过期QPS历史记录失败: %v", err)
	}
	if err := database.CleanOldResourceHistory(sm.retentionDays); err != nil {
		sm.logger.Error("清理过期资源使用历史记录失败: %v", err)
	}
	if err := database.CleanOldNetworkHistory(sm.retentionDays); err != nil {
		sm.logger.Error("清理过期网络流量历史记录失败: %v", err)
	}
}

// LoadFromDatabase 从数据库加载历史数据到内存
func (sm *StatsManager) LoadFromDatabase(timeRange string) error {
	var cutoff time.Time
	now := time.Now()

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

	qpsRecords, err := database.GetQPSHistoryByTimeRange(cutoff, now)
	if err != nil {
		sm.logger.Warn("加载QPS历史数据失败: %v", err)
	}

	resourceRecords, err := database.GetResourceHistoryByTimeRange(cutoff, now)
	if err != nil {
		sm.logger.Warn("加载资源使用历史数据失败: %v", err)
	}

	networkRecords, err := database.GetNetworkHistoryByTimeRange(cutoff, now)
	if err != nil {
		sm.logger.Warn("加载网络流量历史数据失败: %v", err)
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for _, record := range qpsRecords {
		sm.qpsHistory = append(sm.qpsHistory, QPSDataPoint{
			Time: record.Timestamp,
			QPS:  record.QPS,
		})
	}

	for _, record := range resourceRecords {
		sm.resourceHistory = append(sm.resourceHistory, ResourceDataPoint{
			Time:   record.Timestamp,
			CPU:    record.CPU,
			Memory: record.Memory,
			Disk:   record.Disk,
		})
	}

	for _, record := range networkRecords {
		sm.networkHistory = append(sm.networkHistory, NetworkDataPoint{
			Time:        record.Timestamp,
			InboundBps:  record.InboundBps,
			OutboundBps: record.OutboundBps,
		})
	}

	if len(sm.qpsHistory) > 0 {
		sm.lastPersistTime = sm.qpsHistory[len(sm.qpsHistory)-1].Time
	} else if len(sm.resourceHistory) > 0 {
		sm.lastPersistTime = sm.resourceHistory[len(sm.resourceHistory)-1].Time
	} else if len(sm.networkHistory) > 0 {
		sm.lastPersistTime = sm.networkHistory[len(sm.networkHistory)-1].Time
	}

	sm.logger.Info("从数据库加载了 %d 条QPS历史记录、%d 条资源使用历史记录和 %d 条网络流量历史记录",
		len(qpsRecords), len(resourceRecords), len(networkRecords))
	return nil
}

// AggregatedQPSTrend 聚合后的QPS趋势数据
type AggregatedQPSTrend struct {
	TimeLabels []string        `json:"timeLabels"`
	QPSValues  []float64       `json:"qpsValues"`
	Statistics TrendStatistics `json:"statistics"`
}

// TrendStatistics 趋势统计信息
type TrendStatistics struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Avg        float64 `json:"avg"`
	Current    float64 `json:"current"`
	DataPoints int     `json:"dataPoints"`
}

// GetAggregatedQPSTrend 获取聚合后的QPS趋势数据
func (sm *StatsManager) GetAggregatedQPSTrend(timeRange string, points int) *AggregatedQPSTrend {
	cacheKey := fmt.Sprintf("%s_%d", timeRange, points)
	if cached := sm.getCachedAggregatedQPSTrend(cacheKey); cached != nil {
		return cached
	}

	history := sm.GetQPSHistoryWithDB(timeRange)

	if len(history) == 0 {
		return &AggregatedQPSTrend{
			TimeLabels: []string{},
			QPSValues:  []float64{},
			Statistics: TrendStatistics{
				Min:        0,
				Max:        0,
				Avg:        0,
				Current:    0,
				DataPoints: 0,
			},
		}
	}

	result := sm.aggregateQPSData(history, points, timeRange)

	sm.cacheAggregatedQPSTrend(cacheKey, result)

	return result
}

// GetQPSHistoryWithDB 获取QPS历史数据（优先从内存，不足时从数据库补充）
func (sm *StatsManager) GetQPSHistoryWithDB(timeRange string) []QPSDataPoint {
	sm.mutex.RLock()
	memoryData := make([]QPSDataPoint, len(sm.qpsHistory))
	copy(memoryData, sm.qpsHistory)
	sm.mutex.RUnlock()

	var cutoff time.Time
	now := time.Now()

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

	var history []QPSDataPoint
	for _, point := range memoryData {
		if point.Time.After(cutoff) {
			history = append(history, point)
		}
	}

	if len(history) > 0 {
		oldestInMemory := history[0].Time
		if oldestInMemory.After(cutoff) {
			dbRecords, err := database.GetQPSHistoryByTimeRange(cutoff, oldestInMemory)
			if err == nil && len(dbRecords) > 0 {
				prepend := make([]QPSDataPoint, len(dbRecords))
				for i, r := range dbRecords {
					prepend[i] = QPSDataPoint{Time: r.Timestamp, QPS: r.QPS}
				}
				history = append(prepend, history...)
			}
		}
	} else {
		dbRecords, err := database.GetQPSHistoryByTimeRange(cutoff, now)
		if err == nil {
			history = make([]QPSDataPoint, len(dbRecords))
			for i, r := range dbRecords {
				history[i] = QPSDataPoint{Time: r.Timestamp, QPS: r.QPS}
			}
		}
	}

	return history
}

// aggregateQPSData 聚合QPS数据
func (sm *StatsManager) aggregateQPSData(data []QPSDataPoint, points int, timeRange string) *AggregatedQPSTrend {
	result := &AggregatedQPSTrend{
		TimeLabels: make([]string, 0),
		QPSValues:  make([]float64, 0),
	}

	if len(data) == 0 {
		return result
	}

	var timeFormat string
	switch timeRange {
	case "1h", "6h", "24h":
		timeFormat = "15:04"
	case "7d":
		timeFormat = "01-02 15:04"
	default:
		timeFormat = "15:04"
	}

	now := time.Now()
	var totalDuration time.Duration
	switch timeRange {
	case "1h":
		totalDuration = time.Hour
	case "6h":
		totalDuration = 6 * time.Hour
	case "24h":
		totalDuration = 24 * time.Hour
	case "7d":
		totalDuration = 7 * 24 * time.Hour
	default:
		totalDuration = time.Hour
	}

	if points <= 0 {
		points = 12
	}

	interval := totalDuration / time.Duration(points)
	for i := 0; i < points; i++ {
		slotStart := now.Add(-totalDuration + time.Duration(i)*interval)
		slotEnd := slotStart.Add(interval)

		var sum float64
		var count int
		for _, point := range data {
			if point.Time.After(slotStart) && !point.Time.After(slotEnd) {
				sum += point.QPS
				count++
			}
		}

		var avg float64
		if count > 0 {
			avg = round2(sum / float64(count))
		}

		result.TimeLabels = append(result.TimeLabels, slotStart.Format(timeFormat))
		result.QPSValues = append(result.QPSValues, avg)
	}

	stats := sm.calculateTrendStatistics(data)
	result.Statistics = stats

	return result
}

// calculateTrendStatistics 计算趋势统计信息
func (sm *StatsManager) calculateTrendStatistics(data []QPSDataPoint) TrendStatistics {
	if len(data) == 0 {
		return TrendStatistics{}
	}

	min := data[0].QPS
	max := data[0].QPS
	sum := 0.0

	for _, point := range data {
		if point.QPS < min {
			min = point.QPS
		}
		if point.QPS > max {
			max = point.QPS
		}
		sum += point.QPS
	}

	avg := sum / float64(len(data))
	current := data[len(data)-1].QPS

	return TrendStatistics{
		Min:        min,
		Max:        max,
		Avg:        round2(avg),
		Current:    current,
		DataPoints: len(data),
	}
}

// getCachedAggregatedQPSTrend 获取缓存的聚合QPS趋势数据
func (sm *StatsManager) getCachedAggregatedQPSTrend(cacheKey string) *AggregatedQPSTrend {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	if time.Since(sm.lastCacheUpdate) > sm.cacheExpiration {
		return nil
	}

	return nil
}

// cacheAggregatedQPSTrend 缓存聚合QPS趋势数据
func (sm *StatsManager) cacheAggregatedQPSTrend(cacheKey string, data *AggregatedQPSTrend) {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()
	sm.lastCacheUpdate = time.Now()
}

// SetPersistEnabled 设置是否启用持久化
func (sm *StatsManager) SetPersistEnabled(enabled bool) {
	sm.persistEnabled = enabled
}

// SetPersistInterval 设置持久化间隔
func (sm *StatsManager) SetPersistInterval(interval time.Duration) {
	sm.persistInterval = interval
}

// SetRetentionDays 设置数据保留天数
func (sm *StatsManager) SetRetentionDays(days int) {
	sm.retentionDays = days
}

// GetPersistenceStatus 获取持久化状态
func (sm *StatsManager) GetPersistenceStatus() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":         sm.persistEnabled,
		"interval":        sm.persistInterval.String(),
		"lastPersistTime": sm.lastPersistTime.Format("2006-01-02 15:04:05"),
		"retentionDays":   sm.retentionDays,
		"memoryPoints":    len(sm.qpsHistory),
	}
}

// AggregatedResourceUsage 聚合后的资源使用数据
type AggregatedResourceUsage struct {
	TimeLabels []string           `json:"timeLabels"`
	CPUValues  []int              `json:"cpuValues"`
	MemValues  []int              `json:"memValues"`
	DiskValues []int              `json:"diskValues"`
	Statistics ResourceStatistics `json:"statistics"`
}

// ResourceStatistics 资源使用统计信息
type ResourceStatistics struct {
	CPU    ResourceStatItem `json:"cpu"`
	Memory ResourceStatItem `json:"memory"`
	Disk   ResourceStatItem `json:"disk"`
}

// ResourceStatItem 单项资源统计
type ResourceStatItem struct {
	Min        int     `json:"min"`
	Max        int     `json:"max"`
	Avg        float64 `json:"avg"`
	DataPoints int     `json:"dataPoints"`
}

// GetAggregatedResourceUsage 获取聚合后的资源使用数据
func (sm *StatsManager) GetAggregatedResourceUsage(timeRange string, points int) *AggregatedResourceUsage {
	cacheKey := fmt.Sprintf("resource_%s_%d", timeRange, points)
	if cached := sm.getCachedAggregatedResourceUsage(cacheKey); cached != nil {
		return cached
	}

	history := sm.GetResourceHistoryWithDB(timeRange)

	if len(history) == 0 {
		return &AggregatedResourceUsage{
			TimeLabels: []string{},
			CPUValues:  []int{},
			MemValues:  []int{},
			DiskValues: []int{},
			Statistics: ResourceStatistics{},
		}
	}

	result := sm.aggregateResourceData(history, points, timeRange)

	sm.cacheAggregatedResourceUsage(cacheKey, result)

	return result
}

// GetResourceHistoryWithDB 获取资源使用历史数据（优先从内存，不足时从数据库补充）
func (sm *StatsManager) GetResourceHistoryWithDB(timeRange string) []ResourceDataPoint {
	sm.mutex.RLock()
	memoryData := make([]ResourceDataPoint, len(sm.resourceHistory))
	copy(memoryData, sm.resourceHistory)
	sm.mutex.RUnlock()

	var cutoff time.Time
	now := time.Now()

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

	var history []ResourceDataPoint
	for _, point := range memoryData {
		if point.Time.After(cutoff) {
			history = append(history, point)
		}
	}

	if len(history) > 0 {
		oldestInMemory := history[0].Time
		if oldestInMemory.After(cutoff) {
			dbRecords, err := database.GetResourceHistoryByTimeRange(cutoff, oldestInMemory)
			if err == nil && len(dbRecords) > 0 {
				prepend := make([]ResourceDataPoint, len(dbRecords))
				for i, r := range dbRecords {
					prepend[i] = ResourceDataPoint{Time: r.Timestamp, CPU: r.CPU, Memory: r.Memory, Disk: r.Disk}
				}
				history = append(prepend, history...)
			}
		}
	} else {
		dbRecords, err := database.GetResourceHistoryByTimeRange(cutoff, now)
		if err == nil {
			history = make([]ResourceDataPoint, len(dbRecords))
			for i, r := range dbRecords {
				history[i] = ResourceDataPoint{Time: r.Timestamp, CPU: r.CPU, Memory: r.Memory, Disk: r.Disk}
			}
		}
	}

	return history
}

// aggregateResourceData 聚合资源使用数据
func (sm *StatsManager) aggregateResourceData(data []ResourceDataPoint, points int, timeRange string) *AggregatedResourceUsage {
	result := &AggregatedResourceUsage{
		TimeLabels: make([]string, 0),
		CPUValues:  make([]int, 0),
		MemValues:  make([]int, 0),
		DiskValues: make([]int, 0),
	}

	if len(data) == 0 {
		return result
	}

	var timeFormat string
	switch timeRange {
	case "1h", "6h", "24h":
		timeFormat = "15:04"
	case "7d":
		timeFormat = "01-02 15:04"
	default:
		timeFormat = "15:04"
	}

	now := time.Now()
	var totalDuration time.Duration
	switch timeRange {
	case "1h":
		totalDuration = time.Hour
	case "6h":
		totalDuration = 6 * time.Hour
	case "24h":
		totalDuration = 24 * time.Hour
	case "7d":
		totalDuration = 7 * 24 * time.Hour
	default:
		totalDuration = time.Hour
	}

	if points <= 0 {
		points = 12
	}

	interval := totalDuration / time.Duration(points)
	for i := 0; i < points; i++ {
		slotStart := now.Add(-totalDuration + time.Duration(i)*interval)
		slotEnd := slotStart.Add(interval)

		var sumCPU, sumMem, sumDisk int
		var count int
		for _, point := range data {
			if point.Time.After(slotStart) && !point.Time.After(slotEnd) {
				sumCPU += point.CPU
				sumMem += point.Memory
				sumDisk += point.Disk
				count++
			}
		}

		var avgCPU, avgMem, avgDisk int
		if count > 0 {
			avgCPU = sumCPU / count
			avgMem = sumMem / count
			avgDisk = sumDisk / count
		}

		result.TimeLabels = append(result.TimeLabels, slotStart.Format(timeFormat))
		result.CPUValues = append(result.CPUValues, avgCPU)
		result.MemValues = append(result.MemValues, avgMem)
		result.DiskValues = append(result.DiskValues, avgDisk)
	}

	stats := sm.calculateResourceStatistics(data)
	result.Statistics = stats

	return result
}

// calculateResourceStatistics 计算资源使用统计信息
func (sm *StatsManager) calculateResourceStatistics(data []ResourceDataPoint) ResourceStatistics {
	if len(data) == 0 {
		return ResourceStatistics{}
	}

	minCPU := data[0].CPU
	maxCPU := data[0].CPU
	sumCPU := 0
	minMem := data[0].Memory
	maxMem := data[0].Memory
	sumMem := 0
	minDisk := data[0].Disk
	maxDisk := data[0].Disk
	sumDisk := 0

	for _, point := range data {
		if point.CPU < minCPU {
			minCPU = point.CPU
		}
		if point.CPU > maxCPU {
			maxCPU = point.CPU
		}
		sumCPU += point.CPU

		if point.Memory < minMem {
			minMem = point.Memory
		}
		if point.Memory > maxMem {
			maxMem = point.Memory
		}
		sumMem += point.Memory

		if point.Disk < minDisk {
			minDisk = point.Disk
		}
		if point.Disk > maxDisk {
			maxDisk = point.Disk
		}
		sumDisk += point.Disk
	}

	count := len(data)
	return ResourceStatistics{
		CPU: ResourceStatItem{
			Min:        minCPU,
			Max:        maxCPU,
			Avg:        round2(float64(sumCPU) / float64(count)),
			DataPoints: count,
		},
		Memory: ResourceStatItem{
			Min:        minMem,
			Max:        maxMem,
			Avg:        round2(float64(sumMem) / float64(count)),
			DataPoints: count,
		},
		Disk: ResourceStatItem{
			Min:        minDisk,
			Max:        maxDisk,
			Avg:        round2(float64(sumDisk) / float64(count)),
			DataPoints: count,
		},
	}
}

// getCachedAggregatedResourceUsage 获取缓存的聚合资源使用数据
func (sm *StatsManager) getCachedAggregatedResourceUsage(cacheKey string) *AggregatedResourceUsage {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	if time.Since(sm.lastCacheUpdate) > sm.cacheExpiration {
		return nil
	}

	return nil
}

// cacheAggregatedResourceUsage 缓存聚合资源使用数据
func (sm *StatsManager) cacheAggregatedResourceUsage(cacheKey string, data *AggregatedResourceUsage) {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()
	sm.lastCacheUpdate = time.Now()
}

// round2 保留2位小数
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// GetNetworkSpeed 获取当前网络速度
func (sm *StatsManager) GetNetworkSpeed() *NetworkSpeed {
	sm.networkMonitorMutex.RLock()
	defer sm.networkMonitorMutex.RUnlock()

	if sm.currentNetworkSpeed == nil {
		return &NetworkSpeed{}
	}

	speed := *sm.currentNetworkSpeed
	return &speed
}

// collectNetworkStats 采集网络流量统计
func (sm *StatsManager) collectNetworkStats() {
	currentStats, err := sm.readNetworkInterfaceStats()
	if err != nil {
		sm.logger.Debug("读取网络接口统计失败: %v", err)
		return
	}

	sm.networkMonitorMutex.Lock()
	defer sm.networkMonitorMutex.Unlock()

	if sm.lastNetworkStats != nil {
		timeDiff := currentStats.Timestamp.Sub(sm.lastNetworkStats.Timestamp).Seconds()
		if timeDiff > 0 {
			sm.currentNetworkSpeed = &NetworkSpeed{
				InboundBps:  uint64(float64(currentStats.BytesRecv-sm.lastNetworkStats.BytesRecv) / timeDiff),
				OutboundBps: uint64(float64(currentStats.BytesSent-sm.lastNetworkStats.BytesSent) / timeDiff),
				Timestamp:   currentStats.Timestamp,
			}
		}
	}

	sm.lastNetworkStats = currentStats
}

// readNetworkInterfaceStats 读取网络接口统计信息
func (sm *StatsManager) readNetworkInterfaceStats() (*NetworkInterfaceStats, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	var totalBytesRecv, totalBytesSent uint64
	var totalPacketsRecv, totalPacketsSent uint64

	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				continue
			}

			iface := strings.TrimSpace(parts[0])
			if strings.HasPrefix(iface, "lo") || strings.HasPrefix(iface, "docker") ||
				strings.HasPrefix(iface, "veth") || strings.HasPrefix(iface, "br-") {
				continue
			}

			fields := strings.Fields(strings.TrimSpace(parts[1]))
			if len(fields) < 16 {
				continue
			}

			bytesRecv, _ := strconv.ParseUint(fields[0], 10, 64)
			packetsRecv, _ := strconv.ParseUint(fields[1], 10, 64)
			bytesSent, _ := strconv.ParseUint(fields[8], 10, 64)
			packetsSent, _ := strconv.ParseUint(fields[9], 10, 64)

			totalBytesRecv += bytesRecv
			totalBytesSent += bytesSent
			totalPacketsRecv += packetsRecv
			totalPacketsSent += packetsSent
		}
	}

	return &NetworkInterfaceStats{
		BytesRecv:   totalBytesRecv,
		BytesSent:   totalBytesSent,
		PacketsRecv: totalPacketsRecv,
		PacketsSent: totalPacketsSent,
		Timestamp:   time.Now(),
	}, nil
}

// FormatNetworkSpeed 格式化网络速度显示
func FormatNetworkSpeed(bytesPerSecond uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytesPerSecond >= GB:
		return fmt.Sprintf("%.1f GB/s", float64(bytesPerSecond)/float64(GB))
	case bytesPerSecond >= MB:
		return fmt.Sprintf("%.1f MB/s", float64(bytesPerSecond)/float64(MB))
	case bytesPerSecond >= KB:
		return fmt.Sprintf("%.1f KB/s", float64(bytesPerSecond)/float64(KB))
	default:
		return fmt.Sprintf("%d B/s", bytesPerSecond)
	}
}

// UpdateNetworkHistory 更新网络历史数据
// 参数：
//   - inboundBps: 入站流量速率（字节/秒）
//   - outboundBps: 出站流量速率（字节/秒）
func (sm *StatsManager) UpdateNetworkHistory(inboundBps, outboundBps uint64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	sm.networkHistory = append(sm.networkHistory, NetworkDataPoint{
		Time:        now,
		InboundBps:  inboundBps,
		OutboundBps: outboundBps,
	})

	// 限制历史数据点数量
	if len(sm.networkHistory) > sm.maxHistoryPoints {
		sm.networkHistory = sm.networkHistory[len(sm.networkHistory)-sm.maxHistoryPoints:]
	}
}

// GetNetworkHistory 获取网络历史数据
// 参数：
//   - timeRange: 时间范围（1h, 6h, 24h, 7d）
//
// 返回：
//   - []NetworkDataPoint: 网络历史数据点数组
func (sm *StatsManager) GetNetworkHistory(timeRange string) []NetworkDataPoint {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var history []NetworkDataPoint
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

	for _, point := range sm.networkHistory {
		if point.Time.After(cutoff) {
			history = append(history, point)
		}
	}

	return history
}

// GetNetworkHistoryWithDB 获取网络历史数据（优先从内存，不足时从数据库补充）
// 参数：
//   - timeRange: 时间范围（1h, 6h, 24h, 7d）
//
// 返回：
//   - []NetworkDataPoint: 网络历史数据点数组
func (sm *StatsManager) GetNetworkHistoryWithDB(timeRange string) []NetworkDataPoint {
	sm.mutex.RLock()
	memoryData := make([]NetworkDataPoint, len(sm.networkHistory))
	copy(memoryData, sm.networkHistory)
	sm.mutex.RUnlock()

	var cutoff time.Time
	now := time.Now()

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

	var history []NetworkDataPoint
	for _, point := range memoryData {
		if point.Time.After(cutoff) {
			history = append(history, point)
		}
	}

	// 如果内存数据不足，从数据库补充
	if len(history) > 0 {
		oldestInMemory := history[0].Time
		if oldestInMemory.After(cutoff) {
			dbRecords, err := database.GetNetworkHistoryByTimeRange(cutoff, oldestInMemory)
			if err == nil && len(dbRecords) > 0 {
				prepend := make([]NetworkDataPoint, len(dbRecords))
				for i, r := range dbRecords {
					prepend[i] = NetworkDataPoint{Time: r.Timestamp, InboundBps: r.InboundBps, OutboundBps: r.OutboundBps}
				}
				history = append(prepend, history...)
			}
		}
	} else {
		dbRecords, err := database.GetNetworkHistoryByTimeRange(cutoff, now)
		if err == nil {
			history = make([]NetworkDataPoint, len(dbRecords))
			for i, r := range dbRecords {
				history[i] = NetworkDataPoint{Time: r.Timestamp, InboundBps: r.InboundBps, OutboundBps: r.OutboundBps}
			}
		}
	}

	return history
}

// AggregatedNetworkUsage 聚合后的网络使用数据
type AggregatedNetworkUsage struct {
	TimeLabels     []string          `json:"timeLabels"`
	InboundValues  []uint64          `json:"inboundValues"`
	OutboundValues []uint64          `json:"outboundValues"`
	Statistics     NetworkStatistics `json:"statistics"`
}

// NetworkStatistics 网络流量统计信息
type NetworkStatistics struct {
	Inbound  NetworkStatItem `json:"inbound"`
	Outbound NetworkStatItem `json:"outbound"`
}

// NetworkStatItem 网络流量统计项
type NetworkStatItem struct {
	Min        uint64  `json:"min"`
	Max        uint64  `json:"max"`
	Avg        float64 `json:"avg"`
	DataPoints int     `json:"dataPoints"`
}

// GetAggregatedNetworkUsage 获取聚合后的网络使用数据
// 参数：
//   - timeRange: 时间范围（1h, 6h, 24h, 7d）
//   - points: 数据点数量
//
// 返回：
//   - *AggregatedNetworkUsage: 聚合后的网络使用数据
func (sm *StatsManager) GetAggregatedNetworkUsage(timeRange string, points int) *AggregatedNetworkUsage {
	cacheKey := fmt.Sprintf("network_%s_%d", timeRange, points)
	if cached := sm.getCachedAggregatedNetworkUsage(cacheKey); cached != nil {
		return cached
	}

	history := sm.GetNetworkHistoryWithDB(timeRange)

	if len(history) == 0 {
		return &AggregatedNetworkUsage{
			TimeLabels:     []string{},
			InboundValues:  []uint64{},
			OutboundValues: []uint64{},
			Statistics:     NetworkStatistics{},
		}
	}

	result := sm.aggregateNetworkData(history, points, timeRange)

	sm.cacheAggregatedNetworkUsage(cacheKey, result)

	return result
}

// aggregateNetworkData 聚合网络流量数据
// 参数：
//   - data: 网络历史数据点数组
//   - points: 目标数据点数量
//   - timeRange: 时间范围
//
// 返回：
//   - *AggregatedNetworkUsage: 聚合后的网络使用数据
func (sm *StatsManager) aggregateNetworkData(data []NetworkDataPoint, points int, timeRange string) *AggregatedNetworkUsage {
	result := &AggregatedNetworkUsage{
		TimeLabels:     make([]string, 0),
		InboundValues:  make([]uint64, 0),
		OutboundValues: make([]uint64, 0),
	}

	if len(data) == 0 {
		return result
	}

	var timeFormat string
	switch timeRange {
	case "1h", "6h", "24h":
		timeFormat = "15:04"
	case "7d":
		timeFormat = "01-02 15:04"
	default:
		timeFormat = "15:04"
	}

	now := time.Now()
	var totalDuration time.Duration
	switch timeRange {
	case "1h":
		totalDuration = time.Hour
	case "6h":
		totalDuration = 6 * time.Hour
	case "24h":
		totalDuration = 24 * time.Hour
	case "7d":
		totalDuration = 7 * 24 * time.Hour
	default:
		totalDuration = time.Hour
	}

	if points <= 0 {
		points = 12
	}

	interval := totalDuration / time.Duration(points)
	for i := 0; i < points; i++ {
		slotStart := now.Add(-totalDuration + time.Duration(i)*interval)
		slotEnd := slotStart.Add(interval)

		var sumInbound, sumOutbound uint64
		var count int
		for _, point := range data {
			if point.Time.After(slotStart) && !point.Time.After(slotEnd) {
				sumInbound += point.InboundBps
				sumOutbound += point.OutboundBps
				count++
			}
		}

		var avgInbound, avgOutbound uint64
		if count > 0 {
			avgInbound = sumInbound / uint64(count)
			avgOutbound = sumOutbound / uint64(count)
		}

		result.TimeLabels = append(result.TimeLabels, slotStart.Format(timeFormat))
		result.InboundValues = append(result.InboundValues, avgInbound)
		result.OutboundValues = append(result.OutboundValues, avgOutbound)
	}

	stats := sm.calculateNetworkStatistics(data)
	result.Statistics = stats

	return result
}

// calculateNetworkStatistics 计算网络流量统计信息
// 参数：
//   - data: 网络历史数据点数组
//
// 返回：
//   - NetworkStatistics: 网络流量统计信息
func (sm *StatsManager) calculateNetworkStatistics(data []NetworkDataPoint) NetworkStatistics {
	if len(data) == 0 {
		return NetworkStatistics{}
	}

	minInbound := data[0].InboundBps
	maxInbound := data[0].InboundBps
	sumInbound := uint64(0)
	minOutbound := data[0].OutboundBps
	maxOutbound := data[0].OutboundBps
	sumOutbound := uint64(0)

	for _, point := range data {
		if point.InboundBps < minInbound {
			minInbound = point.InboundBps
		}
		if point.InboundBps > maxInbound {
			maxInbound = point.InboundBps
		}
		sumInbound += point.InboundBps

		if point.OutboundBps < minOutbound {
			minOutbound = point.OutboundBps
		}
		if point.OutboundBps > maxOutbound {
			maxOutbound = point.OutboundBps
		}
		sumOutbound += point.OutboundBps
	}

	count := len(data)
	return NetworkStatistics{
		Inbound: NetworkStatItem{
			Min:        minInbound,
			Max:        maxInbound,
			Avg:        round2(float64(sumInbound) / float64(count)),
			DataPoints: count,
		},
		Outbound: NetworkStatItem{
			Min:        minOutbound,
			Max:        maxOutbound,
			Avg:        round2(float64(sumOutbound) / float64(count)),
			DataPoints: count,
		},
	}
}

// getCachedAggregatedNetworkUsage 获取缓存的聚合网络使用数据
// 参数：
//   - cacheKey: 缓存键
//
// 返回：
//   - *AggregatedNetworkUsage: 缓存的聚合网络使用数据
func (sm *StatsManager) getCachedAggregatedNetworkUsage(cacheKey string) *AggregatedNetworkUsage {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	if time.Since(sm.lastCacheUpdate) > sm.cacheExpiration {
		return nil
	}

	return nil
}

// cacheAggregatedNetworkUsage 缓存聚合网络使用数据
// 参数：
//   - cacheKey: 缓存键
//   - data: 聚合网络使用数据
func (sm *StatsManager) cacheAggregatedNetworkUsage(cacheKey string, data *AggregatedNetworkUsage) {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()
	sm.lastCacheUpdate = time.Now()
}
