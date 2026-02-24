// core/webapi/dashboardapi.go

package api

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"

	"github.com/gin-gonic/gin"
)

// DashboardAPIHandlerGin 处理dashboard相关的API请求（Gin版本）
func DashboardAPIHandlerGin(c *gin.Context) {
	// 认证中间件已在路由中统一应用
	dashboardHandlerGin(c)
}

// dashboardHandlerGin 实际处理dashboard请求的函数（Gin版本）
func dashboardHandlerGin(c *gin.Context) {
	// 获取路径参数
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 检查路径长度
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "dashboard" {
		c.JSON(http.StatusNotFound, gin.H{"error": "无效的API端点"})
		return
	}

	switch len(parts) {
	case 2: // /api/dashboard
		switch c.Request.Method {
		case http.MethodGet:
			c.JSON(http.StatusBadRequest, gin.H{"error": "需要指定具体的dashboard API端点"})
			return
		default:
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
			return
		}
	case 3: // /api/dashboard/{endpoint}
		endpoint := parts[2]
		switch endpoint {
		case "summary":
			if c.Request.Method == http.MethodGet {
				getDashboardSummaryGin(c)
				return
			}
		case "trends":
			if c.Request.Method == http.MethodGet {
				getDashboardTrendsGin(c)
				return
			}
		case "top":
			if c.Request.Method == http.MethodGet {
				getDashboardTopGin(c)
				return
			}
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "无效的dashboard API端点"})
			return
		}
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
		return
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "无效的API端点"})
		return
	}
}

// 系统概览统计结构
type SystemStats struct {
	TotalQueries  int     `json:"totalQueries"`
	QPS           float64 `json:"qps"`        // 峰值QPS（最近5分钟）
	CurrentQPS    float64 `json:"currentQps"` // 当前QPS
	AvgQPS        float64 `json:"avgQps"`     // 平均QPS（最近5分钟）
	CacheHitRate  float64 `json:"cacheHitRate"`
	SystemHealth  int     `json:"systemHealth"`
	ActiveServers int     `json:"activeServers"`
}

// 转发服务器状态结构
type ForwardServerStatus struct {
	ID      int     `json:"id"`
	Address string  `json:"address"`
	QPS     float64 `json:"qps"`
	Latency float64 `json:"latency"`
	Status  string  `json:"status"`
}

// 缓存状态结构
type CacheStats struct {
	Size     string `json:"size"`
	MaxSize  string `json:"maxSize"`
	HitRate  int    `json:"hitRate"`
	MissRate int    `json:"missRate"`
	Items    int    `json:"items"`
}

// 系统资源使用情况结构
type SystemResources struct {
	CPU     int `json:"cpu"`
	Memory  int `json:"memory"`
	Disk    int `json:"disk"`
	Network struct {
		Inbound  string `json:"inbound"`
		Outbound string `json:"outbound"`
	} `json:"network"`
}

// QPS趋势数据结构
type QPSTrend struct {
	Time string  `json:"time"`
	QPS  float64 `json:"qps"`
}

// QPS趋势聚合数据结构
type QPSTrendAggregated struct {
	TimeLabels []string           `json:"timeLabels"`
	QPSValues  []float64          `json:"qpsValues"`
	Statistics QPSTrendStatistics `json:"statistics"`
}

// QPS趋势统计信息
type QPSTrendStatistics struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Avg        float64 `json:"avg"`
	Current    float64 `json:"current"`
	DataPoints int     `json:"dataPoints"`
}

// 延迟分布数据结构
type LatencyData struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// 资源使用趋势数据结构
type ResourceUsage struct {
	Time   string `json:"time"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory"`
	Disk   int    `json:"disk"`
}

// 资源使用聚合数据结构
type ResourceUsageAggregated struct {
	TimeLabels []string                `json:"timeLabels"`
	CPUValues  []int                   `json:"cpuValues"`
	MemValues  []int                   `json:"memValues"`
	DiskValues []int                   `json:"diskValues"`
	Statistics ResourceUsageStatistics `json:"statistics"`
}

// ResourceUsageStatistics 资源使用统计信息
type ResourceUsageStatistics struct {
	CPU    ResourceStatItem `json:"cpu"`
	Memory ResourceStatItem `json:"memory"`
	Disk   ResourceStatItem `json:"disk"`
}

