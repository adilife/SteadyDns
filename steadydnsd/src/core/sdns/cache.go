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

package sdns

import (
	"SteadyDNS/core/common"

	"github.com/miekg/dns"
)

// CacheUpdater 缓存更新器接口
type CacheUpdater struct {
	cache  *MemoryCache
	logger *common.Logger
}

// NewCacheUpdater 创建缓存更新器
func NewCacheUpdater() *CacheUpdater {
	return &CacheUpdater{
		cache:  NewMemoryCache(),
		logger: common.NewLogger(),
	}
}

// CheckCache 查询缓存
func (c *CacheUpdater) CheckCache(query *dns.Msg) (*dns.Msg, error) {
	return c.cache.Get(query), nil
}

// UpdateCacheWithResult 更新缓存中的查询结果
func (c *CacheUpdater) UpdateCacheWithResult(result *dns.Msg) error {
	return c.cache.Set(result)
}

// GetCacheStats 获取缓存统计信息
func (c *CacheUpdater) GetCacheStats() map[string]interface{} {
	return c.cache.Stats()
}

// ClearCacheByDomain 清除与指定域名相关的所有缓存条目
func (c *CacheUpdater) ClearCacheByDomain(domain string) {
	c.cache.DeleteByDomain(domain)
}

// ClearCache 清空缓存
func ClearCache() error {
	if GlobalCacheUpdater != nil {
		if GlobalCacheUpdater.logger != nil {
			GlobalCacheUpdater.logger.Info("清空缓存...")
		}
		GlobalCacheUpdater.cache.Clear()
		if GlobalCacheUpdater.logger != nil {
			GlobalCacheUpdater.logger.Info("缓存清空成功")
		}
		return nil
	}

	return nil
}

// ClearCacheByDomain 按域名清空缓存
func ClearCacheByDomain(domain string) error {
	if GlobalCacheUpdater != nil {
		if GlobalCacheUpdater.logger != nil {
			GlobalCacheUpdater.logger.Info("按域名清空缓存: %s", domain)
		}
		GlobalCacheUpdater.ClearCacheByDomain(domain)
		if GlobalCacheUpdater.logger != nil {
			GlobalCacheUpdater.logger.Info("按域名清空缓存成功: %s", domain)
		}
		return nil
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
func GetCacheStats() map[string]interface{} {
	if GlobalCacheUpdater != nil {
		return GlobalCacheUpdater.GetCacheStats()
	}
	return nil
}
