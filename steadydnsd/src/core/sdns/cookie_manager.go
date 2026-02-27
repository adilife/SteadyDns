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
// core/sdns/cookie_manager.go
// 自适应Cookie管理器 - 实现DNS Cookie机制，支持高并发访问

package sdns

import (
	"crypto/rand"
	"hash/fnv"
	"net"
	"sync"
	"time"
)

const (
	// ShardCount 分片数量，用于减少锁竞争
	ShardCount = 256

	// CookieExpiration Cookie过期时间
	CookieExpiration = 1 * time.Hour

	// FailureWindow 失效记录窗口时间，防止重复刷新
	FailureWindow = 5 * time.Second

	// CleanupInterval 清理过期失效记录的时间间隔
	CleanupInterval = 1 * time.Minute
)

// CookieEntry 存储单个服务器的Cookie信息
type CookieEntry struct {
	// ClientCookie 客户端生成的Cookie（8字节）
	ClientCookie []byte
	// ServerCookie 服务器返回的Cookie
	ServerCookie []byte
	// ExpiresAt Cookie过期时间
	ExpiresAt time.Time
}

// CookieFailureRecord 记录Cookie失效信息
type CookieFailureRecord struct {
	// ServerAddr 服务器地址
	ServerAddr string
	// FailedAt 失效时间
	FailedAt time.Time
}

// cookieShard 单个分片，包含一组服务器的Cookie信息
type cookieShard struct {
	// mu 读写锁，保护分片内的数据
	mu sync.RWMutex
	// entries 服务器地址到Cookie条目的映射
	entries map[string]*CookieEntry
}

// AdaptiveCookieManager 自适应Cookie管理器
type AdaptiveCookieManager struct {
	// shards 分片数组，每个分片独立加锁
	shards [ShardCount]*cookieShard
	// pool 用于预生成Client Cookie的对象池，减少GC压力
	// 使用cookie_utils.go中定义的ClientCookieSize常量
	pool sync.Pool
	// failureMu 保护失效记录的互斥锁
	failureMu sync.RWMutex
	// failures 失效记录映射
	failures map[string]*CookieFailureRecord
	// stopCleanup 停止清理goroutine的信号通道
	stopCleanup chan struct{}
}

// NewAdaptiveCookieManager 创建新的自适应Cookie管理器实例
//
// 返回值:
//   - *AdaptiveCookieManager: 新创建的Cookie管理器实例
func NewAdaptiveCookieManager() *AdaptiveCookieManager {
	acm := &AdaptiveCookieManager{
		shards:      [ShardCount]*cookieShard{},
		failures:    make(map[string]*CookieFailureRecord),
		stopCleanup: make(chan struct{}),
	}

	// 初始化对象池，用于预分配Client Cookie内存
	// 使用cookie_utils.go中定义的ClientCookieSize常量（8字节）
	acm.pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 8) // ClientCookieSize = 8
		},
	}

	// 初始化所有分片
	for i := 0; i < ShardCount; i++ {
		acm.shards[i] = &cookieShard{
			entries: make(map[string]*CookieEntry),
		}
	}

	// 启动定期清理goroutine
	go acm.cleanupLoop()

	return acm
}

// Stop 停止Cookie管理器，清理资源
//
// 应在程序退出时调用，确保清理goroutine正确退出
func (acm *AdaptiveCookieManager) Stop() {
	close(acm.stopCleanup)
}

// getShard 根据服务器地址获取对应的分片
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回值:
//   - *cookieShard: 对应的分片实例
func (acm *AdaptiveCookieManager) getShard(serverAddr string) *cookieShard {
	h := fnv.New32a()
	h.Write([]byte(serverAddr))
	index := h.Sum32() % ShardCount
	return acm.shards[index]
}

// generateClientCookie 生成8字节随机Client Cookie
//
// 返回值:
//   - []byte: 生成的8字节随机数据
//   - error: 生成过程中的错误
func (acm *AdaptiveCookieManager) generateClientCookie() ([]byte, error) {
	// 从对象池获取内存
	cookie := acm.pool.Get().([]byte)
	// 确保切片长度正确（ClientCookieSize = 8字节）
	cookie = cookie[:8]

	// 生成随机数据
	if _, err := rand.Read(cookie); err != nil {
		// 发生错误时，将内存归还对象池
		acm.pool.Put(cookie)
		return nil, err
	}

	return cookie, nil
}

// releaseClientCookie 将Client Cookie内存归还对象池
//
// 参数:
//   - cookie: 要归还的Cookie字节切片
func (acm *AdaptiveCookieManager) releaseClientCookie(cookie []byte) {
	if cookie != nil {
		acm.pool.Put(cookie)
	}
}

