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
// core/sdns/dns_forward.go
// DNS转发模块 - 实现支持Cookie、TCP管道化和动态协议升级的DNS查询

package sdns

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/miekg/dns"
)

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

// ForwardQuery 转发DNS查询
// 如果BIND插件禁用，跳过权威域转发逻辑
func (f *DNSForwarder) ForwardQuery(query *dns.Msg) (*dns.Msg, error) {
	startTime := time.Now()

	var queryDomain, queryType string
	if len(query.Question) > 0 {
		queryDomain = query.Question[0].Name
		queryType = dns.TypeToString[query.Question[0].Qtype]
	}

	// 检查BIND插件是否启用，只有启用时才进行权威域匹配
	if f.authorityForwarder.IsBindPluginEnabled() {
		// 检查是否匹配权威域
		isAuthority, authorityZone := f.authorityForwarder.MatchAuthorityZone(queryDomain)
		if isAuthority {
			// 匹配权威域，转发至BIND服务器
			bindAddr := f.authorityForwarder.GetBindAddress()
			f.logger.Debug("转发查询 - 匹配权威域: %s, 转发至BIND服务器: %s", authorityZone, bindAddr)
			result, err := f.forwardToServer(bindAddr, query, nil)
			if err == nil && result != nil {
				return result, nil
			}
			// 权威域查询失败，直接返回错误，不再尝试其他服务器
			f.logger.Error("权威域查询失败: %v", err)
			return nil, fmt.Errorf("权威域查询失败: %v", err)
		}
	}

	// 非权威域查询或BIND插件禁用，使用最长匹配算法选择合适的转发组
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

// tryForwardWithPriority 尝试按优先级转发查询
func (f *DNSForwarder) tryForwardWithPriority(group *ForwardGroup, query *dns.Msg) (*dns.Msg, error) {
	// 整体查询超时时间（5秒）
	overallTimeout := 5 * time.Second
	// 优先级队列启动间隔（从配置读取）
	priorityInterval := f.priorityTimeout

	// 创建通道来接收查询结果（容量设置为服务器总数的估计值）
	resultChan := make(chan *dns.Msg, 10)
	errorChan := make(chan error, 10)
	// 创建备用结果队列，用于存放被启动队列处理逻辑消费的非NOERROR结果
	spareResultChan := make(chan *dns.Msg, 10)
	// 创建整体取消通道
	cancelChan := make(chan struct{})

	// 统计所有健康服务器数量
	totalHealthyServers := 0
	// 收集所有健康服务器
	var allHealthyServers []*DNSServer

	// 按优先级顺序（1 -> 2 -> 3）启动查询
	for priority := 1; priority <= 3; priority++ {
		// 使用新的健康检查机制获取健康服务器
		healthyStatsList := f.GetHealthyServersByPriority(group, priority)
		if len(healthyStatsList) == 0 {
			f.logger.Debug("转发查询 - 优先级队列 %d 中没有健康服务器，跳过", priority)
			continue
		}

		f.logger.Debug("转发查询 - 启动优先级队列 %d, 健康服务器数量: %d", priority, len(healthyStatsList))

		// 按EWMA评分排序（高到低）
		sort.Slice(healthyStatsList, func(i, j int) bool {
			return healthyStatsList[i].EWMAScore > healthyStatsList[j].EWMAScore
		})

		// 为每台服务器创建任务，根据评分分层延迟启动
		// 同一优先级内的服务器按评分延迟启动
		// 不同优先级之间由priorityInterval控制
		for _, stats := range healthyStatsList {
			addr := stats.Address
			score := stats.EWMAScore
			// 计算评分延迟（同一优先级内的相对延迟）
			scoreDelay := calculateTieredDelay(score)
			// 计算优先级延迟（不同优先级之间的基础延迟）
			priorityDelay := time.Duration(priority-1) * priorityInterval
			// 总延迟 = 优先级延迟 + 评分延迟
			totalDelay := priorityDelay + scoreDelay

			f.logger.Debug("转发查询 - 服务器 %s (评分: %.3f, 优先级: %d, 总延迟: %v)", addr, score, priority, totalDelay)

			// 创建转发任务
			task := &DNSForwardTask{
				address:    addr,
				query:      query,
				resultChan: resultChan,
				errorChan:  errorChan,
				forwarder:  f,
				cancelChan: cancelChan,
			}

			// 根据总延迟启动
			go func(t *DNSForwardTask, d time.Duration) {
				if d > 0 {
					time.Sleep(d)
				}
				f.forwardPool.SubmitTask(t)
			}(task, totalDelay)

			totalHealthyServers++
			allHealthyServers = append(allHealthyServers, &DNSServer{Address: addr})
		}

		// 如果不是最后一个优先级队列，等待指定间隔后再启动下一个队列
		if priority < 3 {
			f.logger.Debug("转发查询 - 等待 %v 后启动下一优先级队列", priorityInterval)
			// 只等待优先级间隔，但仍然检查是否有NOERROR响应
			select {
			case result := <-resultChan:
				// 检查是否是NOERROR响应
				if result.Rcode == dns.RcodeSuccess {
					// NOERROR响应，直接返回
					f.logger.Debug("转发查询 - 已收到NOERROR结果，停止启动更多优先级队列")
					close(cancelChan)
					return result, nil
				} else {
					// 非NOERROR响应，存入备用结果队列，然后继续等待
					f.logger.Debug("转发查询 - 收到非NOERROR结果，存入备用队列并继续启动下一优先级队列")
					spareResultChan <- result
					// 继续等待优先级间隔，然后启动下一队列
					<-time.After(priorityInterval)
				}
			case <-time.After(priorityInterval):
				// 等待完成，继续启动下一优先级队列
			}
		}
	}

	// 如果没有健康服务器，返回错误
	if totalHealthyServers == 0 {
		close(cancelChan)
		close(spareResultChan)
		return nil, fmt.Errorf("没有健康的转发服务器")
	}

	// 等待整体超时或结果
	processedCount := 0
	var lastError error
	var firstNonNoErrorResult *dns.Msg

	// 设置超时定时器
	timeoutTimer := time.NewTimer(overallTimeout)
	defer timeoutTimer.Stop()

	for processedCount < totalHealthyServers {
		select {
		case result := <-resultChan:
			// 处理正式结果队列中的结果
			if result.Rcode == dns.RcodeSuccess {
				f.logger.Debug("转发查询 - 成功收到NOERROR结果")
				close(cancelChan)
				return result, nil
			} else {
				// 非NOERROR响应，记录但继续等待
				f.logger.Debug("转发查询 - 收到非NOERROR结果，继续等待其他服务器")
				if firstNonNoErrorResult == nil {
					firstNonNoErrorResult = result
				}
				processedCount++
			}
		case result := <-spareResultChan:
			// 处理备用结果队列中的结果
			f.logger.Debug("转发查询 - 收到备用队列中的非NOERROR结果")
			if firstNonNoErrorResult == nil {
				firstNonNoErrorResult = result
			}
			processedCount++
		case err := <-errorChan:
			// 收到错误，记录并继续等待
			processedCount++
			lastError = err
			f.logger.Debug("转发查询 - 收到服务器错误，已处理服务器数: %d/%d, 错误: %v",
				processedCount, totalHealthyServers, err)
		case <-timeoutTimer.C:
			// 整体超时
			f.logger.Debug("转发查询 - 整体超时")
			select {
			case result := <-resultChan:
				f.logger.Debug("转发查询 - 超时后收到结果")
				close(cancelChan)
				return result, nil
			case result := <-spareResultChan:
				f.logger.Debug("转发查询 - 超时后收到备用队列结果")
				close(cancelChan)
				return result, nil
			default:
				close(cancelChan)
				return nil, fmt.Errorf("整体查询超时")
			}
		}
	}

	// 所有服务器都已处理，立即结束查询
	f.logger.Debug("转发查询 - 所有服务器都已处理完毕，立即结束查询")
	close(cancelChan)
	close(spareResultChan)

	// 如果有非NOERROR响应，返回第一个收到的
	if firstNonNoErrorResult != nil {
		f.logger.Debug("转发查询 - 所有服务器都返回非NOERROR响应，返回第一个收到的响应")
		return firstNonNoErrorResult, nil
	}

	// 所有服务器都返回错误
	if lastError != nil {
		return nil, fmt.Errorf("所有转发服务器都返回错误: %v", lastError)
	}

	return nil, fmt.Errorf("所有转发服务器都不可用")
}

// forwardToServer 向单个DNS服务器转发查询
// 使用ExchangeWithCookie替代直接Exchange，支持Cookie、TCP管道化和动态协议升级
func (f *DNSForwarder) forwardToServer(addr string, query *dns.Msg, cancelChan chan struct{}) (*dns.Msg, error) {
	startTime := time.Now()

	// 首先检查是否已被取消
	if cancelChan != nil {
		select {
		case <-cancelChan:
			f.logger.Debug("转发查询 - 查询已被取消，跳过服务器: %s", addr)
			return nil, fmt.Errorf("查询被取消")
		default:
		}
	}

	// 获取或创建服务器统计信息
	stats := f.getOrCreateServerStats(addr)

	// 使用ExchangeWithCookie进行查询，支持Cookie、TCP管道化和动态协议升级
	result, err := f.ExchangeWithCookie(addr, query)

	// 再次检查是否被取消（查询完成后）
	if cancelChan != nil {
		select {
		case <-cancelChan:
			f.logger.Debug("转发查询 - 查询被取消，服务器: %s", addr)
			return nil, fmt.Errorf("查询被取消")
		default:
		}
	}

	if err != nil {
		duration := time.Since(startTime)
		f.logger.Debug("转发查询 - 服务器 %s 响应失败, 耗时: %v, 错误: %v", addr, duration, err)

		now := time.Now()

		// 更新服务器统计信息
		stats.Mu.Lock()
		stats.Queries++
		stats.FailedQueries++
		stats.LastQueryTime = now
		stats.Mu.Unlock()

		// 更新EWMA评分和滑动窗口
		// rcode=-1表示网络错误，使用默认半衰期10秒
		UpdateTimeDecayEWMA(stats, -1, -1, now, 0)
		UpdateSlidingWindow(stats, false)
		RecordQueryResult(stats, false)

		// 检查是否触发熔断
		if CheckCircuitBreaker(stats) {
			f.logger.Warn("服务器 %s 触发熔断，连续失败次数达到阈值", addr)
		}

		return nil, err
	}

	duration := time.Since(startTime)
	f.logger.Debug("转发查询 - 服务器 %s 响应成功, 耗时: %v", addr, duration)

	now := time.Now()
	latency := float64(duration.Milliseconds())

	// 更新服务器统计信息
	stats.Mu.Lock()
	stats.Queries++
	stats.SuccessfulQueries++
	stats.TotalResponseTime += duration
	stats.LastQueryTime = now
	stats.LastSuccessfulQueryTime = now
	stats.WindowQueries++
	stats.Status = "healthy"
	stats.Mu.Unlock()

	// 更新EWMA评分和滑动窗口
	// rcode=0表示NOERROR，使用默认半衰期10秒
	UpdateTimeDecayEWMA(stats, result.Rcode, latency, now, 0)
	UpdateSlidingWindow(stats, true)
	RecordQueryResult(stats, true)

	return result, nil
}

// ExchangeWithCookie 统一的DNS查询接口，支持Cookie、TCP管道化和动态协议升级
//
// 参数:
//   - serverAddr: 服务器地址
//   - query: DNS查询消息
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) ExchangeWithCookie(serverAddr string, query *dns.Msg) (*dns.Msg, error) {
	// 复制查询消息，避免修改原始查询
	msg := query.Copy()

	// 检查查询大小，>512字节优先走TCP
	querySize := msg.Len()
	if querySize > 512 {
		f.logger.Debug("查询大小 %d 字节超过512字节，优先使用TCP", querySize)
		return f.handleLargeQuery(serverAddr, msg)
	}

	// 查询服务器状态表，选择最优协议
	protocol := f.selectProtocol(serverAddr)
	f.logger.Debug("选择协议: %s, 服务器: %s", protocol, serverAddr)

	switch protocol {
	case "tcp":
		// 使用TCP管道化
		result, err := f.exchangeWithTCP(serverAddr, msg)
		if err != nil {
			f.logger.Debug("TCP查询失败，尝试降级: %v", err)
			return f.handleProtocolDowngrade(serverAddr, msg, "tcp", err)
		}
		return result, nil

	case "cookie":
		// 使用UDP+Cookie
		result, err := f.exchangeWithCookie(serverAddr, msg)
		if err != nil {
			f.logger.Debug("Cookie查询失败，尝试降级到UDP Plain: %v", err)
			return f.handleProtocolDowngrade(serverAddr, msg, "cookie", err)
		}
		return result, nil

	default:
		// 使用UDP Plain
		return f.exchangeWithUDP(serverAddr, msg)
	}
}

