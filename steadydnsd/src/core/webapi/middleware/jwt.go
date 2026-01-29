// /core/webapi/middleWare/jwt.go

package middleware

import (
	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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

type JWTManager struct {
	logger                 *common.Logger  // 日志记录器
	jwtKey                 []byte          // JWT密钥，从配置文件读取
	RefreshTokens          map[string]uint // 刷新令牌存储
	AccessTokenExpiration  time.Duration   // 访问令牌过期时间
	RefreshTokenExpiration time.Duration   // 刷新令牌过期时间
	Claims                 *claims         // 自定义JWT声明
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
		RefreshTokens:          make(map[string]uint),
		AccessTokenExpiration:  0,
		RefreshTokenExpiration: 0,
		Claims:                 &claims{},
	}

	// 从配置文件读取JWT过期时间
	j.loadJWTExpirationConfig()

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

	// 存储刷新令牌
	j.RefreshTokens[refreshToken] = userID

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
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 只接受POST请求
	if r.Method != http.MethodPost {
		SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshTokenRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证刷新令牌
	userID, valid := ValidateRefreshToken(req.RefreshToken)
	if !valid {
		SendErrorResponse(w, "无效的刷新令牌", http.StatusUnauthorized)
		return
	}

	// 获取用户信息
	user, err := database.GetUserByID(userID)
	if err != nil {
		SendErrorResponse(w, "用户不存在", http.StatusNotFound)
		return
	}

	// 生成新的访问令牌和刷新令牌
	accessToken, refreshToken, err := jwtManager.GenerateToken(user)
	if err != nil {
		SendErrorResponse(w, "生成token失败", http.StatusInternalServerError)
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
	SendSuccessResponse(w, resp, "令牌刷新成功")
}

// ValidateRefreshToken 验证刷新令牌
func ValidateRefreshToken(refreshToken string) (uint, bool) {
	userID, exists := jwtManager.RefreshTokens[refreshToken]
	return userID, exists
}
