// core/plugin/plugins/bind_plugin.go
//
// SteadyDNS BIND Plugin Implementation
// Copyright (C) 2024 SteadyDNS Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package plugins

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"SteadyDNS/core/bind"
	"SteadyDNS/core/bind/namedconf"
	"SteadyDNS/core/common"
	"SteadyDNS/core/plugin"
	"SteadyDNS/core/sdns"
)

// BindPlugin BIND权威域管理插件
// 提供权威域管理、BIND服务器管理、转发查询和备份功能
type BindPlugin struct {
	// bindManager BIND管理器实例
	bindManager *bind.BindManager
	// logger 日志记录器
	logger *common.Logger
	// initialized 插件是否已初始化
	initialized bool
}

// NewBindPlugin 创建BIND插件实例
// 返回值: BIND插件实例指针
func NewBindPlugin() *BindPlugin {
	return &BindPlugin{
		logger:      common.NewLogger(),
		initialized: false,
	}
}

// Name 返回插件的唯一标识名称
// 返回值: 插件名称字符串 "bind"
func (p *BindPlugin) Name() string {
	return "bind"
}

// Description 返回插件的功能描述
// 返回值: 插件描述字符串
func (p *BindPlugin) Description() string {
	return "BIND权威域管理插件 - 提供权威域管理、BIND服务器管理、转发查询和备份功能"
}

// Version 返回插件的版本号
// 返回值: 版本号字符串 "1.0.0"
func (p *BindPlugin) Version() string {
	return "1.0.0"
}

// Initialize 初始化插件
// 创建BIND管理器实例并执行必要的初始化操作
// 返回值: 初始化错误信息，nil表示成功
func (p *BindPlugin) Initialize() error {
	p.logger.Info("初始化BIND插件...")

	// 创建BIND管理器实例
	p.bindManager = bind.NewBindManager()
	if p.bindManager == nil {
		return fmt.Errorf("创建BIND管理器实例失败")
	}

	p.initialized = true
	p.logger.Info("BIND插件初始化完成")
	return nil
}

// Shutdown 关闭插件
// 清理插件资源
// 返回值: 关闭错误信息，nil表示成功
func (p *BindPlugin) Shutdown() error {
	p.logger.Info("关闭BIND插件...")

	// 清理资源
	p.bindManager = nil
	p.initialized = false

	p.logger.Info("BIND插件已关闭")
	return nil
}

// Routes 返回插件提供的HTTP路由定义列表
// 返回值: 路由定义切片
func (p *BindPlugin) Routes() []plugin.RouteDefinition {
	return []plugin.RouteDefinition{
		// ==================== 权威域管理路由 ====================
		{
			Method:       "GET",
			Path:         "/api/bind-zones",
			Handler:      p.handleGetBindZones,
			Description:  "获取所有权威域列表",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "GET",
			Path:         "/api/bind-zones/history",
			Handler:      p.handleGetBindHistory,
			Description:  "获取操作历史记录",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "GET",
			Path:         "/api/bind-zones/:domain",
			Handler:      p.handleGetBindZone,
			Description:  "获取单个权威域详情",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-zones",
			Handler:      p.handleCreateBindZone,
			Description:  "创建权威域",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-zones/:domain/reload",
			Handler:      p.handleReloadBindZone,
			Description:  "刷新权威域配置",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-zones/history/:id/restore",
			Handler:      p.handleRestoreBindHistory,
			Description:  "恢复历史记录",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "PUT",
			Path:         "/api/bind-zones/:domain",
			Handler:      p.handleUpdateBindZone,
			Description:  "更新权威域",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "DELETE",
			Path:         "/api/bind-zones/:domain",
			Handler:      p.handleDeleteBindZone,
			Description:  "删除权威域",
			AuthRequired: true,
			Middlewares:  nil,
		},

		// ==================== BIND服务器管理路由 ====================
		{
			Method:       "GET",
			Path:         "/api/bind-server/status",
			Handler:      p.handleBindServerStatus,
			Description:  "获取BIND服务器状态",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/start",
			Handler:      p.handleBindServerStart,
			Description:  "启动BIND服务器",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/stop",
			Handler:      p.handleBindServerStop,
			Description:  "停止BIND服务器",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/restart",
			Handler:      p.handleBindServerRestart,
			Description:  "重启BIND服务器",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/reload",
			Handler:      p.handleBindServerReload,
			Description:  "重载BIND服务器配置",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "GET",
			Path:         "/api/bind-server/stats",
			Handler:      p.handleBindServerStats,
			Description:  "获取BIND服务器统计信息",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "GET",
			Path:         "/api/bind-server/health",
			Handler:      p.handleBindServerHealth,
			Description:  "检查BIND服务健康状态",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/validate",
			Handler:      p.handleBindServerValidate,
			Description:  "验证BIND配置",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "GET",
			Path:         "/api/bind-server/config",
			Handler:      p.handleBindServerConfig,
			Description:  "获取BIND配置信息",
			AuthRequired: true,
			Middlewares:  nil,
		},

		// ==================== named.conf 配置管理路由 ====================
		{
			Method:       "GET",
			Path:         "/api/bind-server/named-conf/content",
			Handler:      p.handleGetNamedConfContent,
			Description:  "获取named.conf文件内容",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "PUT",
			Path:         "/api/bind-server/named-conf",
			Handler:      p.handleUpdateNamedConfContent,
			Description:  "更新named.conf文件内容",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/named-conf/validate",
			Handler:      p.handleValidateNamedConfContent,
			Description:  "验证named.conf配置内容",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/named-conf/diff",
			Handler:      p.handleDiffNamedConf,
			Description:  "获取配置差异",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "GET",
			Path:         "/api/bind-server/named-conf/parse",
			Handler:      p.handleParseNamedConf,
			Description:  "解析named.conf配置结构",
			AuthRequired: true,
			Middlewares:  nil,
		},

		// ==================== named.conf 备份管理路由 ====================
		{
			Method:       "GET",
			Path:         "/api/bind-server/named-conf/backups",
			Handler:      p.handleListNamedConfBackups,
			Description:  "列出所有named.conf备份",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "POST",
			Path:         "/api/bind-server/named-conf/restore",
			Handler:      p.handleRestoreNamedConfBackup,
			Description:  "从备份恢复named.conf配置",
			AuthRequired: true,
			Middlewares:  nil,
		},
		{
			Method:       "DELETE",
			Path:         "/api/bind-server/named-conf/backups/:id",
			Handler:      p.handleDeleteNamedConfBackup,
			Description:  "删除指定备份文件",
			AuthRequired: true,
			Middlewares:  nil,
		},
	}
}