// selectProtocol 根据服务器状态选择协议
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - string: 选择的协议 ("tcp", "cookie", "udp")
func (f *DNSForwarder) selectProtocol(serverAddr string) string {
	// 获取服务器状态
	state, exists := f.ServerCapabilityProber.GetServerState(serverAddr)
	if !exists {
		// 服务器未探测，提交探测任务
		f.ServerCapabilityProber.SubmitProbe(serverAddr)
		// 默认使用UDP Plain
		return "udp"
	}

	caps := state.GetCapabilities()

	// 检查TCP支持
	if caps.HasCapability(CapabilityTCP) && caps.HasCapability(CapabilityPipeline) {
		// 检查TCP连接池是否已有已建立的健康连接
		if f.TCPConnectionPool != nil && f.TCPConnectionPool.HasHealthyConnection(serverAddr) {
			return "tcp"
		}

		// 没有健康连接，触发异步创建
		// EnsureConnections 是非阻塞的，直接调用即可
		f.TCPConnectionPool.EnsureConnections(serverAddr)
	}

	// 检查Cookie支持
	if caps.HasCapability(CapabilityEDNS0) {
		// 检查是否有有效的Server Cookie
		_, serverCookie, exists, _ := f.AdaptiveCookieManager.GetServerCookie(serverAddr)
		if exists && serverCookie != nil {
			return "cookie"
		}
	}

	// 默认使用UDP Plain
	return "udp"
}

