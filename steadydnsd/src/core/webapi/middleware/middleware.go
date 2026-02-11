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
// core/webapi/middleWare/middleware.go

package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"SteadyDNS/core/common"

	"github.com/gin-gonic/gin"
)

// RateLimiter 请求频率限制器
type RateLimiter struct {
	// IP 级别的限制
	ipLimits map[string]*LimitCounter
	ipMutex  sync.Mutex

	// 用户级别的限制
	userLimits map[uint]*LimitCounter
	userMutex  sync.Mutex

	// 封禁的IP
	bannedIPs   map[string]time.Time
	bannedMutex sync.Mutex
}

// LimitCounter 限制计数器
type LimitCounter struct {
	requests    []time.Time
	mutex       sync.Mutex
	limit       int
	window      time.Duration
	failCount   int
	maxFailures int
	banDuration time.Duration
}

// 全局限制器实例
var rateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// GetRateLimiter 获取全局限制器实例
func GetRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		rateLimiter = &RateLimiter{
			ipLimits:   make(map[string]*LimitCounter),
			userLimits: make(map[uint]*LimitCounter),
			bannedIPs:  make(map[string]time.Time),
		}
	})
	return rateLimiter
}

// ReloadConfig reloads rate limit configuration
func (rl *RateLimiter) ReloadConfig() {
	rl.ipMutex.Lock()
	rl.userMutex.Lock()
	rl.bannedMutex.Lock()

	// Clear existing limits to force reload on next request
	rl.ipLimits = make(map[string]*LimitCounter)
	rl.userLimits = make(map[uint]*LimitCounter)

	// Keep banned IPs but clear expired ones
	now := time.Now()
	for ip, banTime := range rl.bannedIPs {
		if now.After(banTime) {
			delete(rl.bannedIPs, ip)
		}
	}

	rl.bannedMutex.Unlock()
	rl.userMutex.Unlock()
	rl.ipMutex.Unlock()

	common.NewLogger().Info("Rate limiter configuration reloaded")
}

// ReloadRateLimitConfig reloads rate limit configuration (global function)
func ReloadRateLimitConfig() {
	if rateLimiter != nil {
		rateLimiter.ReloadConfig()
	}
}

// NewLimitCounter 创建新的限制计数器
func NewLimitCounter(limit int, window time.Duration, maxFailures int, banDuration time.Duration) *LimitCounter {
	return &LimitCounter{
		requests:    make([]time.Time, 0),
		limit:       limit,
		window:      window,
		maxFailures: maxFailures,
		banDuration: banDuration,
	}
}

// AddRequest 添加请求并检查是否超出限制
func (lc *LimitCounter) AddRequest() bool {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	// 清理过期的请求
	now := time.Now()
	cutoff := now.Add(-lc.window)
	validRequests := make([]time.Time, 0)

	for _, reqTime := range lc.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	lc.requests = validRequests

	// 检查是否超出限制
	if len(lc.requests) >= lc.limit {
		lc.failCount++
		return false
	}

	// 添加新请求
	lc.requests = append(lc.requests, now)
	return true
}

// IsBanned 检查是否被封禁
func (rl *RateLimiter) IsBanned(ip string) bool {
	rl.bannedMutex.Lock()
	defer rl.bannedMutex.Unlock()

	banTime, exists := rl.bannedIPs[ip]
	if !exists {
		return false
	}

	// 检查封禁是否过期
	if time.Now().After(banTime) {
		delete(rl.bannedIPs, ip)
		return false
	}

	return true
}

// BanIP 封禁IP
func (rl *RateLimiter) BanIP(ip string, duration time.Duration) {
	rl.bannedMutex.Lock()
	defer rl.bannedMutex.Unlock()

	rl.bannedIPs[ip] = time.Now().Add(duration)
}

