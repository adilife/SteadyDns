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
// core/sdns/tcp_pool_test.go
// TCP连接池单元测试

package sdns

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// TestDefaultPoolConfig 测试默认连接池配置
func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	if config.MaxConnectionsPerServer != 3 {
		t.Errorf("MaxConnectionsPerServer = %d, want 3", config.MaxConnectionsPerServer)
	}

	if config.MaxPipelineDepth != 100 {
		t.Errorf("MaxPipelineDepth = %d, want 100", config.MaxPipelineDepth)
	}

	if config.IdleTimeout != 30*time.Second {
		t.Errorf("IdleTimeout = %v, want 30s", config.IdleTimeout)
	}

	if config.ConnectTimeout != 5*time.Second {
		t.Errorf("ConnectTimeout = %v, want 5s", config.ConnectTimeout)
	}

	if config.MaxConnectionLifetime != 10*time.Minute {
		t.Errorf("MaxConnectionLifetime = %v, want 10m", config.MaxConnectionLifetime)
	}

	if config.HealthCheckInterval != 30*time.Second {
		t.Errorf("HealthCheckInterval = %v, want 30s", config.HealthCheckInterval)
	}

	if config.OutOfOrderThreshold != 10.0 {
		t.Errorf("OutOfOrderThreshold = %f, want 10.0", config.OutOfOrderThreshold)
	}
}

// TestNewTCPConnectionPool 测试创建TCP连接池
func TestNewTCPConnectionPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewTCPConnectionPool(config)

	if pool == nil {
		t.Fatal("NewTCPConnectionPool() returned nil")
	}

	if pool.config != config {
		t.Error("Pool config mismatch")
	}

	if pool.pools == nil {
		t.Error("Pools map is nil")
	}

	if pool.stopCleanup == nil {
		t.Error("stopCleanup channel is nil")
	}

	if pool.stopHealthCheck == nil {
		t.Error("stopHealthCheck channel is nil")
	}

	// 清理
	pool.Close()
}

// TestNewTCPConnectionPoolNilConfig 测试使用nil配置创建连接池
func TestNewTCPConnectionPoolNilConfig(t *testing.T) {
	pool := NewTCPConnectionPool(nil)

	if pool == nil {
		t.Fatal("NewTCPConnectionPool(nil) returned nil")
	}

	if pool.config == nil {
		t.Error("Pool config is nil when using nil config")
	}

	// 验证使用了默认配置
	if pool.config.MaxConnectionsPerServer != 3 {
		t.Error("Pool did not use default config")
	}

	pool.Close()
}