// ResourceStatItem 单项资源统计
type ResourceStatItem struct {
	Min        int     `json:"min"`
	Max        int     `json:"max"`
	Avg        float64 `json:"avg"`
	DataPoints int     `json:"dataPoints"`
}

// 热门域名结构
type TopDomain struct {
	Rank       int     `json:"rank"`
	Domain     string  `json:"domain"`
	Queries    int     `json:"queries"`
	Percentage float64 `json:"percentage"`
}

// 热门客户端结构
type TopClient struct {
	Rank       int     `json:"rank"`
	IP         string  `json:"ip"`
	Queries    int     `json:"queries"`
	Percentage float64 `json:"percentage"`
}

// 综合数据响应结构
type DashboardSummaryResponse struct {
	SystemStats     SystemStats           `json:"systemStats"`
	ForwardServers  []ForwardServerStatus `json:"forwardServers"`
	CacheStats      CacheStats            `json:"cacheStats"`
	SystemResources SystemResources       `json:"systemResources"`
}

// 趋势数据响应结构
type DashboardTrendsResponse struct {
	QPSTrend      interface{}       `json:"qpsTrend"`
	LatencyData   []LatencyData     `json:"latencyData"`
	ResourceUsage interface{}       `json:"resourceUsage"`
	RequestInfo   *TrendRequestInfo `json:"requestInfo,omitempty"`
}

// TrendRequestInfo 请求信息
type TrendRequestInfo struct {
	Type      string `json:"type"`
	TimeRange string `json:"timeRange"`
	Points    int    `json:"points"`
	RequestAt string `json:"requestAt"`
}

// 排行榜数据响应结构
type DashboardTopResponse struct {
	TopDomains []TopDomain `json:"topDomains"`
	TopClients []TopClient `json:"topClients"`
}

// getDashboardSummaryGin 获取dashboard综合数据（Gin版本）
func getDashboardSummaryGin(c *gin.Context) {
	// 获取系统概览统计
	systemStats := getSystemStats()

	// 获取转发服务器状态
	forwardServers := getForwardServerStatus()

	// 获取缓存状态
	cacheStats := getCacheStats()

	// 获取系统资源使用情况
	systemResources := getSystemResources()

	// 构建响应
	response := DashboardSummaryResponse{
		SystemStats:     systemStats,
		ForwardServers:  forwardServers,
		CacheStats:      cacheStats,
		SystemResources: systemResources,
	}

	// 发送成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "获取dashboard综合数据成功",
	})
}

// getDashboardTrendsGin 获取dashboard趋势数据（Gin版本）
func getDashboardTrendsGin(c *gin.Context) {
	startTime := time.Now()

	dataType := c.Query("type")
	if dataType == "" {
		dataType = "all"
	}

	validTypes := map[string]bool{"qps": true, "resource": true, "all": true}
	if !validTypes[dataType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的type参数，可选值：qps, resource, all",
		})
		return
	}

	timeRange := c.Query("timeRange")
	if timeRange == "" {
		timeRange = "1h"
	}

	validTimeRanges := map[string]bool{"1h": true, "6h": true, "24h": true, "7d": true}
	if !validTimeRanges[timeRange] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的timeRange参数，可选值：1h, 6h, 24h, 7d",
		})
		return
	}

	points := 0
	pointsStr := c.Query("points")
	if pointsStr != "" {
		p, err := strconv.Atoi(pointsStr)
		if err != nil || p < 0 || p > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "无效的points参数，必须为0或1-1000之间的正整数",
			})
			return
		}
		points = p
	}

	response := DashboardTrendsResponse{}

	if dataType == "qps" || dataType == "all" {
		if points > 0 {
			response.QPSTrend = getQPSTrendAggregated(timeRange, points)
		} else {
			response.QPSTrend = getQPSTrend(timeRange)
		}
	}

	if dataType == "all" {
		response.LatencyData = getLatencyData()
	}

	if dataType == "resource" || dataType == "all" {
		if points > 0 {
			response.ResourceUsage = getResourceUsageAggregated(timeRange, points)
		} else {
			response.ResourceUsage = getResourceUsage(timeRange)
		}
	}

	if points > 0 {
		response.RequestInfo = &TrendRequestInfo{
			Type:      dataType,
			TimeRange: timeRange,
			Points:    points,
			RequestAt: startTime.Format("2006-01-02 15:04:05"),
		}
	}

	responseTime := time.Since(startTime)
	if responseTime > 200*time.Millisecond {
		common.NewLogger().Warn("QPS趋势接口响应时间超过200ms: %v", responseTime)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "获取dashboard趋势数据成功",
	})
}

