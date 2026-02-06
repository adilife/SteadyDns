package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheckHandler 健康检查处理函数
// 检查系统、数据库和DNS服务的健康状态
func HealthCheckHandler(c *gin.Context) {
	// 执行健康检查
	healthStatus := PerformHealthCheck()

	// 发送响应
	c.JSON(http.StatusOK, gin.H{"success": true, "data": healthStatus, "message": "健康检查完成"})
}
