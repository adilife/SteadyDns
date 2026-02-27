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
// core/sdns/cookie_manager_test.go
// Cookie管理器单元测试

package sdns

import (
	"sync"
	"testing"
	"time"
)

// TestNewAdaptiveCookieManager 测试创建新的Cookie管理器
func TestNewAdaptiveCookieManager(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	if acm == nil {
		t.Fatal("NewAdaptiveCookieManager() returned nil")
	}
	defer acm.Stop()

	// 验证分片已初始化
	for i := 0; i < ShardCount; i++ {
		if acm.shards[i] == nil {
			t.Errorf("Shard %d is nil", i)
		}
		if acm.shards[i].entries == nil {
			t.Errorf("Shard %d entries map is nil", i)
		}
	}

	// 验证失效记录映射已初始化
	if acm.failures == nil {
		t.Error("failures map is nil")
	}

	// 验证清理通道已初始化
	if acm.stopCleanup == nil {
		t.Error("stopCleanup channel is nil")
	}
}

// TestCookieEntryStructure 测试CookieEntry结构体
func TestCookieEntryStructure(t *testing.T) {
	entry := &CookieEntry{
		ClientCookie: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		ServerCookie: []byte{9, 10, 11, 12, 13, 14, 15, 16},
		ExpiresAt:    time.Now().Add(CookieExpiration),
	}

	if len(entry.ClientCookie) != 8 {
		t.Errorf("ClientCookie length = %d, want 8", len(entry.ClientCookie))
	}

	if len(entry.ServerCookie) != 8 {
		t.Errorf("ServerCookie length = %d, want 8", len(entry.ServerCookie))
	}

	if time.Now().After(entry.ExpiresAt) {
		t.Error("ExpiresAt should be in the future")
	}
}

// TestGetServerCookieNew 测试获取新服务器的Cookie
func TestGetServerCookieNew(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"

	// 获取新服务器的Cookie
	clientCookie, serverCookie, exists, err := acm.GetServerCookie(serverAddr)
	if err != nil {
		t.Fatalf("GetServerCookie() error = %v", err)
	}

	if exists {
		t.Error("GetServerCookie() exists = true for new server, want false")
	}

	if clientCookie == nil || len(clientCookie) != 8 {
		t.Errorf("GetServerCookie() clientCookie length = %d, want 8", len(clientCookie))
	}

	if serverCookie != nil {
		t.Error("GetServerCookie() serverCookie should be nil for new server")
	}
}

// TestGetServerCookieExisting 测试获取已存在服务器Cookie
func TestGetServerCookieExisting(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 先设置Cookie
	acm.SetServerCookie(serverAddr, clientCookie, serverCookie)

	// 再获取Cookie
	gotClientCookie, gotServerCookie, exists, err := acm.GetServerCookie(serverAddr)
	if err != nil {
		t.Fatalf("GetServerCookie() error = %v", err)
	}

	if !exists {
		t.Error("GetServerCookie() exists = false for existing server, want true")
	}

	if !bytesEqual(gotClientCookie, clientCookie) {
		t.Error("GetServerCookie() clientCookie mismatch")
	}

	if !bytesEqual(gotServerCookie, serverCookie) {
		t.Error("GetServerCookie() serverCookie mismatch")
	}
}

// TestGetServerCookieExpired 测试获取过期Cookie
func TestGetServerCookieExpired(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 创建过期条目
	entry := &CookieEntry{
		ClientCookie: make([]byte, 8),
		ServerCookie: make([]byte, 8),
		ExpiresAt:    time.Now().Add(-1 * time.Second), // 已过期
	}
	copy(entry.ClientCookie, clientCookie)
	copy(entry.ServerCookie, serverCookie)

	// 直接设置过期条目
	shard := acm.getShard(serverAddr)
	shard.mu.Lock()
	shard.entries[serverAddr] = entry
	shard.mu.Unlock()

	// 获取过期Cookie应该返回新的Client Cookie
	gotClientCookie, gotServerCookie, exists, err := acm.GetServerCookie(serverAddr)
	if err != nil {
		t.Fatalf("GetServerCookie() error = %v", err)
	}

	if exists {
		t.Error("GetServerCookie() exists = true for expired cookie, want false")
	}

	if gotClientCookie == nil || len(gotClientCookie) != 8 {
		t.Errorf("GetServerCookie() clientCookie length = %d, want 8", len(gotClientCookie))
	}

	if gotServerCookie != nil {
		t.Error("GetServerCookie() serverCookie should be nil for expired cookie")
	}
}