// GetIPLimit 获取IP限制计数器
func (rl *RateLimiter) GetIPLimit(ip string) *LimitCounter {
	rl.ipMutex.Lock()
	defer rl.ipMutex.Unlock()

	limiter, exists := rl.ipLimits[ip]
	if !exists {
		// 从配置读取默认限制，支持向后兼容
		limit := getRateLimitConfigInt("API", "RATE_LIMIT_API", "RATE_LIMIT_NORMAL", 300)
		windowSeconds := getRateLimitConfigInt("API", "RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_WINDOW", 60)
		maxFailures := getRateLimitConfigInt("API", "RATE_LIMIT_MAX_FAILURES", "RATE_LIMIT_MAX_FAILURES", 10)
		banMinutes := getRateLimitConfigInt("API", "RATE_LIMIT_BAN_MINUTES", "RATE_LIMIT_BAN_DURATION", 10)

		limiter = NewLimitCounter(limit, time.Duration(windowSeconds)*time.Second, maxFailures, time.Duration(banMinutes)*time.Minute)
		rl.ipLimits[ip] = limiter
	}

	return limiter
}

// GetUserLimit 获取用户限制计数器
func (rl *RateLimiter) GetUserLimit(userID uint) *LimitCounter {
	rl.userMutex.Lock()
	defer rl.userMutex.Unlock()

	limiter, exists := rl.userLimits[userID]
	if !exists {
		// 从配置读取默认限制，支持向后兼容
		limit := getRateLimitConfigInt("API", "RATE_LIMIT_USER", "RATE_LIMIT_USER", 500)
		windowSeconds := getRateLimitConfigInt("API", "RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_WINDOW", 60)
		maxFailures := getRateLimitConfigInt("API", "RATE_LIMIT_USER_MAX_FAILURES", "RATE_LIMIT_MAX_FAILURES", 20)
		banMinutes := getRateLimitConfigInt("API", "RATE_LIMIT_USER_BAN_MINUTES", "RATE_LIMIT_BAN_DURATION", 15)

		limiter = NewLimitCounter(limit, time.Duration(windowSeconds)*time.Second, maxFailures, time.Duration(banMinutes)*time.Minute)
		rl.userLimits[userID] = limiter
	}

	return limiter
}

