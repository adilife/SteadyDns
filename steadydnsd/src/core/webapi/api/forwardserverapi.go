// core/webapi/forwardserverapi.go
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
)

// ForwardServerAPIHandlerGin 处理服务器API请求
func ForwardServerAPIHandlerGin(c *gin.Context) {
	// 认证中间件已在路由中统一应用
	forwardServerHandlerGin(c)
}

// forwardServerHandlerGin 实际处理服务器请求的函数
func forwardServerHandlerGin(c *gin.Context) {
	// 获取路径参数
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 检查路径长度
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "forward-servers" {
		c.JSON(http.StatusNotFound, gin.H{"error": "无效的API端点"})
		return
	}

	switch len(parts) {
	case 2: // /api/forward-servers
		// 批量操作
		if c.Request.Method == http.MethodPost && c.Query("batch") == "true" {
			batchAddForwardServersGin(c)
			return
		}
		if c.Request.Method == http.MethodDelete && c.Query("batch") == "true" {
			batchDeleteForwardServersGin(c)
			return
		}
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
		return
	case 3: // /api/forward-servers/{id}
		serverIDStr := parts[2]
		serverID, err := strconv.ParseUint(serverIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
			return
		}

		switch c.Request.Method {
		case http.MethodGet:
			if c.Query("health") == "true" {
				checkForwardServerHealthGin(c, uint(serverID))
				return
			}
			getForwardServerByIDGin(c, uint(serverID))
			return
		case http.MethodPut:
			updateForwardServerGin(c, uint(serverID))
			return
		case http.MethodDelete:
			deleteForwardServerGin(c, uint(serverID))
			return
		default:
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
			return
		}
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "无效的API端点"})
		return
	}
}

// getForwardServerByIDGin 根据ID获取服务器
func getForwardServerByIDGin(c *gin.Context, serverID uint) {
	server, err := database.GetDNSServerByID(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   fmt.Sprintf("获取服务器失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    server,
		"message": "获取服务器成功",
	})
}

// updateForwardServerGin 更新服务器
func updateForwardServerGin(c *gin.Context, serverID uint) {
	var server database.DNSServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 验证服务器配置
	if err := database.ValidateDNSServerDB(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("验证失败: %v", err),
		})
		return
	}

	// 设置ID
	server.ID = serverID

	// 更新服务器
	if err := database.UpdateDNSServer(&server); err != nil {
		if strings.Contains(err.Error(), "服务器不存在") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("更新服务器失败: %v", err),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("更新服务器失败: %v", err),
			})
		}
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    server,
		"message": "服务器更新成功",
	})
}

// deleteForwardServerGin 删除服务器
func deleteForwardServerGin(c *gin.Context, serverID uint) {
	// 检查服务器是否存在
	_, err := database.GetDNSServerByID(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
		return
	}

	// 删除服务器
	if err := database.DeleteDNSServer(serverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("删除服务器失败: %v", err),
		})
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "服务器删除成功",
	})
}

// batchAddForwardServersGin 批量添加服务器
func batchAddForwardServersGin(c *gin.Context) {
	var servers []database.DNSServer
	if err := c.ShouldBindJSON(&servers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	if len(servers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务器列表不能为空"})
		return
	}

	// 验证并添加服务器
	successCount := 0
	var errors []string

	for i := range servers {
		// 验证服务器配置
		if err := database.ValidateDNSServerDB(&servers[i]); err != nil {
			errors = append(errors, fmt.Sprintf("服务器 %d: %v", i+1, err))
			continue
		}

		// 添加服务器
		if err := database.CreateDNSServer(&servers[i]); err != nil {
			errors = append(errors, fmt.Sprintf("服务器 %d: %v", i+1, err))
			continue
		}

		successCount++
	}

	// 返回结果
	response := map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(servers),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": fmt.Sprintf("成功添加 %d 个服务器", successCount),
	})
}

// batchDeleteForwardServersGin 批量删除服务器
func batchDeleteForwardServersGin(c *gin.Context) {
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID列表不能为空"})
		return
	}

	// 批量删除服务器
	successCount := 0
	var errors []string

	for _, id := range ids {
		if err := database.DeleteDNSServer(id); err != nil {
			errors = append(errors, fmt.Sprintf("服务器 %d: %v", id, err))
			continue
		}
		successCount++
	}

	// 返回结果
	response := map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(ids),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	// 发送成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": fmt.Sprintf("成功删除 %d 个服务器", successCount),
	})
}

// checkForwardServerHealthGin 检查服务器健康状态
func checkForwardServerHealthGin(c *gin.Context, serverID uint) {
	// 获取服务器信息
	server, err := database.GetDNSServerByID(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("获取服务器失败: %v", err),
		})
		return
	}

	// 构建服务器地址
	serverAddr := fmt.Sprintf("%s:%d", server.Address, server.Port)

	// 创建DNS查询消息
	query := new(dns.Msg)
	query.SetQuestion("healthcheck.local.", dns.TypeA)
	query.RecursionDesired = true

	// 发送DNS查询
	startTime := time.Now()
	client := new(dns.Client)
	client.Timeout = 5 * time.Second

	result, _, err := client.Exchange(query, serverAddr)
	duration := time.Since(startTime)
	responseTime := float64(duration.Milliseconds())

	// 分析响应
	isHealthy := false

	if err == nil && result != nil && (result.Rcode == dns.RcodeSuccess || result.Rcode == dns.RcodeNameError || result.Rcode == dns.RcodeRefused) {
		isHealthy = true
	}

	// 发送成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"server_id":     server.ID,
			"address":       server.Address,
			"port":          server.Port,
			"is_healthy":    isHealthy,
			"response_time": responseTime,
		},
		"message": "服务器健康检查完成",
	})
}
