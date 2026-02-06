// core/webapi/cacheapi.go

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"SteadyDNS/core/sdns"
)

// CacheAPIHandler 处理缓存相关的API请求
func CacheAPIHandler(w http.ResponseWriter, r *http.Request) {
	// 应用认证中间件
	authHandler := AuthMiddleware(cacheHandler)
	authHandler(w, r)
}

// cacheHandler 缓存管理API处理函数
func cacheHandler(w http.ResponseWriter, r *http.Request) {
	// 获取服务器管理器实例

	// 解析请求路径
	path := strings.TrimPrefix(r.URL.Path, "/api/cache/")
	pathParts := strings.Split(path, "/")

	// 根据请求路径和方法处理不同的API端点
	switch {
	case path == "stats" && r.Method == http.MethodGet:
		// 获取缓存统计信息
		handleGetCacheStats(w, r)

	case path == "clear" && r.Method == http.MethodPost:
		// 清空缓存
		handleClearCache(w, r)

	case strings.HasPrefix(path, "clear/") && r.Method == http.MethodPost:
		// 按域名清除缓存
		domain := pathParts[1]
		handleClearCacheByDomain(w, r, domain)

	default:
		// 未找到的API端点
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// handleGetCacheStats 处理获取缓存统计信息的请求
func handleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	// 获取缓存统计信息
	stats := sdns.GetCacheStats()

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// handleClearCache 处理清空缓存的请求
func handleClearCache(w http.ResponseWriter, r *http.Request) {
	// 执行清空缓存操作
	err := sdns.ClearCache()

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cache cleared successfully",
	})
}

// handleClearCacheByDomain 处理按域名清除缓存的请求
func handleClearCacheByDomain(w http.ResponseWriter, r *http.Request, domain string) {
	// 执行按域名清除缓存操作
	err := sdns.ClearCacheByDomain(domain)

	// 处理错误
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cache cleared successfully for domain: " + domain,
		"domain":  domain,
	})
}
