// core/webapi/forwardserverapi.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
	"SteadyDNS/core/webapi/middleware"

	"github.com/miekg/dns"
)

// ForwardServerAPIHandler 处理服务器API请求
func ForwardServerAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(forwardServerHandler)
	authHandler(w, r)
}

// forwardServerHandler 实际处理服务器请求的函数
func forwardServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取路径参数
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 检查路径长度
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "forward-servers" {
		middleware.SendErrorResponse(w, "无效的API端点", http.StatusNotFound)
		return
	}

	switch len(parts) {
	case 2: // /api/forward-servers
		// 批量操作
		if r.Method == http.MethodPost && r.URL.Query().Get("batch") == "true" {
			batchAddForwardServers(w, r)
			return
		}
		if r.Method == http.MethodDelete && r.URL.Query().Get("batch") == "true" {
			batchDeleteForwardServers(w, r)
			return
		}
		middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	case 3: // /api/forward-servers/{id}
		serverIDStr := parts[2]
		serverID, err := strconv.ParseUint(serverIDStr, 10, 32)
		if err != nil {
			middleware.SendErrorResponse(w, "无效的服务器ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("health") == "true" {
				checkForwardServerHealth(w, uint(serverID))
				return
			}
			getForwardServerByID(w, uint(serverID))
			return
		case http.MethodPut:
			updateForwardServer(w, r, uint(serverID))
			return
		case http.MethodDelete:
			deleteForwardServer(w, uint(serverID))
			return
		default:
			middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}
	default:
		middleware.SendErrorResponse(w, "无效的API端点", http.StatusNotFound)
		return
	}
}

// getForwardServerByID 根据ID获取服务器
func getForwardServerByID(w http.ResponseWriter, serverID uint) {
	server, err := database.GetDNSServerByID(serverID)
	if err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("获取服务器失败: %v", err), http.StatusNotFound)
		return
	}

	middleware.SendSuccessResponse(w, server, "获取服务器成功")
}

// updateForwardServer 更新服务器
func updateForwardServer(w http.ResponseWriter, r *http.Request, serverID uint) {
	var server database.DNSServer
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&server); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证服务器配置
	if err := database.ValidateDNSServerDB(&server); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("验证失败: %v", err), http.StatusBadRequest)
		return
	}

	// 设置ID
	server.ID = serverID

	// 更新服务器
	if err := database.UpdateDNSServer(&server); err != nil {
		if strings.Contains(err.Error(), "服务器不存在") {
			middleware.SendErrorResponse(w, fmt.Sprintf("更新服务器失败: %v", err), http.StatusNotFound)
		} else {
			middleware.SendErrorResponse(w, fmt.Sprintf("更新服务器失败: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	middleware.SendSuccessResponse(w, server, "服务器更新成功")
}

// deleteForwardServer 删除服务器
func deleteForwardServer(w http.ResponseWriter, serverID uint) {
	// 检查服务器是否存在
	_, err := database.GetDNSServerByID(serverID)
	if err != nil {
		middleware.SendErrorResponse(w, "服务器不存在", http.StatusNotFound)
		return
	}

	// 删除服务器
	if err := database.DeleteDNSServer(serverID); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("删除服务器失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	middleware.SendSuccessResponse(w, nil, "服务器删除成功")
}

// batchAddForwardServers 批量添加服务器
func batchAddForwardServers(w http.ResponseWriter, r *http.Request) {
	var servers []database.DNSServer
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&servers); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	if len(servers) == 0 {
		middleware.SendErrorResponse(w, "服务器列表不能为空", http.StatusBadRequest)
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

	middleware.SendSuccessResponse(w, response, fmt.Sprintf("成功添加 %d 个服务器", successCount))
}

// batchDeleteForwardServers 批量删除服务器
func batchDeleteForwardServers(w http.ResponseWriter, r *http.Request) {
	var ids []uint
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&ids); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	if len(ids) == 0 {
		middleware.SendErrorResponse(w, "ID列表不能为空", http.StatusBadRequest)
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
	middleware.SendSuccessResponse(w, response, fmt.Sprintf("成功删除 %d 个服务器", successCount))
}

// checkForwardServerHealth 检查服务器健康状态
func checkForwardServerHealth(w http.ResponseWriter, serverID uint) {
	// 获取服务器信息
	server, err := database.GetDNSServerByID(serverID)
	if err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("获取服务器失败: %v", err), http.StatusNotFound)
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
	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	result, _, err := c.Exchange(query, serverAddr)
	duration := time.Since(startTime)
	responseTime := float64(duration.Milliseconds())

	// 分析响应
	isHealthy := false

	if err == nil && result != nil && (result.Rcode == dns.RcodeSuccess || result.Rcode == dns.RcodeNameError || result.Rcode == dns.RcodeRefused) {
		isHealthy = true
	}

	// 发送成功响应
	middleware.SendSuccessResponse(w, map[string]interface{}{
		"server_id":     server.ID,
		"address":       server.Address,
		"port":          server.Port,
		"is_healthy":    isHealthy,
		"response_time": responseTime,
	}, "服务器健康检查完成")
}