// shouldUseTCP 判断是否应该使用TCP
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - bool: 是否应该使用TCP
func (f *DNSForwarder) shouldUseTCP(serverAddr string) bool {
	state, exists := f.ServerCapabilityProber.GetServerState(serverAddr)
	if !exists {
		return false
	}

	caps := state.GetCapabilities()
	return caps.HasCapability(CapabilityTCP) && caps.HasCapability(CapabilityPipeline)
}

// handleLargeQuery 处理大数据包查询，TCP不可用时两层降级
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: DNS查询消息
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) handleLargeQuery(serverAddr string, msg *dns.Msg) (*dns.Msg, error) {
	f.logger.Debug("处理大数据包查询，服务器: %s", serverAddr)

	// 第一层：尝试TCP
	if f.shouldUseTCP(serverAddr) {
		result, err := f.exchangeWithTCP(serverAddr, msg)
		if err == nil {
			return result, nil
		}
		f.logger.Debug("TCP查询失败，尝试降级到UDP+Cookie: %v", err)
	}

	// 第二层：尝试UDP+Cookie
	state, exists := f.ServerCapabilityProber.GetServerState(serverAddr)
	if exists {
		caps := state.GetCapabilities()
		if caps.HasCapability(CapabilityEDNS0) {
			result, err := f.exchangeWithCookie(serverAddr, msg)
			if err == nil {
				return result, nil
			}
			f.logger.Debug("UDP+Cookie查询失败，尝试降级到UDP Plain: %v", err)
		}
	}

	// 第三层：UDP Plain（可能失败，因为数据包太大）
	f.logger.Warn("大数据包查询降级到UDP Plain，可能因数据包过大而失败")
	return f.exchangeWithUDP(serverAddr, msg)
}

