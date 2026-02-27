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

// core/sdns/advanced_health.go
// 高级健康检查模块 - 实现时间衰减EWMA、滑动窗口熔断和主动探测

package sdns

import (
	"math"
	"time"
)

// 健康检查常量定义
const (
	// EWMA参数
	DefaultEWMHalfLife = 10 * time.Second // 半衰期10秒

	// 健康检查专用EWMA参数
	HealthCheckEWMHalfLife = 5 * time.Second // 健康检查半衰期5秒，加快评分恢复

	// 熔断参数
	DefaultFailureThreshold = 5 // 连续5次失败触发熔断
	DefaultWindowSize       = 5 // 滑动窗口大小5

	// 探测间隔
	DefaultProbeInterval    = 1 * time.Second  // 熔断服务器探测间隔1秒
	DefaultStaleThreshold   = 60 * time.Second // 僵尸服务器阈值60秒
	DefaultPeriodicInterval = 30 * time.Second // 定时检查间隔30秒

	// EWMA评分阈值
	DefaultEWMAHealthyThreshold = 0.5 // EWMA评分高于此值视为健康

	// DNS响应码评分
	ScoreNoError      = 1.0 // NOERROR/NXDOMAIN等成功响应
	ScoreServerError  = 0.3 // SERVFAIL/REFUSED/NOTIMP等服务器错误
	ScoreNetworkError = 0.0 // 网络错误（超时、连接失败等）

	// 分层延迟阈值和延迟时间
	HighScoreThreshold   = 0.8 // 高评分阈值
	MediumScoreThreshold = 0.6 // 中评分阈值
	HighScoreDelay       = 0   // 高评分延迟（ms）
	MediumScoreDelay     = 5   // 中评分延迟（ms）
	LowScoreDelay        = 15  // 低评分延迟（ms）

	// 中评分服务器评分回升机制
	MediumScoreRecoveryTarget   = 0.9 // 回升目标评分
	MediumScoreRecoveryHalfLife = 60  // 回升半衰期（秒）
	MediumScoreRecoveryInterval = 10  // 回升计算间隔（秒）

	// 低评分服务器探测
	LowScoreProbeInterval = 10 * time.Second // 低评分服务器探测间隔
)

// calculateScoreByRcode 根据DNS响应码计算EWMA评分值
// 参数：
//   - rcode: DNS响应码
//
// 返回：
//   - EWMA评分值（0.0-1.0）
func calculateScoreByRcode(rcode int) float64 {
	switch rcode {
	case 0: // NOERROR
		return ScoreNoError
	case 3: // NXDOMAIN
		return ScoreNoError // 服务器正常，只是无此记录
	case 1: // FORMERR
		return ScoreNoError // 格式错误，但服务器响应了
	case 2: // SERVFAIL
		return ScoreServerError // 服务器故障
	case 4: // NOTIMP
		return ScoreServerError // 未实现
	case 5: // REFUSED
		return ScoreServerError // 拒绝查询
	default:
		return ScoreServerError // 其他错误视为服务器错误
	}
}

// calculateHealthCheckScore 根据DNS响应码计算健康检查EWMA评分值
// 使用宽松的评分策略，只有SERVFAIL和网络错误才降低评分
// 参数：
//   - rcode: DNS响应码（-1表示网络错误）
//
// 返回：
//   - EWMA评分值（0.0-1.0）
func calculateHealthCheckScore(rcode int) float64 {
	if rcode < 0 {
		// 网络错误
		return ScoreNetworkError
	}

	switch rcode {
	case 0: // NOERROR
		return ScoreNoError
	case 3: // NXDOMAIN
		return ScoreNoError // 服务器正常，只是无此记录
	case 1: // FORMERR
		return ScoreNoError // 格式错误，但服务器响应了
	case 2: // SERVFAIL
		return ScoreServerError // 服务器故障，降低评分
	case 4: // NOTIMP
		return ScoreNoError // 未实现，但服务器响应了（权威服务器常见）
	case 5: // REFUSED
		return ScoreNoError // 拒绝查询，但服务器响应了（权威服务器常见）
	default:
		return ScoreNoError // 其他响应，只要服务器响应了就视为正常
	}
}

