// core/sdns/server_stats.go

package sdns

import (
	"net"
	"strconv"
	"sync"
	"time"

	"SteadyDNS/core/database"
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

	// 时间衰减EWMA相关字段
	EWMAScore      float64   // EWMA健康评分 (0.0-1.0)
	EWMALatency    float64   // EWMA延迟评分
	EWMAHalfLife   float64   // EWMA半衰期（秒）
	EWMALastUpdate time.Time // EWMA上次更新时间

	// 滑动窗口相关字段
	RecentResults []bool // 最近N次查询结果，true=成功, false=失败
	WindowIndex   int    // 滑动窗口当前索引
	WindowSize    int    // 滑动窗口大小

	// 熔断状态相关字段
	CircuitBroken    bool // 是否处于熔断状态
	ProbeMode        bool // 是否处于主动探测模式
	ConsecutiveFails int  // 连续失败次数
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
		addr := net.JoinHostPort(server.Address, strconv.Itoa(server.Port))
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

// getOrCreateServerStats 获取或创建服务器统计信息
func (f *DNSForwarder) getOrCreateServerStats(addr string) *ServerStats {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()

	if stats, exists := f.serverStats[addr]; exists {
		return stats
	}

	now := time.Now()
	stats := &ServerStats{
		Address:         addr,
		Status:          "healthy",
		WindowStartTime: now,
		LastQueryTime:   now,
		// 初始化EWMA相关字段
		EWMAScore:      1.0,  // 初始评分为1.0（健康）
		EWMALatency:    0.0,  // 初始延迟为0
		EWMAHalfLife:   10.0, // 默认半衰期10秒
		EWMALastUpdate: now,
		// 初始化滑动窗口相关字段
		RecentResults: make([]bool, 0, 5),
		WindowIndex:   0,
		WindowSize:    5,
		// 初始化熔断状态相关字段
		CircuitBroken:    false,
		ProbeMode:        false,
		ConsecutiveFails: 0,
	}
	f.serverStats[addr] = stats
	return stats
}
