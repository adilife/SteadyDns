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

// ClearCacheByDomain 清除与指定域名相关的所有缓存条目
func (c *CacheUpdater) ClearCacheByDomain(domain string) {
	c.cache.DeleteByDomain(domain)
}

// ClearCache 清空缓存
func ClearCache() error {
	GlobalCacheUpdater.logger.Info("清空缓存...")

	if GlobalCacheUpdater != nil {
		GlobalCacheUpdater.cache.Clear()
		GlobalCacheUpdater.logger.Info("缓存清空成功")
		return nil
	}

	GlobalCacheUpdater.logger.Error("缓存更新器未初始化")
	return nil
}

// ClearCacheByDomain 按域名清空缓存
func ClearCacheByDomain(domain string) error {
	GlobalCacheUpdater.logger.Info("按域名清空缓存: %s", domain)

	if GlobalCacheUpdater != nil {
		GlobalCacheUpdater.ClearCacheByDomain(domain)
		GlobalCacheUpdater.logger.Info("按域名清空缓存成功: %s", domain)
		return nil
	}

	GlobalCacheUpdater.logger.Error("缓存更新器未初始化")
	return nil
}

// GetCacheStats 获取缓存统计信息
func GetCacheStats() map[string]interface{} {
	if GlobalCacheUpdater != nil {
		return GlobalCacheUpdater.GetCacheStats()
	}
	return nil
}
