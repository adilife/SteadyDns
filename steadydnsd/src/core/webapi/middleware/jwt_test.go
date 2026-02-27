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
// core/webapi/middleware/jwt_test.go
// JWT管理器测试

package middleware

import (
	"testing"
	"time"

	"SteadyDNS/core/database"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupJWTTestDB 创建测试用的内存数据库
func setupJWTTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	database.DB = db

	if err := database.DB.AutoMigrate(&database.User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}

	return func() {
		sqlDB, _ := database.DB.DB()
		sqlDB.Close()
		database.DB = nil
	}
}

// TestValidateSecretKey 测试密钥强度验证
func TestValidateSecretKey(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected bool
	}{
		{"有效强密钥", "ThisIsAVeryStrongSecretKey123!@#abcdef", true},
		{"有效中等密钥-含特殊字符", "MySecretKey1234567890123456!@#$%", true},
		{"过短密钥", "short", false},
		{"仅小写字母-32字符", "abcdefghijklmnopqrstuvwxyzabcdef", false},
		{"仅数字-32字符", "12345678901234567890123456789012", false},
		{"空密钥", "", false},
		{"混合类型密钥-32字符", "Abc123!@#Def456$%^Ghi789&*()Jkl012", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSecretKey(tt.secret)
			if result != tt.expected {
				t.Errorf("validateSecretKey(%q) = %v, want %v", tt.secret, result, tt.expected)
			}
		})
	}
}

// TestGenerateStrongSecret 测试强密钥生成
func TestGenerateStrongSecret(t *testing.T) {
	secret1 := generateStrongSecret()
	secret2 := generateStrongSecret()

	if len(secret1) < minSecretKeyLength {
		t.Errorf("生成的密钥长度 %d 小于最小长度 %d", len(secret1), minSecretKeyLength)
	}

	if secret1 == secret2 {
		t.Error("两次生成的密钥不应该相同")
	}
}

// TestJWTManagerGenerateToken 测试JWT令牌生成
func TestJWTManagerGenerateToken(t *testing.T) {
	cleanup := setupJWTTestDB(t)
	defer cleanup()

	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	database.DB.Create(user)

	jwtMgr := &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  30 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Claims:                 &claims{},
	}

	accessToken, refreshToken, err := jwtMgr.GenerateToken(user)
	if err != nil {
		t.Errorf("GenerateToken() error = %v", err)
		return
	}

	if accessToken == "" {
		t.Error("accessToken 不应该为空")
	}

	if refreshToken == "" {
		t.Error("refreshToken 不应该为空")
	}

	_, exists := jwtMgr.RefreshTokens[refreshToken]
	if !exists {
		t.Error("refreshToken 应该存储在 RefreshTokens map 中")
	}
}

// TestJWTManagerGetUserFromToken 测试从令牌获取用户信息
func TestJWTManagerGetUserFromToken(t *testing.T) {
	cleanup := setupJWTTestDB(t)
	defer cleanup()

	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	database.DB.Create(user)

	jwtMgr := &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  30 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Claims:                 &claims{},
	}

	accessToken, _, err := jwtMgr.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := jwtMgr.GetUserFromToken(accessToken)
	if err != nil {
		t.Errorf("GetUserFromToken() error = %v", err)
		return
	}

	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %d, want %d", claims.UserID, user.ID)
	}

	if claims.Username != user.Username {
		t.Errorf("claims.Username = %s, want %s", claims.Username, user.Username)
	}
}

// TestJWTManagerGetUserFromToken_InvalidToken 测试无效令牌
func TestJWTManagerGetUserFromToken_InvalidToken(t *testing.T) {
	jwtMgr := &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  30 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Claims:                 &claims{},
	}

	tests := []struct {
		name        string
		token       string
		wantErr     bool
	}{
		{"空令牌", "", true},
		{"无效格式", "invalid-token", true},
		{"随机字符串", "abc.def.ghi", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtMgr.GetUserFromToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserFromToken(%q) error = %v, wantErr %v", tt.token, err, tt.wantErr)
			}
		})
	}
}

