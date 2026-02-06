// core/webapi/forwardgroupapi.go

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"SteadyDNS/core/database"
	"SteadyDNS/core/sdns"
	"SteadyDNS/core/webapi/middleware"
)

// ForwardGroupAPIHandler 处理转发组API请求
func ForwardGroupAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(forwardGroupHandler)
	authHandler(w, r)
}

// forwardGroupHandler 实际处理转发组请求的函数
func forwardGroupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取路径参数
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 检查路径长度
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "forward-groups" {
		middleware.SendErrorResponse(w, "无效的API端点", http.StatusNotFound)
		return
	}

	switch len(parts) {
	case 2: // /api/forward-groups
		switch r.Method {
		case http.MethodGet:
			getForwardGroups(w)
			return
		case http.MethodPost:
			createForwardGroup(w, r)
			return
		case http.MethodDelete:
			if r.URL.Query().Get("batch") == "true" {
				batchDeleteForwardGroups(w, r)
				return
			}
			middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		default:
			middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}
	case 3: // /api/forward-groups/{id}
		groupIDStr := parts[2]
		groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err != nil {
			// 检查是否为域名匹配测试端点
			if groupIDStr == "test-domain-match" {
				testDomainMatchHandler(w, r)
				return
			}
			middleware.SendErrorResponse(w, "无效的转发组ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			getForwardGroupByID(w, uint(groupID))
			return
		case http.MethodPut:
			updateForwardGroup(w, r, uint(groupID))
			return
		case http.MethodDelete:
			deleteForwardGroup(w, uint(groupID))
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

// getForwardGroups 获取所有转发组
func getForwardGroups(w http.ResponseWriter) {
	groups, err := database.GetForwardGroups()
	if err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("获取转发组列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 确保即使没有数据也返回空数组而不是nil
	if groups == nil {
		groups = []database.ForwardGroup{}
	}

	middleware.SendSuccessResponse(w, groups, "获取转发组列表成功")
}

// createForwardGroup 创建转发组
func createForwardGroup(w http.ResponseWriter, r *http.Request) {
	var group database.ForwardGroup
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&group); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证输入
	if err := database.ValidateForwardGroupDB(&group); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("验证失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查域名是否已存在
	existingGroup, err := database.GetForwardGroupByDomain(group.Domain)
	if err == nil && existingGroup != nil {
		middleware.SendErrorResponse(w, "转发组域名已存在", http.StatusBadRequest)
		return
	}

	// 创建转发组
	if err := database.CreateForwardGroup(&group); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("创建转发组失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	middleware.SendSuccessResponse(w, group, "转发组创建成功")
}

// updateForwardGroup 更新转发组
func updateForwardGroup(w http.ResponseWriter, r *http.Request, groupID uint) {
	var group database.ForwardGroup
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&group); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 设置ID
	group.ID = groupID

	// 检查ID=1的默认转发组
	if groupID == 1 {
		// 获取当前转发组信息
		currentGroup, err := database.GetForwardGroupByID(groupID)
		if err != nil {
			middleware.SendErrorResponse(w, fmt.Sprintf("获取转发组失败: %v", err), http.StatusNotFound)
			return
		}
		// 检查是否尝试修改域名或描述
		if group.Domain != currentGroup.Domain || group.Description != currentGroup.Description {
			middleware.SendErrorResponse(w, "ID=1的转发组是默认组，域名和描述不可修改", http.StatusBadRequest)
			return
		}
	}

	// 验证输入
	if err := database.ValidateForwardGroupDB(&group); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("验证失败: %v", err), http.StatusBadRequest)
		return
	}

	// 更新转发组
	if err := database.UpdateForwardGroup(&group); err != nil {
		if strings.Contains(err.Error(), "转发组不存在") {
			middleware.SendErrorResponse(w, fmt.Sprintf("更新转发组失败: %v", err), http.StatusNotFound)
		} else {
			middleware.SendErrorResponse(w, fmt.Sprintf("更新转发组失败: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// 刷新DNS转发器配置
	if err := sdns.ReloadForwardGroups(); err != nil {
		fmt.Printf("刷新DNS转发器配置失败: %v\n", err)
	}

	middleware.SendSuccessResponse(w, group, "转发组更新成功")
}

// deleteForwardGroup 删除转发组
func deleteForwardGroup(w http.ResponseWriter, groupID uint) {
	// 检查是否为ID=1的转发组，ID=1的转发组是默认组，不允许删除
	if groupID == 1 {
		middleware.SendErrorResponse(w, "ID=1的转发器组是默认组，不允许删除", http.StatusBadRequest)
		return
	}

	// 检查转发组是否存在
	_, err := database.GetForwardGroupByID(groupID)
	if err != nil {
		middleware.SendErrorResponse(w, "转发组不存在", http.StatusNotFound)
		return
	}

	// 删除转发组
	if err := database.DeleteForwardGroup(groupID); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("删除转发组失败: %v", err), http.StatusInternalServerError)
		return
	}

	middleware.SendSuccessResponse(w, nil, "转发组删除成功")
}

// getForwardGroupByID 根据ID获取转发组
func getForwardGroupByID(w http.ResponseWriter, groupID uint) {
	group, err := database.GetForwardGroupByID(groupID)
	if err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("获取转发组失败: %v", err), http.StatusNotFound)
		return
	}

	middleware.SendSuccessResponse(w, group, "获取转发组成功")
}

// batchDeleteForwardGroups 批量删除转发组
func batchDeleteForwardGroups(w http.ResponseWriter, r *http.Request) {
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

	// 检查是否包含ID=1的转发组，ID=1的转发组是默认组，不允许删除
	for _, id := range ids {
		if id == 1 {
			middleware.SendErrorResponse(w, "ID=1的转发器组是默认组，不允许删除", http.StatusBadRequest)
			return
		}
	}

	if err := database.BatchDeleteForwardGroups(ids); err != nil {
		middleware.SendErrorResponse(w, fmt.Sprintf("批量删除转发组失败: %v", err), http.StatusInternalServerError)
		return
	}

	middleware.SendSuccessResponse(w, nil, fmt.Sprintf("成功删除 %d 个转发组", len(ids)))
}

// testDomainMatchHandler 处理域名匹配测试请求
func testDomainMatchHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许GET请求
	if r.Method != http.MethodGet {
		middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数中的域名
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		middleware.SendErrorResponse(w, "域名参数不能为空", http.StatusBadRequest)
		return
	}

	// 调用DNS转发器的TestDomainMatch方法
	matchedGroup := sdns.GlobalDNSForwarder.TestDomainMatch(domain)

	// 返回匹配结果
	middleware.SendSuccessResponse(w, map[string]interface{}{
		"domain":        domain,
		"matched_group": matchedGroup,
	}, "域名匹配测试成功")
}