// getDashboardTopGin 获取dashboard排行榜数据（Gin版本）
func getDashboardTopGin(c *gin.Context) {
	// 获取限制参数
	limitStr := c.Query("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 获取热门域名
	topDomains := getTopDomains(limit)

	// 获取热门客户端
	topClients := getTopClients(limit)

	// 构建响应
	response := DashboardTopResponse{
		TopDomains: topDomains,
		TopClients: topClients,
	}

	// 发送成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "获取dashboard排行榜数据成功",
	})
}

// getSystemStats 获取系统概览统计
func getSystemStats() SystemStats {
	// 获取统计管理器
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		// 统计管理器不可用，返回模拟数据
		return SystemStats{
			TotalQueries:  0,
			QPS:           0,
			CurrentQPS:    0,
			AvgQPS:        0,
			CacheHitRate:  0,
			SystemHealth:  0,
			ActiveServers: 0,
		}
	}

	// 获取网络统计信息
	networkStats := statsManager.GetNetworkStats()

	// 计算峰值QPS（最近5分钟）
	peakQPS := statsManager.GetPeakQPS(5 * time.Minute)

	// 计算当前QPS
	currentQPS := statsManager.GetCurrentQPS()

	// 计算平均QPS（最近5分钟）
	avgQPS := statsManager.GetAverageQPS(5 * time.Minute)

	// 获取系统健康度
	health := statsManager.GetSystemHealth()

	// 计算活跃服务器数量（简单实现，实际应从配置或监控中获取）
	activeServers := 2 // UDP和TCP服务器

	return SystemStats{
		TotalQueries:  int(networkStats.TotalRequests),
		QPS:           peakQPS,
		CurrentQPS:    currentQPS,
		AvgQPS:        avgQPS,
		CacheHitRate:  0, // 暂时返回0，需要从缓存系统获取
		SystemHealth:  health,
		ActiveServers: activeServers,
	}
}

// getForwardServerStatus 获取转发服务器状态
func getForwardServerStatus() []ForwardServerStatus {
	// 尝试从数据库获取转发服务器，添加超时保护
	var servers []database.DNSServer
	var err error

	// 使用通道实现简单的超时控制
	resultChan := make(chan struct{})

	go func() {
		servers, err = database.GetDNSServersByGroupID(1)
		close(resultChan)
	}()

	// 等待数据库操作完成或超时
	select {
	case <-resultChan:
		// 操作完成
	case <-time.After(5 * time.Second):
		// 数据库操作超时，返回模拟数据
		logger := common.NewLogger()
		logger.Warn("获取转发服务器超时，返回模拟数据")
		return []ForwardServerStatus{
			{ID: 1, Address: "8.8.8.8", QPS: 12.5, Latency: 15.2, Status: "healthy"},
			{ID: 2, Address: "1.1.1.1", QPS: 10.2, Latency: 12.5, Status: "healthy"},
			{ID: 3, Address: "114.114.114.114", QPS: 8.7, Latency: 8.3, Status: "healthy"},
		}
	}

	if err != nil {
		common.NewLogger().Error("获取转发服务器失败: %v", err)
		// 返回模拟数据
		return []ForwardServerStatus{
			{ID: 1, Address: "8.8.8.8", QPS: 12.5, Latency: 15.2, Status: "healthy"},
			{ID: 2, Address: "1.1.1.1", QPS: 10.2, Latency: 12.5, Status: "healthy"},
			{ID: 3, Address: "114.114.114.114", QPS: 8.7, Latency: 8.3, Status: "healthy"},
		}
	}

	// 构建响应数据
	serverStatuses := make([]ForwardServerStatus, len(servers))
	for i, server := range servers {
		// 构建服务器地址（使用net.JoinHostPort处理IPv6地址格式）
		addr := net.JoinHostPort(server.Address, strconv.Itoa(server.Port))

		// 获取服务器统计信息
		qps := 0.0
		latency := 0.0
		status := "healthy"

		// 从全局DNS转发器获取统计信息
		if sdns.GlobalDNSForwarder != nil {
			// 添加超时保护
			statsChan := make(chan *sdns.ServerStats)
			go func() {
				stats := sdns.GlobalDNSForwarder.GetServerStats(addr)
				statsChan <- stats
			}()

			select {
			case stats := <-statsChan:
				if stats != nil {
					stats.Mu.RLock()
					qps = stats.QPS
					latency = stats.Latency
					status = stats.Status
					stats.Mu.RUnlock()
				}
			case <-time.After(2 * time.Second):
				// 获取统计信息超时，使用默认值
				logger := common.NewLogger()
				logger.Warn("获取服务器统计信息超时: %s", addr)
			}
		}

		serverStatuses[i] = ForwardServerStatus{
			ID:      int(server.ID),
			Address: server.Address,
			QPS:     qps,
			Latency: latency,
			Status:  status,
		}
	}

	return serverStatuses
}

