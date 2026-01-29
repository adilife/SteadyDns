// core/sdns/memorycache.go

package sdns

import (
	"SteadyDNS/core/common"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// 记录类型到默认TTL的映射（秒）
var recordTypeTTLMap = map[uint16]time.Duration{
	dns.TypeA:      3600 * time.Second,  // A记录默认1小时
	dns.TypeAAAA:   3600 * time.Second,  // AAAA记录默认1小时
	dns.TypeCNAME:  7200 * time.Second,  // CNAME记录默认2小时
	dns.TypeMX:     86400 * time.Second, // MX记录默认1天
	dns.TypeNS:     86400 * time.Second, // NS记录默认1天
	dns.TypeSOA:    86400 * time.Second, // SOA记录默认1天
	dns.TypeTXT:    3600 * time.Second,  // TXT记录默认1小时
	dns.TypeSRV:    3600 * time.Second,  // SRV记录默认1小时
	dns.TypePTR:    3600 * time.Second,  // PTR记录默认1小时
	dns.TypeNAPTR:  3600 * time.Second,  // NAPTR记录默认1小时
	dns.TypeDS:     86400 * time.Second, // DS记录默认1天
	dns.TypeRRSIG:  86400 * time.Second, // RRSIG记录默认1天
	dns.TypeDNSKEY: 86400 * time.Second, // DNSKEY记录默认1天
	dns.TypeNSEC:   86400 * time.Second, // NSEC记录默认1天
	dns.TypeNSEC3:  86400 * time.Second, // NSEC3记录默认1天
	dns.TypeANY:    300 * time.Second,   // ANY记录默认5分钟
}

// 默认TTL（当记录类型未在映射中定义时使用）
const defaultRecordTTL = 3600 * time.Second

// CacheEntry 缓存条目
type CacheEntry struct {
	Response   *dns.Msg  // DNS响应消息
	ExpireTime time.Time // 过期时间
	Size       int       // 条目大小（字节）
	LastAccess time.Time // 最后访问时间
}

// MemoryCache 内存缓存
type MemoryCache struct {
	cache            map[string]*CacheEntry // 缓存存储
	mutex            sync.RWMutex           // 并发锁
	maxSize          int64                  // 最大缓存大小（字节）
	currentSize      int64                  // 当前缓存大小（字节）
	cleanupInterval  time.Duration          // 清理间隔
	errorTTL         time.Duration          // 错误响应过期时间
	cleanupThreshold float64                // 清理阈值（0-1）
	hitCount         int64                  // 缓存命中次数
	missCount        int64                  // 缓存未命中次数
	evictionCount    int64                  // 缓存驱逐次数
	cleanupCount     int64                  // 清理执行次数
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	// 从配置获取配置
	var maxSizeMB int
	if size := common.GetConfig("Cache", "DNS_CACHE_SIZE_MB"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			maxSizeMB = s
		} else {
			maxSizeMB = 100 // 默认100MB
		}
	} else {
		maxSizeMB = 100 // 默认100MB
	}

	var cleanupInterval time.Duration
	if interval := common.GetConfig("Cache", "DNS_CACHE_CLEANUP_INTERVAL"); interval != "" {
		if i, err := strconv.Atoi(interval); err == nil && i > 0 {
			cleanupInterval = time.Duration(i) * time.Second
		} else {
			cleanupInterval = 60 * time.Second // 默认60秒
		}
	} else {
		cleanupInterval = 60 * time.Second // 默认60秒
	}

	var errorTTL time.Duration
	if t := common.GetConfig("Cache", "DNS_CACHE_ERROR_TTL"); t != "" {
		if i, err := strconv.Atoi(t); err == nil && i > 0 {
			errorTTL = time.Duration(i) * time.Second
		} else {
			errorTTL = 3600 * time.Second // 默认3600秒
		}
	} else {
		errorTTL = 3600 * time.Second // 默认3600秒
	}

	// 清理阈值，默认75%
	cleanupThreshold := 0.75
	if threshold := common.GetConfig("Cache", "DNS_CACHE_CLEANUP_THRESHOLD"); threshold != "" {
		if t, err := strconv.ParseFloat(threshold, 64); err == nil && t > 0 && t < 1 {
			cleanupThreshold = t
		}
	}

	cache := &MemoryCache{
		cache:            make(map[string]*CacheEntry),
		maxSize:          int64(maxSizeMB) * 1024 * 1024, // 转换为字节
		cleanupInterval:  cleanupInterval,
		errorTTL:         errorTTL,
		cleanupThreshold: cleanupThreshold,
		hitCount:         0,
		missCount:        0,
		evictionCount:    0,
		cleanupCount:     0,
	}

	// 启动定期清理过期条目
	go cache.startCleanup()

	return cache
}