// TestConnectionHealthString 测试连接健康状态字符串表示
func TestConnectionHealthString(t *testing.T) {
	tests := []struct {
		health ConnectionHealth
		want   string
	}{
		{ConnectionHealthHealthy, "healthy"},
		{ConnectionHealthDegraded, "degraded"},
		{ConnectionHealthUnhealthy, "unhealthy"},
		{ConnectionHealth(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.health.String()
		if got != tt.want {
			t.Errorf("ConnectionHealth(%d).String() = %s, want %s", tt.health, got, tt.want)
		}
	}
}

// TestNewPipelineStats 测试创建管道化统计信息
func TestNewPipelineStats(t *testing.T) {
	stats := NewPipelineStats()

	if stats == nil {
		t.Fatal("NewPipelineStats() returned nil")
	}

	if stats.PipelineDepthHistory == nil {
		t.Error("PipelineDepthHistory is nil")
	}

	if stats.TotalQueries != 0 {
		t.Errorf("TotalQueries = %d, want 0", stats.TotalQueries)
	}

	if stats.OutOfOrderResponses != 0 {
		t.Errorf("OutOfOrderResponses = %d, want 0", stats.OutOfOrderResponses)
	}
}

// TestPipelineStatsRecordQuery 测试记录查询
func TestPipelineStatsRecordQuery(t *testing.T) {
	stats := NewPipelineStats()

	for i := 0; i < 100; i++ {
		stats.RecordQuery()
	}

	if stats.TotalQueries != 100 {
		t.Errorf("TotalQueries = %d, want 100", stats.TotalQueries)
	}
}

// TestPipelineStatsRecordOutOfOrder 测试记录乱序响应
func TestPipelineStatsRecordOutOfOrder(t *testing.T) {
	stats := NewPipelineStats()

	for i := 0; i < 50; i++ {
		stats.RecordOutOfOrder()
	}

	if stats.OutOfOrderResponses != 50 {
		t.Errorf("OutOfOrderResponses = %d, want 50", stats.OutOfOrderResponses)
	}
}

// TestPipelineStatsGetOutOfOrderRate 测试获取乱序率
func TestPipelineStatsGetOutOfOrderRate(t *testing.T) {
	stats := NewPipelineStats()

	// 初始状态
	rate := stats.GetOutOfOrderRate()
	if rate != 0.0 {
		t.Errorf("GetOutOfOrderRate() = %f, want 0.0", rate)
	}

	// 记录100个查询，10个乱序
	for i := 0; i < 100; i++ {
		stats.RecordQuery()
	}
	for i := 0; i < 10; i++ {
		stats.RecordOutOfOrder()
	}

	rate = stats.GetOutOfOrderRate()
	expectedRate := 10.0 // 10%
	if rate != expectedRate {
		t.Errorf("GetOutOfOrderRate() = %f, want %f", rate, expectedRate)
	}
}

// TestPipelineStatsUpdateExpectedMsgID 测试更新期望消息ID
func TestPipelineStatsUpdateExpectedMsgID(t *testing.T) {
	stats := NewPipelineStats()

	stats.UpdateExpectedMsgID(100)

	if stats.GetExpectedMsgID() != 100 {
		t.Errorf("GetExpectedMsgID() = %d, want 100", stats.GetExpectedMsgID())
	}
}

// TestPipelineStatsRecordPipelineDepth 测试记录管道深度
func TestPipelineStatsRecordPipelineDepth(t *testing.T) {
	stats := NewPipelineStats()

	// 记录一些深度值
	for i := 0; i < 50; i++ {
		stats.RecordPipelineDepth(int32(i))
	}

	avg := stats.GetAveragePipelineDepth()
	expectedAvg := 24.5 // (0+49)/2
	if avg != expectedAvg {
		t.Errorf("GetAveragePipelineDepth() = %f, want %f", avg, expectedAvg)
	}
}

// TestPipelineStatsRecordPipelineDepthLimit 测试记录管道深度限制
func TestPipelineStatsRecordPipelineDepthLimit(t *testing.T) {
	stats := NewPipelineStats()

	// 记录超过100个深度值
	for i := 0; i < 150; i++ {
		stats.RecordPipelineDepth(int32(i))
	}

	// 应该只保留最近100个
	avg := stats.GetAveragePipelineDepth()
	// 最近100个是50-149，平均值是(50+149)/2 = 99.5
	expectedAvg := 99.5
	if avg != expectedAvg {
		t.Errorf("GetAveragePipelineDepth() = %f, want %f", avg, expectedAvg)
	}
}

// TestPipelineStatsCanAdjust 测试检查是否可以调整
func TestPipelineStatsCanAdjust(t *testing.T) {
	stats := NewPipelineStats()

	// 初始状态不应该可以调整（因为lastAdjustTime被初始化为time.Now()）
	if stats.CanAdjust() {
		t.Error("CanAdjust() = true initially, want false (cooldown period)")
	}

	// 标记调整后
	stats.MarkAdjusted()

	// 立即检查应该返回false
	if stats.CanAdjust() {
		t.Error("CanAdjust() = true immediately after MarkAdjusted, want false")
	}
}

// TestPooledConnectionHealth 测试连接健康状态
func TestPooledConnectionHealth(t *testing.T) {
	conn := &PooledConnection{
		health: ConnectionHealthHealthy,
	}

	if conn.GetHealth() != ConnectionHealthHealthy {
		t.Error("GetHealth() mismatch")
	}

	conn.SetHealth(ConnectionHealthDegraded)
	if conn.GetHealth() != ConnectionHealthDegraded {
		t.Error("SetHealth() did not work")
	}
}

// TestPooledConnectionIsHealthy 测试检查连接是否健康
func TestPooledConnectionIsHealthy(t *testing.T) {
	tests := []struct {
		health ConnectionHealth
		want   bool
	}{
		{ConnectionHealthHealthy, true},
		{ConnectionHealthDegraded, true},
		{ConnectionHealthUnhealthy, false},
	}

	for _, tt := range tests {
		conn := &PooledConnection{
			health: tt.health,
		}
		if got := conn.IsHealthy(); got != tt.want {
			t.Errorf("IsHealthy() with health=%d = %v, want %v", tt.health, got, tt.want)
		}
	}
}

// TestPooledConnectionIsExpired 测试检查连接是否过期
func TestPooledConnectionIsExpired(t *testing.T) {
	// 新连接不应该过期
	conn := &PooledConnection{
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
	}

	if conn.IsExpired() {
		t.Error("New connection should not be expired")
	}

	// 空闲时间过长的连接应该过期
	conn.lastUsedAt = time.Now().Add(-35 * time.Second)
	if !conn.IsExpired() {
		t.Error("Idle connection should be expired")
	}

	// 寿命过长的连接应该过期
	conn = &PooledConnection{
		createdAt:  time.Now().Add(-11 * time.Minute),
		lastUsedAt: time.Now(),
	}
	if !conn.IsExpired() {
		t.Error("Old connection should be expired")
	}
}

// TestPooledConnectionUpdateLastUsed 测试更新最后使用时间
func TestPooledConnectionUpdateLastUsed(t *testing.T) {
	conn := &PooledConnection{
		lastUsedAt: time.Now().Add(-1 * time.Hour),
	}

	conn.UpdateLastUsed()

	if time.Since(conn.lastUsedAt) > time.Second {
		t.Error("UpdateLastUsed() did not update lastUsedAt")
	}
}

// TestTCPConnectionPoolClose 测试关闭连接池
func TestTCPConnectionPoolClose(t *testing.T) {
	pool := NewTCPConnectionPool(nil)

	// 关闭应该成功
	err := pool.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 重复关闭应该返回nil
	err = pool.Close()
	if err != nil {
		t.Errorf("Close() second time error = %v", err)
	}
}

// TestTCPConnectionPoolGetStats 测试获取连接池统计信息
func TestTCPConnectionPoolGetStats(t *testing.T) {
	pool := NewTCPConnectionPool(nil)
	defer pool.Close()

	stats := pool.GetStats()

	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}

	// 初始状态应该有0个连接
	totalConns, ok := stats["total_connections"].(int)
	if !ok {
		t.Error("total_connections not found or wrong type")
	}
	if totalConns != 0 {
		t.Errorf("total_connections = %d, want 0", totalConns)
	}
}

// TestTCPConnectionPoolGetConnectionStats 测试获取连接统计信息
func TestTCPConnectionPoolGetConnectionStats(t *testing.T) {
	pool := NewTCPConnectionPool(nil)
	defer pool.Close()

	// 不存在的服务器应该返回false
	_, exists := pool.GetConnectionStats("192.168.1.1:53")
	if exists {
		t.Error("GetConnectionStats() = true for non-existent server, want false")
	}
}

// TestTCPConnectionPoolGetConnectionClosedPool 测试从已关闭的连接池获取连接
func TestTCPConnectionPoolGetConnectionClosedPool(t *testing.T) {
	pool := NewTCPConnectionPool(nil)
	pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := pool.GetConnection("192.168.1.1:53", ctx)
	if err == nil {
		t.Error("GetConnection() from closed pool should return error")
	}
}

// TestPooledConnectionClose 测试关闭连接
func TestPooledConnectionClose(t *testing.T) {
	conn := &PooledConnection{
		inflight: make(map[uint16]*InflightQuery),
		stopRead: make(chan struct{}),
		closed:   0,
	}

	err := conn.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 重复关闭应该返回nil
	err = conn.Close()
	if err != nil {
		t.Errorf("Close() second time error = %v", err)
	}
}

// TestPipelineStatsConcurrency 测试管道统计并发安全
func TestPipelineStatsConcurrency(t *testing.T) {
	stats := NewPipelineStats()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// 并发记录查询
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				stats.RecordQuery()
			}
		}()
	}

	// 并发记录乱序
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				stats.RecordOutOfOrder()
			}
		}()
	}

	// 并发读取乱序率
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = stats.GetOutOfOrderRate()
			}
		}()
	}

	wg.Wait()

	// 验证计数正确
	expectedQueries := uint64(numGoroutines * numOperations)
	if stats.TotalQueries != expectedQueries {
		t.Errorf("TotalQueries = %d, want %d", stats.TotalQueries, expectedQueries)
	}

	expectedOutOfOrder := uint64(numGoroutines * numOperations)
	if stats.OutOfOrderResponses != expectedOutOfOrder {
		t.Errorf("OutOfOrderResponses = %d, want %d", stats.OutOfOrderResponses, expectedOutOfOrder)
	}
}

