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

// core/webapi/cacheapi.go

package api

import (
	"net/http"
	"strings"

	"SteadyDNS/core/sdns"

	"github.com/gin-gonic/gin"
)

// CacheAPIHandlerGin 处理缓存相关的API请求
func CacheAPIHandlerGin(c *gin.Context) {
	// 认证中间件已在路由中统一应用
	cacheHandlerGin(c)
}

// cacheHandlerGin 缓存管理API处理函数
func cacheHandlerGin(c *gin.Context) {
	// 解析请求路径
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/cache/")
	pathParts := strings.Split(path, "/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "stats" && c.Request.Method == http.MethodGet:
		// 获取缓存统计信息
		handleGetCacheStatsGin(c)

	case path == "clear" && c.Request.Method == http.MethodPost:
		// 清空缓存
		handleClearCacheGin(c)

	case strings.HasPrefix(path, "clear/") && c.Request.Method == http.MethodPost:
		// 按域名清除缓存
		domain := pathParts[1]
		handleClearCacheByDomainGin(c, domain)

	default:
		// 未找到的API端点
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	}
}

// handleGetCacheStatsGin 处理获取缓存统计信息的请求
func handleGetCacheStatsGin(c *gin.Context) {
	// 获取缓存统计信息
	stats := sdns.GetCacheStats()

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// handleClearCacheGin 处理清空缓存的请求
func handleClearCacheGin(c *gin.Context) {
	// 执行清空缓存操作
	err := sdns.ClearCache()

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
		"message": "Cache cleared successfully",
	})
}

// handleClearCacheByDomainGin 处理按域名清除缓存的请求
func handleClearCacheByDomainGin(c *gin.Context, domain string) {
	// 执行按域名清除缓存操作
	err := sdns.ClearCacheByDomain(domain)

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
		"message": "Cache cleared successfully for domain: " + domain,
		"domain":  domain,
	})
}