// TestSetServerCookie 测试设置服务器Cookie
func TestSetServerCookie(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	acm.SetServerCookie(serverAddr, clientCookie, serverCookie)

	// 验证Cookie已设置
	shard := acm.getShard(serverAddr)
	shard.mu.RLock()
	entry, exists := shard.entries[serverAddr]
	shard.mu.RUnlock()

	if !exists {
		t.Fatal("Cookie entry not found after SetServerCookie")
	}

	if !bytesEqual(entry.ClientCookie, clientCookie) {
		t.Error("ClientCookie mismatch")
	}

	if !bytesEqual(entry.ServerCookie, serverCookie) {
		t.Error("ServerCookie mismatch")
	}

	if time.Now().After(entry.ExpiresAt) {
		t.Error("ExpiresAt should be in the future")
	}
}

// TestRefreshServerCookie 测试刷新服务器Cookie
func TestRefreshServerCookie(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"
	oldClientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 先设置旧Cookie
	acm.SetServerCookie(serverAddr, oldClientCookie, serverCookie)

	// 刷新Cookie
	newClientCookie, err := acm.RefreshServerCookie(serverAddr)
	if err != nil {
		t.Fatalf("RefreshServerCookie() error = %v", err)
	}

	if newClientCookie == nil || len(newClientCookie) != 8 {
		t.Errorf("RefreshServerCookie() returned cookie length = %d, want 8", len(newClientCookie))
	}

	// 验证新Cookie与旧Cookie不同
	if bytesEqual(newClientCookie, oldClientCookie) {
		t.Error("RefreshServerCookie() returned same cookie, expected different")
	}

	// 验证条目已更新
	shard := acm.getShard(serverAddr)
	shard.mu.RLock()
	entry, exists := shard.entries[serverAddr]
	shard.mu.RUnlock()

	if !exists {
		t.Fatal("Cookie entry not found after RefreshServerCookie")
	}

	if entry.ServerCookie != nil {
		t.Error("ServerCookie should be nil after refresh")
	}
}

// TestIsRecentlyFailed 测试检查最近失效
func TestIsRecentlyFailed(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"

	// 初始状态应该返回false
	if acm.IsRecentlyFailed(serverAddr) {
		t.Error("IsRecentlyFailed() = true for new server, want false")
	}

	// 记录失效
	acm.RecordFailure(serverAddr)

	// 记录后应该返回true
	if !acm.IsRecentlyFailed(serverAddr) {
		t.Error("IsRecentlyFailed() = false after RecordFailure, want true")
	}
}

// TestIsRecentlyFailedExpired 测试失效记录过期
func TestIsRecentlyFailedExpired(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"

	// 手动设置过期失效记录
	acm.failureMu.Lock()
	acm.failures[serverAddr] = &CookieFailureRecord{
		ServerAddr: serverAddr,
		FailedAt:   time.Now().Add(-10 * time.Second), // 10秒前失效
	}
	acm.failureMu.Unlock()

	// 过期后应该返回false
	if acm.IsRecentlyFailed(serverAddr) {
		t.Error("IsRecentlyFailed() = true for expired failure record, want false")
	}
}

// TestRecordFailure 测试记录失效
func TestRecordFailure(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"

	acm.RecordFailure(serverAddr)

	acm.failureMu.RLock()
	record, exists := acm.failures[serverAddr]
	acm.failureMu.RUnlock()

	if !exists {
		t.Fatal("Failure record not found after RecordFailure")
	}

	if record.ServerAddr != serverAddr {
		t.Errorf("Record.ServerAddr = %s, want %s", record.ServerAddr, serverAddr)
	}

	if time.Since(record.FailedAt) > time.Second {
		t.Error("Record.FailedAt is too old")
	}
}

