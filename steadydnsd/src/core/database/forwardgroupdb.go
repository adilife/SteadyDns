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

// core/database/forwardgroupdb.go

package database

import (
	"fmt"
	"net"
	"time"

	"gorm.io/gorm"
)

// ForwardGroup 转发组模型
type ForwardGroup struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	Domain      string      `json:"domain" gorm:"size:255;not null;unique"` // 转发组域名，长度0-255
	Description string      `json:"description" gorm:"size:65535"`          // 描述，长度0-65535
	Enable      bool        `json:"enable" gorm:"default:true"`             // 是否启用，默认启用
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Servers     []DNSServer `json:"servers" gorm:"foreignKey:GroupID"` // 关联的DNS服务器
}

// DNSServer DNS服务器模型
type DNSServer struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GroupID     uint      `json:"group_id" gorm:"index"`         // 关联的转发组ID
	Address     string    `json:"address" gorm:"not null"`       // DNS服务器地址，支持IPv4/IPv6
	Port        int       `json:"port" gorm:"default:53"`        // 端口，默认53
	Description string    `json:"description" gorm:"size:65535"` // 描述，长度0-65535
	QueueIndex  int       `json:"queue_index"`                   // 队列序号
	Priority    int       `json:"priority" gorm:"default:1"`     // 优先级 (1-3)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetForwardGroups 获取所有转发组
func GetForwardGroups() ([]ForwardGroup, error) {
	var groups []ForwardGroup
	result := DB.Preload("Servers").Find(&groups)
	if result.Error != nil {
		return nil, fmt.Errorf("获取转发组列表失败: %v", result.Error)
	}
	return groups, nil
}

// GetForwardGroupByID 根据ID获取转发组
func GetForwardGroupByID(id uint) (*ForwardGroup, error) {
	var group ForwardGroup
	if err := DB.Preload("Servers").First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("转发组不存在")
		}
		return nil, err
	}
	return &group, nil
}

// CreateForwardGroup 创建转发组
func CreateForwardGroup(group *ForwardGroup) error {
	// 处理ID=1的默认转发组
	if group.ID == 1 {
		// 设置默认域名和描述
		group.Domain = "Default"
		group.Description = "Default Domain"
	}

	// 验证组域名唯一性
	var existingGroup ForwardGroup
	if err := DB.Where("domain = ?", group.Domain).First(&existingGroup).Error; err == nil {
		return fmt.Errorf("转发组域名已存在")
	}

	// 创建转发组
	if err := DB.Create(group).Error; err != nil {
		return fmt.Errorf("创建转发组失败: %v", err)
	}

	return nil
}

