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
// core/database/statsdb_test.go
// 统计数据库操作测试

package database

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupStatsTestDB 创建测试用的内存数据库
func setupStatsTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	DB = db

	if err := DB.AutoMigrate(&QPSHistory{}, &ResourceHistory{}, &NetworkHistory{}); err != nil {
		t.Fatalf("迁移表失败: %v", err)
	}

	return func() {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
		DB = nil
	}
}

// TestQPSHistoryCRUD 测试QPS历史记录CRUD操作
func TestQPSHistoryCRUD(t *testing.T) {
	cleanup := setupStatsTestDB(t)
	defer cleanup()

	now := time.Now()
	records := []QPSHistory{
		{Timestamp: now.Add(-2 * time.Hour), QPS: 100.5},
		{Timestamp: now.Add(-1 * time.Hour), QPS: 150.2},
		{Timestamp: now, QPS: 200.8},
	}

	t.Run("批量保存QPS历史记录", func(t *testing.T) {
		err := SaveQPSHistoryBatch(records)
		if err != nil {
			t.Errorf("SaveQPSHistoryBatch() error = %v", err)
			return
		}

		count, err := GetQPSHistoryCount()
		if err != nil {
			t.Errorf("GetQPSHistoryCount() error = %v", err)
			return
		}
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("保存空记录", func(t *testing.T) {
		err := SaveQPSHistoryBatch([]QPSHistory{})
		if err != nil {
			t.Errorf("SaveQPSHistoryBatch([]) error = %v", err)
		}
	})

	t.Run("按时间范围查询", func(t *testing.T) {
		startTime := now.Add(-90 * time.Minute)
		endTime := now.Add(30 * time.Minute)

		results, err := GetQPSHistoryByTimeRange(startTime, endTime)
		if err != nil {
			t.Errorf("GetQPSHistoryByTimeRange() error = %v", err)
			return
		}
		if len(results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(results))
		}
	})

	t.Run("获取最近N条记录", func(t *testing.T) {
		results, err := GetLatestQPSHistory(2)
		if err != nil {
			t.Errorf("GetLatestQPSHistory() error = %v", err)
			return
		}
		if len(results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(results))
		}
		if results[0].QPS < results[1].QPS {
			t.Error("结果应该按时间降序排列")
		}
	})

	t.Run("获取统计信息", func(t *testing.T) {
		stats, err := GetQPSHistoryStats()
		if err != nil {
			t.Errorf("GetQPSHistoryStats() error = %v", err)
			return
		}

		count := stats["count"].(int64)
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("获取时间范围统计", func(t *testing.T) {
		startTime := now.Add(-3 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		stats, err := GetQPSHistoryStatsByRange(startTime, endTime)
		if err != nil {
			t.Errorf("GetQPSHistoryStatsByRange() error = %v", err)
			return
		}

		minQPS := stats["min"]
		maxQPS := stats["max"]
		avgQPS := stats["avg"]

		if minQPS != 100.5 {
			t.Errorf("min = %v, want 100.5", minQPS)
		}
		if maxQPS != 200.8 {
			t.Errorf("max = %v, want 200.8", maxQPS)
		}
		if avgQPS < 150 || avgQPS > 152 {
			t.Errorf("avg = %v, want around 150.5", avgQPS)
		}
	})

	t.Run("获取最小/最大/平均值", func(t *testing.T) {
		startTime := now.Add(-3 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		minQPS, err := GetQPSHistoryMinValue(startTime, endTime)
		if err != nil {
			t.Errorf("GetQPSHistoryMinValue() error = %v", err)
			return
		}
		if minQPS != 100.5 {
			t.Errorf("min = %v, want 100.5", minQPS)
		}

		maxQPS, err := GetQPSHistoryMaxValue(startTime, endTime)
		if err != nil {
			t.Errorf("GetQPSHistoryMaxValue() error = %v", err)
			return
		}
		if maxQPS != 200.8 {
			t.Errorf("max = %v, want 200.8", maxQPS)
		}

		avgQPS, err := GetQPSHistoryAvgValue(startTime, endTime)
		if err != nil {
			t.Errorf("GetQPSHistoryAvgValue() error = %v", err)
			return
		}
		if avgQPS < 150 || avgQPS > 152 {
			t.Errorf("avg = %v, want around 150.5", avgQPS)
		}
	})

	t.Run("清理过期记录", func(t *testing.T) {
		err := CleanOldQPSHistory(1)
		if err != nil {
			t.Errorf("CleanOldQPSHistory() error = %v", err)
			return
		}

		count, _ := GetQPSHistoryCount()
		if count == 0 {
			t.Error("清理后应该还有记录（1天内的）")
		}
	})

	t.Run("删除所有记录", func(t *testing.T) {
		err := DeleteAllQPSHistory()
		if err != nil {
			t.Errorf("DeleteAllQPSHistory() error = %v", err)
			return
		}

		count, _ := GetQPSHistoryCount()
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})
}

// TestQPSHistoryConversion 测试QPS历史记录转换
func TestQPSHistoryConversion(t *testing.T) {
	records := []QPSHistory{
		{Timestamp: time.Now(), QPS: 100.5},
		{Timestamp: time.Now().Add(time.Hour), QPS: 200.0},
	}

	points := ConvertToQPSHistoryPoints(records)
	if len(points) != 2 {
		t.Errorf("len(points) = %d, want 2", len(points))
	}
	if points[0].QPS != 100.5 {
		t.Errorf("points[0].QPS = %v, want 100.5", points[0].QPS)
	}

	converted := ConvertToQPSHistory(points)
	if len(converted) != 2 {
		t.Errorf("len(converted) = %d, want 2", len(converted))
	}
	if converted[0].QPS != 100.5 {
		t.Errorf("converted[0].QPS = %v, want 100.5", converted[0].QPS)
	}
}

// TestResourceHistoryCRUD 测试资源使用历史记录CRUD操作
func TestResourceHistoryCRUD(t *testing.T) {
	cleanup := setupStatsTestDB(t)
	defer cleanup()

	now := time.Now()
	records := []ResourceHistory{
		{Timestamp: now.Add(-2 * time.Hour), CPU: 30, Memory: 40, Disk: 50},
		{Timestamp: now.Add(-1 * time.Hour), CPU: 40, Memory: 50, Disk: 55},
		{Timestamp: now, CPU: 50, Memory: 60, Disk: 60},
	}

	t.Run("批量保存资源历史记录", func(t *testing.T) {
		err := SaveResourceHistoryBatch(records)
		if err != nil {
			t.Errorf("SaveResourceHistoryBatch() error = %v", err)
			return
		}

		count, err := GetResourceHistoryCount()
		if err != nil {
			t.Errorf("GetResourceHistoryCount() error = %v", err)
			return
		}
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("按时间范围查询", func(t *testing.T) {
		startTime := now.Add(-90 * time.Minute)
		endTime := now.Add(30 * time.Minute)

		results, err := GetResourceHistoryByTimeRange(startTime, endTime)
		if err != nil {
			t.Errorf("GetResourceHistoryByTimeRange() error = %v", err)
			return
		}
		if len(results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(results))
		}
	})

	t.Run("获取时间范围统计", func(t *testing.T) {
		startTime := now.Add(-3 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		stats, err := GetResourceHistoryStatsByRange(startTime, endTime)
		if err != nil {
			t.Errorf("GetResourceHistoryStatsByRange() error = %v", err)
			return
		}

		cpuStats := stats["cpu"].(map[string]interface{})
		if cpuStats["min"].(int) != 30 {
			t.Errorf("CPU min = %v, want 30", cpuStats["min"])
		}
		if cpuStats["max"].(int) != 50 {
			t.Errorf("CPU max = %v, want 50", cpuStats["max"])
		}

		memoryStats := stats["memory"].(map[string]interface{})
		if memoryStats["min"].(int) != 40 {
			t.Errorf("Memory min = %v, want 40", memoryStats["min"])
		}
		if memoryStats["max"].(int) != 60 {
			t.Errorf("Memory max = %v, want 60", memoryStats["max"])
		}
	})

	t.Run("清理过期记录", func(t *testing.T) {
		err := CleanOldResourceHistory(1)
		if err != nil {
			t.Errorf("CleanOldResourceHistory() error = %v", err)
		}
	})
}

// TestResourceHistoryConversion 测试资源历史记录转换
func TestResourceHistoryConversion(t *testing.T) {
	records := []ResourceHistory{
		{Timestamp: time.Now(), CPU: 30, Memory: 40, Disk: 50},
	}

	points := ConvertToResourceHistoryPoints(records)
	if len(points) != 1 {
		t.Errorf("len(points) = %d, want 1", len(points))
	}
	if points[0].CPU != 30 || points[0].Memory != 40 || points[0].Disk != 50 {
		t.Errorf("points[0] = %+v, want CPU=30, Memory=40, Disk=50", points[0])
	}

	converted := ConvertToResourceHistory(points)
	if len(converted) != 1 {
		t.Errorf("len(converted) = %d, want 1", len(converted))
	}
}

// TestNetworkHistoryCRUD 测试网络流量历史记录CRUD操作
func TestNetworkHistoryCRUD(t *testing.T) {
	cleanup := setupStatsTestDB(t)
	defer cleanup()

	now := time.Now()
	records := []NetworkHistory{
		{Timestamp: now.Add(-2 * time.Hour), InboundBps: 1000000, OutboundBps: 500000},
		{Timestamp: now.Add(-1 * time.Hour), InboundBps: 2000000, OutboundBps: 800000},
		{Timestamp: now, InboundBps: 3000000, OutboundBps: 1000000},
	}

	t.Run("批量保存网络历史记录", func(t *testing.T) {
		err := SaveNetworkHistoryBatch(records)
		if err != nil {
			t.Errorf("SaveNetworkHistoryBatch() error = %v", err)
			return
		}
	})

	t.Run("按时间范围查询", func(t *testing.T) {
		startTime := now.Add(-90 * time.Minute)
		endTime := now.Add(30 * time.Minute)

		results, err := GetNetworkHistoryByTimeRange(startTime, endTime)
		if err != nil {
			t.Errorf("GetNetworkHistoryByTimeRange() error = %v", err)
			return
		}
		if len(results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(results))
		}
	})

	t.Run("获取时间范围统计", func(t *testing.T) {
		startTime := now.Add(-3 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		stats, err := GetNetworkHistoryStatsByRange(startTime, endTime)
		if err != nil {
			t.Errorf("GetNetworkHistoryStatsByRange() error = %v", err)
			return
		}

		inboundStats := stats["inbound"].(map[string]interface{})
		if inboundStats["min"].(uint64) != 1000000 {
			t.Errorf("Inbound min = %v, want 1000000", inboundStats["min"])
		}
		if inboundStats["max"].(uint64) != 3000000 {
			t.Errorf("Inbound max = %v, want 3000000", inboundStats["max"])
		}

		outboundStats := stats["outbound"].(map[string]interface{})
		if outboundStats["min"].(uint64) != 500000 {
			t.Errorf("Outbound min = %v, want 500000", outboundStats["min"])
		}
		if outboundStats["max"].(uint64) != 1000000 {
			t.Errorf("Outbound max = %v, want 1000000", outboundStats["max"])
		}
	})

	t.Run("清理过期记录", func(t *testing.T) {
		err := CleanOldNetworkHistory(1)
		if err != nil {
			t.Errorf("CleanOldNetworkHistory() error = %v", err)
		}
	})
}

// TestNetworkHistoryConversion 测试网络历史记录转换
func TestNetworkHistoryConversion(t *testing.T) {
	records := []NetworkHistory{
		{Timestamp: time.Now(), InboundBps: 1000000, OutboundBps: 500000},
	}

	points := ConvertToNetworkHistoryPoints(records)
	if len(points) != 1 {
		t.Errorf("len(points) = %d, want 1", len(points))
	}
	if points[0].InboundBps != 1000000 || points[0].OutboundBps != 500000 {
		t.Errorf("points[0] = %+v, want InboundBps=1000000, OutboundBps=500000", points[0])
	}

	converted := ConvertToNetworkHistory(points)
	if len(converted) != 1 {
		t.Errorf("len(converted) = %d, want 1", len(converted))
	}
}

// TestEnsureTablesExist 测试确保表存在
func TestEnsureTablesExist(t *testing.T) {
	cleanup := setupStatsTestDB(t)
	defer cleanup()

	t.Run("QPS历史表已存在", func(t *testing.T) {
		err := EnsureQPSHistoryTableExists()
		if err != nil {
			t.Errorf("EnsureQPSHistoryTableExists() error = %v", err)
		}
	})

	t.Run("资源历史表已存在", func(t *testing.T) {
		err := EnsureResourceHistoryTableExists()
		if err != nil {
			t.Errorf("EnsureResourceHistoryTableExists() error = %v", err)
		}
	})

	t.Run("网络历史表已存在", func(t *testing.T) {
		err := EnsureNetworkHistoryTableExists()
		if err != nil {
			t.Errorf("EnsureNetworkHistoryTableExists() error = %v", err)
		}
	})
}

// TestQPSHistoryAggregated 测试聚合查询
func TestQPSHistoryAggregated(t *testing.T) {
	cleanup := setupStatsTestDB(t)
	defer cleanup()

	now := time.Now()
	var records []QPSHistory
	for i := 0; i < 10; i++ {
		records = append(records, QPSHistory{
			Timestamp: now.Add(time.Duration(i*10) * time.Minute),
			QPS:       float64(100 + i*10),
		})
	}

	err := SaveQPSHistoryBatch(records)
	if err != nil {
		t.Fatalf("SaveQPSHistoryBatch() error = %v", err)
	}

	startTime := now.Add(-5 * time.Minute)
	endTime := now.Add(105 * time.Minute)

	results, err := GetQPSHistoryAggregated(startTime, endTime, 30)
	if err != nil {
		t.Errorf("GetQPSHistoryAggregated() error = %v", err)
		return
	}

	if len(results) == 0 {
		t.Error("聚合查询应该返回结果")
	}

	for _, r := range results {
		if r.Count == 0 {
			t.Error("聚合结果的Count应该大于0")
		}
	}
}
