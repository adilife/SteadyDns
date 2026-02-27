/*
SteadyDNS - DNS服务器实现

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// core/webapi/bind_server_api.go

package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"SteadyDNS/core/bind"
	"SteadyDNS/core/bind/namedconf"

	"github.com/gin-gonic/gin"
)

// BindServerAPIHandlerGin 处理BIND服务器管理API请求（Gin版本）
func BindServerAPIHandlerGin(c *gin.Context) {
	// 认证中间件已在路由中统一应用
	bindServerHandlerGin(c)
}

// bindServerHandlerGin BIND服务器管理API处理函数（Gin版本）
func bindServerHandlerGin(c *gin.Context) {
	// 获取BIND管理器实例
	bindManager := bind.NewBindManager()

	// 解析请求路径
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/bind-server/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "status" && c.Request.Method == http.MethodGet:
		// 获取BIND服务器状态
		handleBindServerStatusGin(c, bindManager)

	case path == "start" && c.Request.Method == http.MethodPost:
		// 启动BIND服务器
		handleBindServerActionGin(c, bindManager, "start")

	case path == "stop" && c.Request.Method == http.MethodPost:
		// 停止BIND服务器
		handleBindServerActionGin(c, bindManager, "stop")

	case path == "restart" && c.Request.Method == http.MethodPost:
		// 重启BIND服务器
		handleBindServerActionGin(c, bindManager, "restart")

	case path == "reload" && c.Request.Method == http.MethodPost:
		// 重载BIND服务器
		handleBindServerActionGin(c, bindManager, "reload")

	case path == "stats" && c.Request.Method == http.MethodGet:
		// 获取BIND服务器统计信息
		handleBindServerStatsGin(c, bindManager)

	case path == "health" && c.Request.Method == http.MethodGet:
		// 检查BIND服务健康状态
		handleBindServerHealthGin(c, bindManager)

	case path == "validate" && c.Request.Method == http.MethodPost:
		// 验证BIND配置
		handleBindServerValidateGin(c, bindManager)

	case path == "config" && c.Request.Method == http.MethodGet:
		// 获取BIND配置
		handleBindServerConfigGin(c, bindManager)

	case path == "named-conf/content" && c.Request.Method == http.MethodGet:
		// 获取 named.conf 文件内容
		handleGetNamedConfContentGin(c, bindManager)

	case path == "named-conf" && c.Request.Method == http.MethodPut:
		// 更新 named.conf 文件内容
		handleUpdateNamedConfContentGin(c, bindManager)

	case path == "named-conf/validate" && c.Request.Method == http.MethodPost:
		// 验证 named.conf 配置内容
		handleValidateNamedConfContentGin(c, bindManager)

	case path == "named-conf/diff" && c.Request.Method == http.MethodPost:
		// 获取配置差异
		handleDiffNamedConfGin(c, bindManager)

	case path == "named-conf/parse" && c.Request.Method == http.MethodGet:
		// 解析 named.conf 配置结构
		handleParseNamedConfGin(c, bindManager)

	case path == "named-conf/backups" && c.Request.Method == http.MethodGet:
		// 列出所有 named.conf 备份
		handleListNamedConfBackupsGin(c, bindManager)

	case path == "named-conf/restore" && c.Request.Method == http.MethodPost:
		// 从指定备份恢复配置
		handleRestoreNamedConfBackupGin(c, bindManager)

	case strings.HasPrefix(path, "named-conf/backups/") && c.Request.Method == http.MethodDelete:
		// 删除指定备份文件
		backupId := strings.TrimPrefix(path, "named-conf/backups/")
		handleDeleteNamedConfBackupGin(c, bindManager, backupId)

	default:
		// 未找到的API端点
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	}
}

// handleBindServerStatusGin 处理获取BIND服务器状态的请求（Gin版本）
func handleBindServerStatusGin(c *gin.Context, bindManager *bind.BindManager) {
	// 获取BIND服务器状态
	status, err := bindManager.GetBindStatus()

	// 处理错误
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

// handleBindServerActionGin 处理BIND服务器操作的请求（Gin版本）
func handleBindServerActionGin(c *gin.Context, bindManager *bind.BindManager, action string) {
	var err error

	// 根据操作类型执行相应的操作
	switch action {
	case "start":
		err = bindManager.StartBind()
	case "stop":
		err = bindManager.StopBind()
	case "restart":
		err = bindManager.RestartBind()
	case "reload":
		err = bindManager.ReloadBind()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	// 处理错误
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
		"message": "BIND server " + action + "ed successfully",
	})
}

// handleBindServerStatsGin 处理获取BIND服务器统计信息的请求（Gin版本）
func handleBindServerStatsGin(c *gin.Context, bindManager *bind.BindManager) {
	// 获取BIND服务器统计信息
	stats, err := bindManager.GetBindStats()

	// 处理错误
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

// handleBindServerHealthGin 处理检查BIND服务健康状态的请求（Gin版本）
func handleBindServerHealthGin(c *gin.Context, bindManager *bind.BindManager) {
	// 检查BIND服务健康状态
	health, err := bindManager.CheckBindHealth()

	// 处理错误
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

// handleBindServerValidateGin 处理验证BIND配置的请求（Gin版本）
func handleBindServerValidateGin(c *gin.Context, bindManager *bind.BindManager) {
	// 验证BIND配置
	err := bindManager.ValidateConfig()

	// 处理错误
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
		"message": "BIND configuration validated successfully",
	})
}

// handleBindServerConfigGin 处理获取BIND配置的请求（Gin版本）
func handleBindServerConfigGin(c *gin.Context, bindManager *bind.BindManager) {
	// 获取BIND配置
	config, err := bindManager.GetBindConfig()

	// 处理错误
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

// handleGetNamedConfContentGin 处理获取 named.conf 文件内容的请求（Gin版本）
func handleGetNamedConfContentGin(c *gin.Context, bindManager *bind.BindManager) {
	// 获取 named.conf 文件内容
	content, err := bindManager.GetNamedConfContent()

	// 处理错误
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回 JSON 响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"content": content,
		},
	})
}

// handleUpdateNamedConfContentGin 处理更新 named.conf 文件内容的请求（Gin版本）
func handleUpdateNamedConfContentGin(c *gin.Context, bindManager *bind.BindManager) {
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

	// 更新 named.conf 文件内容
	if err := bindManager.UpdateNamedConfContent(request.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "named.conf 配置文件更新成功",
	})
}

// handleValidateNamedConfContentGin 处理验证 named.conf 配置内容的请求（Gin版本）
func handleValidateNamedConfContentGin(c *gin.Context, bindManager *bind.BindManager) {
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
	validationResult, err := bindManager.ValidateNamedConfContent(request.Content)
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

// handleDiffNamedConfGin 处理获取配置差异的请求（Gin版本）
func handleDiffNamedConfGin(c *gin.Context, bindManager *bind.BindManager) {
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
	diffResult := bindManager.DiffNamedConf(request.OldContent, request.NewContent)

	// 打印差异结果
	fmt.Printf("接口返回差异结果: unchanged %d, added %d, removed %d, total %d\n",
		diffResult.Stats.Unchanged, diffResult.Stats.Added, diffResult.Stats.Removed, diffResult.Stats.Total)

	// 返回差异结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    diffResult,
	})
}

// handleParseNamedConfGin 处理解析 named.conf 配置结构的请求（Gin版本）
func handleParseNamedConfGin(c *gin.Context, bindManager *bind.BindManager) {
	// 解析 named.conf 文件
	config, err := bindManager.ParseNamedConf()

	// 处理错误
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

// handleListNamedConfBackupsGin 处理列出 named.conf 备份的请求（Gin版本）
func handleListNamedConfBackupsGin(c *gin.Context, bindManager *bind.BindManager) {
	// 获取 named.conf 文件路径
	namedConfPath := bindManager.GetNamedConfPath()

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

// handleRestoreNamedConfBackupGin 处理从备份恢复 named.conf 配置的请求（Gin版本）
// 注意：此方法调用 ReloadBind()，该方法内部会获取 bm.mu 锁。
// 请勿在此方法中添加锁，以避免死锁。
func handleRestoreNamedConfBackupGin(c *gin.Context, bindManager *bind.BindManager) {
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

	// 获取 named.conf 文件路径
	namedConfPath := bindManager.GetNamedConfPath()

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

	// 重载 BIND 服务使配置生效
	if err := bindManager.ReloadBind(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "恢复备份成功，但重载 BIND 服务失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "从备份恢复 named.conf 配置成功",
	})
}

// handleDeleteNamedConfBackupGin 处理删除 named.conf 备份的请求（Gin版本）
func handleDeleteNamedConfBackupGin(c *gin.Context, bindManager *bind.BindManager, backupId string) {
	// 验证备份ID
	if backupId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "备份ID不能为空",
		})
		return
	}

	// 构建备份文件路径
	backupPath := "./backup/" + backupId

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