// TestInflightQueryStructure 测试InflightQuery结构体
func TestInflightQueryStructure(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	query := &InflightQuery{
		Msg:      msg,
		Response: make(chan *dns.Msg, 1),
		Error:    make(chan error, 1),
		SentAt:   time.Now(),
		MsgID:    12345,
	}

	if query.MsgID != 12345 {
		t.Errorf("MsgID = %d, want 12345", query.MsgID)
	}

	if query.Msg == nil {
		t.Error("Msg is nil")
	}

	if query.Response == nil {
		t.Error("Response channel is nil")
	}

	if query.Error == nil {
		t.Error("Error channel is nil")
	}
}

// TestPooledConnectionStructure 测试PooledConnection结构体
func TestPooledConnectionStructure(t *testing.T) {
	conn := &PooledConnection{
		inflight:         make(map[uint16]*InflightQuery),
		health:           ConnectionHealthHealthy,
		createdAt:        time.Now(),
		lastUsedAt:       time.Now(),
		serverAddr:       "192.168.1.1:53",
		maxPipelineDepth: 100,
		stats:            NewPipelineStats(),
		stopRead:         make(chan struct{}),
	}

	if conn.serverAddr != "192.168.1.1:53" {
		t.Errorf("serverAddr = %s, want 192.168.1.1:53", conn.serverAddr)
	}

	if conn.maxPipelineDepth != 100 {
		t.Errorf("maxPipelineDepth = %d, want 100", conn.maxPipelineDepth)
	}

	if conn.stats == nil {
		t.Error("stats is nil")
	}

	if conn.inflight == nil {
		t.Error("inflight map is nil")
	}
}