// GetServerCookie 获取指定服务器的Server Cookie
//
// 如果该服务器没有有效的Cookie，会生成新的Client Cookie并返回
//
// 参数:
//   - serverAddr: 服务器地址（IP:Port格式）
//
// 返回值:
//   - clientCookie: 客户端Cookie（8字节）
//   - serverCookie: 服务器Cookie（可能为nil）
//   - exists: 是否存在有效的Server Cookie
//   - err: 错误信息
func (acm *AdaptiveCookieManager) GetServerCookie(serverAddr string) (clientCookie, serverCookie []byte, exists bool, err error) {
	shard := acm.getShard(serverAddr)

	// 先尝试读锁获取
	shard.mu.RLock()
	entry, found := shard.entries[serverAddr]
	if found && time.Now().Before(entry.ExpiresAt) {
		// 找到有效的Cookie
		clientCookie = make([]byte, len(entry.ClientCookie))
		copy(clientCookie, entry.ClientCookie)
		if entry.ServerCookie != nil {
			serverCookie = make([]byte, len(entry.ServerCookie))
			copy(serverCookie, entry.ServerCookie)
		}
		shard.mu.RUnlock()
		return clientCookie, serverCookie, true, nil
	}
	shard.mu.RUnlock()

	// 没有找到有效Cookie，生成新的Client Cookie
	clientCookie, err = acm.generateClientCookie()
	if err != nil {
		return nil, nil, false, err
	}

	return clientCookie, nil, false, nil
}

// SetServerCookie 存储服务器的Server Cookie
//
// 参数:
//   - serverAddr: 服务器地址
//   - clientCookie: 客户端Cookie
//   - serverCookie: 服务器返回的Cookie
func (acm *AdaptiveCookieManager) SetServerCookie(serverAddr string, clientCookie, serverCookie []byte) {
	shard := acm.getShard(serverAddr)

	// 创建新的Cookie条目
	entry := &CookieEntry{
		ClientCookie: make([]byte, len(clientCookie)),
		ServerCookie: make([]byte, len(serverCookie)),
		ExpiresAt:    time.Now().Add(CookieExpiration),
	}
	copy(entry.ClientCookie, clientCookie)
	copy(entry.ServerCookie, serverCookie)

	// 使用写锁存储
	shard.mu.Lock()
	// 如果之前有旧的Client Cookie，尝试归还对象池
	if oldEntry, exists := shard.entries[serverAddr]; exists {
		acm.releaseClientCookie(oldEntry.ClientCookie)
	}
	shard.entries[serverAddr] = entry
	shard.mu.Unlock()
}

// RefreshServerCookie 刷新失效的Server Cookie
//
// 当Server Cookie验证失败时调用，生成新的Client Cookie
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回值:
//   - []byte: 新生成的Client Cookie
//   - error: 生成过程中的错误
func (acm *AdaptiveCookieManager) RefreshServerCookie(serverAddr string) ([]byte, error) {
	shard := acm.getShard(serverAddr)

	// 生成新的Client Cookie
	newClientCookie, err := acm.generateClientCookie()
	if err != nil {
		return nil, err
	}

	// 更新条目，清除Server Cookie
	shard.mu.Lock()
	if oldEntry, exists := shard.entries[serverAddr]; exists {
		// 归还旧的Client Cookie内存
		acm.releaseClientCookie(oldEntry.ClientCookie)
	}
	// 创建新条目，只包含Client Cookie
	shard.entries[serverAddr] = &CookieEntry{
		ClientCookie: newClientCookie,
		ServerCookie: nil,
		ExpiresAt:    time.Now().Add(CookieExpiration),
	}
	shard.mu.Unlock()

	// 返回复制的Client Cookie（调用者负责释放）
	result := make([]byte, len(newClientCookie))
	copy(result, newClientCookie)
	return result, nil
}

// IsRecentlyFailed 检查指定服务器的Cookie是否最近失效过
//
// 用于防止在短时间内重复刷新同一服务器的Cookie
//
// 参数:
//   - serverAddr: 服务器地址
//
// 返回值:
//   - bool: 是否在失效窗口期内失效过
func (acm *AdaptiveCookieManager) IsRecentlyFailed(serverAddr string) bool {
	acm.failureMu.RLock()
	record, exists := acm.failures[serverAddr]
	acm.failureMu.RUnlock()

	if !exists {
		return false
	}

	return time.Since(record.FailedAt) < FailureWindow
}

