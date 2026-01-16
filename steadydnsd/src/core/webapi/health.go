// core/webapi/health.go

package webapi

import (
	"net/http"
	"runtime"
	"time"

	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
)

// HealthCheckResponse 健康检查响应结构
type HealthCheckResponse struct {
	Status     string                     `json:"status"`     // 整体状态 (healthy/unhealthy)
	Timestamp  time.Time                  `json:"timestamp"`  // 检查时间戳
	System     SystemHealth               `json:"system"`     // 系统状态
	Database   DatabaseHealth             `json:"database"`   // 数据库状态
	DNS        DNSHealth                  `json:"dns"`        // DNS服务状态
	Components map[string]ComponentHealth `json:"components"` // 组件状态
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	CPU             int    `json:"cpu"`              // CPU核心数
	GoRoutines      int    `json:"goroutines"`       // 当前goroutine数
	MemoryAllocated uint64 `json:"memory_allocated"` // 已分配内存(字节)
	MemoryTotal     uint64 `json:"memory_total"`     // 总内存(字节)
}

// DatabaseHealth 数据库健康状态
type DatabaseHealth struct {
	Status    string  `json:"status"`               // 数据库状态 (healthy/unhealthy)
	Latency   float64 `json:"latency_ms"`           // 数据库响应延迟(毫秒)
	LastError string  `json:"last_error,omitempty"` // 最后错误信息
}

// DNSHealth DNS服务健康状态
type DNSHealth struct {
	Status    string `json:"status"`               // DNS服务状态 (healthy/unhealthy)
	IsRunning bool   `json:"is_running"`           // DNS服务是否运行中
	LastError string `json:"last_error,omitempty"` // 最后错误信息
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Status    string `json:"status"`               // 组件状态 (healthy/unhealthy)
	LastError string `json:"last_error,omitempty"` // 最后错误信息
}

// HealthCheckHandler 健康检查处理函数
// 检查系统、数据库和DNS服务的健康状态
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// 执行健康检查
	healthStatus := performHealthCheck()

	// 发送响应
	sendSuccessResponse(w, healthStatus, "健康检查完成")
}

// performHealthCheck 执行健康检查
func performHealthCheck() HealthCheckResponse {
	// 获取系统状态
	systemHealth := getSystemHealth()

	// 检查数据库状态
	dbHealth := checkDatabaseHealth()

	// 检查DNS服务状态
	dnsHealth := checkDNSHealth()

	// 检查各组件状态
	componentsHealth := checkComponentsHealth()

	// 确定整体状态
	overallStatus := "healthy"
	if dbHealth.Status != "healthy" || dnsHealth.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// 构建响应
	return HealthCheckResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		System:     systemHealth,
		Database:   dbHealth,
		DNS:        dnsHealth,
		Components: componentsHealth,
	}
}

// getSystemHealth 获取系统状态
func getSystemHealth() SystemHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemHealth{
		CPU:             runtime.NumCPU(),
		GoRoutines:      runtime.NumGoroutine(),
		MemoryAllocated: m.Alloc,
		MemoryTotal:     m.Sys,
	}
}

// checkDatabaseHealth 检查数据库健康状态
func checkDatabaseHealth() DatabaseHealth {
	start := time.Now()

	// 执行简单的数据库查询
	err := database.CheckConnection()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return DatabaseHealth{
			Status:    "unhealthy",
			Latency:   float64(latency),
			LastError: err.Error(),
		}
	}

	return DatabaseHealth{
		Status:  "healthy",
		Latency: float64(latency),
	}
}

// checkDNSHealth 检查DNS服务健康状态
func checkDNSHealth() DNSHealth {
	// 检查DNS服务器是否运行
	isRunning := sdns.IsDNSServerRunning()

	if !isRunning {
		return DNSHealth{
			Status:    "unhealthy",
			IsRunning: false,
			LastError: "DNS server is not running",
		}
	}

	return DNSHealth{
		Status:    "healthy",
		IsRunning: true,
	}
}

// checkComponentsHealth 检查各组件健康状态
func checkComponentsHealth() map[string]ComponentHealth {
	components := make(map[string]ComponentHealth)

	// 检查统计管理器
	components["stats_manager"] = checkStatsManagerHealth()

	// 检查配置
	components["config"] = checkConfigHealth()

	return components
}

// checkStatsManagerHealth 检查统计管理器健康状态
func checkStatsManagerHealth() ComponentHealth {
	// 检查统计管理器是否正常
	// 这里可以添加具体的检查逻辑
	return ComponentHealth{
		Status: "healthy",
	}
}

// checkConfigHealth 检查配置健康状态
func checkConfigHealth() ComponentHealth {
	// 检查配置是否正常
	// 这里可以添加具体的检查逻辑
	return ComponentHealth{
		Status: "healthy",
	}
}