// TestServerPoolStructure 测试ServerPool结构体
func TestServerPoolStructure(t *testing.T) {
	serverPool := &ServerPool{
		connections: make([]*PooledConnection, 0),
		serverAddr:  "192.168.1.1:53",
		nextIndex:   0,
	}

	if serverPool.serverAddr != "192.168.1.1:53" {
		t.Errorf("serverAddr = %s, want 192.168.1.1:53", serverPool.serverAddr)
	}

	if serverPool.connections == nil {
		t.Error("connections slice is nil")
	}
}

// TestTCPConnectionPoolConcurrentAccess 测试连接池并发访问
func TestTCPConnectionPoolConcurrentAccess(t *testing.T) {
	pool := NewTCPConnectionPool(nil)
	defer pool.Close()

	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 并发获取统计信息
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = pool.GetStats()
		}()
	}

	wg.Wait()
}

// TestPoolConfigStructure 测试PoolConfig结构体
func TestPoolConfigStructure(t *testing.T) {
	config := &PoolConfig{
		MaxConnectionsPerServer: 5,
		MaxPipelineDepth:        200,
		IdleTimeout:             60 * time.Second,
		ConnectTimeout:          10 * time.Second,
		MaxConnectionLifetime:   20 * time.Minute,
		HealthCheckInterval:     60 * time.Second,
		OutOfOrderThreshold:     20.0,
	}

	if config.MaxConnectionsPerServer != 5 {
		t.Errorf("MaxConnectionsPerServer = %d, want 5", config.MaxConnectionsPerServer)
	}

	if config.MaxPipelineDepth != 200 {
		t.Errorf("MaxPipelineDepth = %d, want 200", config.MaxPipelineDepth)
	}

	if config.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", config.IdleTimeout)
	}

	if config.OutOfOrderThreshold != 20.0 {
		t.Errorf("OutOfOrderThreshold = %f, want 20.0", config.OutOfOrderThreshold)
	}
}