// getCacheKey 生成缓存键
func getCacheKey(query *dns.Msg) string {
	if len(query.Question) == 0 {
		return ""
	}
	q := query.Question[0]
	return q.Name + "|" + dns.TypeToString[q.Qtype] + "|" + dns.ClassToString[q.Qclass]
}

// calculateSize 计算缓存条目大小
func calculateSize(msg *dns.Msg) int {
	data, err := json.Marshal(msg)
	if err != nil {
		return 0
	}
	return len(data)
}

// Set 添加或更新缓存条目
func (c *MemoryCache) Set(msg *dns.Msg) error {
	if len(msg.Question) == 0 {
		return nil
	}

	key := getCacheKey(msg)
	if key == "" {
		return nil
	}

	// 计算TTL
	ttl := c.errorTTL
	if len(msg.Answer) > 0 {
		// 首先尝试使用第一条记录的TTL
		if msg.Answer[0].Header().Ttl > 0 {
			ttl = time.Duration(msg.Answer[0].Header().Ttl) * time.Second
		} else {
			// 如果记录中没有指定TTL，使用基于记录类型的默认TTL
			recordType := msg.Answer[0].Header().Rrtype
			if defaultTTL, exists := recordTypeTTLMap[recordType]; exists {
				ttl = defaultTTL
			} else {
				ttl = defaultRecordTTL
			}
		}
	}

	// 创建缓存条目
	size := calculateSize(msg)
	entry := &CacheEntry{
		Response:   msg.Copy(),
		ExpireTime: time.Now().Add(ttl),
		Size:       size,
		LastAccess: time.Now(),
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查缓存使用百分比，如果超过阈值，触发清理
	usagePercent := float64(c.currentSize) / float64(c.maxSize)
	if usagePercent > c.cleanupThreshold {
		// 根据使用百分比确定清理强度
		cleanupPercentage := 0.2 // 默认清理20%
		if usagePercent > 0.9 {
			cleanupPercentage = 0.5 // 如果使用超过90%，清理50%
		} else if usagePercent > 0.8 {
			cleanupPercentage = 0.3 // 如果使用超过80%，清理30%
		}
		c.cleanupByPercentage(cleanupPercentage)
	}

	// 检查是否已存在该条目
	if existingEntry, ok := c.cache[key]; ok {
		// 更新现有条目大小
		c.currentSize -= int64(existingEntry.Size)
	}

	// 检查缓存大小是否超过限制
	for c.currentSize+int64(size) > c.maxSize {
		// 移除最久未使用的条目
		c.evictLRU()
	}

	// 添加或更新条目
	c.cache[key] = entry
	c.currentSize += int64(size)

	return nil
}

// Get 获取缓存条目
func (c *MemoryCache) Get(query *dns.Msg) *dns.Msg {
	key := getCacheKey(query)
	if key == "" {
		c.mutex.Lock()
		c.missCount++
		c.mutex.Unlock()
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, ok := c.cache[key]
	if !ok {
		c.missCount++
		return nil
	}

	// 检查是否过期
	if time.Now().After(entry.ExpireTime) {
		// 移除过期条目
		c.currentSize -= int64(entry.Size)
		delete(c.cache, key)
		c.missCount++
		return nil
	}

	// 更新最后访问时间
	entry.LastAccess = time.Now()
	c.hitCount++

	// 复制响应消息
	response := entry.Response.Copy()
	// 更新消息 ID 以匹配查询 ID
	response.Id = query.Id

	return response
}

// Delete 删除缓存条目
func (c *MemoryCache) Delete(query *dns.Msg) {
	key := getCacheKey(query)
	if key == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, ok := c.cache[key]; ok {
		c.currentSize -= int64(entry.Size)
		delete(c.cache, key)
	}
}

// Clear 清空缓存
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry)
	c.currentSize = 0
}

