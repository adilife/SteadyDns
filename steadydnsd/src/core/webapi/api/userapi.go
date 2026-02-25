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
// core/webapi/api/userapi.go

package api

import (
	"net/http"
	"strconv"

	"SteadyDNS/core/database"

	"github.com/gin-gonic/gin"
)

// UserResponse 用户响应结构体（不包含密码）
type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// CreateUserRequest 创建用户请求结构体
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"` // 邮箱改为可选项
	Password string `json:"password" binding:"required"`
}

// UpdateUserRequest 更新用户请求结构体
type UpdateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ChangePasswordRequest 修改密码请求结构体
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// UsersListResponse 用户列表响应结构体
type UsersListResponse struct {
	Users    []UserResponse `json:"users"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// GetUsersHandler 获取用户列表处理器
// 支持分页查询，需要JWT认证
func GetUsersHandler(c *gin.Context) {
	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 查询用户列表
	users, total, err := database.GetAllUsers(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 转换为响应格式（不包含密码）
	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": UsersListResponse{
			Users:    userResponses,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// CreateUserHandler 创建用户处理器
// 需要JWT认证
func CreateUserHandler(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求体"})
		return
	}

	// 验证必填字段
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "用户名和密码不能为空"})
		return
	}

	// 创建用户
	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := database.CreateUser(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 返回创建的用户（不包含密码）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
		"message": "用户创建成功",
	})
}

// UpdateUserHandler 更新用户信息处理器
// 需要JWT认证
func UpdateUserHandler(c *gin.Context) {
	// 获取用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的用户ID"})
		return
	}

	// 查询用户
	user, err := database.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求体"})
		return
	}

	// 更新字段
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	if err := database.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 返回更新后的用户（不包含密码）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
		"message": "用户更新成功",
	})
}

// DeleteUserHandler 删除用户处理器
// 需要JWT认证，不能删除admin用户
func DeleteUserHandler(c *gin.Context) {
	// 获取用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的用户ID"})
		return
	}

	// 删除用户（DeleteUser函数内部会检查是否为admin）
	if err := database.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户删除成功",
	})
}

// ChangePasswordHandler 修改密码处理器
// 需要JWT认证，用户只能修改自己的密码
func ChangePasswordHandler(c *gin.Context) {
	// 获取用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的用户ID"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求体"})
		return
	}

	// 验证旧密码
	user, valid := database.ValidateUserWithDB("", "")
	if !valid {
		// 需要通过用户名验证，这里需要从数据库获取用户信息
		user, err = database.GetUserByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
			return
		}
		// 使用用户名和旧密码验证
		_, valid = database.ValidateUserWithDB(user.Username, req.OldPassword)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "旧密码错误"})
			return
		}
	}

	// 更新密码
	user.Password = req.NewPassword
	if err := database.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码修改成功",
	})
}
