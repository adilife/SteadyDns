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
// core/database/usersdb_test.go
// 用户数据库操作测试

package database

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试用的内存数据库
func setupTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	DB = db

	if err := DB.AutoMigrate(&User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}

	return func() {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
		DB = nil
	}
}

// TestIsBcryptHash 测试bcrypt哈希检测函数
func TestIsBcryptHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"bcrypt $2a$ 格式", "$2a$12$abcdefghijklmnopqrstuvwx", true},
		{"bcrypt $2b$ 格式", "$2b$12$abcdefghijklmnopqrstuvwx", true},
		{"bcrypt $2y$ 格式", "$2y$12$abcdefghijklmnopqrstuvwx", true},
		{"普通密码", "admin123", false},
		{"空密码", "", false},
		{"其他哈希格式", "$5$rounds=5000$salt$hash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBcryptHash(tt.password)
			if result != tt.expected {
				t.Errorf("isBcryptHash(%q) = %v, want %v", tt.password, result, tt.expected)
			}
		})
	}
}

// TestHashPassword 测试密码哈希函数
func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"普通密码", "admin123", false},
		{"空密码", "", false},
		{"中等长度密码", strings.Repeat("a", 50), false},
		{"已经是bcrypt格式", "$2a$12$abcdefghijklmnopqrstuvwx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("hashPassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
				return
			}

			if isBcryptHash(tt.password) {
				if result != tt.password {
					t.Errorf("hashPassword(已哈希密码) 应该返回原密码")
				}
			} else if tt.password != "" {
				if !isBcryptHash(result) {
					t.Errorf("hashPassword(%q) = %q, 应该是bcrypt格式", tt.password, result)
				}
			}
		})
	}
}

// TestCreateUser 测试创建用户
func TestCreateUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		user    *User
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常创建用户",
			user:    &User{Username: "testuser", Email: "test@example.com", Password: "password123"},
			wantErr: false,
		},
		{
			name:    "创建无邮箱用户",
			user:    &User{Username: "noemail", Password: "password123"},
			wantErr: false,
		},
		{
			name:    "重复用户名",
			user:    &User{Username: "testuser", Email: "another@example.com", Password: "password456"},
			wantErr: true,
			errMsg:  "用户名已存在",
		},
		{
			name:    "重复邮箱",
			user:    &User{Username: "anotheruser", Email: "test@example.com", Password: "password789"},
			wantErr: true,
			errMsg:  "邮箱已存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateUser(tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("CreateUser() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

// TestGetUserByID 测试根据ID获取用户
func TestGetUserByID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user := &User{Username: "testuser", Email: "test@example.com", Password: "password123"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name     string
		id       uint
		wantErr  bool
		username string
	}{
		{"存在的用户", user.ID, false, "testuser"},
		{"不存在的用户", 9999, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetUserByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID(%d) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.Username != tt.username {
				t.Errorf("GetUserByID(%d).Username = %v, want %v", tt.id, result.Username, tt.username)
			}
		})
	}
}

// TestGetUserByUsername 测试根据用户名获取用户
func TestGetUserByUsername(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user := &User{Username: "testuser", Email: "test@example.com", Password: "password123"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"存在的用户", "testuser", false},
		{"不存在的用户", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetUserByUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.Username != tt.username {
				t.Errorf("GetUserByUsername(%q).Username = %v, want %v", tt.username, result.Username, tt.username)
			}
		})
	}
}

// TestGetUserByEmail 测试根据邮箱获取用户
func TestGetUserByEmail(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user := &User{Username: "testuser", Email: "test@example.com", Password: "password123"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"存在的邮箱", "test@example.com", false},
		{"不存在的邮箱", "nonexistent@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetUserByEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.Email != tt.email {
				t.Errorf("GetUserByEmail(%q).Email = %v, want %v", tt.email, result.Email, tt.email)
			}
		})
	}
}

// TestUpdateUser 测试更新用户
func TestUpdateUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user := &User{Username: "testuser", Email: "test@example.com", Password: "password123"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	t.Run("更新邮箱", func(t *testing.T) {
		user.Email = "newemail@example.com"
		err := UpdateUser(user)
		if err != nil {
			t.Errorf("UpdateUser() error = %v", err)
			return
		}

		result, err := GetUserByID(user.ID)
		if err != nil {
			t.Errorf("GetUserByID() error = %v", err)
			return
		}
		if result.Email != "newemail@example.com" {
			t.Errorf("Email = %v, want newemail@example.com", result.Email)
		}
	})

	t.Run("更新密码", func(t *testing.T) {
		user.Password = "newpassword456"
		err := UpdateUser(user)
		if err != nil {
			t.Errorf("UpdateUser() error = %v", err)
			return
		}

		result, err := GetUserByID(user.ID)
		if err != nil {
			t.Errorf("GetUserByID() error = %v", err)
			return
		}
		if !isBcryptHash(result.Password) {
			t.Errorf("Password 应该是bcrypt格式")
		}
	})
}

// TestDeleteUser 测试删除用户
func TestDeleteUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	adminUser := &User{Username: "admin", Email: "admin@example.com", Password: "admin123"}
	if err := CreateUser(adminUser); err != nil {
		t.Fatalf("创建admin用户失败: %v", err)
	}

	normalUser := &User{Username: "normaluser", Email: "normal@example.com", Password: "password123"}
	if err := CreateUser(normalUser); err != nil {
		t.Fatalf("创建普通用户失败: %v", err)
	}

	tests := []struct {
		name    string
		id      uint
		wantErr bool
		errMsg  string
	}{
		{"删除普通用户", normalUser.ID, false, ""},
		{"删除admin用户", adminUser.ID, true, "不能删除默认管理员用户"},
		{"删除不存在的用户", 9999, true, "用户不存在"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteUser(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteUser(%d) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("DeleteUser() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

// TestValidateUserWithDB 测试用户验证
func TestValidateUserWithDB(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user := &User{Username: "testuser", Email: "test@example.com", Password: "password123"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		wantOk   bool
	}{
		{"正确的凭据", "testuser", "password123", true},
		{"错误的密码", "testuser", "wrongpassword", false},
		{"不存在的用户", "nonexistent", "password123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ValidateUserWithDB(tt.username, tt.password)
			if ok != tt.wantOk {
				t.Errorf("ValidateUserWithDB(%q, %q) = %v, want %v", tt.username, tt.password, ok, tt.wantOk)
			}
		})
	}
}

// TestGetAllUsers 测试获取所有用户
func TestGetAllUsers(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	for i := 1; i <= 15; i++ {
		user := &User{
			Username: "testuser" + string(rune('A'+i-1)),
			Email:    "testuser" + string(rune('A'+i-1)) + "@example.com",
			Password: "password123",
		}
		if err := CreateUser(user); err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}
	}

	t.Run("第一页", func(t *testing.T) {
		users, total, err := GetAllUsers(1, 10)
		if err != nil {
			t.Errorf("GetAllUsers() error = %v", err)
			return
		}
		if total != 15 {
			t.Errorf("total = %d, want 15", total)
		}
		if len(users) > 10 {
			t.Errorf("len(users) = %d, should <= 10", len(users))
		}
	})

	t.Run("第二页", func(t *testing.T) {
		users, total, err := GetAllUsers(2, 10)
		if err != nil {
			t.Errorf("GetAllUsers() error = %v", err)
			return
		}
		if total != 15 {
			t.Errorf("total = %d, want 15", total)
		}
		if len(users) > 5 {
			t.Errorf("len(users) = %d, should <= 5", len(users))
		}
	})
}
