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
// core/database/statsdb.go

package database

import (
	"fmt"
	"time"
)

// QPSHistory QPS历史记录表
type QPSHistory struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Timestamp time.Time `json:"timestamp" gorm:"index;not null"`
	QPS       float64   `json:"qps" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (QPSHistory) TableName() string {
	return "qps_history"
}

// SaveQPSHistoryBatch 批量保存QPS历史记录
func SaveQPSHistoryBatch(records []QPSHistory) error {
	if len(records) == 0 {
		return nil
	}

	if err := DB.CreateInBatches(records, 100).Error; err != nil {
		return fmt.Errorf("批量保存QPS历史记录失败: %v", err)
	}

	return nil
}

// GetQPSHistoryByTimeRange 根据时间范围获取QPS历史记录
func GetQPSHistoryByTimeRange(startTime, endTime time.Time) ([]QPSHistory, error) {
	var records []QPSHistory

	err := DB.Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Order("timestamp ASC").
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("查询QPS历史记录失败: %v", err)
	}

	return records, nil
}

// CleanOldQPSHistory 清理过期的QPS历史记录
func CleanOldQPSHistory(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result := DB.Where("timestamp < ?", cutoff).Delete(&QPSHistory{})
	if result.Error != nil {
		return fmt.Errorf("清理QPS历史记录失败: %v", result.Error)
	}

	if result.RowsAffected > 0 {
		GetLogManager().logger.Info("清理了 %d 条过期的QPS历史记录", result.RowsAffected)
	}

	return nil
}

// GetQPSHistoryCount 获取QPS历史记录总数
func GetQPSHistoryCount() (int64, error) {
	var count int64
	if err := DB.Model(&QPSHistory{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取QPS历史记录数量失败: %v", err)
	}
	return count, nil
}

// GetLatestQPSHistory 获取最近N条QPS历史记录
func GetLatestQPSHistory(limit int) ([]QPSHistory, error) {
	var records []QPSHistory

	err := DB.Order("timestamp DESC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("获取最新QPS历史记录失败: %v", err)
	}

	return records, nil
}

// GetQPSHistoryStats 获取QPS历史记录统计信息
func GetQPSHistoryStats() (map[string]interface{}, error) {
	var count int64
	var oldestTime, newestTime time.Time

	if err := DB.Model(&QPSHistory{}).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("获取QPS历史记录统计失败: %v", err)
	}

	if count > 0 {
		var oldest QPSHistory
		if err := DB.Order("timestamp ASC").First(&oldest).Error; err == nil {
			oldestTime = oldest.Timestamp
		}

		var newest QPSHistory
		if err := DB.Order("timestamp DESC").First(&newest).Error; err == nil {
			newestTime = newest.Timestamp
		}
	}

	return map[string]interface{}{
		"count":      count,
		"oldestTime": oldestTime,
		"newestTime": newestTime,
	}, nil
}

// DeleteAllQPSHistory 删除所有QPS历史记录
func DeleteAllQPSHistory() error {
	if err := DB.Where("1 = 1").Delete(&QPSHistory{}).Error; err != nil {
		return fmt.Errorf("删除所有QPS历史记录失败: %v", err)
	}
	return nil
}

// QPSHistoryAgg QPS历史聚合结果
type QPSHistoryAgg struct {
	TimeBucket time.Time
	AvgQPS     float64
	Count      int64
}

// GetQPSHistoryAggregated 获取聚合后的QPS历史数据
func GetQPSHistoryAggregated(startTime, endTime time.Time, intervalMinutes int) ([]QPSHistoryAgg, error) {
	var results []QPSHistoryAgg

	query := `
		SELECT 
			datetime(strftime('%s', timestamp) / (? * 60) * (? * 60), 'unixepoch') as time_bucket,
			AVG(qps) as avg_qps,
			COUNT(*) as count
		FROM qps_history
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	rows, err := DB.Raw(query, intervalMinutes, intervalMinutes, startTime, endTime).Rows()
	if err != nil {
		return nil, fmt.Errorf("聚合查询QPS历史记录失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agg QPSHistoryAgg
		var timeStr string
		if err := rows.Scan(&timeStr, &agg.AvgQPS, &agg.Count); err != nil {
			continue
		}
		parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
		if err == nil {
			agg.TimeBucket = parsedTime
			results = append(results, agg)
		}
	}

	return results, nil
}

