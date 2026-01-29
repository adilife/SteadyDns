// core/sdns/domain_match.go

package sdns

import (
	"strings"
	"time"
)

// initDomainIndex 初始化域名索引，按域名长度降序排序
func (f *DNSForwarder) initDomainIndex() {

	// 提取所有非默认转发组的域名
	domains := make([]string, 0, len(f.groups))
	for domain := range f.groups {
		// 跳过默认转发组（使用"Default"作为key）
		if domain != "Default" {
			domains = append(domains, domain)
		}
	}

	// 按域名长度降序排序，用于最长匹配
	for i := 0; i < len(domains); i++ {
		for j := i + 1; j < len(domains); j++ {
			if len(domains[i]) < len(domains[j]) {
				domains[i], domains[j] = domains[j], domains[i]
			}
		}
	}

	// 更新域名索引
	f.domainIndex = domains
}

// matchDomain 根据查询域名匹配最合适的转发组
// 实现最长匹配机制：完全匹配或前缀+当前域名
func (f *DNSForwarder) matchDomain(queryDomain string) *ForwardGroup {
	// 移除末尾的点
	queryDomain = strings.TrimSuffix(queryDomain, ".")

	// 检查缓存
	f.matchCacheMu.RLock()
	entry, found := f.matchCache[queryDomain]
	f.matchCacheMu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.group
	}

	// 尝试最长匹配
	var matchedGroup *ForwardGroup
	var matchedZone string

	// 先尝试匹配转发域
	for _, domain := range f.domainIndex {
		// 完全匹配（如jcgov.gov.cn）
		if queryDomain == domain {
			f.mu.RLock()
			matchedGroup = f.groups[domain]
			matchedZone = domain
			f.mu.RUnlock()
			break
		}
		// 前缀+当前域名（如www.jcgov.gov.cn）
		if strings.HasSuffix(queryDomain, "."+domain) {
			f.mu.RLock()
			matchedGroup = f.groups[domain]
			matchedZone = domain
			f.mu.RUnlock()
			break
		}
	}

	// 检查是否与权威域冲突
	if matchedGroup != nil {
		isAuthority, authorityZone := f.authorityForwarder.MatchAuthorityZone(queryDomain)
		if isAuthority {
			// 检查权威域是否更具体
			authorityLen := len(authorityZone)
			matchedLen := len(matchedZone)

			// 权威域更具体，优先级更高
			if authorityLen > matchedLen {
				// 此域名应转发至权威域，返回默认组让后续逻辑处理
				f.mu.RLock()
				matchedGroup = f.defaultGroup
				f.mu.RUnlock()
			}
		}
	}

	// 没有匹配到，返回默认转发组
	if matchedGroup == nil {
		f.mu.RLock()
		matchedGroup = f.defaultGroup
		f.mu.RUnlock()
	}

	// 更新缓存
	f.matchCacheMu.Lock()
	f.matchCache[queryDomain] = &cacheEntry{
		group:     matchedGroup,
		expiresAt: time.Now().Add(f.cacheTTL),
	}
	f.matchCacheMu.Unlock()

	return matchedGroup
}

// TestDomainMatch 测试域名匹配，返回匹配到的转发组名称
func (f *DNSForwarder) TestDomainMatch(domain string) string {
	matchedGroup := f.matchDomain(domain)
	if matchedGroup != nil {
		return matchedGroup.Name
	}
	return ""
}
