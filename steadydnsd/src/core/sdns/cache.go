package sdns

import (
	"github.com/miekg/dns"
)

// CacheUpdater 缓存更新器接口
type CacheUpdater struct {
	cache *MemoryCache
}

// NewCacheUpdater 创建缓存更新器
func NewCacheUpdater() *CacheUpdater {
	return &CacheUpdater{
		cache: NewMemoryCache(),
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

// ClearCache 清空缓存
func (c *CacheUpdater) ClearCache() {
	c.cache.Clear()
}
