// core/webapi/middleWare/response.go

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Success bool   `json:"success"`         // 是否成功
	Message string `json:"message"`         // 错误消息
	Code    int    `json:"code,omitempty"`  // 错误代码
	Error   string `json:"error,omitempty"` // 原始错误信息
}

// SuccessResponse 成功响应结构
type SuccessResponse struct {
	Success bool        `json:"success"`           // 是否成功
	Data    interface{} `json:"data,omitempty"`    // 响应数据
	Message string      `json:"message,omitempty"` // 成功消息
}

// GetTokenFromRequest 从请求中获取token
func GetTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}



// SendErrorResponseGin 发送错误响应（Gin版本）
func SendErrorResponseGin(c *gin.Context, message string, statusCode int) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
		Code:    statusCode,
	})
}



// SendDetailedErrorResponseGin 发送详细错误响应（Gin版本）
func SendDetailedErrorResponseGin(c *gin.Context, message string, statusCode int, err error) {
	errorResponse := ErrorResponse{
		Success: false,
		Message: message,
		Code:    statusCode,
	}
	if err != nil {
		errorResponse.Error = err.Error()
	}
	c.JSON(statusCode, errorResponse)
}



// SendSuccessResponseGin 发送成功响应（Gin版本）
func SendSuccessResponseGin(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}



// RespondWithErrorGin 发送错误响应（Gin版本）
func RespondWithErrorGin(c *gin.Context, message string, statusCode int) {
	c.JSON(statusCode, map[string]string{
		"message": message,
	})
}