// EnsureQPSHistoryTableExists 确保QPS历史表存在
func EnsureQPSHistoryTableExists() error {
	if !DB.Migrator().HasTable(&QPSHistory{}) {
		if err := DB.AutoMigrate(&QPSHistory{}); err != nil {
			return fmt.Errorf("创建QPS历史表失败: %v", err)
		}
		GetLogManager().logger.Info("QPS历史表创建成功")
	}
	return nil
}

// QPSHistoryPoint 用于内存和API传输的数据点
type QPSHistoryPoint struct {
	Time time.Time
	QPS  float64
}

// ConvertToQPSHistoryPoints 将数据库记录转换为内存数据点格式
func ConvertToQPSHistoryPoints(records []QPSHistory) []QPSHistoryPoint {
	points := make([]QPSHistoryPoint, len(records))
	for i, r := range records {
		points[i] = QPSHistoryPoint{
			Time: r.Timestamp,
			QPS:  r.QPS,
		}
	}
	return points
}

// ConvertToQPSHistory 将内存数据点转换为数据库记录格式
func ConvertToQPSHistory(points []QPSHistoryPoint) []QPSHistory {
	records := make([]QPSHistory, len(points))
	for i, p := range points {
		records[i] = QPSHistory{
			Timestamp: p.Time,
			QPS:       p.QPS,
		}
	}
	return records
}