// ==================== 权威域管理处理函数 ====================

// handleGetBindZones 处理获取所有权威域的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleGetBindZones(c *gin.Context) {
	p.logger.Debug("获取所有权威域请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取所有权威域
	zones, err := p.bindManager.GetAuthZones()
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

// handleGetBindZone 处理获取单个权威域的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleGetBindZone(c *gin.Context) {
	domain := c.Param("domain")
	p.logger.Debug("获取单个权威域请求，域名: %s", domain)

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取权威域
	zone, err := p.bindManager.GetAuthZone(domain)
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

// handleCreateBindZone 处理创建权威域的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleCreateBindZone(c *gin.Context) {
	p.logger.Debug("创建权威域请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

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
	if err := p.bindManager.CreateAuthZone(zone); err != nil {
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

	// 刷新权威域转发列表
	if sdns.GlobalDNSForwarder != nil {
		if err := sdns.GlobalDNSForwarder.GetAuthorityForwarder().ReloadAuthorityZones(); err != nil {
			p.logger.Warn("刷新权威域转发列表失败: %v", err)
		}
	}

	// 清除与域名相关的缓存
	p.clearCacheAsync(zone.Domain)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "权威域创建成功",
			"domain":  zone.Domain,
		},
	})
}

// handleUpdateBindZone 处理更新权威域的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleUpdateBindZone(c *gin.Context) {
	domain := c.Param("domain")
	p.logger.Debug("更新权威域请求，域名: %s", domain)

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

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
	if err := p.bindManager.UpdateAuthZone(zone); err != nil {
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

	// 刷新权威域转发列表
	if sdns.GlobalDNSForwarder != nil {
		if err := sdns.GlobalDNSForwarder.GetAuthorityForwarder().ReloadAuthorityZones(); err != nil {
			p.logger.Warn("刷新权威域转发列表失败: %v", err)
		}
	}

	// 清除与域名相关的缓存
	p.clearCacheAsync(zone.Domain)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "权威域更新成功",
			"domain":  zone.Domain,
		},
	})
}

