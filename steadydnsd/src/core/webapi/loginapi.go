// core/webapi/loginapi.go

package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"

	"github.com/golang-jwt/jwt/v4"
)

// User 用户模型
type User struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // 不在JSON响应中包含密码
	CreatedAt time.Time `json:"created_at"`
}

// 登录请求结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 登录响应结构体
type LoginResponse struct {
	Token   string      `json:"token"`
	User    interface{} `json:"user"`
	Message string      `json:"message,omitempty"`
}

// Claims 自定义JWT声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWT相关配置
const (
	// 最小密钥长度
	minSecretKeyLength = 32
)

// JWT过期时间配置
var (
	// 访问令牌过期时间
	accessTokenExpiration time.Duration
	// 刷新令牌过期时间
	refreshTokenExpiration time.Duration
)

// JWT密钥，从配置文件读取
var jwtKey []byte

// 刷新令牌存储
var refreshTokens = make(map[string]uint)

// init 初始化函数，在包加载时执行
func init() {
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

	jwtKey = []byte(jwtSecret)

	// 从配置文件读取JWT过期时间
	loadJWTExpirationConfig()
}

// loadJWTExpirationConfig 从配置文件加载JWT过期时间配置
func loadJWTExpirationConfig() {
	// 读取访问令牌过期时间（分钟）
	accessExpStr := common.GetConfig("JWT", "ACCESS_TOKEN_EXPIRATION")
	accessExp := 30 // 默认30分钟
	if accessExpStr != "" {
		if exp, err := strconv.Atoi(accessExpStr); err == nil && exp > 0 {
			accessExp = exp
		}
	}
	accessTokenExpiration = time.Duration(accessExp) * time.Minute

	// 读取刷新令牌过期时间（天）
	refreshExpStr := common.GetConfig("JWT", "REFRESH_TOKEN_EXPIRATION")
	refreshExp := 7 // 默认7天
	if refreshExpStr != "" {
		if exp, err := strconv.Atoi(refreshExpStr); err == nil && exp > 0 {
			refreshExp = exp
		}
	}
	refreshTokenExpiration = time.Duration(refreshExp) * 24 * time.Hour
}

// getJWTSecret 从多来源获取JWT密钥
func getJWTSecret() string {
	// 1. 首先尝试从环境变量获取（优先级最高）
	if secret := os.Getenv("JWT_SECRET_KEY"); secret != "" {
		return secret
	}

	// 2. 从配置文件获取
	if secret := common.GetConfig("Server", "JWT_SECRET_KEY"); secret != "" {
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

// LoginHandler 处理登录请求
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 只接受POST请求
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		sendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证输入
	if req.Username == "" || req.Password == "" {
		sendErrorResponse(w, "用户名和密码不能为空", http.StatusBadRequest)
		return
	}

	// 使用数据库验证用户凭据
	user, valid := database.ValidateUserWithDB(req.Username, req.Password)
	if !valid {
		sendErrorResponse(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	// 生成JWT token
	accessToken, refreshToken, err := generateToken(user)
	if err != nil {
		sendErrorResponse(w, "生成token失败", http.StatusInternalServerError)
		return
	}

	// 返回登录响应，不包含密码
	userInfo := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	}

	resp := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userInfo,
		ExpiresIn:    int64(accessTokenExpiration / time.Second),
	}

	sendSuccessResponse(w, resp, "登录成功")
}

// TokenResponse 令牌响应结构体
type TokenResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         interface{} `json:"user"`
	ExpiresIn    int64       `json:"expires_in"` // 访问令牌过期时间（秒）
}

// generateToken 生成JWT token
func generateToken(user *database.User) (string, string, error) {
	// 创建访问令牌声明
	accessClaims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "SteadyDNS",
			Subject:   "access token",
		},
	}

	// 创建访问令牌
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)

	// 生成访问令牌字符串
	accessTokenString, err := accessToken.SignedString(jwtKey)
	if err != nil {
		return "", "", err
	}

	// 生成刷新令牌
	refreshTokenString := generateRefreshToken(user.ID)

	return accessTokenString, refreshTokenString, nil
}

// generateRefreshToken 生成刷新令牌
func generateRefreshToken(userID uint) string {
	// 生成基于时间戳和用户ID的刷新令牌
	refreshToken := fmt.Sprintf("%d_%d_%d", userID, time.Now().UnixNano(), time.Now().Unix())

	// 存储刷新令牌
	refreshTokens[refreshToken] = userID

	return refreshToken
}

// respondWithError 发送错误响应
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// GetUserFromToken 从token中获取用户信息（用于后续的认证中间件）
func GetUserFromToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("无效的token")
	}

	return claims, nil
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
		sendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshTokenRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		sendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证刷新令牌
	userID, valid := validateRefreshToken(req.RefreshToken)
	if !valid {
		sendErrorResponse(w, "无效的刷新令牌", http.StatusUnauthorized)
		return
	}

	// 获取用户信息
	user, err := database.GetUserByID(userID)
	if err != nil {
		sendErrorResponse(w, "用户不存在", http.StatusNotFound)
		return
	}

	// 生成新的访问令牌和刷新令牌
	accessToken, refreshToken, err := generateToken(user)
	if err != nil {
		sendErrorResponse(w, "生成token失败", http.StatusInternalServerError)
		return
	}

	// 删除旧的刷新令牌
	delete(refreshTokens, req.RefreshToken)

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
		ExpiresIn:    int64(accessTokenExpiration / time.Second),
	}

	sendSuccessResponse(w, resp, "令牌刷新成功")
}

// validateRefreshToken 验证刷新令牌
func validateRefreshToken(refreshToken string) (uint, bool) {
	userID, exists := refreshTokens[refreshToken]
	return userID, exists
}

// LogoutRequest 登出请求结构体
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutHandler 处理登出请求
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 只接受POST请求
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req LogoutRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		sendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证刷新令牌
	_, valid := validateRefreshToken(req.RefreshToken)
	if !valid {
		sendErrorResponse(w, "无效的刷新令牌", http.StatusUnauthorized)
		return
	}

	// 删除刷新令牌
	delete(refreshTokens, req.RefreshToken)

	sendSuccessResponse(w, nil, "登出成功")
}