// UpdateTimeDecayEWMA 更新时间衰减EWMA评分
// 参数：
//   - stats: 服务器统计信息
//   - rcode: DNS响应码（-1表示网络错误）
//   - latency: 查询延迟（毫秒）
//   - now: 当前时间
//   - halfLife: 半衰期（秒），如果为0则使用默认值10秒
func UpdateTimeDecayEWMA(stats *ServerStats, rcode int, latency float64, now time.Time, halfLife float64) {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	// 计算距离上次更新的时间差（秒）
	dt := now.Sub(stats.EWMALastUpdate).Seconds()
	if dt < 0 {
		dt = 0
	}

	// 计算动态Alpha值
	// Alpha = 1 - exp(-ln(2) * dt / halfLife)
	if halfLife <= 0 {
		halfLife = 10.0 // 使用默认值
	}
	alpha := 1 - math.Exp(-math.Ln2*dt/halfLife)

	// 根据响应码计算评分值
	var value float64
	if rcode < 0 {
		// 网络错误
		value = ScoreNetworkError
	} else {
		value = calculateScoreByRcode(rcode)
	}

	// 更新EWMA评分
	stats.EWMAScore = alpha*value + (1-alpha)*stats.EWMAScore

	// 更新EWMA延迟（只对成功的查询，rcode=0表示NOERROR）
	if rcode == 0 && latency >= 0 {
		if stats.EWMALatency == 0 {
			stats.EWMALatency = latency
		} else {
			stats.EWMALatency = alpha*latency + (1-alpha)*stats.EWMALatency
		}
	}

	// 更新时间戳
	stats.EWMALastUpdate = now
}

// UpdateTimeDecayEWMAForHealthCheck 健康检查专用的EWMA评分更新
// 使用宽松的评分策略和较短的半衰期，加快评分恢复
// 参数：
//   - stats: 服务器统计信息
//   - rcode: DNS响应码（-1表示网络错误）
//   - now: 当前时间
func UpdateTimeDecayEWMAForHealthCheck(stats *ServerStats, rcode int, now time.Time) {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	// 计算距离上次更新的时间差（秒）
	dt := now.Sub(stats.EWMALastUpdate).Seconds()
	if dt < 0 {
		dt = 0
	}

	// 健康检查使用较短的半衰期，加快评分恢复
	halfLife := HealthCheckEWMHalfLife.Seconds()
	alpha := 1 - math.Exp(-math.Ln2*dt/halfLife)

	// 使用宽松的评分策略
	value := calculateHealthCheckScore(rcode)

	// 更新EWMA评分
	stats.EWMAScore = alpha*value + (1-alpha)*stats.EWMAScore

	// 更新时间戳
	stats.EWMALastUpdate = now
}

// DecayEWMAToTarget EWMA评分向目标值衰减
// 用于中评分服务器的评分回升机制
// 参数：
//   - stats: 服务器统计信息
//   - target: 目标评分值
//   - halfLife: 半衰期（秒）
//   - now: 当前时间
func DecayEWMAToTarget(stats *ServerStats, target float64, halfLife float64, now time.Time) {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	dt := now.Sub(stats.EWMALastUpdate).Seconds()
	if dt <= 0 {
		return
	}

	if halfLife <= 0 {
		halfLife = 60.0 // 使用默认值60秒
	}
	alpha := 1 - math.Exp(-math.Ln2*dt/halfLife)

	// 向目标值衰减
	stats.EWMAScore = (1-alpha)*stats.EWMAScore + alpha*target
	stats.EWMALastUpdate = now
}

// calculateTieredDelay 根据EWMA评分计算分层延迟
// 参数：
//   - score: EWMA评分
//
// 返回：
//   - 延迟时间（毫秒）
func calculateTieredDelay(score float64) time.Duration {
	switch {
	case score >= HighScoreThreshold:
		// 高评分：立即启动
		return HighScoreDelay * time.Millisecond
	case score >= MediumScoreThreshold:
		// 中评分：轻微延迟
		return MediumScoreDelay * time.Millisecond
	default:
		// 低评分：明显延迟
		return LowScoreDelay * time.Millisecond
	}
}

// UpdateSlidingWindow 更新滑动窗口
// 返回当前窗口内的失败次数
func UpdateSlidingWindow(stats *ServerStats, success bool) int {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	// 添加新结果到窗口
	stats.RecentResults = append(stats.RecentResults, success)

	// 如果窗口已满，移除最旧的结果
	if len(stats.RecentResults) > stats.WindowSize {
		stats.RecentResults = stats.RecentResults[1:]
	}

	// 计算失败次数
	failCount := 0
	for _, result := range stats.RecentResults {
		if !result {
			failCount++
		}
	}

	return failCount
}