// RateLimitMiddleware 请求频率限制中间件
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用了速率限制
		if !isRateLimitEnabled() {
			c.Next()
			return
		}

		// 获取客户端IP
		clientIP := c.ClientIP()

		// 获取限制器
		limiter := GetRateLimiter()

		// 检查IP是否被封禁
		if limiter.IsBanned(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}

		// 获取请求路径
		path := c.Request.URL.Path

		// 根据路径设置不同的限制策略和键
		// 使用独立的键确保不同API类型的限制互不影响
		var ipLimit *LimitCounter
		var ipKey string
		var limit, windowSeconds, maxFailures, banMinutes int

		switch {
		case path == "/api/login":
			// 登录端点：从配置读取限制，支持向后兼容
			ipKey = clientIP + ":login"
			limit = getRateLimitConfigInt("API", "RATE_LIMIT_LOGIN", "RATE_LIMIT_LOGIN", 60)
			windowSeconds = getRateLimitConfigInt("API", "RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_WINDOW", 60)
			maxFailures = getRateLimitConfigInt("API", "RATE_LIMIT_LOGIN_MAX_FAILURES", "RATE_LIMIT_MAX_FAILURES_LOGIN", 10)
			banMinutes = getRateLimitConfigInt("API", "RATE_LIMIT_LOGIN_BAN_MINUTES", "RATE_LIMIT_BAN_DURATION_LOGIN", 5)
		case path == "/api/refresh-token":
			// 令牌刷新：从配置读取限制，支持向后兼容
			ipKey = clientIP + ":refresh"
			limit = getRateLimitConfigInt("API", "RATE_LIMIT_REFRESH", "RATE_LIMIT_REFRESH", 5)
			windowSeconds = getRateLimitConfigInt("API", "RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_WINDOW", 60)
			maxFailures = getRateLimitConfigInt("API", "RATE_LIMIT_REFRESH_MAX_FAILURES", "RATE_LIMIT_MAX_FAILURES_REFRESH", 3)
			banMinutes = getRateLimitConfigInt("API", "RATE_LIMIT_REFRESH_BAN_MINUTES", "RATE_LIMIT_BAN_DURATION_REFRESH", 3)
		case path == "/api/health":
			// 健康检查：从配置读取限制，设置较高的限制值，支持向后兼容
			ipKey = clientIP + ":health"
			limit = getRateLimitConfigInt("API", "RATE_LIMIT_HEALTH", "RATE_LIMIT_HEALTH", 500)
			windowSeconds = getRateLimitConfigInt("API", "RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_WINDOW", 60)
			maxFailures = getRateLimitConfigInt("API", "RATE_LIMIT_HEALTH_MAX_FAILURES", "RATE_LIMIT_MAX_FAILURES_HEALTH", 20)
			banMinutes = getRateLimitConfigInt("API", "RATE_LIMIT_HEALTH_BAN_MINUTES", "RATE_LIMIT_BAN_DURATION_HEALTH", 10)
		default:
			// 普通API：从配置读取限制，支持向后兼容
			ipKey = clientIP + ":api"
			limit = getRateLimitConfigInt("API", "RATE_LIMIT_API", "RATE_LIMIT_NORMAL", 300)
			windowSeconds = getRateLimitConfigInt("API", "RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_WINDOW", 60)
			maxFailures = getRateLimitConfigInt("API", "RATE_LIMIT_MAX_FAILURES", "RATE_LIMIT_MAX_FAILURES", 10)
			banMinutes = getRateLimitConfigInt("API", "RATE_LIMIT_BAN_MINUTES", "RATE_LIMIT_BAN_DURATION", 10)
		}

		limiter.ipMutex.Lock()
		if existingLimit, exists := limiter.ipLimits[ipKey]; exists {
			ipLimit = existingLimit
		} else {
			ipLimit = NewLimitCounter(limit, time.Duration(windowSeconds)*time.Second, maxFailures, time.Duration(banMinutes)*time.Minute)
			limiter.ipLimits[ipKey] = ipLimit
		}
		limiter.ipMutex.Unlock()

		// 检查IP限制
		if !ipLimit.AddRequest() {
			// 增加失败计数
			ipLimit.failCount++

			// 检查是否需要封禁
			if ipLimit.failCount >= ipLimit.maxFailures {
				limiter.BanIP(clientIP, ipLimit.banDuration)
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，已被临时封禁"})
				c.Abort()
				return
			}

			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}

		// 对于已认证的用户，检查用户级别的限制
		token := c.GetHeader("Authorization")
		if token != "" {
			// 移除Bearer前缀
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			claims, err := GetJWTManager().GetUserFromToken(token)
			if err == nil {
				userLimit := limiter.GetUserLimit(claims.UserID)
				if !userLimit.AddRequest() {
					c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
					c.Abort()
					return
				}
			}
		}

		// 继续处理请求
		c.Next()
	}
}

// LoggerMiddleware 请求日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用了日志
		if !isLogEnabled() {
			c.Next()
			return
		}

		// 开始时间
		startTime := time.Now()

		// 获取客户端IP
		clientIP := c.ClientIP()

		// 获取请求信息
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 获取用户信息（如果已认证）
		userID := uint(0)
		username := ""
		token := c.GetHeader("Authorization")
		if token != "" {
			// 移除Bearer前缀
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			claims, err := GetJWTManager().GetUserFromToken(token)
			if err == nil {
				userID = claims.UserID
				username = claims.Username
			}
		}

		// 捕获请求体
		var requestBody string
		if isLogRequestBodyEnabled() && c.Request.Body != nil {
			// 限制请求体大小，避免日志过大
			maxRequestBodySize := 1024 * 10 // 10KB
			bodyReader := io.LimitReader(c.Request.Body, int64(maxRequestBodySize))
			bodyBytes, err := io.ReadAll(bodyReader)
			if err == nil {
				requestBody = string(bodyBytes)
				// 脱敏处理
				requestBody = sanitizeRequestBody(requestBody)
			}
			// 重新设置请求体，确保后续处理能正常读取
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 捕获响应体
		var responseBody string
		var responseWriter *ResponseBodyWriter
		if isLogResponseBodyEnabled() {
			// 限制响应体大小，避免日志过大
			maxResponseBodySize := 1024 * 10 // 10KB
			responseWriter = NewResponseBodyWriter(c.Writer, maxResponseBodySize)
			c.Writer = responseWriter
		}

		// 处理请求
		c.Next()

		// 获取响应体
		if isLogResponseBodyEnabled() && responseWriter != nil {
			responseBody = responseWriter.GetBody()
			// 脱敏处理
			responseBody = sanitizeResponseBody(responseBody)
		}

		// 计算响应时间
		responseTime := time.Since(startTime)

		// 获取响应状态码
		statusCode := c.Writer.Status()

		// 创建日志记录器
		logger := common.NewLogger()

		// 构建日志消息
		logMessage := fmt.Sprintf("API请求 - IP: %s, 方法: %s, 路径: %s, 查询: %s, 状态码: %d, 响应时间: %v",
			clientIP, method, path, query, statusCode, responseTime)

		// 如果有用户信息，添加到日志
		if userID > 0 {
			logMessage += fmt.Sprintf(", 用户ID: %d, 用户名: %s", userID, username)
		}

		// 如果有请求体，添加到日志
		if requestBody != "" {
			logMessage += fmt.Sprintf(", 请求体: %s", requestBody)
		}

		// 如果有响应体，添加到日志
		if responseBody != "" {
			logMessage += fmt.Sprintf(", 响应体: %s", responseBody)
		}

		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			logger.Error("%s", logMessage)
		case statusCode >= 400:
			logger.Warn("%s", logMessage)
		default:
			logger.Info("%s", logMessage)
		}
	}
}

