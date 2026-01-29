// core/sdns/health_check.go

package sdns

import (
	"fmt"
	"time"

	"github.com/miekg/dns"
)

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

	if result == nil {
		f.logger.Debug("健康检查 - 服务器 %s 失败，返回空结果", addr)
		return false
	}

	// 服务器有返回，认为健康，记录返回码
	f.logger.Debug("健康检查 - 服务器 %s 成功，返回码: %d", addr, result.Rcode)
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

	// 只检查服务器状态
	return stats.Status == "healthy"
}

// runHealthChecks 执行健康检查
func (f *DNSForwarder) runHealthChecks() {
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

// StartHealthChecks 启动健康检查协程
func (f *DNSForwarder) StartHealthChecks() {
	go func() {
		// 启动时立即执行一次健康检查
		f.logger.Debug("启动健康检查协程，立即执行首次健康检查")
		f.runHealthChecks()

		// 然后启动定时健康检查
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			f.runHealthChecks()
		}
	}()
}
