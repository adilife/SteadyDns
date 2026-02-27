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
// core/webapi/loginapi.go

package api

import (
	"net/http"
	"time"

	"SteadyDNS/core/database"
	"SteadyDNS/core/webapi/middleware"

	"github.com/gin-gonic/gin"
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

// getJWTManager 获取JWT管理器实例
func getJWTManager() *middleware.JWTManager {
	return middleware.GetJWTManager()
}

// LoginHandler 处理登录请求
func LoginHandler(c *gin.Context) {
	// 只接受POST请求
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 验证输入
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	// 使用数据库验证用户凭据
	user, valid := database.ValidateUserWithDB(req.Username, req.Password)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成JWT token
	jwtMgr := getJWTManager()
	accessToken, refreshToken, err := jwtMgr.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	// 返回登录响应，不包含密码
	userInfo := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	}

	resp := middleware.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userInfo,
		ExpiresIn:    int64(jwtMgr.AccessTokenExpiration / time.Second),
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp, "message": "登录成功"})
}

// LogoutRequest 登出请求结构体
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutHandler 处理登出请求
func LogoutHandler(c *gin.Context) {
	// 只接受POST请求
	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "方法不允许"})
		return
	}

	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 验证刷新令牌
	_, valid := middleware.ValidateRefreshToken(req.RefreshToken)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的刷新令牌"})
		return
	}

	// 删除刷新令牌
	delete(getJWTManager().RefreshTokens, req.RefreshToken)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "登出成功"})
}