// TestGenerateClientCookie 测试生成Client Cookie
func TestGenerateClientCookie(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	// 生成多个Cookie，验证随机性
	cookies := make(map[string]bool)
	for i := 0; i < 100; i++ {
		cookie, err := acm.generateClientCookie()
		if err != nil {
			t.Fatalf("generateClientCookie() error = %v", err)
		}

		if len(cookie) != 8 {
			t.Errorf("generateClientCookie() length = %d, want 8", len(cookie))
		}

		// 检查是否重复
		key := string(cookie)
		if cookies[key] {
			t.Error("generateClientCookie() generated duplicate cookie")
		}
		cookies[key] = true

		// 归还内存
		acm.releaseClientCookie(cookie)
	}
}

// TestGetStats 测试获取统计信息
func TestGetStats(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	// 初始状态
	entries, failures := acm.GetStats()
	if entries != 0 {
		t.Errorf("GetStats() entries = %d, want 0", entries)
	}
	if failures != 0 {
		t.Errorf("GetStats() failures = %d, want 0", failures)
	}

	// 添加一些条目
	for i := 0; i < 5; i++ {
		addr := string(rune('A' + i))
		acm.SetServerCookie(addr+":53", []byte{1, 2, 3, 4, 5, 6, 7, 8}, []byte{9, 10, 11, 12, 13, 14, 15, 16})
	}

	// 记录一些失效
	for i := 0; i < 3; i++ {
		addr := string(rune('A' + i))
		acm.RecordFailure(addr + ":53")
	}

	entries, failures = acm.GetStats()
	if entries != 5 {
		t.Errorf("GetStats() entries = %d, want 5", entries)
	}
	if failures != 3 {
		t.Errorf("GetStats() failures = %d, want 3", failures)
	}
}

// TestRemoveServer 测试移除服务器
func TestRemoveServer(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 先设置Cookie
	acm.SetServerCookie(serverAddr, clientCookie, serverCookie)

	// 验证存在
	_, _, exists, _ := acm.GetServerCookie(serverAddr)
	if !exists {
		t.Fatal("Cookie should exist before removal")
	}

	// 移除服务器
	acm.RemoveServer(serverAddr)

	// 验证已移除
	_, _, exists, _ = acm.GetServerCookie(serverAddr)
	if exists {
		t.Error("Cookie should not exist after removal")
	}
}

// TestClear 测试清空所有Cookie
func TestClear(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	// 添加多个条目
	for i := 0; i < 10; i++ {
		addr := string(rune('A' + i))
		acm.SetServerCookie(addr+":53", []byte{1, 2, 3, 4, 5, 6, 7, 8}, []byte{9, 10, 11, 12, 13, 14, 15, 16})
	}

	// 验证条目存在
	entries, _ := acm.GetStats()
	if entries != 10 {
		t.Fatalf("Expected 10 entries before clear, got %d", entries)
	}

	// 清空
	acm.Clear()

	// 验证已清空
	entries, _ = acm.GetStats()
	if entries != 0 {
		t.Errorf("GetStats() entries = %d after Clear(), want 0", entries)
	}
}

// TestValidateServerCookie 测试验证Server Cookie
func TestValidateServerCookie(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 设置Cookie
	acm.SetServerCookie(serverAddr, clientCookie, serverCookie)

	// 验证正确的Cookie
	if !acm.ValidateServerCookie(serverAddr, clientCookie, serverCookie) {
		t.Error("ValidateServerCookie() = false for valid cookie, want true")
	}

	// 验证错误的Client Cookie
	wrongClientCookie := []byte{8, 7, 6, 5, 4, 3, 2, 1}
	if acm.ValidateServerCookie(serverAddr, wrongClientCookie, serverCookie) {
		t.Error("ValidateServerCookie() = true for wrong client cookie, want false")
	}

	// 验证错误的Server Cookie
	wrongServerCookie := []byte{16, 15, 14, 13, 12, 11, 10, 9}
	if acm.ValidateServerCookie(serverAddr, clientCookie, wrongServerCookie) {
		t.Error("ValidateServerCookie() = true for wrong server cookie, want false")
	}

	// 验证不存在的服务器
	if acm.ValidateServerCookie("192.168.1.2:53", clientCookie, serverCookie) {
		t.Error("ValidateServerCookie() = true for non-existent server, want false")
	}
}

