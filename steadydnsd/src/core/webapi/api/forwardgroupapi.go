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

// core/webapi/forwardgroupapi.go

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"

	"github.com/gin-gonic/gin"
)

// ForwardGroupAPIHandler 处理转发组API请求
func ForwardGroupAPIHandler(c *gin.Context) {
	// 认证中间件已在路由中统一应用
	forwardGroupHandlerGin(c)
}

// forwardGroupHandlerGin 实际处理转发组请求的函数（Gin版本）
func forwardGroupHandlerGin(c *gin.Context) {
	// 获取路径参数
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 检查路径长度
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "forward-groups" {
		c.JSON(http.StatusNotFound, gin.H{"error": "无效的API端点"})
		return
	}

	switch len(parts) {
	case 2: // /api/forward-groups
		switch c.Request.Method {
		case http.MethodGet:
			getForwardGroupsGin(c)
			return
		case http.MethodPost:
			createForwardGroupGin(c)
			return
		case http.MethodDelete:
			if c.Request.URL.Query().Get("batch") == "true" {
				batchDeleteForwardGroupsGin(c)
				return
			}
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
			return
		default:
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
			return
		}
	case 3: // /api/forward-groups/{id}
		groupIDStr := parts[2]
		groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err != nil {
			// 检查是否为域名匹配测试端点
			if groupIDStr == "test-domain-match" {
				testDomainMatchHandlerGin(c)
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的转发组ID"})
			return
		}

		switch c.Request.Method {
		case http.MethodGet:
			getForwardGroupByIDGin(c, uint(groupID))
			return
		case http.MethodPut:
			updateForwardGroupGin(c, uint(groupID))
			return
		case http.MethodDelete:
			deleteForwardGroupGin(c, uint(groupID))
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

// getForwardGroupsGin 获取所有转发组（Gin版本）
func getForwardGroupsGin(c *gin.Context) {
	groups, err := database.GetForwardGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取转发组列表失败: %v", err)})
		return
	}

	// 确保即使没有数据也返回空数组而不是nil
	if groups == nil {
		groups = []database.ForwardGroup{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": groups, "message": "获取转发组列表成功"})
}

// createForwardGroupGin 创建转发组（Gin版本）
func createForwardGroupGin(c *gin.Context) {
	var group database.ForwardGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 验证输入
	if err := database.ValidateForwardGroupDB(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("验证失败: %v", err)})
		return
	}

	// 检查域名是否已存在
	existingGroup, err := database.GetForwardGroupByDomain(group.Domain)
	if err == nil && existingGroup != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "转发组域名已存在"})
		return
	}

	// 创建转发组
	if err := database.CreateForwardGroup(&group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建转发组失败: %v", err)})
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": group, "message": "转发组创建成功"})
}

// updateForwardGroupGin 更新转发组（Gin版本）
func updateForwardGroupGin(c *gin.Context, groupID uint) {
	var group database.ForwardGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 设置ID
	group.ID = groupID

	// 检查ID=1的默认转发组
	if groupID == 1 {
		// 获取当前转发组信息
		currentGroup, err := database.GetForwardGroupByID(groupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("获取转发组失败: %v", err)})
			return
		}
		// 检查是否尝试修改域名或描述
		if group.Domain != currentGroup.Domain || group.Description != currentGroup.Description {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID=1的转发组是默认组，域名和描述不可修改"})
			return
		}
	}

	// 验证输入
	if err := database.ValidateForwardGroupDB(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("验证失败: %v", err)})
		return
	}

	// 更新转发组
	if err := database.UpdateForwardGroup(&group); err != nil {
		if strings.Contains(err.Error(), "转发组不存在") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("更新转发组失败: %v", err)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新转发组失败: %v", err)})
		}
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": group, "message": "转发组更新成功"})
}

// deleteForwardGroupGin 删除转发组（Gin版本）
func deleteForwardGroupGin(c *gin.Context, groupID uint) {
	// 检查是否为ID=1的转发组，ID=1的转发组是默认组，不允许删除
	if groupID == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID=1的转发器组是默认组，不允许删除"})
		return
	}

	// 检查转发组是否存在
	_, err := database.GetForwardGroupByID(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "转发组不存在"})
		return
	}

	// 删除转发组
	if err := database.DeleteForwardGroup(groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除转发组失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "转发组删除成功"})
}

// getForwardGroupByIDGin 根据ID获取转发组（Gin版本）
func getForwardGroupByIDGin(c *gin.Context, groupID uint) {
	group, err := database.GetForwardGroupByID(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("获取转发组失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": group, "message": "获取转发组成功"})
}

// batchDeleteForwardGroupsGin 批量删除转发组（Gin版本）
func batchDeleteForwardGroupsGin(c *gin.Context) {
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID列表不能为空"})
		return
	}

	// 检查是否包含ID=1的转发组，ID=1的转发组是默认组，不允许删除
	for _, id := range ids {
		if id == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID=1的转发器组是默认组，不允许删除"})
			return
		}
	}

	if err := database.BatchDeleteForwardGroups(ids); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("批量删除转发组失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功删除 %d 个转发组", len(ids))})
}

// testDomainMatchHandlerGin 处理域名匹配测试请求（Gin版本）
func testDomainMatchHandlerGin(c *gin.Context) {
	// 只允许GET请求
	if c.Request.Method != http.MethodGet {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
		return
	}

	// 获取查询参数中的域名
	domain := c.Request.URL.Query().Get("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "域名参数不能为空"})
		return
	}

	// 调用DNS转发器的TestDomainMatch方法
	matchedGroup := sdns.GlobalDNSForwarder.TestDomainMatch(domain)

	// 返回匹配结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"domain":        domain,
			"matched_group": matchedGroup,
		},
		"message": "域名匹配测试成功",
	})
}