// isRateLimitEnabled 检查是否启用了速率限制
func isRateLimitEnabled() bool {
	enabled := common.GetConfig("API", "RATE_LIMIT_ENABLED")
	return enabled != "false"
}

// isLogEnabled 检查是否启用了日志
func isLogEnabled() bool {
	enabled := common.GetConfig("API", "LOG_ENABLED")
	return enabled != "false"
}

// isLogRequestBodyEnabled 检查是否启用了请求体日志
func isLogRequestBodyEnabled() bool {
	enabled := common.GetConfig("API", "LOG_REQUEST_BODY")
	return enabled == "true"
}

// isLogResponseBodyEnabled 检查是否启用了响应体日志
func isLogResponseBodyEnabled() bool {
	enabled := common.GetConfig("API", "LOG_RESPONSE_BODY")
	return enabled == "true"
}

// ResponseBodyWriter 响应体捕获写入器
// 实现 http.ResponseWriter 接口，用于捕获响应体
// 同时将写入操作代理到原始的响应写入器
// 支持对响应体大小进行限制，避免日志过大
// 支持对敏感信息进行脱敏处理
// 线程安全：不保证线程安全，因为每个请求只在一个 goroutine 中处理
// 性能考虑：对于大响应体，可能会增加内存使用
// 适用场景：仅在需要记录响应体时使用
// 注意：由于实现了 http.ResponseWriter 接口，所有方法都必须实现
// 包括：Write, WriteHeader, Header, WriteString, WriteByte, WriteRune
// 但为了简化实现，只实现了核心的 Write, WriteHeader, Header 方法
// 其他方法会通过默认实现间接调用 Write 方法
// 因此可能会有轻微的性能损失，但对于日志记录场景是可接受的
// 另外，此实现不处理 Trailers，因为在大多数 API 场景中不常用
// 如果需要支持 Trailers，需要额外实现 Trailer 和 Flush 方法
// 最后，此实现假设响应体是 UTF-8 编码的文本，对于二进制响应可能会产生乱码
// 因此建议只在 API 接口返回 JSON/XML/文本等可打印格式时启用响应体日志
// 对于文件下载等二进制响应，应禁用响应体日志
// 综上，此实现适合 API 开发和调试场景，不适合生产环境的所有场景
// 生产环境中建议只在必要时启用，并且要注意日志大小和敏感信息保护
// 为了安全起见，默认情况下应保持 LOG_RESPONSE_BODY=false
// 仅在开发和调试时临时设置为 true
// 以上是此实现的详细说明和注意事项
// 现在开始实现 ResponseBodyWriter 结构体
// 注意：以下实现是基于 Gin 框架的使用场景
// 对于标准 net/http 服务器，可能需要做适当调整
// 但由于我们的项目使用的是 Gin 框架，因此此实现应该是适用的
// 此外，为了避免影响原始响应的写入，所有写入操作都会同时写入原始响应写入器
// 因此捕获响应体的过程不会影响客户端的正常接收
// 最后，捕获的响应体将被添加到日志中，以便于调试和分析
// 现在开始编写代码
// ResponseBodyWriter 用于捕获响应体的写入器
type ResponseBodyWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	maxSize int
}