// tryReconnectTCP 尝试重建TCP连接
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回:
//   - bool: 是否成功重建
func (f *DNSForwarder) tryReconnectTCP(serverAddr string) bool {
	if f.TCPConnectionPool == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试获取连接，这会触发新连接的创建
	conn, err := f.TCPConnectionPool.GetConnection(serverAddr, ctx)
	if err != nil {
		f.logger.Debug("重建TCP连接失败: %v", err)
		return false
	}

	// 连接健康，可以使用
	if conn.IsHealthy() {
		return true
	}

	return false
}

// shouldRetryWithNewCookie 判断是否需要重试
//
// 参数:
//   - resp: DNS响应消息
//
// 返回:
//   - bool: 是否需要重试
func (f *DNSForwarder) shouldRetryWithNewCookie(resp *dns.Msg) bool {
	if resp == nil {
		return false
	}

	// 检查是否为BADCOOKIE响应
	if resp.Rcode == dns.RcodeBadCookie {
		return true
	}

	// 检查是否为REFUSED且包含echoed Cookie
	if f.isRefusedWithEchoedCookie(resp) {
		return true
	}

	return false
}

// handleBadCookie 处理BADCOOKIE响应
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: 原始查询消息
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) handleBadCookie(serverAddr string, msg *dns.Msg) (*dns.Msg, error) {
	f.logger.Debug("处理BADCOOKIE响应，刷新Cookie后重试，服务器: %s", serverAddr)

	// 记录Cookie失效
	f.AdaptiveCookieManager.RecordFailure(serverAddr)

	// 刷新Server Cookie
	newClientCookie, err := f.AdaptiveCookieManager.RefreshServerCookie(serverAddr)
	if err != nil {
		return nil, fmt.Errorf("刷新Cookie失败: %w", err)
	}

	// 复制消息并注入新的Client Cookie
	retryMsg := msg.Copy()
	RemoveCookie(retryMsg)
	if err := InjectCookie(retryMsg, newClientCookie, nil); err != nil {
		return nil, fmt.Errorf("注入新Cookie失败: %w", err)
	}

	// 使用UDP Plain发送（只有Client Cookie，等待服务器返回Server Cookie）
	return f.exchangeWithUDP(serverAddr, retryMsg)
}

