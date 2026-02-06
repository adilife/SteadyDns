// core/webapi/loginapi.go

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"SteadyDNS/core/database"
	"SteadyDNS/core/webapi/middleware"
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

// 获取JWT管理器实例
var jwtManager = middleware.GetJWTManager()

// LoginHandler 处理登录请求
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 只接受POST请求
	if r.Method != http.MethodPost {
		middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证输入
	if req.Username == "" || req.Password == "" {
		middleware.SendErrorResponse(w, "用户名和密码不能为空", http.StatusBadRequest)
		return
	}

	// 使用数据库验证用户凭据
	user, valid := database.ValidateUserWithDB(req.Username, req.Password)
	if !valid {
		middleware.SendErrorResponse(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	// 生成JWT token
	accessToken, refreshToken, err := jwtManager.GenerateToken(user)
	if err != nil {
		middleware.SendErrorResponse(w, "生成token失败", http.StatusInternalServerError)
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
		ExpiresIn:    int64(jwtManager.AccessTokenExpiration / time.Second),
	}

	// 返回成功响应
	middleware.SendSuccessResponse(w, resp, "登录成功")
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
		middleware.SendErrorResponse(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req LogoutRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		middleware.SendErrorResponse(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 验证刷新令牌
	_, valid := middleware.ValidateRefreshToken(req.RefreshToken)
	if !valid {
		middleware.SendErrorResponse(w, "无效的刷新令牌", http.StatusUnauthorized)
		return
	}

	// 删除刷新令牌
	delete(jwtManager.RefreshTokens, req.RefreshToken)

	// 返回成功响应
	middleware.SendSuccessResponse(w, nil, "登出成功")
}