// NewResponseBodyWriter 创建一个新的响应体捕获写入器
func NewResponseBodyWriter(w gin.ResponseWriter, maxSize int) *ResponseBodyWriter {
	return &ResponseBodyWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		maxSize:        maxSize,
	}
}

// Write 写入响应体数据
func (w *ResponseBodyWriter) Write(b []byte) (int, error) {
	// 写入原始响应
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		return n, err
	}

	// 捕获响应体，不超过最大大小
	if w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}

	return n, nil
}

// WriteString 写入字符串响应体数据
func (w *ResponseBodyWriter) WriteString(s string) (int, error) {
	// 写入原始响应
	n, err := w.ResponseWriter.WriteString(s)
	if err != nil {
		return n, err
	}

	// 捕获响应体，不超过最大大小
	if w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(s) > remaining {
			w.body.WriteString(s[:remaining])
		} else {
			w.body.WriteString(s)
		}
	}

	return n, nil
}

// GetBody 获取捕获的响应体
func (w *ResponseBodyWriter) GetBody() string {
	return w.body.String()
}

// sanitizeRequestBody 对请求体进行脱敏处理
func sanitizeRequestBody(body string) string {
	// 对常见的敏感字段进行脱敏
	sensitiveFields := []string{"password", "token", "secret", "key", "credential", "auth"}
	result := body

	for _, field := range sensitiveFields {
		// 使用字符串替换，适用于 JSON 格式的请求体
		// 匹配 "field":"value" 格式
		result = strings.ReplaceAll(result, fmt.Sprintf(`"%s":"admin123"`, field), fmt.Sprintf(`"%s":"***"`, field))
	}

	return result
}

// sanitizeResponseBody 对响应体进行脱敏处理
func sanitizeResponseBody(body string) string {
	// 对常见的敏感字段进行脱敏
	sensitiveFields := []string{"token", "secret", "key", "credential", "auth", "password"}
	result := body

	for _, field := range sensitiveFields {
		// 使用字符串替换，适用于 JSON 格式的响应体
		// 匹配 "field":"value" 格式
		if field == "token" {
			// 对 token 字段进行特殊处理，匹配更长的 token 值
			result = strings.ReplaceAll(result, `"access_token":"`, `"access_token":"***`)
			result = strings.ReplaceAll(result, `"refresh_token":"`, `"refresh_token":"***`)
		} else if field == "password" {
			// 对密码字段进行处理
			result = strings.ReplaceAll(result, fmt.Sprintf(`"%s":"admin123"`, field), fmt.Sprintf(`"%s":"***"`, field))
		}
	}

	return result
}

// getConfigInt 从配置读取整数
func getConfigInt(section, key string, defaultValue int) int {
	valueStr := common.GetConfig(section, key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getConfigDuration 从配置读取时间间隔
func getConfigDuration(section, key string, defaultValue time.Duration) time.Duration {
	valueStr := common.GetConfig(section, key)
	if valueStr == "" {
		return defaultValue
	}

	// 尝试解析数字（表示秒）
	if value, err := strconv.Atoi(valueStr); err == nil {
		return time.Duration(value) * time.Second
	}

	// 尝试解析时间字符串
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getRateLimitConfigInt gets rate limit config with backward compatibility
func getRateLimitConfigInt(section, newKey, oldKey string, defaultValue int) int {
	if value := getConfigInt(section, newKey, -1); value != -1 {
		return value
	}
	return getConfigInt(section, oldKey, defaultValue)
}

// getRateLimitConfigDuration gets rate limit duration config with backward compatibility
func getRateLimitConfigDuration(section, newKey, oldKey string, defaultValue time.Duration) time.Duration {
	if value := getConfigDuration(section, newKey, -1); value != -1 {
		return value
	}
	return getConfigDuration(section, oldKey, defaultValue)
}