// DeleteByDomain 清除与指定域名相关的所有缓存条目
func (c *MemoryCache) DeleteByDomain(domain string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	deletedCount := 0
	for key, entry := range c.cache {
		// 检查缓存键是否包含指定域名
		if strings.Contains(key, domain+".") || strings.Contains(key, domain) {
			c.currentSize -= int64(entry.Size)
			delete(c.cache, key)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		common.NewLogger().Debug("清除域名 %s 的缓存条目，共 %d 个", domain, deletedCount)
	}
}

// evictLRU 移除最久未使用的条目
func (c *MemoryCache) evictLRU() {
	if len(c.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		c.currentSize -= int64(c.cache[oldestKey].Size)
		delete(c.cache, oldestKey)
		c.evictionCount++
	}
}

// cleanupByPercentage 根据百分比清理缓存
func (c *MemoryCache) cleanupByPercentage(percentage float64) {
	if len(c.cache) == 0 {
		return
	}

	// 计算需要清理的条目数量
	targetCount := int(float64(len(c.cache)) * percentage)
	if targetCount == 0 {
		targetCount = 1 // 至少清理一个
	}

	// 收集所有条目的访问时间和键
	type entryInfo struct {
		key        string
		lastAccess time.Time
	}
	var entries []entryInfo

	for key, entry := range c.cache {
		entries = append(entries, entryInfo{
			key:        key,
			lastAccess: entry.LastAccess,
		})
	}

	// 按最后访问时间排序（最旧的在前）
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].lastAccess.After(entries[j].lastAccess) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// 清理最旧的条目
	cleanedCount := 0
	for _, info := range entries {
		if cleanedCount >= targetCount {
			break
		}
		if entry, ok := c.cache[info.key]; ok {
			c.currentSize -= int64(entry.Size)
			delete(c.cache, info.key)
			c.evictionCount++
			cleanedCount++
		}
	}

	c.cleanupCount++
}

// startCleanup 启动定期清理过期条目
func (c *MemoryCache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

// cleanupExpired 清理过期条目
func (c *MemoryCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredCount := 0
	for key, entry := range c.cache {
		if now.After(entry.ExpireTime) {
			c.currentSize -= int64(entry.Size)
			delete(c.cache, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		c.cleanupCount++
	}
}

// Stats 获取缓存统计信息
func (c *MemoryCache) Stats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// 计算命中率
	var hitRate float64 = 0
	totalRequests := c.hitCount + c.missCount
	if totalRequests > 0 {
		hitRate = float64(c.hitCount) / float64(totalRequests) * 100
	}

	// 计算缓存使用百分比
	usagePercent := float64(c.currentSize) / float64(c.maxSize) * 100

	stats := map[string]interface{}{
		"count":            len(c.cache),
		"currentSize":      c.currentSize,
		"maxSize":          c.maxSize,
		"usagePercent":     usagePercent,
		"cleanupInterval":  c.cleanupInterval,
		"errorTTL":         c.errorTTL,
		"cleanupThreshold": c.cleanupThreshold,
		"hitCount":         c.hitCount,
		"missCount":        c.missCount,
		"totalRequests":    totalRequests,
		"hitRate":          hitRate,
		"evictionCount":    c.evictionCount,
		"cleanupCount":     c.cleanupCount,
	}

	return stats
}
