// /core/webapi/middleWare/jwt.go

package middleware

import (
	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// claims 自定义JWT声明
type claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWT相关配置
const (
	// 最小密钥长度
	minSecretKeyLength = 32
)

// RefreshTokenInfo 刷新令牌信息结构体
type RefreshTokenInfo struct {
	UserID    uint  // 用户ID
	ExpiresAt int64 // 过期时间（Unix时间戳）
}

type JWTManager struct {
	logger                 *common.Logger              // 日志记录器
	jwtKey                 []byte                      // JWT密钥，从配置文件读取
	RefreshTokens          map[string]RefreshTokenInfo // 刷新令牌存储
	AccessTokenExpiration  time.Duration               // 访问令牌过期时间
	RefreshTokenExpiration time.Duration               // 刷新令牌过期时间
	Claims                 *claims                     // 自定义JWT声明
}

// 全局JWT管理器实例
var jwtManager *JWTManager

// GetJWTManager 获取JWT管理器实例
func GetJWTManager() *JWTManager {
	if jwtManager == nil {
		jwtManager = newJWTManager()
	}
	return jwtManager
}

func newJWTManager() *JWTManager {
	// 从多来源获取JWT密钥
	jwtSecret := getJWTSecret()

	// 验证密钥强度
	if !validateSecretKey(jwtSecret) {
		fmt.Println("警告: JWT密钥强度不足，建议使用至少32字节的随机字符串")
		// 如果密钥强度不足，生成一个临时的强密钥
		if len(jwtSecret) < minSecretKeyLength {
			jwtSecret = generateStrongSecret()
			fmt.Println("已生成临时强密钥，请在生产环境中配置安全的密钥")
		}
	}

	jwtKey := []byte(jwtSecret)

	j := &JWTManager{
		logger:                 common.NewLogger(),
		jwtKey:                 jwtKey,
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  0,
		RefreshTokenExpiration: 0,
		Claims:                 &claims{},
	}

	// 从配置文件读取JWT过期时间
	j.loadJWTExpirationConfig()

	// 启动定期清理过期令牌的后台任务
	StartTokenCleanupTask()

	return j
}

// loadJWTExpirationConfig 从配置文件加载JWT过期时间配置
func (j *JWTManager) loadJWTExpirationConfig() {
	// 读取访问令牌过期时间（分钟）
	accessExpStr := common.GetConfig("JWT", "ACCESS_TOKEN_EXPIRATION")
	accessExp := 30 // 默认30分钟
	if accessExpStr != "" {
		if exp, err := strconv.Atoi(accessExpStr); err == nil && exp > 0 {
			accessExp = exp
		}
	}
	j.AccessTokenExpiration = time.Duration(accessExp) * time.Minute

	// 读取刷新令牌过期时间（天）
	refreshExpStr := common.GetConfig("JWT", "REFRESH_TOKEN_EXPIRATION")
	refreshExp := 7 // 默认7天
	if refreshExpStr != "" {
		if exp, err := strconv.Atoi(refreshExpStr); err == nil && exp > 0 {
			refreshExp = exp
		}
	}
	j.RefreshTokenExpiration = time.Duration(refreshExp) * 24 * time.Hour
}

// ReloadConfig 重新加载JWT相关配置
func (j *JWTManager) ReloadConfig() {
	// 从多来源获取JWT密钥
	jwtSecret := getJWTSecret()

	// 验证密钥强度
	if !validateSecretKey(jwtSecret) {
		j.logger.Warn("JWT密钥强度不足，建议使用至少32字节的随机字符串")
		// 如果密钥强度不足，生成一个临时的强密钥
		if len(jwtSecret) < minSecretKeyLength {
			jwtSecret = generateStrongSecret()
			j.logger.Warn("已生成临时强密钥，请在生产环境中配置安全的密钥")
		}
	}

	// 更新JWT密钥
	j.jwtKey = []byte(jwtSecret)

	// 重新加载JWT过期时间配置
	j.loadJWTExpirationConfig()

	j.logger.Info("JWT配置重载成功")
}

// getJWTSecret 从多来源获取JWT密钥
func getJWTSecret() string {
	// 1. 首先尝试从环境变量获取（优先级最高）
	if secret := os.Getenv("JWT_SECRET_KEY"); secret != "" {
		return secret
	}

	// 2. 从配置文件获取
	if secret := common.GetConfig("JWT", "JWT_SECRET_KEY"); secret != "" {
		return secret
	}

	// 3. 使用默认值作为最后后备
	return "your-default-jwt-secret-key-change-this-in-production"
}

// validateSecretKey 验证密钥强度
func validateSecretKey(secret string) bool {
	// 检查密钥长度
	return len(secret) >= minSecretKeyLength
}

// generateStrongSecret 生成强密钥
func generateStrongSecret() string {
	// 生成32字节的随机字符串
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
	secret := make([]byte, minSecretKeyLength)
	for i := range secret {
		secret[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // 增加随机性
	}
	return string(secret)
}

// GetUserFromToken 从token中获取用户信息（用于后续的认证中间件）
func (j *JWTManager) GetUserFromToken(tokenString string) (*claims, error) {
	claims := j.Claims

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return j.jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("无效的token")
	}

	return claims, nil
}

// GenerateToken 生成JWT token
func (j *JWTManager) GenerateToken(user *database.User) (string, string, error) {
	// 创建访问令牌声明
	accessClaims := *j.Claims
	accessClaims.UserID = user.ID
	accessClaims.Username = user.Username
	accessClaims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.AccessTokenExpiration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "SteadyDNS",
		Subject:   "access token",
	}

	// 创建访问令牌
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)

	// 生成访问令牌字符串
	accessTokenString, err := accessToken.SignedString(j.jwtKey)
	if err != nil {
		return "", "", err
	}

	// 生成刷新令牌
	refreshTokenString := j.GenerateRefreshToken(user.ID)

	return accessTokenString, refreshTokenString, nil
}