// CheckCircuitBreaker 检查是否应该触发熔断
// 返回：是否触发熔断
func CheckCircuitBreaker(stats *ServerStats) bool {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	// 如果已经在熔断状态，不重复触发
	if stats.CircuitBroken {
		return false
	}

	// 检查连续失败次数
	if stats.ConsecutiveFails >= DefaultFailureThreshold {
		stats.CircuitBroken = true
		stats.ProbeMode = true
		return true
	}

	return false
}

// RecordQueryResult 记录查询结果，更新连续失败计数
func RecordQueryResult(stats *ServerStats, success bool) {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	if success {
		stats.ConsecutiveFails = 0
	} else {
		stats.ConsecutiveFails++
	}
}

// ResetCircuitBreaker 重置熔断状态
func ResetCircuitBreaker(stats *ServerStats) {
	stats.Mu.Lock()
	defer stats.Mu.Unlock()

	stats.CircuitBroken = false
	stats.ProbeMode = false
	stats.ConsecutiveFails = 0
	// 重置EWMA评分为中性值，给服务器一个重新开始的机会
	stats.EWMAScore = 0.5
}

// IsServerAvailable 检查服务器是否可用（未被熔断且EWMA评分健康）
func IsServerAvailable(stats *ServerStats) bool {
	stats.Mu.RLock()
	defer stats.Mu.RUnlock()

	// 如果处于熔断状态，不可用
	if stats.CircuitBroken {
		return false
	}

	// 检查EWMA评分
	return stats.EWMAScore >= DefaultEWMAHealthyThreshold
}

// GetStaleServers 获取长时间无查询的服务器（僵尸服务器）
func (f *DNSForwarder) GetStaleServers(threshold time.Duration) []*ServerStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	now := time.Now()
	var staleServers []*ServerStats

	for _, stats := range f.serverStats {
		stats.Mu.RLock()
		timeSinceLastQuery := now.Sub(stats.LastQueryTime)
		stats.Mu.RUnlock()

		if timeSinceLastQuery >= threshold {
			staleServers = append(staleServers, stats)
		}
	}

	return staleServers
}

// GetCircuitBrokenServers 获取处于熔断状态的服务器
func (f *DNSForwarder) GetCircuitBrokenServers() []*ServerStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	var brokenServers []*ServerStats

	for _, stats := range f.serverStats {
		stats.Mu.RLock()
		isBroken := stats.CircuitBroken
		stats.Mu.RUnlock()

		if isBroken {
			brokenServers = append(brokenServers, stats)
		}
	}

	return brokenServers
}

// GetMediumScoreServers 获取中评分服务器（评分在0.6-0.8之间）
func (f *DNSForwarder) GetMediumScoreServers() []*ServerStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	var mediumServers []*ServerStats

	for _, stats := range f.serverStats {
		stats.Mu.RLock()
		score := stats.EWMAScore
		stats.Mu.RUnlock()

		if score >= MediumScoreThreshold && score < HighScoreThreshold {
			mediumServers = append(mediumServers, stats)
		}
	}

	return mediumServers
}

// GetLowScoreServers 获取低评分服务器（评分低于0.6）
func (f *DNSForwarder) GetLowScoreServers() []*ServerStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	var lowServers []*ServerStats

	for _, stats := range f.serverStats {
		stats.Mu.RLock()
		score := stats.EWMAScore
		isBroken := stats.CircuitBroken
		stats.Mu.RUnlock()

		// 低评分且未熔断的服务器
		if score < MediumScoreThreshold && !isBroken {
			lowServers = append(lowServers, stats)
		}
	}

	return lowServers
}

// GetHealthyServersByPriority 获取指定优先级组中健康的服务器
func (f *DNSForwarder) GetHealthyServersByPriority(group *ForwardGroup, priority int) []*ServerStats {
	servers, exists := group.PriorityQueues[priority]
	if !exists || len(servers) == 0 {
		return nil
	}

	var healthyServers []*ServerStats

	for _, server := range servers {
		addr := server.GetAddress()
		stats := f.GetServerStats(addr)
		if stats == nil {
			// 如果没有统计信息，创建一个新的
			stats = f.getOrCreateServerStats(addr)
		}

		// 检查服务器是否可用
		if IsServerAvailable(stats) {
			healthyServers = append(healthyServers, stats)
		}
	}

	return healthyServers
}
