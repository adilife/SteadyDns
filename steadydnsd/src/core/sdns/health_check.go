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
// core/sdns/health_check.go
// 健康检查模块 - 实现多层次健康检查策略

package sdns

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// getServerGroupDomain 根据服务器地址查找所属的转发组域名
// 参数:
//   - addr: 服务器地址（格式为"host:port"）
//
// 返回:
//   - string: 转发组域名（Default或其他域名）
func (f *DNSForwarder) getServerGroupDomain(addr string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 遍历所有转发组，查找服务器所属的组
	for domain, group := range f.groups {
		for _, priorityServers := range group.PriorityQueues {
			for _, server := range priorityServers {
				serverAddr := net.JoinHostPort(server.Address, strconv.Itoa(server.Port))
				if serverAddr == addr {
					return domain
				}
			}
		}
	}

	// 如果未找到，返回Default
	return "Default"
}

// CheckServerHealth 检查服务器健康状态
// 发送探测请求并更新EWMA评分
// 使用ExchangeWithCookie方法，支持Cookie、TCP管道化和动态协议升级
//
// 参数:
//   - addr: 服务器地址（格式为"host:port"）
//   - groupDomain: 所属转发组的域名（Default或其他域名）
//
// 返回:
//   - bool: 服务器是否健康（返回码为NOERROR）
func (f *DNSForwarder) CheckServerHealth(addr string, groupDomain string) bool {
	// 根据转发组域名确定健康检查查询目标
	// Default组查询根域名的SOA记录，其他组查询该组域名的SOA记录
	checkDomain := groupDomain
	if groupDomain == "Default" {
		checkDomain = "."
	} else {
		// 确保域名以点结尾（fully qualified domain name）
		if !strings.HasSuffix(checkDomain, ".") {
			checkDomain = checkDomain + "."
		}
	}

	// 构造健康检查查询消息（查询SOA记录）
	query := new(dns.Msg)
	query.SetQuestion(checkDomain, dns.TypeSOA)
	query.RecursionDesired = true

	// 执行查询，使用ExchangeWithCookie支持Cookie和动态协议选择
	result, err := f.ExchangeWithCookie(addr, query)

	// 获取或创建统计信息
	stats := f.getOrCreateServerStats(addr)
	now := time.Now()

	if err != nil {
		f.logger.Debug("健康检查 - 服务器 %s 失败: %v", addr, err)
		// 更新EWMA评分为失败（rcode=-1表示网络错误）
		UpdateTimeDecayEWMAForHealthCheck(stats, -1, now)
		UpdateSlidingWindow(stats, false)
		RecordQueryResult(stats, false)
		return false
	}

	if result == nil {
		f.logger.Debug("健康检查 - 服务器 %s 失败，返回空结果", addr)
		// 更新EWMA评分为失败（rcode=-1表示网络错误）
		UpdateTimeDecayEWMAForHealthCheck(stats, -1, now)
		UpdateSlidingWindow(stats, false)
		RecordQueryResult(stats, false)
		return false
	}

	// 服务器有返回，根据响应码更新评分
	f.logger.Debug("健康检查 - 服务器 %s 成功，返回码: %d", addr, result.Rcode)

	// 判断服务器是否健康
	// 对于权威服务器，返回REFUSED(5)是正常的（表示不处理该域名）
	// 返回NXDOMAIN(3)也是正常的（表示域名不存在）
	// 只有SERVFAIL(2)或网络错误才认为不健康
	isHealthy := result.Rcode != dns.RcodeServerFailure

	// 更新EWMA评分（使用健康检查专用的宽松评分策略）
	UpdateTimeDecayEWMAForHealthCheck(stats, result.Rcode, now)
	UpdateSlidingWindow(stats, isHealthy)
	RecordQueryResult(stats, isHealthy)

	return isHealthy
}

// IsServerHealthy 检查服务器是否健康（兼容旧接口）
// 现在使用新的IsServerAvailable函数
//
// 参数:
//   - addr: 服务器地址
//
// 返回:
//   - bool: 服务器是否健康
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
			// 获取服务器所属的转发组域名
			groupDomain := f.getServerGroupDomain(addr)
			success := f.CheckServerHealth(addr, groupDomain)

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
			// 获取服务器所属的转发组域名
			groupDomain := f.getServerGroupDomain(addr)
			f.CheckServerHealth(addr, groupDomain)
		}(stats)
	}
}

// runHealthChecks 执行健康检查
// 实现多层次健康检查策略：
// 1. 启动时全量检测（初始化EWMA评分）
// 2. 定时检查僵尸服务器（超过DefaultStaleThreshold无查询）
// 3. 熔断服务器高频探测（由单独协程处理）
//
// 参数:
//   - isStartup: 是否为启动时的全量检测
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
	// 从当前活跃的groups中获取服务器地址和所属组域名
	f.mu.RLock()
	type serverInfo struct {
		addr        string
		groupDomain string
	}
	var servers []serverInfo
	for domain, group := range f.groups {
		for _, priorityServers := range group.PriorityQueues {
			for _, server := range priorityServers {
				addr := net.JoinHostPort(server.Address, strconv.Itoa(server.Port))
				servers = append(servers, serverInfo{addr: addr, groupDomain: domain})
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
	for _, info := range servers {
		go func(address, groupDomain string) {
			f.CheckServerHealth(address, groupDomain)
		}(info.addr, info.groupDomain)
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
			// 获取服务器所属的转发组域名
			groupDomain := f.getServerGroupDomain(addr)
			success := f.CheckServerHealth(addr, groupDomain)
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