// handleDeleteBindZone 处理删除权威域的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleDeleteBindZone(c *gin.Context) {
	domain := c.Param("domain")
	p.logger.Debug("删除权威域请求，域名: %s", domain)

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 删除权威域
	if err := p.bindManager.DeleteAuthZone(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "删除权威域失败: " + err.Error(),
		})
		return
	}

	// 刷新权威域转发列表
	if sdns.GlobalDNSForwarder != nil {
		if err := sdns.GlobalDNSForwarder.GetAuthorityForwarder().ReloadAuthorityZones(); err != nil {
			p.logger.Warn("刷新权威域转发列表失败: %v", err)
		}
	}

	// 清除与域名相关的缓存
	p.clearCacheAsync(domain)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "权威域删除成功",
			"domain":  domain,
		},
	})
}

// handleReloadBindZone 处理刷新权威域的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleReloadBindZone(c *gin.Context) {
	domain := c.Param("domain")
	p.logger.Debug("刷新权威域请求，域名: %s", domain)

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 刷新BIND服务器
	if err := p.bindManager.ReloadBind(); err != nil {
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

// handleGetBindHistory 处理获取操作历史的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleGetBindHistory(c *gin.Context) {
	p.logger.Debug("获取操作历史请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取格式化的历史记录
	history, err := p.bindManager.HistoryMgr.GetHistoryRecordsForAPI()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取历史记录失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// handleRestoreBindHistory 处理恢复历史记录的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleRestoreBindHistory(c *gin.Context) {
	historyID := c.Param("id")
	p.logger.Debug("恢复历史记录请求，历史ID: %s", historyID)

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 解析历史记录ID
	recordID, err := strconv.ParseUint(historyID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的历史记录ID",
		})
		return
	}

	// 调用恢复功能
	if err := p.bindManager.HistoryMgr.RestoreBackup(recordID); err != nil {
		p.logger.Error("恢复历史记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("恢复历史记录失败: %v", err),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message":    "历史记录恢复成功",
			"history_id": historyID,
		},
	})
}

// ==================== BIND服务器管理处理函数 ====================

// handleBindServerStatus 处理获取BIND服务器状态的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerStatus(c *gin.Context) {
	p.logger.Debug("获取BIND服务器状态请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取BIND服务器状态
	status, err := p.bindManager.GetBindStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"status": status,
		},
	})
}

// handleBindServerStart 处理启动BIND服务器的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerStart(c *gin.Context) {
	p.logger.Debug("启动BIND服务器请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 启动BIND服务器
	if err := p.bindManager.StartBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "BIND服务器启动成功",
	})
}

// handleBindServerStop 处理停止BIND服务器的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerStop(c *gin.Context) {
	p.logger.Debug("停止BIND服务器请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 停止BIND服务器
	if err := p.bindManager.StopBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "BIND服务器停止成功",
	})
}

// handleBindServerRestart 处理重启BIND服务器的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerRestart(c *gin.Context) {
	p.logger.Debug("重启BIND服务器请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 重启BIND服务器
	if err := p.bindManager.RestartBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "BIND服务器重启成功",
	})
}

// handleBindServerReload 处理重载BIND服务器的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerReload(c *gin.Context) {
	p.logger.Debug("重载BIND服务器请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 重载BIND服务器
	if err := p.bindManager.ReloadBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "BIND服务器重载成功",
	})
}

// handleBindServerStats 处理获取BIND服务器统计信息的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerStats(c *gin.Context) {
	p.logger.Debug("获取BIND服务器统计信息请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取BIND服务器统计信息
	stats, err := p.bindManager.GetBindStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// handleBindServerHealth 处理检查BIND服务健康状态的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerHealth(c *gin.Context) {
	p.logger.Debug("检查BIND服务健康状态请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 检查BIND服务健康状态
	health, err := p.bindManager.CheckBindHealth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    health,
	})
}

// handleBindServerValidate 处理验证BIND配置的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerValidate(c *gin.Context) {
	p.logger.Debug("验证BIND配置请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 验证BIND配置
	if err := p.bindManager.ValidateConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "BIND配置验证成功",
	})
}

// handleBindServerConfig 处理获取BIND配置的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleBindServerConfig(c *gin.Context) {
	p.logger.Debug("获取BIND配置请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取BIND配置
	config, err := p.bindManager.GetBindConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// ==================== named.conf 配置管理处理函数 ====================

// handleGetNamedConfContent 处理获取named.conf文件内容的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleGetNamedConfContent(c *gin.Context) {
	p.logger.Debug("获取named.conf文件内容请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取named.conf文件内容
	content, err := p.bindManager.GetNamedConfContent()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"content": content,
		},
	})
}

