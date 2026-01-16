// core/database/usersdb.go

package database

import (
	"SteadyDNS/core/common"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Username string `json:"username" gorm:"uniqueIndex;not null"`
	Email    string `json:"email" gorm:"uniqueIndex;not null"`
	Password string `json:"-" gorm:"column:password;not null"` // 不在JSON中输出
}

// CreateUser 创建用户
func CreateUser(user *User) error {
	// 检查用户名是否已存在
	var existingUser User
	if err := DB.Where("username = ?", user.Username).First(&existingUser).Error; err == nil {
		return fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	if err := DB.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		return fmt.Errorf("邮箱已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("加密密码失败: %v", err)
	}
	user.Password = string(hashedPassword)

	// 创建用户
	if err := DB.Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %v", err)
	}

	return nil
}

// GetUserByID 根据ID获取用户
func GetUserByID(id uint) (*User, error) {
	var user User
	if err := DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func GetUserByUsername(username string) (*User, error) {
	var user User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户
func GetUserByEmail(email string) (*User, error) {
	var user User
	if err := DB.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func UpdateUser(user *User) error {
	if err := DB.Save(user).Error; err != nil {
		return fmt.Errorf("更新用户失败: %v", err)
	}
	return nil
}

// DeleteUser 删除用户
func DeleteUser(id uint) error {
	if err := DB.Delete(&User{}, id).Error; err != nil {
		return fmt.Errorf("删除用户失败: %v", err)
	}
	return nil
}

// ValidateUserWithDB 使用数据库验证用户凭据
func ValidateUserWithDB(username, password string) (*User, bool) {
	var user User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false
		}
		// 记录错误日志
		common.NewLogger().Warn("查询用户失败: %v", err)
		return nil, false
	}

	// 验证密码（密码已用bcrypt加密存储）
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, false
	}

	// 返回时不包含密码字段
	return &user, true
}

// GetAllUsers 获取所有用户（分页）
func GetAllUsers(page, pageSize int) ([]User, int64, error) {
	var users []User
	var total int64

	offset := (page - 1) * pageSize

	// 获取总数
	DB.Model(&User{}).Count(&total)

	// 获取分页数据
	if err := DB.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %v", err)
	}

	return users, total, nil
}

// CreateDefaultAdminUser 创建默认管理员用户
func CreateDefaultAdminUser() error {
	username := "admin"
	password := "admin123"

	// 检查是否存在管理员用户
	var count int64
	DB.Model(&User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		common.NewLogger().Warn("管理员用户已存在")
		return nil
	}

	// 创建默认管理员用户
	user := &User{
		Username: username,
		Email:    "admin@steadydns.local",
		Password: password,
	}

	if err := CreateUser(user); err != nil {
		return fmt.Errorf("创建管理员用户失败: %v", err)
	}

	common.NewLogger().Info("默认管理员用户创建成功: %s / %s", username, password)
	return nil
}