// GenerateRefreshToken 生成刷新令牌
func (j *JWTManager) GenerateRefreshToken(userID uint) string {
	// 生成基于时间戳和用户ID的刷新令牌
	refreshToken := fmt.Sprintf("%d_%d_%d", userID, time.Now().UnixNano(), time.Now().Unix())

	// 计算过期时间
	expiresAt := time.Now().Add(j.RefreshTokenExpiration).Unix()

	// 存储刷新令牌和过期信息
	j.RefreshTokens[refreshToken] = RefreshTokenInfo{
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	return refreshToken
}

// TokenResponse 令牌响应结构体
type TokenResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         interface{} `json:"user"`
	ExpiresIn    int64       `json:"expires_in"` // 访问令牌过期时间（秒）
}

// RefreshTokenRequest 刷新令牌请求结构体
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenHandler 处理令牌刷新请求
func RefreshTokenHandler(c *gin.Context) {
	// 只接受POST请求
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
		return
	}

	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 验证刷新令牌
	userID, valid := ValidateRefreshToken(req.RefreshToken)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的刷新令牌"})
		return
	}

	// 获取用户信息
	user, err := database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 生成新的访问令牌和刷新令牌
	accessToken, refreshToken, err := jwtManager.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	// 删除旧的刷新令牌
	delete(jwtManager.RefreshTokens, req.RefreshToken)

	// 返回新的令牌
	userInfo := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	}

	resp := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userInfo,
		ExpiresIn:    int64(jwtManager.AccessTokenExpiration / time.Second),
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp, "message": "令牌刷新成功"})
}

// ValidateRefreshToken 验证刷新令牌
func ValidateRefreshToken(refreshToken string) (uint, bool) {
	tokenInfo, exists := jwtManager.RefreshTokens[refreshToken]
	if !exists {
		return 0, false
	}

	// 检查是否过期
	if time.Now().Unix() > tokenInfo.ExpiresAt {
		// 过期的令牌，从映射中删除
		delete(jwtManager.RefreshTokens, refreshToken)
		return 0, false
	}

	return tokenInfo.UserID, true
}

// StartTokenCleanupTask 启动定期清理过期令牌的后台任务
func StartTokenCleanupTask() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredTokens()
		}
	}()
}

// cleanupExpiredTokens 清理过期的刷新令牌
func cleanupExpiredTokens() {
	now := time.Now().Unix()
	for token, info := range jwtManager.RefreshTokens {
		if now > info.ExpiresAt {
			delete(jwtManager.RefreshTokens, token)
		}
	}
}