// isRefusedWithEchoedCookie 判断是否为REFUSED + echoed Cookie
//
// 参数:
//   - resp: DNS响应消息
//
// 返回:
//   - bool: 是否为REFUSED + echoed Cookie
func (f *DNSForwarder) isRefusedWithEchoedCookie(resp *dns.Msg) bool {
	if resp == nil {
		return false
	}

	// 检查是否为REFUSED
	if resp.Rcode != dns.RcodeRefused {
		return false
	}

	// 检查是否包含echoed Cookie（只有Client Cookie，没有Server Cookie）
	isEchoed, err := IsEchoedCookie(resp)
	if err != nil {
		return false
	}

	return isEchoed
}

// handleRefusedWithCookie 处理REFUSED + echoed Cookie场景
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: 原始查询消息
//   - resp: 包含echoed Cookie的响应
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) handleRefusedWithCookie(serverAddr string, msg *dns.Msg, resp *dns.Msg) (*dns.Msg, error) {
	f.logger.Debug("处理REFUSED + echoed Cookie，获取Server Cookie后重试，服务器: %s", serverAddr)

	// 从响应中提取Client Cookie
	clientCookie, err := ExtractClientCookie(resp)
	if err != nil {
		return nil, fmt.Errorf("提取Client Cookie失败: %w", err)
	}

	// 从响应中提取Server Cookie（如果有的话）
	serverCookie, err := ExtractServerCookie(resp)
	if err == nil && serverCookie != nil {
		// 缓存Server Cookie
		f.AdaptiveCookieManager.SetServerCookie(serverAddr, clientCookie, serverCookie)
		f.logger.Debug("从REFUSED响应中提取并缓存Server Cookie")
	}

	// 使用获取到的Cookie重试
	retryMsg := msg.Copy()
	RemoveCookie(retryMsg)
	if err := InjectCookie(retryMsg, clientCookie, serverCookie); err != nil {
		return nil, fmt.Errorf("注入Cookie失败: %w", err)
	}

	return f.exchangeWithUDP(serverAddr, retryMsg)
}

// handleProtocolDowngrade 处理协议降级
//
// 降级路径:
// - TCP失败 → UDP+Cookie → UDP Plain
// - Cookie失败 → UDP Plain
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: DNS查询消息
//   - currentProtocol: 当前失败的协议
//   - originalErr: 原始错误
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) handleProtocolDowngrade(serverAddr string, msg *dns.Msg, currentProtocol string, originalErr error) (*dns.Msg, error) {
	f.logger.Debug("协议降级: %s -> 更低级别协议, 服务器: %s", currentProtocol, serverAddr)

	switch currentProtocol {
	case "tcp":
		// TCP失败，尝试UDP+Cookie
		state, exists := f.ServerCapabilityProber.GetServerState(serverAddr)
		if exists {
			caps := state.GetCapabilities()
			if caps.HasCapability(CapabilityEDNS0) {
				result, err := f.exchangeWithCookie(serverAddr, msg)
				if err == nil {
					return result, nil
				}
				// Cookie也失败，继续降级到UDP Plain
				return f.exchangeWithUDP(serverAddr, msg)
			}
		}
		// 服务器不支持EDNS0，直接降级到UDP Plain
		return f.exchangeWithUDP(serverAddr, msg)

	case "cookie":
		// Cookie失败，降级到UDP Plain
		return f.exchangeWithUDP(serverAddr, msg)

	default:
		// 已经是UDP Plain，无法降级
		return nil, originalErr
	}
}