// handleUpdateNamedConfContent 处理更新named.conf文件内容的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleUpdateNamedConfContent(c *gin.Context) {
	p.logger.Debug("更新named.conf文件内容请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 解析请求体
	var request struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求体",
		})
		return
	}

	// 验证输入
	if request.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "配置内容不能为空",
		})
		return
	}

	// 更新named.conf文件内容
	if err := p.bindManager.UpdateNamedConfContent(request.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "named.conf配置文件更新成功",
	})
}

// handleValidateNamedConfContent 处理验证named.conf配置内容的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleValidateNamedConfContent(c *gin.Context) {
	p.logger.Debug("验证named.conf配置内容请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 解析请求体
	var request struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求体",
		})
		return
	}

	// 验证输入
	if request.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "配置内容不能为空",
		})
		return
	}

	// 验证配置内容
	validationResult, err := p.bindManager.ValidateNamedConfContent(request.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回验证结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    validationResult,
	})
}

// handleDiffNamedConf 处理获取配置差异的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleDiffNamedConf(c *gin.Context) {
	p.logger.Debug("获取配置差异请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 解析请求体
	var request struct {
		OldContent string `json:"oldContent"`
		NewContent string `json:"newContent"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求体",
		})
		return
	}

	// 验证输入
	if request.OldContent == "" || request.NewContent == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "旧内容和新内容不能为空",
		})
		return
	}

	// 获取配置差异
	diffResult := p.bindManager.DiffNamedConf(request.OldContent, request.NewContent)

	// 打印差异结果
	p.logger.Debug("接口返回差异结果: unchanged %d, added %d, removed %d, total %d",
		diffResult.Stats.Unchanged, diffResult.Stats.Added, diffResult.Stats.Removed, diffResult.Stats.Total)

	// 返回差异结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    diffResult,
	})
}

// handleParseNamedConf 处理解析named.conf配置结构的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleParseNamedConf(c *gin.Context) {
	p.logger.Debug("解析named.conf配置结构请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 解析named.conf文件
	config, err := p.bindManager.ParseNamedConf()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回解析结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// ==================== named.conf 备份管理处理函数 ====================

// handleListNamedConfBackups 处理列出named.conf备份的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleListNamedConfBackups(c *gin.Context) {
	p.logger.Debug("列出named.conf备份请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 获取named.conf文件路径
	namedConfPath := p.bindManager.GetNamedConfPath()

	// 创建备份管理器实例
	backupManager := namedconf.NewBackupManager("./backup", 10)

	// 获取备份列表
	backups, err := backupManager.ListBackups(namedConfPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回备份列表
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    backups,
	})
}

// handleRestoreNamedConfBackup 处理从备份恢复named.conf配置的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleRestoreNamedConfBackup(c *gin.Context) {
	p.logger.Debug("从备份恢复named.conf配置请求")

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 解析请求体
	var request struct {
		BackupPath string `json:"backupPath"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求体",
		})
		return
	}

	// 验证输入
	if request.BackupPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "备份文件路径不能为空",
		})
		return
	}

	// 获取named.conf文件路径
	namedConfPath := p.bindManager.GetNamedConfPath()

	// 创建备份管理器实例
	backupManager := namedconf.NewBackupManager("./backup", 10)

	// 恢复备份
	if err := backupManager.RestoreBackup(request.BackupPath, namedConfPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 重载BIND服务使配置生效
	if err := p.bindManager.ReloadBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "恢复备份成功，但重载BIND服务失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "从备份恢复named.conf配置成功",
	})
}

// handleDeleteNamedConfBackup 处理删除named.conf备份的请求
// 参数:
//   - c: Gin上下文
func (p *BindPlugin) handleDeleteNamedConfBackup(c *gin.Context) {
	backupID := c.Param("id")
	p.logger.Debug("删除named.conf备份请求，备份ID: %s", backupID)

	// 检查插件是否已初始化
	if !p.initialized || p.bindManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "BIND插件未初始化",
		})
		return
	}

	// 验证备份ID
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "备份ID不能为空",
		})
		return
	}

	// 构建备份文件路径
	backupPath := "./backup/" + backupID

	// 检查文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "备份文件不存在",
		})
		return
	}

	// 删除备份文件
	if err := os.Remove(backupPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "删除备份文件失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除备份文件成功",
	})
}

// ==================== 辅助函数 ====================

// clearCacheAsync 异步清除与指定域名相关的所有缓存条目
// 参数:
//   - domain: 域名
func (p *BindPlugin) clearCacheAsync(domain string) {
	go func() {
		// 清除与域名相关的所有缓存
		sdns.ClearCacheByDomain(domain)
	}()
}

// 确保BindPlugin实现了Plugin接口
var _ plugin.Plugin = (*BindPlugin)(nil)