// TestParseServerAddr 测试解析服务器地址
func TestParseServerAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		want    string
		wantErr bool
	}{
		{
			name:    "IPv4 with port",
			addr:    "192.168.1.1:53",
			want:    "192.168.1.1:53",
			wantErr: false,
		},
		{
			name:    "IPv6 with port",
			addr:    "[::1]:53",
			want:    "[::1]:53",
			wantErr: false,
		},
		{
			name:    "IPv6 without brackets",
			addr:    "::1:53",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Invalid address",
			addr:    "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Missing port",
			addr:    "192.168.1.1",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServerAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseServerAddr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseServerAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBytesEqual 测试字节切片比较
func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{
			name: "Equal slices",
			a:    []byte{1, 2, 3, 4},
			b:    []byte{1, 2, 3, 4},
			want: true,
		},
		{
			name: "Different content",
			a:    []byte{1, 2, 3, 4},
			b:    []byte{1, 2, 3, 5},
			want: false,
		},
		{
			name: "Different length",
			a:    []byte{1, 2, 3},
			b:    []byte{1, 2, 3, 4},
			want: false,
		},
		{
			name: "Empty slices",
			a:    []byte{},
			b:    []byte{},
			want: true,
		},
		{
			name: "Nil slices",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "One nil",
			a:    []byte{},
			b:    nil,
			want: true, // 两者长度都为0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("bytesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // 4种操作

	// 并发GetServerCookie
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				addr := string(rune('A' + (id % 26)))
				acm.GetServerCookie(addr + ":53")
			}
		}(i)
	}

	// 并发SetServerCookie
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				addr := string(rune('A' + (id % 26)))
				clientCookie := []byte{byte(id), byte(j), 3, 4, 5, 6, 7, 8}
				serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}
				acm.SetServerCookie(addr+":53", clientCookie, serverCookie)
			}
		}(i)
	}

	// 并发RecordFailure
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				addr := string(rune('A' + (id % 26)))
				acm.RecordFailure(addr + ":53")
			}
		}(i)
	}

	// 并发IsRecentlyFailed
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				addr := string(rune('A' + (id % 26)))
				acm.IsRecentlyFailed(addr + ":53")
			}
		}(i)
	}

	wg.Wait()

	// 验证没有panic，统计信息合理
	entries, failures := acm.GetStats()
	if entries < 0 || failures < 0 {
		t.Error("Stats should be non-negative")
	}
}

// TestCleanupExpiredFailures 测试清理过期失效记录
func TestCleanupExpiredFailures(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	serverAddr := "192.168.1.1:53"

	// 手动设置过期失效记录
	acm.failureMu.Lock()
	acm.failures[serverAddr] = &CookieFailureRecord{
		ServerAddr: serverAddr,
		FailedAt:   time.Now().Add(-10 * time.Second), // 10秒前失效
	}
	acm.failureMu.Unlock()

	// 清理过期记录
	acm.cleanupExpiredFailures()

	// 验证已清理
	acm.failureMu.RLock()
	_, exists := acm.failures[serverAddr]
	acm.failureMu.RUnlock()

	if exists {
		t.Error("Expired failure record should be cleaned up")
	}
}

// TestGetShard 测试获取分片
func TestGetShard(t *testing.T) {
	acm := NewAdaptiveCookieManager()
	defer acm.Stop()

	// 测试不同地址获取分片
	shards := make(map[*cookieShard]bool)
	for i := 0; i < 1000; i++ {
		addr := string(rune('A' + (i % 26)))
		shard := acm.getShard(addr + ":53")
		shards[shard] = true
	}

	// 验证分片分布
	if len(shards) < 2 {
		t.Error("Expected multiple shards to be used")
	}

	// 验证同一地址总是返回同一分片
	shard1 := acm.getShard("192.168.1.1:53")
	shard2 := acm.getShard("192.168.1.1:53")
	if shard1 != shard2 {
		t.Error("Same address should return same shard")
	}
}
