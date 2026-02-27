// core/webapi/middleWare/response.go

package middleware

import (
	"net/http"
	"regexp"

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

// sanitizeError 对错误信息进行脱敏处理，防止泄露系统内部信息
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// 过滤文件路径
	pathRegex := regexp.MustCompile(`[a-zA-Z]:\\[^"'\s]+|/[^"'\s]+`)
	errStr = pathRegex.ReplaceAllString(errStr, "[路径已隐藏]")

	// 过滤IP地址
	ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	errStr = ipRegex.ReplaceAllString(errStr, "[IP地址已隐藏]")

	// 过滤堆栈跟踪信息（常见的堆栈模式）
	stackRegex := regexp.MustCompile(`(?:at\s+[\w./]+|goroutine\s+\d+|runtime\.[\w]+)`)
	errStr = stackRegex.ReplaceAllString(errStr, "[堆栈信息已隐藏]")

	// 过滤可能的敏感关键词
	sensitiveKeywords := []string{"password", "secret", "token", "key", "credential", "auth"}
	for _, keyword := range sensitiveKeywords {
		regex := regexp.MustCompile(`(?i)` + keyword + `\s*[:=]\s*[^\s,]+`)
		errStr = regex.ReplaceAllString(errStr, keyword+"=[已隐藏]")
	}

	return errStr
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
// 注意：此函数会对错误信息进行脱敏处理，防止泄露系统内部信息
func SendDetailedErrorResponseGin(c *gin.Context, message string, statusCode int, err error) {
	errorResponse := ErrorResponse{
		Success: false,
		Message: message,
		Code:    statusCode,
	}
	if err != nil {
		errorResponse.Error = sanitizeError(err)
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