// getCacheStats 获取缓存状态
func getCacheStats() CacheStats {
	// 从实际缓存系统获取数据
	cacheStats := sdns.GetCacheStats()
	if cacheStats == nil {
		return CacheStats{}
	}

	// 获取当前缓存大小
	currentSize := int64(0)
	if val, ok := cacheStats["currentSize"].(int64); ok {
		currentSize = val
	}

	// 获取最大缓存大小
	maxSize := int64(0)
	if val, ok := cacheStats["maxSize"].(int64); ok {
		maxSize = val
	}

	// 计算命中率
	hitRate := 0.0
	if val, ok := cacheStats["hitRate"].(float64); ok {
		hitRate = val
	}

	// 获取缓存条目数量
	items := 0
	if val, ok := cacheStats["count"].(int); ok {
		items = val
	}

	return CacheStats{
		Size:     formatSize(currentSize),
		MaxSize:  formatSize(maxSize),
		HitRate:  int(hitRate),
		MissRate: 100 - int(hitRate),
		Items:    items,
	}
}

// formatSize 格式化大小显示
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// getSystemResources 获取系统资源使用情况
func getSystemResources() SystemResources {
	// TODO: 从实际系统中获取数据
	// 暂时返回模拟数据
	var resources SystemResources
	resources.CPU = 45
	resources.Memory = 68
	resources.Disk = 32
	resources.Network.Inbound = "12 MB/s"
	resources.Network.Outbound = "8 MB/s"
	return resources
}

// getQPSTrend 获取QPS趋势数据
func getQPSTrend(timeRange string) []QPSTrend {
	// 获取统计管理器
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		// 统计管理器不可用，返回空数组
		return []QPSTrend{}
	}

	// 获取QPS历史数据
	qpsHistory := statsManager.GetQPSHistory(timeRange)

	// 转换为响应格式
	trend := make([]QPSTrend, 0)
	for _, point := range qpsHistory {
		var timeFormat string
		switch timeRange {
		case "1h", "6h", "24h":
			timeFormat = "15:04"
		case "7d":
			timeFormat = "01-02"
		default:
			timeFormat = "15:04"
		}

		trend = append(trend, QPSTrend{
			Time: point.Time.Format(timeFormat),
			QPS:  point.QPS,
		})
	}

	return trend
}

// getQPSTrendAggregated 获取聚合后的QPS趋势数据
func getQPSTrendAggregated(timeRange string, points int) QPSTrendAggregated {
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		return QPSTrendAggregated{
			TimeLabels: []string{},
			QPSValues:  []float64{},
			Statistics: QPSTrendStatistics{},
		}
	}

	aggregated := statsManager.GetAggregatedQPSTrend(timeRange, points)
	if aggregated == nil {
		return QPSTrendAggregated{
			TimeLabels: []string{},
			QPSValues:  []float64{},
			Statistics: QPSTrendStatistics{},
		}
	}

	return QPSTrendAggregated{
		TimeLabels: aggregated.TimeLabels,
		QPSValues:  aggregated.QPSValues,
		Statistics: QPSTrendStatistics{
			Min:        aggregated.Statistics.Min,
			Max:        aggregated.Statistics.Max,
			Avg:        aggregated.Statistics.Avg,
			Current:    aggregated.Statistics.Current,
			DataPoints: aggregated.Statistics.DataPoints,
		},
	}
}