// RecordFailure 记录Cookie失效
//
// 当Server Cookie验证失败时调用，记录失效时间
//
// 参数:
//   - serverAddr: 服务器地址
func (acm *AdaptiveCookieManager) RecordFailure(serverAddr string) {
	acm.failureMu.Lock()
	acm.failures[serverAddr] = &CookieFailureRecord{
		ServerAddr: serverAddr,
		FailedAt:   time.Now(),
	}
	acm.failureMu.Unlock()
}

// cleanupLoop 定期清理过期的失效记录
//
// 在后台goroutine中运行，直到Stop被调用
func (acm *AdaptiveCookieManager) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			acm.cleanupExpiredFailures()
		case <-acm.stopCleanup:
			return
		}
	}
}

// cleanupExpiredFailures 清理过期的失效记录
//
// 删除超过失效窗口时间的记录
func (acm *AdaptiveCookieManager) cleanupExpiredFailures() {
	acm.failureMu.Lock()
	defer acm.failureMu.Unlock()

	now := time.Now()
	for addr, record := range acm.failures {
		if now.Sub(record.FailedAt) > FailureWindow {
			delete(acm.failures, addr)
		}
	}
}

// GetStats 获取Cookie管理器的统计信息
//
// 返回值:
//   - totalEntries: 总Cookie条目数
//   - totalFailures: 当前失效记录数
func (acm *AdaptiveCookieManager) GetStats() (totalEntries, totalFailures int) {
	// 统计Cookie条目
	for i := 0; i < ShardCount; i++ {
		shard := acm.shards[i]
		shard.mu.RLock()
		totalEntries += len(shard.entries)
		shard.mu.RUnlock()
	}

	// 统计失效记录
	acm.failureMu.RLock()
	totalFailures = len(acm.failures)
	acm.failureMu.RUnlock()

	return totalEntries, totalFailures
}

// RemoveServer 移除指定服务器的Cookie记录
//
// 参数:
//   - serverAddr: 服务器地址
func (acm *AdaptiveCookieManager) RemoveServer(serverAddr string) {
	shard := acm.getShard(serverAddr)

	shard.mu.Lock()
	if entry, exists := shard.entries[serverAddr]; exists {
		acm.releaseClientCookie(entry.ClientCookie)
		delete(shard.entries, serverAddr)
	}
	shard.mu.Unlock()
}

// Clear 清空所有Cookie记录
//
// 谨慎使用，会清空所有服务器的Cookie信息
func (acm *AdaptiveCookieManager) Clear() {
	for i := 0; i < ShardCount; i++ {
		shard := acm.shards[i]
		shard.mu.Lock()
		// 归还所有Client Cookie内存
		for _, entry := range shard.entries {
			acm.releaseClientCookie(entry.ClientCookie)
		}
		shard.entries = make(map[string]*CookieEntry)
		shard.mu.Unlock()
	}
}

// ValidateServerCookie 验证Server Cookie是否匹配
//
// 参数:
//   - serverAddr: 服务器地址
//   - clientCookie: 客户端Cookie
//   - serverCookie: 要验证的服务器Cookie
//
// 返回值:
//   - bool: 是否匹配
func (acm *AdaptiveCookieManager) ValidateServerCookie(serverAddr string, clientCookie, serverCookie []byte) bool {
	shard := acm.getShard(serverAddr)

	shard.mu.RLock()
	entry, exists := shard.entries[serverAddr]
	if !exists || time.Now().After(entry.ExpiresAt) {
		shard.mu.RUnlock()
		return false
	}

	// 验证Client Cookie匹配
	if !bytesEqual(entry.ClientCookie, clientCookie) {
		shard.mu.RUnlock()
		return false
	}

	// 验证Server Cookie匹配
	match := bytesEqual(entry.ServerCookie, serverCookie)
	shard.mu.RUnlock()

	return match
}

// bytesEqual 安全地比较两个字节切片是否相等
//
// 参数:
//   - a: 第一个字节切片
//   - b: 第二个字节切片
//
// 返回值:
//   - bool: 是否相等
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ParseServerAddr 解析服务器地址字符串
//
// 参数:
//   - addr: 地址字符串（如 "192.168.1.1:53" 或 "[::1]:53"）
//
// 返回值:
//   - string: 标准化的地址字符串
//   - error: 解析错误
func ParseServerAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	// 解析IP地址
	ip := net.ParseIP(host)
	if ip == nil {
		return "", &net.AddrError{Err: "invalid IP address", Addr: host}
	}

	// 标准化IPv6地址
	if ip.To4() == nil {
		// IPv6地址
		return net.JoinHostPort(ip.String(), port), nil
	}

	// IPv4地址
	return net.JoinHostPort(ip.String(), port), nil
}
