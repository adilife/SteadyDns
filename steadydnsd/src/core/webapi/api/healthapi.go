package api

import (
	"net/http"

	"SteadyDNS/core/webapi/middleware"
)

// HealthCheckHandler 健康检查处理函数
// 检查系统、数据库和DNS服务的健康状态
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// 执行健康检查
	healthStatus := PerformHealthCheck()

	// 发送响应
	middleware.SendSuccessResponse(w, healthStatus, "健康检查完成")
}
