// core/webapi/bindapi.go
// BIND权威域API接口

package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"SteadyDNS/core/bind"
	"SteadyDNS/core/common"
	"SteadyDNS/core/sdns"
)

// apiLogger API日志记录器
var apiLogger = common.NewLogger()

// bindManager 全局BIND管理器实例
var bindManager *bind.BindManager

// clearCacheAsync 异步清除与指定域名相关的所有缓存条目
func clearCacheAsync(domain string) {
	go func() {
		// 清除与域名相关的所有缓存
		sdns.ClearCacheByDomain(domain)
	}()
}

// initBindManager 初始化BIND管理器
func initBindManager() {
	if bindManager == nil {
		bindManager = bind.NewBindManager()
	}
}

// BindAPIHandlerGin BIND权威域API请求处理函数
func BindAPIHandlerGin(c *gin.Context) {
	// 记录API请求信息
	apiLogger.Debug("API请求: %s %s, 客户端IP: %s",
		c.Request.Method, c.Request.URL.Path, c.ClientIP())

	// 初始化BIND管理器
	initBindManager()

	// 获取路径参数
	path := c.Request.URL.Path
	method := c.Request.Method

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
			GetBindZonesGin(c)
		} else if domain == "history" {
			// 获取操作历史
			GetBindHistoryGin(c)
		} else {
			// 获取单个权威域
			GetBindZoneGin(c, domain)
		}
	case http.MethodPost:
		if domain == "" {
			// 创建权威域
			CreateBindZoneGin(c)
		} else if strings.HasSuffix(domain, "/reload") {
			// 刷新权威域
			domain = strings.TrimSuffix(domain, "/reload")
			ReloadBindZoneGin(c, domain)
		} else if strings.HasPrefix(domain, "history/") {
			// 恢复历史记录
			historyID := strings.TrimPrefix(domain, "history/")
			RestoreBindHistoryGin(c, historyID)
		} else {
			// 不支持的POST请求
			c.JSON(http.StatusMethodNotAllowed, gin.H{"success": false, "error": "不支持的请求"})
		}
	case http.MethodPut:
		if domain != "" {
			// 更新权威域
			UpdateBindZoneGin(c, domain)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "缺少域名参数"})
		}
	case http.MethodDelete:
		if domain != "" {
			// 删除权威域
			DeleteBindZoneGin(c, domain)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "缺少域名参数"})
		}
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"success": false, "error": "不支持的请求方法"})
	}
}

// GetBindZonesGin 获取所有权威域
func GetBindZonesGin(c *gin.Context) {
	apiLogger.Debug("获取所有权威域请求")
	// 初始化BIND管理器
	initBindManager()

	// 获取所有权威域
	zones, err := bindManager.GetAuthZones()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取权威域列表失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    zones,
	})
}

// GetBindZoneGin 获取单个权威域
func GetBindZoneGin(c *gin.Context, domain string) {
	apiLogger.Debug("获取单个权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 获取权威域
	zone, err := bindManager.GetAuthZone(domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "权威域不存在: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    zone,
	})
}

// CreateBindZoneGin 创建权威域
func CreateBindZoneGin(c *gin.Context) {
	apiLogger.Debug("创建权威域请求")
	// 初始化BIND管理器
	initBindManager()

	// 读取请求体
	var zone bind.AuthZone
	if err := c.ShouldBindJSON(&zone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 验证必填字段
	if zone.Domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "缺少域名参数",
		})
		return
	}

	// 创建权威域
	if err := bindManager.CreateAuthZone(zone); err != nil {
		errMsg := "创建权威域失败: " + err.Error()
		// 检查是否为CNAME冲突错误，如果是则返回400状态码
		if strings.Contains(err.Error(), "CNAME") || strings.Contains(err.Error(), "记录名称冲突") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   errMsg,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   errMsg,
			})
		}
		return
	}

	// 清除与域名相关的缓存
	clearCacheAsync(zone.Domain)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "权威域创建成功",
			"domain":  zone.Domain,
		},
	})
}

// UpdateBindZoneGin 更新权威域
func UpdateBindZoneGin(c *gin.Context, domain string) {
	apiLogger.Debug("更新权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 读取请求体
	var zone bind.AuthZone
	if err := c.ShouldBindJSON(&zone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 确保域名一致
	zone.Domain = domain

	// 更新权威域
	if err := bindManager.UpdateAuthZone(zone); err != nil {
		errMsg := "更新权威域失败: " + err.Error()
		// 检查是否为CNAME冲突错误，如果是则返回400状态码
		if strings.Contains(err.Error(), "CNAME") || strings.Contains(err.Error(), "记录名称冲突") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   errMsg,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   errMsg,
			})
		}
		return
	}

	// 清除与域名相关的缓存
	clearCacheAsync(zone.Domain)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "权威域更新成功",
			"domain":  zone.Domain,
		},
	})
}

// DeleteBindZoneGin 删除权威域
func DeleteBindZoneGin(c *gin.Context, domain string) {
	apiLogger.Debug("删除权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 删除权威域
	if err := bindManager.DeleteAuthZone(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "删除权威域失败: " + err.Error(),
		})
		return
	}

	// 清除与域名相关的缓存
	clearCacheAsync(domain)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "权威域删除成功",
			"domain":  domain,
		},
	})
}

// ReloadBindZoneGin 刷新权威域
func ReloadBindZoneGin(c *gin.Context, domain string) {
	apiLogger.Debug("刷新权威域请求，域名: %s", domain)
	// 初始化BIND管理器
	initBindManager()

	// 刷新BIND服务器
	if err := bindManager.ReloadBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "刷新BIND服务器失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "BIND服务器刷新成功",
			"domain":  domain,
		},
	})
}

// GetBindHistoryGin 获取操作历史
func GetBindHistoryGin(c *gin.Context) {
	apiLogger.Debug("获取操作历史请求")
	// 初始化BIND管理器
	initBindManager()

	// 获取所有历史记录
	history, _ := bindManager.HistoryMgr.GetHistoryRecords()

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// RestoreBindHistoryGin 恢复历史记录
func RestoreBindHistoryGin(c *gin.Context, historyID string) {
	apiLogger.Debug("恢复历史记录请求，历史ID: %s", historyID)
	// 初始化BIND管理器
	initBindManager()

	// 返回成功响应（暂时不实现具体的恢复逻辑）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message":    "历史记录恢复功能暂未实现",
			"history_id": historyID,
		},
	})
}
