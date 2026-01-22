// core/webapi/bindapi.go
// BIND权威域API接口

package webapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"SteadyDNS/core/bind"
	"SteadyDNS/core/common"
)

// apiLogger API日志记录器
var apiLogger = common.NewLogger()

// bindManager 全局BIND管理器实例
var bindManager *bind.BindManager

// initBindManager 初始化BIND管理器
func initBindManager() {
	if bindManager == nil {
		bindManager = bind.NewBindManager()
	}
}

// BindAPIHandler BIND权威域API请求处理函数
func BindAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 记录API请求信息
	apiLogger.Debug("API请求: %s %s, 客户端IP: %s",
		r.Method, r.URL.Path, r.RemoteAddr)

	// 初始化BIND管理器
	initBindManager()

	// 获取路径参数
	path := r.URL.Path
	method := r.Method

	// 解析域名参数
	domain := ""
	if strings.HasPrefix(path, "/api/bind-zones/") {
		// 提取域名
		domain = strings.TrimPrefix(path, "/api/bind-zones/")
		domain = strings.TrimSpace(domain)
		// 处理带trailing slash的情况
		domain = strings.TrimSuffix(domain, "/")
	}

	// 根据请求方法和路径调用相应的处理函数
	switch method {
	case http.MethodGet:
		if domain == "" {
			// 获取所有权威域
			GetBindZones(w, r)
		} else if domain == "history" {
			// 获取操作历史
			GetBindHistory(w, r)
		} else {
			// 获取单个权威域
			GetBindZone(w, r, domain)
		}
	case http.MethodPost:
		if domain == "" {
			// 创建权威域
			CreateBindZone(w, r)
		} else if strings.HasSuffix(domain, "/reload") {
			// 刷新权威域
			domain = strings.TrimSuffix(domain, "/reload")
			ReloadBindZone(w, r, domain)
		} else if strings.HasPrefix(domain, "history/") {
			// 恢复历史记录
			historyID := strings.TrimPrefix(domain, "history/")
			RestoreBindHistory(w, r, historyID)
		} else {
			// 不支持的POST请求
			http.Error(w, "不支持的请求", http.StatusMethodNotAllowed)
		}
	case http.MethodPut:
		if domain != "" {
			// 更新权威域
			UpdateBindZone(w, r, domain)
		} else {
			http.Error(w, "缺少域名参数", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if domain != "" {
			// 删除权威域
			DeleteBindZone(w, r, domain)
		} else {
			http.Error(w, "缺少域名参数", http.StatusBadRequest)
		}
	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

// GetBindZones 获取所有权威域
func GetBindZones(w http.ResponseWriter, r *http.Request) {
	apiLogger.Debug("获取所有权威域请求")
	// 初始化BIND管理器
	initBindManager()

	// 获取所有权威域
	zones, err := bindManager.GetAuthZones()
	if err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, "获取权威域列表失败: "+err.Error())
		return
	}

	// 返回成功响应
	SendSuccessResponse(w, zones)
}

// GetBindZone 获取单个权威域
func GetBindZone(w http.ResponseWriter, r *http.Request, domain string) {
	apiLogger.Debug("获取单个权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 获取权威域
	zone, err := bindManager.GetAuthZone(domain)
	if err != nil {
		SendErrorResponse(w, http.StatusNotFound, "权威域不存在: "+err.Error())
		return
	}

	// 返回成功响应
	SendSuccessResponse(w, zone)
}

// CreateBindZone 创建权威域
func CreateBindZone(w http.ResponseWriter, r *http.Request) {
	apiLogger.Debug("创建权威域请求")
	// 初始化BIND管理器
	initBindManager()

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendErrorResponse(w, http.StatusBadRequest, "读取请求体失败: "+err.Error())
		return
	}
	defer r.Body.Close()

	// 直接解析请求体到AuthZone结构体
	var zone bind.AuthZone
	if err := json.Unmarshal(body, &zone); err != nil {
		SendErrorResponse(w, http.StatusBadRequest, "解析请求体失败: "+err.Error())
		return
	}

	// 验证必填字段
	if zone.Domain == "" {
		SendErrorResponse(w, http.StatusBadRequest, "缺少域名参数")
		return
	}

	// 创建权威域
	if err := bindManager.CreateAuthZone(zone); err != nil {
		errMsg := "创建权威域失败: " + err.Error()
		// 检查是否为CNAME冲突错误，如果是则返回400状态码
		if strings.Contains(err.Error(), "CNAME") || strings.Contains(err.Error(), "记录名称冲突") {
			SendErrorResponse(w, http.StatusBadRequest, errMsg)
		} else {
			SendErrorResponse(w, http.StatusInternalServerError, errMsg)
		}
		return
	}

	// 返回成功响应
	SendSuccessResponse(w, map[string]string{
		"message": "权威域创建成功",
		"domain":  zone.Domain,
	})
}

// UpdateBindZone 更新权威域
func UpdateBindZone(w http.ResponseWriter, r *http.Request, domain string) {
	apiLogger.Debug("更新权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendErrorResponse(w, http.StatusBadRequest, "读取请求体失败: "+err.Error())
		return
	}
	defer r.Body.Close()

	// 直接解析请求体到AuthZone结构体
	var zone bind.AuthZone
	if err := json.Unmarshal(body, &zone); err != nil {
		SendErrorResponse(w, http.StatusBadRequest, "解析请求体失败: "+err.Error())
		return
	}

	// 确保域名一致
	zone.Domain = domain

	// 更新权威域
	if err := bindManager.UpdateAuthZone(zone); err != nil {
		errMsg := "更新权威域失败: " + err.Error()
		// 检查是否为CNAME冲突错误，如果是则返回400状态码
		if strings.Contains(err.Error(), "CNAME") || strings.Contains(err.Error(), "记录名称冲突") {
			SendErrorResponse(w, http.StatusBadRequest, errMsg)
		} else {
			SendErrorResponse(w, http.StatusInternalServerError, errMsg)
		}
		return
	}

	// 返回成功响应
	SendSuccessResponse(w, map[string]string{
		"message": "权威域更新成功",
		"domain":  zone.Domain,
	})
}

// DeleteBindZone 删除权威域
func DeleteBindZone(w http.ResponseWriter, r *http.Request, domain string) {
	apiLogger.Debug("删除权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 删除权威域
	if err := bindManager.DeleteAuthZone(domain); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, "删除权威域失败: "+err.Error())
		return
	}

	// 返回成功响应
	SendSuccessResponse(w, map[string]string{
		"message": "权威域删除成功",
		"domain":  domain,
	})
}

