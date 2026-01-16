// core/webapi/response.go

package webapi

import (
	"encoding/json"
	"net/http"
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

// getTokenFromRequest 从请求中获取token
func getTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// sendErrorResponse 发送错误响应
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Message: message,
		Code:    statusCode,
	})
}

// sendDetailedErrorResponse 发送详细错误响应
func sendDetailedErrorResponse(w http.ResponseWriter, message string, statusCode int, err error) {
	w.WriteHeader(statusCode)
	errorResponse := ErrorResponse{
		Success: false,
		Message: message,
		Code:    statusCode,
	}
	if err != nil {
		errorResponse.Error = err.Error()
	}
	json.NewEncoder(w).Encode(errorResponse)
}

// sendSuccessResponse 发送成功响应
func sendSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}
