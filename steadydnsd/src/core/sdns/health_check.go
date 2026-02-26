// core/sdns/health_check.go
// 健康检查模块 - 实现多层次健康检查策略

package sdns

import (
	"net"
	"strconv"
	"time"

	"github.com/miekg/dns"
)

// CheckServerHealth 检查服务器健康状态
// 发送探测请求并更新EWMA评分
func (f *DNSForwarder) CheckServerHealth(addr string) bool {
	// 创建一个简单的DNS查询
	query := new(dns.Msg)
	query.SetQuestion("example.com.", dns.TypeA)
	query.RecursionDesired = true

	// 设置超时
	c := new(dns.Client)
	c.Timeout = 2 * time.Second

	// 执行查询
	start := time.Now()
	result, _, err := c.Exchange(query, addr)
	duration := time.Since(start)

	// 获取或创建统计信息
	stats := f.getOrCreateServerStats(addr)
	now := time.Now()

	if err != nil {
		f.logger.Debug("健康检查 - 服务器 %s 失败: %v", addr, err)
		// 更新EWMA评分为失败（rcode=-1表示网络错误）
		UpdateTimeDecayEWMA(stats, -1, -1, now)
		UpdateSlidingWindow(stats, false)
		RecordQueryResult(stats, false)
		return false
	}

	if result == nil {
		f.logger.Debug("健康检查 - 服务器 %s 失败，返回空结果", addr)
		// 更新EWMA评分为失败（rcode=-1表示网络错误）
		UpdateTimeDecayEWMA(stats, -1, -1, now)
		UpdateSlidingWindow(stats, false)
		RecordQueryResult(stats, false)
		return false
	}

	// 服务器有返回，根据响应码更新评分
	f.logger.Debug("健康检查 - 服务器 %s 成功，返回码: %d", addr, result.Rcode)

	// 更新EWMA评分（使用实际响应码）
	latency := float64(duration.Milliseconds())
	UpdateTimeDecayEWMA(stats, result.Rcode, latency, now)
	UpdateSlidingWindow(stats, result.Rcode == 0)
	RecordQueryResult(stats, result.Rcode == 0)

	return result.Rcode == 0
}

// IsServerHealthy 检查服务器是否健康（兼容旧接口）
// 现在使用新的IsServerAvailable函数
func (f *DNSForwarder) IsServerHealthy(addr string) bool {
	stats := f.GetServerStats(addr)
	if stats == nil {
		return false
	}
	return IsServerAvailable(stats)
}

// probeCircuitBrokenServers 探测熔断状态的服务器
// 每DefaultProbeInterval秒探测一次
func (f *DNSForwarder) probeCircuitBrokenServers() {
	brokenServers := f.GetCircuitBrokenServers()
	if len(brokenServers) == 0 {
		return
	}

	f.logger.Debug("主动探测 - 发现 %d 个熔断服务器，开始探测", len(brokenServers))

	for _, stats := range brokenServers {
		go func(s *ServerStats) {
			addr := s.Address
			success := f.CheckServerHealth(addr)

			if success {
				f.logger.Info("主动探测 - 服务器 %s 恢复成功，重置熔断状态", addr)
				ResetCircuitBreaker(s)
			}
		}(stats)
	}
}

// checkStaleServers 检查长时间无查询的服务器（僵尸服务器）
// 对这些服务器进行探测，更新EWMA评分
func (f *DNSForwarder) checkStaleServers() {
	staleServers := f.GetStaleServers(DefaultStaleThreshold)
	if len(staleServers) == 0 {
		return
	}

	f.logger.Debug("定时检查 - 发现 %d 个僵尸服务器（超过%v无查询），开始探测",
		len(staleServers), DefaultStaleThreshold)

	for _, stats := range staleServers {
		go func(s *ServerStats) {
			addr := s.Address
			f.CheckServerHealth(addr)
		}(stats)
	}
}

// runHealthChecks 执行健康检查
// 实现多层次健康检查策略：
// 1. 启动时全量检测（初始化EWMA评分）
// 2. 定时检查僵尸服务器（超过DefaultStaleThreshold无查询）
// 3. 熔断服务器高频探测（由单独协程处理）
func (f *DNSForwarder) runHealthChecks(isStartup bool) {
	if isStartup {
		// 启动时全量检测
		f.logger.Debug("健康检查 - 启动时全量检测，初始化所有服务器状态")
		f.runFullHealthCheck()
	} else {
		// 定时检查僵尸服务器
		f.checkStaleServers()
	}
}