// TestJWTManagerExpiredToken 测试过期令牌
func TestJWTManagerExpiredToken(t *testing.T) {
	jwtMgr := &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  -1 * time.Hour,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Claims:                 &claims{},
	}

	user := &database.User{
		ID:       1,
		Username: "testuser",
	}

	accessToken, _, err := jwtMgr.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = jwtMgr.GetUserFromToken(accessToken)
	if err == nil {
		t.Error("过期令牌应该返回错误")
	}
}

// TestValidateRefreshToken 测试刷新令牌验证
func TestValidateRefreshToken(t *testing.T) {
	jwtManager = &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  30 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Claims:                 &claims{},
	}

	userID := uint(1)
	refreshToken := jwtManager.GenerateRefreshToken(userID)

	tests := []struct {
		name         string
		token        string
		wantUserID   uint
		wantValid    bool
	}{
		{"有效刷新令牌", refreshToken, userID, true},
		{"无效刷新令牌", "invalid-refresh-token", 0, false},
		{"空刷新令牌", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, valid := ValidateRefreshToken(tt.token)
			if valid != tt.wantValid {
				t.Errorf("ValidateRefreshToken(%q) valid = %v, want %v", tt.token, valid, tt.wantValid)
			}
			if valid && userID != tt.wantUserID {
				t.Errorf("ValidateRefreshToken(%q) userID = %d, want %d", tt.token, userID, tt.wantUserID)
			}
		})
	}
}

// TestValidateRefreshToken_Expired 测试过期刷新令牌
func TestValidateRefreshToken_Expired(t *testing.T) {
	jwtManager = &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  30 * time.Minute,
		RefreshTokenExpiration: -1 * time.Hour,
		Claims:                 &claims{},
	}

	userID := uint(1)
	refreshToken := jwtManager.GenerateRefreshToken(userID)

	validatedUserID, valid := ValidateRefreshToken(refreshToken)
	if valid {
		t.Error("过期刷新令牌应该无效")
	}
	if validatedUserID != 0 {
		t.Errorf("过期刷新令牌应该返回 userID = 0, got %d", validatedUserID)
	}
}

// TestClaimsStructure 测试Claims结构
func TestClaimsStructure(t *testing.T) {
	c := &claims{
		UserID:   1,
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "SteadyDNS",
			Subject:   "access token",
		},
	}

	if c.UserID != 1 {
		t.Errorf("UserID = %d, want 1", c.UserID)
	}

	if c.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", c.Username)
	}

	if c.Issuer != "SteadyDNS" {
		t.Errorf("Issuer = %s, want SteadyDNS", c.Issuer)
	}
}

// TestTokenResponseStructure 测试TokenResponse结构
func TestTokenResponseStructure(t *testing.T) {
	resp := TokenResponse{
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
		User: map[string]interface{}{
			"id":       1,
			"username": "testuser",
		},
		ExpiresIn: 1800,
	}

	if resp.AccessToken != "access-token-value" {
		t.Errorf("AccessToken = %s, want access-token-value", resp.AccessToken)
	}

	if resp.RefreshToken != "refresh-token-value" {
		t.Errorf("RefreshToken = %s, want refresh-token-value", resp.RefreshToken)
	}

	if resp.ExpiresIn != 1800 {
		t.Errorf("ExpiresIn = %d, want 1800", resp.ExpiresIn)
	}
}

// TestCleanupExpiredTokens 测试清理过期令牌
func TestCleanupExpiredTokens(t *testing.T) {
	jwtManager = &JWTManager{
		jwtKey:                 []byte("test-secret-key-for-jwt-testing-12345"),
		RefreshTokens:          make(map[string]RefreshTokenInfo),
		AccessTokenExpiration:  30 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Claims:                 &claims{},
	}

	jwtManager.RefreshTokens["expired_token"] = RefreshTokenInfo{
		UserID:    1,
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}

	jwtManager.RefreshTokens["valid_token"] = RefreshTokenInfo{
		UserID:    2,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	cleanupExpiredTokens()

	if _, exists := jwtManager.RefreshTokens["expired_token"]; exists {
		t.Error("过期令牌应该被清理")
	}

	if _, exists := jwtManager.RefreshTokens["valid_token"]; !exists {
		t.Error("有效令牌不应该被清理")
	}
}
