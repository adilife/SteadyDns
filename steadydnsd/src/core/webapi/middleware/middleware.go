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
	"fmt"
	"net/http"
	"strconv"
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
		// 从配置读取默认限制
		limit := getConfigInt("API", "RATE_LIMIT_NORMAL", 300)
		window := getConfigDuration("API", "RATE_LIMIT_WINDOW", time.Minute)
		maxFailures := getConfigInt("API", "RATE_LIMIT_MAX_FAILURES", 10)
		banDuration := getConfigDuration("API", "RATE_LIMIT_BAN_DURATION", 10*time.Minute)

		limiter = NewLimitCounter(limit, window, maxFailures, banDuration)
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
		// 从配置读取默认限制
		limit := getConfigInt("API", "RATE_LIMIT_USER", 500)
		window := getConfigDuration("API", "RATE_LIMIT_WINDOW", time.Minute)
		maxFailures := getConfigInt("API", "RATE_LIMIT_MAX_FAILURES", 20)
		banDuration := getConfigDuration("API", "RATE_LIMIT_BAN_DURATION", 15*time.Minute)

		limiter = NewLimitCounter(limit, window, maxFailures, banDuration)
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

		// 根据路径设置不同的限制策略
		var ipLimit *LimitCounter
		limiter.ipMutex.Lock()
		switch {
		case path == "/api/login":
			// 登录端点：从配置读取限制
			limit := getConfigInt("API", "RATE_LIMIT_LOGIN", 60)
			window := getConfigDuration("API", "RATE_LIMIT_WINDOW", time.Minute)
			maxFailures := getConfigInt("API", "RATE_LIMIT_MAX_FAILURES_LOGIN", 10)
			banDuration := getConfigDuration("API", "RATE_LIMIT_BAN_DURATION_LOGIN", 5*time.Minute)

			if existingLimit, exists := limiter.ipLimits[clientIP]; exists {
				ipLimit = existingLimit
			} else {
				ipLimit = NewLimitCounter(limit, window, maxFailures, banDuration)
				limiter.ipLimits[clientIP] = ipLimit
			}
		case path == "/api/refresh-token":
			// 令牌刷新：从配置读取限制
			limit := getConfigInt("API", "RATE_LIMIT_REFRESH", 5)
			window := getConfigDuration("API", "RATE_LIMIT_WINDOW", time.Minute)
			maxFailures := getConfigInt("API", "RATE_LIMIT_MAX_FAILURES_REFRESH", 3)
			banDuration := getConfigDuration("API", "RATE_LIMIT_BAN_DURATION_REFRESH", 3*time.Minute)

			if existingLimit, exists := limiter.ipLimits[clientIP]; exists {
				ipLimit = existingLimit
			} else {
				ipLimit = NewLimitCounter(limit, window, maxFailures, banDuration)
				limiter.ipLimits[clientIP] = ipLimit
			}
		case path == "/api/health":
			// 健康检查：从配置读取限制，设置较高的限制值
			limit := getConfigInt("API", "RATE_LIMIT_HEALTH", 500)
			window := getConfigDuration("API", "RATE_LIMIT_WINDOW", time.Minute)
			maxFailures := getConfigInt("API", "RATE_LIMIT_MAX_FAILURES_HEALTH", 20)
			banDuration := getConfigDuration("API", "RATE_LIMIT_BAN_DURATION_HEALTH", 10*time.Minute)

			if existingLimit, exists := limiter.ipLimits[clientIP]; exists {
				ipLimit = existingLimit
			} else {
				ipLimit = NewLimitCounter(limit, window, maxFailures, banDuration)
				limiter.ipLimits[clientIP] = ipLimit
			}
		default:
			// 普通API：从配置读取限制
			limit := getConfigInt("API", "RATE_LIMIT_NORMAL", 300)
			window := getConfigDuration("API", "RATE_LIMIT_WINDOW", time.Minute)
			maxFailures := getConfigInt("API", "RATE_LIMIT_MAX_FAILURES", 10)
			banDuration := getConfigDuration("API", "RATE_LIMIT_BAN_DURATION", 10*time.Minute)

			if existingLimit, exists := limiter.ipLimits[clientIP]; exists {
				ipLimit = existingLimit
			} else {
				ipLimit = NewLimitCounter(limit, window, maxFailures, banDuration)
				limiter.ipLimits[clientIP] = ipLimit
			}
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

		// 处理请求
		c.Next()

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