// exchangeWithTCP 使用TCP管道化进行DNS查询
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: DNS查询消息
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) exchangeWithTCP(serverAddr string, msg *dns.Msg) (*dns.Msg, error) {
	if f.TCPConnectionPool == nil {
		return nil, fmt.Errorf("TCP连接池未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	f.logger.Debug("使用TCP管道化查询，服务器: %s", serverAddr)
	return f.TCPConnectionPool.Exchange(msg, serverAddr, ctx)
}

// exchangeWithCookie 使用UDP+Cookie进行DNS查询
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: DNS查询消息
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) exchangeWithCookie(serverAddr string, msg *dns.Msg) (*dns.Msg, error) {
	f.logger.Debug("使用UDP+Cookie查询，服务器: %s", serverAddr)

	// 复制消息
	cookieMsg := msg.Copy()

	// 获取Client Cookie和Server Cookie
	var clientCookie []byte
	var serverCookie []byte
	var exists bool

	// 检查原始消息中是否有Client Cookie
	origClientCookie, err := ExtractClientCookie(msg)
	if err == nil && origClientCookie != nil {
		// 透传客户端Client Cookie
		clientCookie = origClientCookie
		// 尝试获取对应的Server Cookie
		_, serverCookie, exists, _ = f.AdaptiveCookieManager.GetServerCookie(serverAddr)
		if !exists {
			serverCookie = nil
		}
	} else {
		// 生成系统Client Cookie
		clientCookie, serverCookie, exists, err = f.AdaptiveCookieManager.GetServerCookie(serverAddr)
		if err != nil {
			return nil, fmt.Errorf("获取Cookie失败: %w", err)
		}
		if !exists {
			serverCookie = nil
		}
	}

	// 注入Cookie到消息
	RemoveCookie(cookieMsg)
	if err := InjectCookie(cookieMsg, clientCookie, serverCookie); err != nil {
		return nil, fmt.Errorf("注入Cookie失败: %w", err)
	}

	// 执行UDP查询
	result, err := f.exchangeWithUDP(serverAddr, cookieMsg)
	if err != nil {
		return nil, err
	}

	// 从响应中提取Server Cookie并缓存
	respServerCookie, err := ExtractServerCookie(result)
	if err == nil && respServerCookie != nil {
		f.AdaptiveCookieManager.SetServerCookie(serverAddr, clientCookie, respServerCookie)
		f.logger.Debug("从响应中提取并缓存Server Cookie")
	}

	// 检查是否需要重试（BADCOOKIE或REFUSED + echoed Cookie）
	if f.shouldRetryWithNewCookie(result) {
		if result.Rcode == dns.RcodeBadCookie {
			return f.handleBadCookie(serverAddr, msg)
		}
		if f.isRefusedWithEchoedCookie(result) {
			return f.handleRefusedWithCookie(serverAddr, msg, result)
		}
	}

	return result, nil
}

// exchangeWithUDP 使用UDP Plain进行DNS查询
//
// 参数:
//   - serverAddr: 服务器地址
//   - msg: DNS查询消息
//
// 返回:
//   - *dns.Msg: DNS响应消息
//   - error: 错误信息
func (f *DNSForwarder) exchangeWithUDP(serverAddr string, msg *dns.Msg) (*dns.Msg, error) {
	f.logger.Debug("使用UDP Plain查询，服务器: %s", serverAddr)

	// 移除Cookie选项（UDP Plain不使用Cookie）
	udpMsg := msg.Copy()
	RemoveCookie(udpMsg)

	// 创建UDP客户端
	c := new(dns.Client)
	c.Net = "udp"
	c.Timeout = 5 * time.Second

	// 执行查询
	result, rtt, err := c.Exchange(udpMsg, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("UDP查询失败: %w", err)
	}

	f.logger.Debug("UDP查询成功，服务器: %s, 耗时: %v", serverAddr, rtt)
	return result, nil
}