// GetQPSHistoryMinValue 获取指定时间范围内的最小QPS值
func GetQPSHistoryMinValue(startTime, endTime time.Time) (float64, error) {
	var minQPS float64
	err := DB.Model(&QPSHistory{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Select("MIN(qps)").
		Scan(&minQPS).Error

	if err != nil {
		return 0, fmt.Errorf("获取最小QPS值失败: %v", err)
	}
	return minQPS, nil
}

// GetQPSHistoryMaxValue 获取指定时间范围内的最大QPS值
func GetQPSHistoryMaxValue(startTime, endTime time.Time) (float64, error) {
	var maxQPS float64
	err := DB.Model(&QPSHistory{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Select("MAX(qps)").
		Scan(&maxQPS).Error

	if err != nil {
		return 0, fmt.Errorf("获取最大QPS值失败: %v", err)
	}
	return maxQPS, nil
}

// GetQPSHistoryAvgValue 获取指定时间范围内的平均QPS值
func GetQPSHistoryAvgValue(startTime, endTime time.Time) (float64, error) {
	var avgQPS float64
	err := DB.Model(&QPSHistory{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Select("AVG(qps)").
		Scan(&avgQPS).Error

	if err != nil {
		return 0, fmt.Errorf("获取平均QPS值失败: %v", err)
	}
	return avgQPS, nil
}

// GetQPSHistoryStatsByRange 获取指定时间范围内的QPS统计信息
func GetQPSHistoryStatsByRange(startTime, endTime time.Time) (map[string]float64, error) {
	stats := make(map[string]float64)

	var result struct {
		Min   float64
		Max   float64
		Avg   float64
		Count int64
	}

	err := DB.Model(&QPSHistory{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Select("MIN(qps) as min, MAX(qps) as max, AVG(qps) as avg, COUNT(*) as count").
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("获取QPS统计信息失败: %v", err)
	}

	stats["min"] = result.Min
	stats["max"] = result.Max
	stats["avg"] = result.Avg
	stats["count"] = float64(result.Count)

	return stats, nil
}

// ResourceHistory 资源使用历史记录表
type ResourceHistory struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Timestamp time.Time `json:"timestamp" gorm:"index;not null"`
	CPU       int       `json:"cpu" gorm:"not null"`
	Memory    int       `json:"memory" gorm:"not null"`
	Disk      int       `json:"disk" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (ResourceHistory) TableName() string {
	return "resource_history"
}

// SaveResourceHistoryBatch 批量保存资源使用历史记录
func SaveResourceHistoryBatch(records []ResourceHistory) error {
	if len(records) == 0 {
		return nil
	}

	if err := DB.CreateInBatches(records, 100).Error; err != nil {
		return fmt.Errorf("批量保存资源使用历史记录失败: %v", err)
	}

	return nil
}

// GetResourceHistoryByTimeRange 根据时间范围获取资源使用历史记录
func GetResourceHistoryByTimeRange(startTime, endTime time.Time) ([]ResourceHistory, error) {
	var records []ResourceHistory

	err := DB.Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Order("timestamp ASC").
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("查询资源使用历史记录失败: %v", err)
	}

	return records, nil
}

// CleanOldResourceHistory 清理过期的资源使用历史记录
func CleanOldResourceHistory(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result := DB.Where("timestamp < ?", cutoff).Delete(&ResourceHistory{})
	if result.Error != nil {
		return fmt.Errorf("清理资源使用历史记录失败: %v", result.Error)
	}

	if result.RowsAffected > 0 {
		GetLogManager().logger.Info("清理了 %d 条过期的资源使用历史记录", result.RowsAffected)
	}

	return nil
}

// GetResourceHistoryCount 获取资源使用历史记录总数
func GetResourceHistoryCount() (int64, error) {
	var count int64
	if err := DB.Model(&ResourceHistory{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取资源使用历史记录数量失败: %v", err)
	}
	return count, nil
}

// ResourceHistoryPoint 用于内存和API传输的资源数据点
type ResourceHistoryPoint struct {
	Time   time.Time
	CPU    int
	Memory int
	Disk   int
}

// ConvertToResourceHistoryPoints 将数据库记录转换为内存数据点格式
func ConvertToResourceHistoryPoints(records []ResourceHistory) []ResourceHistoryPoint {
	points := make([]ResourceHistoryPoint, len(records))
	for i, r := range records {
		points[i] = ResourceHistoryPoint{
			Time:   r.Timestamp,
			CPU:    r.CPU,
			Memory: r.Memory,
			Disk:   r.Disk,
		}
	}
	return points
}

// ConvertToResourceHistory 将内存数据点转换为数据库记录格式
func ConvertToResourceHistory(points []ResourceHistoryPoint) []ResourceHistory {
	records := make([]ResourceHistory, len(points))
	for i, p := range points {
		records[i] = ResourceHistory{
			Timestamp: p.Time,
			CPU:       p.CPU,
			Memory:    p.Memory,
			Disk:      p.Disk,
		}
	}
	return records
}

// GetResourceHistoryStatsByRange 获取指定时间范围内的资源使用统计信息
func GetResourceHistoryStatsByRange(startTime, endTime time.Time) (map[string]interface{}, error) {
	var result struct {
		MinCPU    int
		MaxCPU    int
		AvgCPU    float64
		MinMemory int
		MaxMemory int
		AvgMemory float64
		MinDisk   int
		MaxDisk   int
		AvgDisk   float64
		Count     int64
	}

	err := DB.Model(&ResourceHistory{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Select("MIN(cpu) as min_cpu, MAX(cpu) as max_cpu, AVG(cpu) as avg_cpu, " +
			"MIN(memory) as min_memory, MAX(memory) as max_memory, AVG(memory) as avg_memory, " +
			"MIN(disk) as min_disk, MAX(disk) as max_disk, AVG(disk) as avg_disk, " +
			"COUNT(*) as count").
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("获取资源使用统计信息失败: %v", err)
	}

	return map[string]interface{}{
		"cpu": map[string]interface{}{
			"min": result.MinCPU,
			"max": result.MaxCPU,
			"avg": result.AvgCPU,
		},
		"memory": map[string]interface{}{
			"min": result.MinMemory,
			"max": result.MaxMemory,
			"avg": result.AvgMemory,
		},
		"disk": map[string]interface{}{
			"min": result.MinDisk,
			"max": result.MaxDisk,
			"avg": result.AvgDisk,
		},
		"count": result.Count,
	}, nil
}

// EnsureResourceHistoryTableExists 确保资源使用历史表存在
func EnsureResourceHistoryTableExists() error {
	if !DB.Migrator().HasTable(&ResourceHistory{}) {
		if err := DB.AutoMigrate(&ResourceHistory{}); err != nil {
			return fmt.Errorf("创建资源使用历史表失败: %v", err)
		}
		GetLogManager().logger.Info("资源使用历史表创建成功")
	}
	return nil
}

// NetworkHistory 网络流量历史记录表
type NetworkHistory struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Timestamp   time.Time `json:"timestamp" gorm:"index;not null"`
	InboundBps  uint64    `json:"inboundBps" gorm:"not null"`
	OutboundBps uint64    `json:"outboundBps" gorm:"not null"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (NetworkHistory) TableName() string {
	return "network_history"
}

// SaveNetworkHistoryBatch 批量保存网络流量历史记录
// 参数：
//   - records: 网络流量历史记录数组
// 返回：
//   - error: 保存失败时返回错误
func SaveNetworkHistoryBatch(records []NetworkHistory) error {
	if len(records) == 0 {
		return nil
	}

	if err := DB.CreateInBatches(records, 100).Error; err != nil {
		return fmt.Errorf("批量保存网络流量历史记录失败: %v", err)
	}

	return nil
}

// GetNetworkHistoryByTimeRange 根据时间范围获取网络流量历史记录
// 参数：
//   - startTime: 开始时间
//   - endTime: 结束时间
// 返回：
//   - []NetworkHistory: 网络流量历史记录数组
//   - error: 查询失败时返回错误
func GetNetworkHistoryByTimeRange(startTime, endTime time.Time) ([]NetworkHistory, error) {
	var records []NetworkHistory

	err := DB.Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Order("timestamp ASC").
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("查询网络流量历史记录失败: %v", err)
	}

	return records, nil
}

// CleanOldNetworkHistory 清理过期的网络流量历史记录
// 参数：
//   - retentionDays: 数据保留天数
// 返回：
//   - error: 清理失败时返回错误
func CleanOldNetworkHistory(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result := DB.Where("timestamp < ?", cutoff).Delete(&NetworkHistory{})
	if result.Error != nil {
		return fmt.Errorf("清理网络流量历史记录失败: %v", result.Error)
	}

	if result.RowsAffected > 0 {
		GetLogManager().logger.Info("清理了 %d 条过期的网络流量历史记录", result.RowsAffected)
	}

	return nil
}

// EnsureNetworkHistoryTableExists 确保网络流量历史表存在
// 返回：
//   - error: 创建表失败时返回错误
func EnsureNetworkHistoryTableExists() error {
	if !DB.Migrator().HasTable(&NetworkHistory{}) {
		if err := DB.AutoMigrate(&NetworkHistory{}); err != nil {
			return fmt.Errorf("创建网络流量历史表失败: %v", err)
		}
		GetLogManager().logger.Info("网络流量历史表创建成功")
	}
	return nil
}

// NetworkHistoryPoint 用于内存和API传输的网络流量数据点
type NetworkHistoryPoint struct {
	Time        time.Time
	InboundBps  uint64
	OutboundBps uint64
}

// ConvertToNetworkHistoryPoints 将数据库记录转换为内存数据点格式
// 参数：
//   - records: 数据库记录数组
// 返回：
//   - []NetworkHistoryPoint: 内存数据点数组
func ConvertToNetworkHistoryPoints(records []NetworkHistory) []NetworkHistoryPoint {
	points := make([]NetworkHistoryPoint, len(records))
	for i, r := range records {
		points[i] = NetworkHistoryPoint{
			Time:        r.Timestamp,
			InboundBps:  r.InboundBps,
			OutboundBps: r.OutboundBps,
		}
	}
	return points
}

// ConvertToNetworkHistory 将内存数据点转换为数据库记录格式
// 参数：
//   - points: 内存数据点数组
// 返回：
//   - []NetworkHistory: 数据库记录数组
func ConvertToNetworkHistory(points []NetworkHistoryPoint) []NetworkHistory {
	records := make([]NetworkHistory, len(points))
	for i, p := range points {
		records[i] = NetworkHistory{
			Timestamp:   p.Time,
			InboundBps:  p.InboundBps,
			OutboundBps: p.OutboundBps,
		}
	}
	return records
}

// GetNetworkHistoryStatsByRange 获取指定时间范围内的网络流量统计信息
// 参数：
//   - startTime: 开始时间
//   - endTime: 结束时间
// 返回：
//   - map[string]interface{}: 统计信息
//   - error: 查询失败时返回错误
func GetNetworkHistoryStatsByRange(startTime, endTime time.Time) (map[string]interface{}, error) {
	var result struct {
		MinInbound   uint64
		MaxInbound   uint64
		AvgInbound   float64
		MinOutbound  uint64
		MaxOutbound  uint64
		AvgOutbound  float64
		Count        int64
	}

	err := DB.Model(&NetworkHistory{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Select("MIN(inbound_bps) as min_inbound, MAX(inbound_bps) as max_inbound, AVG(inbound_bps) as avg_inbound, " +
			"MIN(outbound_bps) as min_outbound, MAX(outbound_bps) as max_outbound, AVG(outbound_bps) as avg_outbound, " +
			"COUNT(*) as count").
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("获取网络流量统计信息失败: %v", err)
	}

	return map[string]interface{}{
		"inbound": map[string]interface{}{
			"min": result.MinInbound,
			"max": result.MaxInbound,
			"avg": result.AvgInbound,
		},
		"outbound": map[string]interface{}{
			"min": result.MinOutbound,
			"max": result.MaxOutbound,
			"avg": result.AvgOutbound,
		},
		"count": result.Count,
	}, nil
}