// UpdateForwardGroup 更新转发组
func UpdateForwardGroup(group *ForwardGroup) error {
	// 检查转发组是否存在
	var existingGroup ForwardGroup
	if err := DB.First(&existingGroup, group.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("转发组不存在")
		}
		return err
	}

	// ID=1的转发组是默认组，禁止修改域名和描述
	if group.ID == 1 {
		// 检查是否尝试修改域名或描述
		if group.Domain != existingGroup.Domain || group.Description != existingGroup.Description {
			return fmt.Errorf("ID=1的转发组是默认组，域名和描述不可修改")
		}
	}

	// 检查域名是否被其他组使用
	var otherGroup ForwardGroup
	if group.ID != 1 {
		if err := DB.Where("domain = ? AND id != ?", group.Domain, group.ID).First(&otherGroup).Error; err == nil {
			return fmt.Errorf("转发组域名已被其他组使用")
		}
	}

	// 使用事务来确保数据一致性
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新转发组基本信息
	updateData := make(map[string]interface{})
	updateData["enable"] = group.Enable

	// 只有非默认组允许更新域名和描述
	if group.ID != 1 {
		updateData["domain"] = group.Domain
		updateData["description"] = group.Description
	}

	if err := tx.Model(&existingGroup).Updates(updateData).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新转发组失败: %v", err)
	}

	// 只有当提供了服务器信息时才执行服务器相关操作
	if group.Servers != nil {
		// 删除旧的服务器记录
		if err := tx.Where("group_id = ?", group.ID).Delete(&DNSServer{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("删除旧服务器记录失败: %v", err)
		}

		// 添加新的服务器记录
		for i := range group.Servers {
			group.Servers[i].GroupID = group.ID
			if err := tx.Create(&group.Servers[i]).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建服务器记录失败: %v", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// DeleteForwardGroup 删除转发组
func DeleteForwardGroup(id uint) error {
	// ID=1的转发组是默认组，禁止删除
	if id == 1 {
		return fmt.Errorf("ID=1的转发组是默认组，不可删除")
	}

	// 删除相关服务器记录
	if err := DB.Where("group_id = ?", id).Delete(&DNSServer{}).Error; err != nil {
		return fmt.Errorf("删除服务器记录失败: %v", err)
	}

	// 删除转发组
	if err := DB.Delete(&ForwardGroup{}, id).Error; err != nil {
		return fmt.Errorf("删除转发组失败: %v", err)
	}

	return nil
}

// GetForwardGroupByDomain 根据域名获取转发组
func GetForwardGroupByDomain(domain string) (*ForwardGroup, error) {
	var group ForwardGroup
	if err := DB.Preload("Servers").Where("domain = ?", domain).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("转发组不存在")
		}
		return nil, err
	}
	return &group, nil
}

// CountForwardGroups 获取转发组总数
func CountForwardGroups() (int64, error) {
	var count int64
	if err := DB.Model(&ForwardGroup{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取转发组总数失败: %v", err)
	}
	return count, nil
}

// BatchDeleteForwardGroups 批量删除转发组
func BatchDeleteForwardGroups(ids []uint) error {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除相关服务器记录
	if err := tx.Where("group_id IN ?", ids).Delete(&DNSServer{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除服务器记录失败: %v", err)
	}

	// 删除转发组
	if err := tx.Delete(&ForwardGroup{}, "id IN ?", ids).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除转发组失败: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// ValidateForwardGroupDB 验证转发组配置
func ValidateForwardGroupDB(group *ForwardGroup) error {
	// ID=1的转发组是默认组，跳过域名和描述的验证
	if group.ID != 1 {
		if len(group.Domain) == 0 || len(group.Domain) > 255 {
			return fmt.Errorf("组域名长度必须在1-255之间")
		}

		if len(group.Description) > 65535 {
			return fmt.Errorf("描述长度不能超过65535")
		}
	}

	// 检查是否有重复的服务器地址:端口组合
	serverMap := make(map[string]bool)
	for _, server := range group.Servers {
		if err := ValidateDNSServerDB(&server); err != nil {
			return fmt.Errorf("服务器配置错误: %v", err)
		}

		serverKey := fmt.Sprintf("%s:%d", server.Address, server.Port)
		if serverMap[serverKey] {
			return fmt.Errorf("存在重复的服务器地址和端口: %s", serverKey)
		}
		serverMap[serverKey] = true
	}

	return nil
}

func ValidateDNSServerDB(server *DNSServer) error {
	if server.Address == "" {
		return fmt.Errorf("DNS服务器地址不能为空")
	}

	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}

	if len(server.Description) > 65535 {
		return fmt.Errorf("描述长度不能超过65535")
	}

	// 验证地址格式
	ip := net.ParseIP(server.Address)
	if ip == nil {
		return fmt.Errorf("无效的IP地址: %s", server.Address)
	}

	if server.Priority < 1 || server.Priority > 3 {
		return fmt.Errorf("优先级必须在1-3之间")
	}

	return nil
}

// CreateDNSServer 创建DNS服务器
func CreateDNSServer(server *DNSServer) error {
	// 检查关联的转发组是否存在
	var group ForwardGroup
	if err := DB.First(&group, server.GroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("转发组不存在")
		}
		return err
	}

	// 检查该转发组中是否已存在相同的地址和端口
	var existingServer DNSServer
	if err := DB.Where("group_id = ? AND address = ? AND port = ?", server.GroupID, server.Address, server.Port).First(&existingServer).Error; err == nil {
		return fmt.Errorf("该转发组中已存在相同地址和端口的服务器")
	}

	if err := DB.Create(server).Error; err != nil {
		return fmt.Errorf("创建服务器失败: %v", err)
	}

	return nil
}

// GetDNSServerByID 根据ID获取DNS服务器
func GetDNSServerByID(id uint) (*DNSServer, error) {
	var server DNSServer
	if err := DB.First(&server, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("服务器不存在")
		}
		return nil, err
	}
	return &server, nil
}

// UpdateDNSServer 更新DNS服务器
func UpdateDNSServer(server *DNSServer) error {
	// 检查服务器是否存在
	var existingServer DNSServer
	if err := DB.First(&existingServer, server.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("服务器不存在")
		}
		return err
	}

	// 检查组是否存在
	var group ForwardGroup
	if err := DB.First(&group, server.GroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("转发组不存在")
		}
		return err
	}

	// 检查该转发组中是否已存在相同的地址和端口（排除当前服务器）
	var otherServer DNSServer
	if err := DB.Where("group_id = ? AND address = ? AND port = ? AND id != ?", server.GroupID, server.Address, server.Port, server.ID).First(&otherServer).Error; err == nil {
		return fmt.Errorf("该转发组中已存在相同地址和端口的服务器")
	}

	// 更新服务器
	if err := DB.Model(&existingServer).Updates(DNSServer{
		Address:     server.Address,
		Port:        server.Port,
		Description: server.Description,
		QueueIndex:  server.QueueIndex,
		Priority:    server.Priority,
	}).Error; err != nil {
		return fmt.Errorf("更新服务器失败: %v", err)
	}

	return nil
}

// DeleteDNSServer 删除DNS服务器
func DeleteDNSServer(id uint) error {
	// 检查服务器是否存在
	var server DNSServer
	if err := DB.First(&server, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("服务器不存在")
		}
		return err
	}

	// 删除服务器
	if err := DB.Delete(&DNSServer{}, id).Error; err != nil {
		return fmt.Errorf("删除服务器失败: %v", err)
	}

	return nil
}

// GetDNSServersByGroupID 根据转发组ID获取服务器列表
func GetDNSServersByGroupID(groupID uint) ([]DNSServer, error) {
	var servers []DNSServer
	if err := DB.Where("group_id = ?", groupID).Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("获取服务器列表失败: %v", err)
	}
	return servers, nil
}

// GetAllDNSServers 获取所有DNS服务器列表，按组ID、优先级、地址排序
// 返回结果按以下顺序排序：
//  1. 组ID升序
//  2. 优先级降序（高优先级在前）
//  3. 地址升序
func GetAllDNSServers() ([]DNSServer, error) {
	var servers []DNSServer
	if err := DB.Order("group_id ASC, priority DESC, address ASC").Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("获取所有服务器列表失败: %v", err)
	}
	return servers, nil
}
