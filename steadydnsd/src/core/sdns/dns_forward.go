// core/sdns/dns_forward.go

package sdns

import (
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
		for _, stats := range healthyStatsList {
			addr := stats.Address
			score := stats.EWMAScore
			delay := calculateTieredDelay(score)

			f.logger.Debug("转发查询 - 服务器 %s (评分: %.3f, 延迟: %v)", addr, score, delay)

			// 创建转发任务
			task := &DNSForwardTask{
				address:    addr,
				query:      query,
				resultChan: resultChan,
				errorChan:  errorChan,
				forwarder:  f,
				cancelChan: cancelChan,
			}

			// 根据评分延迟启动
			go func(t *DNSForwardTask, d time.Duration) {
				if d > 0 {
					time.Sleep(d)
				}
				f.forwardPool.SubmitTask(t)
			}(task, delay)

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
func (f *DNSForwarder) forwardToServer(addr string, query *dns.Msg, cancelChan chan struct{}) (*dns.Msg, error) {
	startTime := time.Now()

	// 获取或创建服务器统计信息
	stats := f.getOrCreateServerStats(addr)

	// 创建结果通道
	resultChan := make(chan *dns.Msg, 1)
	errorChan := make(chan error, 1)

	// 在goroutine中执行DNS查询
	func() {
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

		// 所有DNS响应码都视为成功（服务器健康）
		if result.Rcode == dns.RcodeSuccess {
			// 成功响应
			f.logger.Debug("转发查询 - DNS查询成功，服务器: %s, 耗时: %v", addr, rtt)
			resultChan <- result
		} else {
			// 其他响应码也视为成功（服务器健康，返回结果）
			f.logger.Debug("转发查询 - DNS查询返回结果码，服务器: %s, 返回码: %d", addr, result.Rcode)
			resultChan <- result
		}
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
		// rcode=0表示NOERROR
		UpdateTimeDecayEWMA(stats, 0, latency, now)
		UpdateSlidingWindow(stats, true)
		RecordQueryResult(stats, true)

		return result, nil
	case err := <-errorChan:
		// 收到错误
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
		// rcode=-1表示网络错误
		UpdateTimeDecayEWMA(stats, -1, -1, now)
		UpdateSlidingWindow(stats, false)
		RecordQueryResult(stats, false)

		// 检查是否触发熔断
		if CheckCircuitBreaker(stats) {
			f.logger.Warn("服务器 %s 触发熔断，连续失败次数达到阈值", addr)
		}

		return nil, err
	}
}
