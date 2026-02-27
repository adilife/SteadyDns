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

// core/sdns/domain_match.go

package sdns

import (
	"strings"
	"time"
)

// initDomainIndex 初始化域名索引，按域名长度降序排序
func (f *DNSForwarder) initDomainIndex() {
	// 清空并重新初始化 Trie 树
	f.domainTrie.Clear()

	// 提取所有非默认转发组的域名
	domains := make([]string, 0, len(f.groups))
	for domain := range f.groups {
		// 跳过默认转发组（使用"Default"作为key）
		if domain != "Default" {
			domains = append(domains, domain)
			// 插入到 Trie 树中
			if err := f.domainTrie.Insert(domain, f.groups[domain]); err != nil {
				f.logger.Warn("插入域名到Trie树失败: %s, 错误: %v", domain, err)
			}
		}
	}

	// 按域名长度降序排序，用于最长匹配（保留旧的索引以保持向后兼容）
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

	now := time.Now()

	// 检查缓存
	f.matchCacheMu.RLock()
	entry, found := f.matchCache[queryDomain]
	f.matchCacheMu.RUnlock()

	if found && now.Before(entry.expiresAt) {
		// 更新最后访问时间
		f.matchCacheMu.Lock()
		if e, ok := f.matchCache[queryDomain]; ok {
			e.lastAccess = now
		}
		f.matchCacheMu.Unlock()
		return entry.group
	}

	// 使用 Trie 树进行最长匹配
	var matchedGroup *ForwardGroup
	var matchedZone string
	matchedGroup, matchedZone = f.domainTrie.SearchWithZone(queryDomain)

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
	defer f.matchCacheMu.Unlock()

	// 检查缓存是否超过上限，如果超过则使用LRU淘汰最旧的条目
	if len(f.matchCache) >= f.maxMatchCacheSize {
		f.evictLRUMatchCache()
	}

	f.matchCache[queryDomain] = &cacheEntry{
		group:     matchedGroup,
		expiresAt: now.Add(f.cacheTTL),
		lastAccess: now,
	}

	return matchedGroup
}

// evictLRUMatchCache 淘汰最久未使用的域名匹配缓存条目
// 调用前必须持有 matchCacheMu 锁
func (f *DNSForwarder) evictLRUMatchCache() {
	if len(f.matchCache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, entry := range f.matchCache {
		if oldestKey == "" || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
		}
	}

	if oldestKey != "" {
		delete(f.matchCache, oldestKey)
	}
}

// TestDomainMatch 测试域名匹配，返回匹配到的转发组名称
func (f *DNSForwarder) TestDomainMatch(domain string) string {
	matchedGroup := f.matchDomain(domain)
	if matchedGroup != nil {
		return matchedGroup.Name
	}
	return ""
}