// runFullHealthCheck 对所有服务器进行全量健康检查
// 用于服务启动时初始化服务器状态
func (f *DNSForwarder) runFullHealthCheck() {
	// 从当前活跃的groups中获取服务器地址
	f.mu.RLock()
	var servers []string
	for _, group := range f.groups {
		for _, priorityServers := range group.PriorityQueues {
			for _, server := range priorityServers {
				addr := net.JoinHostPort(server.Address, strconv.Itoa(server.Port))
				servers = append(servers, addr)
			}
		}
	}
	f.mu.RUnlock()

	if len(servers) == 0 {
		f.logger.Debug("健康检查 - 没有配置转发服务器")
		return
	}

	f.logger.Debug("健康检查 - 全量检测 %d 个服务器", len(servers))

	// 对每个服务器进行健康检查
	for _, addr := range servers {
		go func(address string) {
			f.CheckServerHealth(address)
		}(addr)
	}
}

// recoverMediumScoreServers 中评分服务器评分回升机制
// 每MediumScoreRecoveryInterval秒执行一次，帮助中评分服务器回升到高评分
func (f *DNSForwarder) recoverMediumScoreServers() {
	mediumServers := f.GetMediumScoreServers()
	if len(mediumServers) == 0 {
		return
	}

	f.logger.Debug("评分回升 - 发现 %d 个中评分服务器，开始回升计算", len(mediumServers))

	now := time.Now()
	for _, stats := range mediumServers {
		// 向目标评分0.9衰减（回升）
		DecayEWMAToTarget(stats, MediumScoreRecoveryTarget, MediumScoreRecoveryHalfLife, now)
		f.logger.Debug("评分回升 - 服务器 %s 评分回升到 %.3f", stats.Address, stats.EWMAScore)
	}
}

// probeLowScoreServers 低评分服务器主动探测
// 每LowScoreProbeInterval秒探测一次，帮助低评分服务器有机会回升
func (f *DNSForwarder) probeLowScoreServers() {
	lowServers := f.GetLowScoreServers()
	if len(lowServers) == 0 {
		return
	}

	f.logger.Debug("主动探测 - 发现 %d 个低评分服务器，开始探测", len(lowServers))

	for _, stats := range lowServers {
		go func(s *ServerStats) {
			addr := s.Address
			success := f.CheckServerHealth(addr)
			if success {
				f.logger.Debug("主动探测 - 低评分服务器 %s 探测成功，评分已更新", addr)
			}
		}(stats)
	}
}

// StartHealthChecks 启动健康检查协程
// 实现多层次健康检查：
// 1. 启动时立即全量检测
// 2. 熔断服务器高频探测（每DefaultProbeInterval秒）
// 3. 中评分服务器评分回升（每MediumScoreRecoveryInterval秒）
// 4. 低评分服务器主动探测（每LowScoreProbeInterval秒）
// 5. 僵尸服务器定时检查（每DefaultPeriodicInterval秒）
func (f *DNSForwarder) StartHealthChecks() {
	// 启动熔断服务器探测协程
	go func() {
		f.logger.Debug("启动熔断服务器探测协程，探测间隔: %v", DefaultProbeInterval)
		probeTicker := time.NewTicker(DefaultProbeInterval)
		defer probeTicker.Stop()

		for range probeTicker.C {
			f.probeCircuitBrokenServers()
		}
	}()

	// 启动中评分服务器评分回升协程
	go func() {
		f.logger.Debug("启动中评分服务器评分回升协程，回升间隔: %v", MediumScoreRecoveryInterval*time.Second)
		recoveryTicker := time.NewTicker(MediumScoreRecoveryInterval * time.Second)
		defer recoveryTicker.Stop()

		for range recoveryTicker.C {
			f.recoverMediumScoreServers()
		}
	}()

	// 启动低评分服务器主动探测协程
	go func() {
		f.logger.Debug("启动低评分服务器主动探测协程，探测间隔: %v", LowScoreProbeInterval)
		lowScoreTicker := time.NewTicker(LowScoreProbeInterval)
		defer lowScoreTicker.Stop()

		for range lowScoreTicker.C {
			f.probeLowScoreServers()
		}
	}()

	// 启动定时健康检查协程
	go func() {
		// 启动时立即执行一次全量检测
		f.logger.Debug("启动健康检查协程，立即执行首次全量检测")
		f.runHealthChecks(true)

		// 然后启动定时健康检查（只检查僵尸服务器）
		ticker := time.NewTicker(DefaultPeriodicInterval)
		defer ticker.Stop()

		for range ticker.C {
			f.runHealthChecks(false)
		}
	}()
}