// ReloadBindZone 刷新权威域
func ReloadBindZone(w http.ResponseWriter, r *http.Request, domain string) {
	apiLogger.Debug("刷新权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 刷新BIND服务器
	if err := bindManager.ReloadBind(); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, "刷新BIND服务器失败: "+err.Error())
		return
	}

	// 返回成功响应
	SendSuccessResponse(w, map[string]string{
		"message": "BIND服务器刷新成功",
		"domain":  domain,
	})
}

// GetBindHistory 获取操作历史
func GetBindHistory(w http.ResponseWriter, r *http.Request) {
	apiLogger.Debug("获取操作历史请求")
	// 初始化BIND管理器
	initBindManager()

	// 获取所有历史记录
	history, _ := bindManager.HistoryMgr.GetHistoryRecords()

	// 返回成功响应
	SendSuccessResponse(w, history)
}

// RestoreBindHistory 恢复历史记录
func RestoreBindHistory(w http.ResponseWriter, r *http.Request, historyID string) {
	apiLogger.Debug("恢复历史记录请求，历史ID: %s", historyID)
	// 初始化BIND管理器
	initBindManager()

	// 返回成功响应（暂时不实现具体的恢复逻辑）
	SendSuccessResponse(w, map[string]string{
		"message":    "历史记录恢复功能暂未实现",
		"history_id": historyID,
	})
}

// SendSuccessResponse 发送成功响应
func SendSuccessResponse(w http.ResponseWriter, data interface{}) {
	apiLogger.Debug("API响应成功")
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 编码响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "编码响应失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// SendErrorResponse 发送错误响应
func SendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	apiLogger.Debug("API响应错误，状态码: %d, 错误信息: %s", statusCode, message)
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 设置状态码
	w.WriteHeader(statusCode)

	// 编码响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "编码响应失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
