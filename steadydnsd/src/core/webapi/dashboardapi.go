// core/webapi/dashboardapi.go

package webapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
)

// DashboardAPIHandler 处理dashboard相关的API请求
func DashboardAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(dashboardHandler)
	authHandler(w, r)
}

// dashboardHandler 实际处理dashboard请求的函数
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取路径参数
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 检查路径长度
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "dashboard" {
		sendErrorResponse(w, "无效的API端点", http.StatusNotFound)
		return
	}

	switch len(parts) {
	case 2: // /api/dashboard
		switch r.Method {
		case http.MethodGet:
			sendErrorResponse(w, "需要指定具体的dashboard API端点", http.StatusBadRequest)
			return
		default:
			sendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}
	case 3: // /api/dashboard/{endpoint}
		endpoint := parts[2]
		switch endpoint {
		case "summary":
			if r.Method == http.MethodGet {
				getDashboardSummary(w, r)
				return
			}
		case "trends":
			if r.Method == http.MethodGet {
				getDashboardTrends(w, r)
				return
			}
		case "top":
			if r.Method == http.MethodGet {
				getDashboardTop(w, r)
				return
			}
		default:
			sendErrorResponse(w, "无效的dashboard API端点", http.StatusNotFound)
			return
		}
		sendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	default:
		sendErrorResponse(w, "无效的API端点", http.StatusNotFound)
		return
	}
}

// 系统概览统计结构
type SystemStats struct {
	TotalQueries  int     `json:"totalQueries"`
	QPS           float64 `json:"qps"`
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
	QPSTrend      []QPSTrend      `json:"qpsTrend"`
	LatencyData   []LatencyData   `json:"latencyData"`
	ResourceUsage []ResourceUsage `json:"resourceUsage"`
}

// 排行榜数据响应结构
type DashboardTopResponse struct {
	TopDomains []TopDomain `json:"topDomains"`
	TopClients []TopClient `json:"topClients"`
}

// getDashboardSummary 获取dashboard综合数据
func getDashboardSummary(w http.ResponseWriter, r *http.Request) {
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

	sendSuccessResponse(w, response, "获取dashboard综合数据成功")
}

// getDashboardTrends 获取dashboard趋势数据
func getDashboardTrends(w http.ResponseWriter, r *http.Request) {
	// 获取时间范围参数
	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "1h"
	}

	// 获取QPS趋势数据
	qpsTrend := getQPSTrend(timeRange)

	// 获取延迟分布数据
	latencyData := getLatencyData()

	// 获取资源使用趋势数据
	resourceUsage := getResourceUsage(timeRange)

	// 构建响应
	response := DashboardTrendsResponse{
		QPSTrend:      qpsTrend,
		LatencyData:   latencyData,
		ResourceUsage: resourceUsage,
	}

	sendSuccessResponse(w, response, "获取dashboard趋势数据成功")
}

// getDashboardTop 获取dashboard排行榜数据
func getDashboardTop(w http.ResponseWriter, r *http.Request) {
	// 获取限制参数
	limitStr := r.URL.Query().Get("limit")
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

	sendSuccessResponse(w, response, "获取dashboard排行榜数据成功")
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
			CacheHitRate:  0,
			SystemHealth:  0,
			ActiveServers: 0,
		}
	}

	// 获取网络统计信息
	networkStats := statsManager.GetNetworkStats()

	// 计算QPS
	qps := statsManager.CalculateQPS()

	// 获取系统健康度
	health := statsManager.GetSystemHealth()

	// 计算活跃服务器数量（简单实现，实际应从配置或监控中获取）
	activeServers := 2 // UDP和TCP服务器

	return SystemStats{
		TotalQueries:  int(networkStats.TotalRequests),
		QPS:           qps,
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
		// 构建服务器地址
		addr := server.Address
		if server.Port != 53 {
			addr = fmt.Sprintf("%s:%d", server.Address, server.Port)
		} else {
			addr = fmt.Sprintf("%s:53", server.Address)
		}

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
	// TODO: 从实际缓存系统中获取数据
	// 暂时返回模拟数据
	return CacheStats{
		Size:     "1.2 GB",
		MaxSize:  "2 GB",
		HitRate:  85,
		MissRate: 15,
		Items:    150000,
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