// getLatencyData 获取延迟分布数据
func getLatencyData() []LatencyData {
	// 获取统计管理器
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		// 统计管理器不可用，返回空数组
		return []LatencyData{}
	}

	// 获取延迟分布数据
	distribution := statsManager.GetLatencyDistribution()

	// 转换为响应格式
	var latencyData []LatencyData
	for rangeStr, count := range distribution {
		latencyData = append(latencyData, LatencyData{
			Range: rangeStr,
			Count: count,
		})
	}

	return latencyData
}

// getResourceUsage 获取资源使用趋势数据
func getResourceUsage(timeRange string) []ResourceUsage {
	// 获取统计管理器
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		// 统计管理器不可用，返回空数组
		return []ResourceUsage{}
	}

	// 获取资源使用历史数据
	resourceHistory := statsManager.GetResourceHistory(timeRange)

	// 转换为响应格式
	usage := make([]ResourceUsage, 0)
	for _, point := range resourceHistory {
		var timeFormat string
		switch timeRange {
		case "1h", "6h", "24h":
			timeFormat = "15:04"
		case "7d":
			timeFormat = "01-02"
		default:
			timeFormat = "15:04"
		}

		usage = append(usage, ResourceUsage{
			Time:   point.Time.Format(timeFormat),
			CPU:    point.CPU,
			Memory: point.Memory,
			Disk:   point.Disk,
		})
	}

	return usage
}

// getResourceUsageAggregated 获取聚合后的资源使用趋势数据
func getResourceUsageAggregated(timeRange string, points int) ResourceUsageAggregated {
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		return ResourceUsageAggregated{
			TimeLabels: []string{},
			CPUValues:  []int{},
			MemValues:  []int{},
			DiskValues: []int{},
			Statistics: ResourceUsageStatistics{},
		}
	}

	aggregated := statsManager.GetAggregatedResourceUsage(timeRange, points)
	if aggregated == nil {
		return ResourceUsageAggregated{
			TimeLabels: []string{},
			CPUValues:  []int{},
			MemValues:  []int{},
			DiskValues: []int{},
			Statistics: ResourceUsageStatistics{},
		}
	}

	return ResourceUsageAggregated{
		TimeLabels: aggregated.TimeLabels,
		CPUValues:  aggregated.CPUValues,
		MemValues:  aggregated.MemValues,
		DiskValues: aggregated.DiskValues,
		Statistics: ResourceUsageStatistics{
			CPU: ResourceStatItem{
				Min:        aggregated.Statistics.CPU.Min,
				Max:        aggregated.Statistics.CPU.Max,
				Avg:        aggregated.Statistics.CPU.Avg,
				DataPoints: aggregated.Statistics.CPU.DataPoints,
			},
			Memory: ResourceStatItem{
				Min:        aggregated.Statistics.Memory.Min,
				Max:        aggregated.Statistics.Memory.Max,
				Avg:        aggregated.Statistics.Memory.Avg,
				DataPoints: aggregated.Statistics.Memory.DataPoints,
			},
			Disk: ResourceStatItem{
				Min:        aggregated.Statistics.Disk.Min,
				Max:        aggregated.Statistics.Disk.Max,
				Avg:        aggregated.Statistics.Disk.Avg,
				DataPoints: aggregated.Statistics.Disk.DataPoints,
			},
		},
	}
}

// getTopDomains 获取热门域名
func getTopDomains(limit int) []TopDomain {
	// 获取统计管理器
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		// 统计管理器不可用，返回空数组
		return []TopDomain{}
	}

	// 获取热门域名
	domainStats := statsManager.GetTopDomains(limit)

	// 转换为响应格式
	domains := make([]TopDomain, 0)
	for _, stat := range domainStats {
		domains = append(domains, TopDomain{
			Rank:       stat.Rank,
			Domain:     stat.Domain,
			Queries:    stat.Queries,
			Percentage: stat.Percentage,
		})
	}

	return domains
}

// getTopClients 获取热门客户端
func getTopClients(limit int) []TopClient {
	// 获取统计管理器
	statsManager := sdns.GetStatsManager()
	if statsManager == nil {
		// 统计管理器不可用，返回空数组
		return []TopClient{}
	}

	// 获取热门客户端
	clientStats := statsManager.GetTopClients(limit)

	// 转换为响应格式
	clients := make([]TopClient, 0)
	for _, stat := range clientStats {
		clients = append(clients, TopClient{
			Rank:       stat.Rank,
			IP:         stat.IP,
			Queries:    stat.Queries,
			Percentage: stat.Percentage,
		})
	}

	return clients
}
