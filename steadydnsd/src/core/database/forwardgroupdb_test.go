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
// core/database/forwardgroupdb_test.go
// 转发组数据库操作测试

package database

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupForwardGroupTestDB 创建测试用的内存数据库
func setupForwardGroupTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	DB = db

	if err := DB.AutoMigrate(&User{}, &ForwardGroup{}, &DNSServer{}); err != nil {
		t.Fatalf("迁移表失败: %v", err)
	}

	return func() {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
		DB = nil
	}
}

// TestValidateDNSServerDB 测试DNS服务器验证
func TestValidateDNSServerDB(t *testing.T) {
	tests := []struct {
		name    string
		server  *DNSServer
		wantErr bool
		errMsg  string
	}{
		{"有效IPv4地址", &DNSServer{Address: "192.168.1.1", Port: 53, Priority: 1}, false, ""},
		{"有效IPv6地址", &DNSServer{Address: "2001:4860:4860::8888", Port: 53, Priority: 2}, false, ""},
		{"空地址", &DNSServer{Address: "", Port: 53, Priority: 1}, true, "地址不能为空"},
		{"无效端口-零", &DNSServer{Address: "192.168.1.1", Port: 0, Priority: 1}, true, "端口号必须在1-65535之间"},
		{"无效端口-负数", &DNSServer{Address: "192.168.1.1", Port: -1, Priority: 1}, true, "端口号必须在1-65535之间"},
		{"无效端口-过大", &DNSServer{Address: "192.168.1.1", Port: 65536, Priority: 1}, true, "端口号必须在1-65535之间"},
		{"无效IP地址", &DNSServer{Address: "invalid", Port: 53, Priority: 1}, true, "无效的IP地址"},
		{"优先级-零", &DNSServer{Address: "192.168.1.1", Port: 53, Priority: 0}, true, "优先级必须在1-3之间"},
		{"优先级-过大", &DNSServer{Address: "192.168.1.1", Port: 53, Priority: 4}, true, "优先级必须在1-3之间"},
		{"有效优先级边界-1", &DNSServer{Address: "192.168.1.1", Port: 53, Priority: 1}, false, ""},
		{"有效优先级边界-3", &DNSServer{Address: "192.168.1.1", Port: 53, Priority: 3}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSServerDB(tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDNSServerDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateDNSServerDB() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

// TestValidateForwardGroupDB 测试转发组验证
func TestValidateForwardGroupDB(t *testing.T) {
	tests := []struct {
		name    string
		group   *ForwardGroup
		wantErr bool
		errMsg  string
	}{
		{"有效转发组", &ForwardGroup{Domain: "example.com", Description: "Test Group", Servers: []DNSServer{{Address: "192.168.1.1", Port: 53, Priority: 1}}}, false, ""},
		{"空域名", &ForwardGroup{Domain: "", Description: "Test", Servers: []DNSServer{}}, true, "组域名长度必须在1-255之间"},
		{"超长域名", &ForwardGroup{Domain: strings.Repeat("a", 256), Description: "Test", Servers: []DNSServer{}}, true, "组域名长度必须在1-255之间"},
		{"ID=1跳过验证", &ForwardGroup{ID: 1, Domain: "", Description: "", Servers: []DNSServer{}}, false, ""},
		{"无效服务器", &ForwardGroup{Domain: "example.com", Servers: []DNSServer{{Address: "", Port: 53, Priority: 1}}}, true, "服务器配置错误"},
		{"重复服务器", &ForwardGroup{Domain: "example.com", Servers: []DNSServer{{Address: "192.168.1.1", Port: 53, Priority: 1}, {Address: "192.168.1.1", Port: 53, Priority: 2}}}, true, "存在重复的服务器地址和端口"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateForwardGroupDB(tt.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForwardGroupDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateForwardGroupDB() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

// TestCreateForwardGroup 测试创建转发组
func TestCreateForwardGroup(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		group   *ForwardGroup
		wantErr bool
		errMsg  string
	}{
		{"正常创建", &ForwardGroup{Domain: "example.com", Description: "Test Group"}, false, ""},
		{"重复域名", &ForwardGroup{Domain: "example.com", Description: "Another Group"}, true, "转发组域名已存在"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateForwardGroup(tt.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateForwardGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("CreateForwardGroup() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

// TestGetForwardGroupByID 测试根据ID获取转发组
func TestGetForwardGroupByID(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	group := &ForwardGroup{Domain: "example.com", Description: "Test Group"}
	if err := CreateForwardGroup(group); err != nil {
		t.Fatalf("创建测试转发组失败: %v", err)
	}

	tests := []struct {
		name   string
		id     uint
		domain string
		wantOk bool
	}{
		{"存在的转发组", group.ID, "example.com", true},
		{"不存在的转发组", 9999, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetForwardGroupByID(tt.id)
			if tt.wantOk {
				if err != nil {
					t.Errorf("GetForwardGroupByID(%d) error = %v", tt.id, err)
					return
				}
				if result.Domain != tt.domain {
					t.Errorf("Domain = %v, want %v", result.Domain, tt.domain)
				}
			} else {
				if err == nil {
					t.Errorf("GetForwardGroupByID(%d) 应该返回错误", tt.id)
				}
			}
		})
	}
}

// TestGetForwardGroupByDomain 测试根据域名获取转发组
func TestGetForwardGroupByDomain(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	group := &ForwardGroup{Domain: "example.com", Description: "Test Group"}
	if err := CreateForwardGroup(group); err != nil {
		t.Fatalf("创建测试转发组失败: %v", err)
	}

	tests := []struct {
		name   string
		domain string
		wantOk bool
	}{
		{"存在的域名", "example.com", true},
		{"不存在的域名", "nonexistent.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetForwardGroupByDomain(tt.domain)
			if tt.wantOk {
				if err != nil {
					t.Errorf("GetForwardGroupByDomain(%q) error = %v", tt.domain, err)
					return
				}
				if result.Domain != tt.domain {
					t.Errorf("Domain = %v, want %v", result.Domain, tt.domain)
				}
			} else {
				if err == nil {
					t.Errorf("GetForwardGroupByDomain(%q) 应该返回错误", tt.domain)
				}
			}
		})
	}
}

// TestUpdateForwardGroup 测试更新转发组
func TestUpdateForwardGroup(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	defaultGroup := &ForwardGroup{ID: 1, Domain: "Default", Description: "Default Domain"}
	if err := CreateForwardGroup(defaultGroup); err != nil {
		t.Fatalf("创建默认转发组失败: %v", err)
	}

	normalGroup := &ForwardGroup{Domain: "example.com", Description: "Test Group"}
	if err := CreateForwardGroup(normalGroup); err != nil {
		t.Fatalf("创建普通转发组失败: %v", err)
	}

	t.Run("更新普通组", func(t *testing.T) {
		normalGroup.Description = "Updated Description"
		err := UpdateForwardGroup(normalGroup)
		if err != nil {
			t.Errorf("UpdateForwardGroup() error = %v", err)
			return
		}

		result, err := GetForwardGroupByID(normalGroup.ID)
		if err != nil {
			t.Errorf("GetForwardGroupByID() error = %v", err)
			return
		}
		if result.Description != "Updated Description" {
			t.Errorf("Description = %v, want Updated Description", result.Description)
		}
	})

	t.Run("修改默认组域名失败", func(t *testing.T) {
		defaultGroup.Domain = "modified"
		err := UpdateForwardGroup(defaultGroup)
		if err == nil {
			t.Error("修改默认组域名应该返回错误")
			return
		}
		if !strings.Contains(err.Error(), "域名和描述不可修改") {
			t.Errorf("错误信息应该包含'域名和描述不可修改', 实际: %v", err)
		}
	})

	t.Run("更新不存在的组", func(t *testing.T) {
		nonExistGroup := &ForwardGroup{ID: 9999, Domain: "nonexistent.com"}
		err := UpdateForwardGroup(nonExistGroup)
		if err == nil {
			t.Error("更新不存在的组应该返回错误")
		}
	})
}

// TestDeleteForwardGroup 测试删除转发组
func TestDeleteForwardGroup(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	defaultGroup := &ForwardGroup{ID: 1, Domain: "Default", Description: "Default Domain"}
	if err := CreateForwardGroup(defaultGroup); err != nil {
		t.Fatalf("创建默认转发组失败: %v", err)
	}

	normalGroup := &ForwardGroup{Domain: "example.com", Description: "Test Group"}
	if err := CreateForwardGroup(normalGroup); err != nil {
		t.Fatalf("创建普通转发组失败: %v", err)
	}

	tests := []struct {
		name    string
		id      uint
		wantErr bool
		errMsg  string
	}{
		{"删除普通组", normalGroup.ID, false, ""},
		{"删除默认组", 1, true, "ID=1的转发组是默认组，不可删除"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteForwardGroup(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteForwardGroup(%d) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("DeleteForwardGroup() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

// TestGetForwardGroups 测试获取所有转发组
func TestGetForwardGroups(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	for i := 1; i <= 3; i++ {
		group := &ForwardGroup{
			Domain:      "group" + string(rune('0'+i)),
			Description: "Test Group " + string(rune('0'+i)),
		}
		if err := CreateForwardGroup(group); err != nil {
			t.Fatalf("创建测试转发组失败: %v", err)
		}
	}

	groups, err := GetForwardGroups()
	if err != nil {
		t.Errorf("GetForwardGroups() error = %v", err)
		return
	}

	if len(groups) != 3 {
		t.Errorf("len(groups) = %d, want 3", len(groups))
	}
}

// TestCountForwardGroups 测试统计转发组数量
func TestCountForwardGroups(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	count, err := CountForwardGroups()
	if err != nil {
		t.Errorf("CountForwardGroups() error = %v", err)
		return
	}
	if count != 0 {
		t.Errorf("初始 count = %d, want 0", count)
	}

	for i := 1; i <= 5; i++ {
		group := &ForwardGroup{
			Domain:      "group" + string(rune('0'+i)),
			Description: "Test Group",
		}
		if err := CreateForwardGroup(group); err != nil {
			t.Fatalf("创建测试转发组失败: %v", err)
		}
	}

	count, err = CountForwardGroups()
	if err != nil {
		t.Errorf("CountForwardGroups() error = %v", err)
		return
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

// TestDNSServerCRUD 测试DNS服务器CRUD操作
func TestDNSServerCRUD(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	group := &ForwardGroup{Domain: "example.com", Description: "Test Group"}
	if err := CreateForwardGroup(group); err != nil {
		t.Fatalf("创建测试转发组失败: %v", err)
	}

	t.Run("创建DNS服务器", func(t *testing.T) {
		server := &DNSServer{
			GroupID:     group.ID,
			Address:     "192.168.1.1",
			Port:        53,
			Description: "Primary DNS",
			Priority:    1,
		}
		err := CreateDNSServer(server)
		if err != nil {
			t.Errorf("CreateDNSServer() error = %v", err)
			return
		}
		if server.ID == 0 {
			t.Error("创建后ID应该不为0")
		}
	})

	t.Run("创建重复服务器失败", func(t *testing.T) {
		server := &DNSServer{
			GroupID:  group.ID,
			Address:  "192.168.1.1",
			Port:     53,
			Priority: 2,
		}
		err := CreateDNSServer(server)
		if err == nil {
			t.Error("创建重复服务器应该返回错误")
		}
	})

	t.Run("获取DNS服务器", func(t *testing.T) {
		servers, err := GetDNSServersByGroupID(group.ID)
		if err != nil {
			t.Errorf("GetDNSServersByGroupID() error = %v", err)
			return
		}
		if len(servers) != 1 {
			t.Errorf("len(servers) = %d, want 1", len(servers))
		}
	})

	t.Run("更新DNS服务器", func(t *testing.T) {
		servers, _ := GetDNSServersByGroupID(group.ID)
		server := servers[0]
		server.Description = "Updated DNS"
		server.Priority = 2

		err := UpdateDNSServer(&server)
		if err != nil {
			t.Errorf("UpdateDNSServer() error = %v", err)
			return
		}

		updated, err := GetDNSServerByID(server.ID)
		if err != nil {
			t.Errorf("GetDNSServerByID() error = %v", err)
			return
		}
		if updated.Description != "Updated DNS" {
			t.Errorf("Description = %v, want Updated DNS", updated.Description)
		}
		if updated.Priority != 2 {
			t.Errorf("Priority = %d, want 2", updated.Priority)
		}
	})

	t.Run("删除DNS服务器", func(t *testing.T) {
		servers, _ := GetDNSServersByGroupID(group.ID)
		serverID := servers[0].ID

		err := DeleteDNSServer(serverID)
		if err != nil {
			t.Errorf("DeleteDNSServer() error = %v", err)
			return
		}

		_, err = GetDNSServerByID(serverID)
		if err == nil {
			t.Error("删除后获取应该返回错误")
		}
	})
}

// TestBatchDeleteForwardGroups 测试批量删除转发组
func TestBatchDeleteForwardGroups(t *testing.T) {
	cleanup := setupForwardGroupTestDB(t)
	defer cleanup()

	var groupIDs []uint
	for i := 1; i <= 3; i++ {
		group := &ForwardGroup{
			Domain:      "batch" + string(rune('0'+i)),
			Description: "Batch Test",
		}
		if err := CreateForwardGroup(group); err != nil {
			t.Fatalf("创建测试转发组失败: %v", err)
		}
		groupIDs = append(groupIDs, group.ID)
	}

	err := BatchDeleteForwardGroups(groupIDs)
	if err != nil {
		t.Errorf("BatchDeleteForwardGroups() error = %v", err)
		return
	}

	count, _ := CountForwardGroups()
	if count != 0 {
		t.Errorf("批量删除后 count = %d, want 0", count)
	}
}
