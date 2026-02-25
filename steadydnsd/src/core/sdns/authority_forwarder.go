// core/sdns/authority_forwarder.go
// 权威域转发管理器

package sdns

import (
	"SteadyDNS/core/bind"
	"SteadyDNS/core/common"
	"SteadyDNS/core/plugin"
	"fmt"
	"strings"
	"sync"
)

// AuthorityForwarder 权威域转发管理器
type AuthorityForwarder struct {
	bindManager    *bind.BindManager // 现有的BIND管理器
	authorityZones []string          // 权威域列表，按长度降序排列
	bindAddress    string            // BIND服务器地址
	enabled        bool              // BIND插件是否启用
	mu             sync.RWMutex      // 保护权威域列表的锁
}

// NewAuthorityForwarder 创建权威域转发管理器实例
func NewAuthorityForwarder() *AuthorityForwarder {
	// 创建BIND管理器实例
	bindManager := bind.NewBindManager()

	// 从配置读取BIND服务器地址
	bindAddress := common.GetConfig("BIND", "BIND_ADDRESS")
	if bindAddress == "" {
		bindAddress = "127.0.0.1:5300" // 默认值
	}

	// 检查BIND插件是否启用
	enabled := plugin.GetPluginManager().IsPluginEnabled("bind")

	forwarder := &AuthorityForwarder{
		bindManager:    bindManager,
		authorityZones: []string{},
		bindAddress:    bindAddress,
		enabled:        enabled,
	}

	// 只有BIND插件启用时才加载权威域列表
	if enabled {
		// 初始化权威域列表
		forwarder.LoadAuthorityZones()
	}

	return forwarder
}

// LoadAuthorityZones 加载权威域列表
func (af *AuthorityForwarder) LoadAuthorityZones() error {
	af.mu.Lock()
	defer af.mu.Unlock()

	// 清空现有权威域列表
	af.authorityZones = []string{}

	// 使用现有BindManager获取所有权威域
	zones, err := af.bindManager.GetAuthZones()
	if err != nil {
		return fmt.Errorf("获取权威域失败: %v", err)
	}

	// 提取权威域域名
	for _, zone := range zones {
		af.authorityZones = append(af.authorityZones, zone.Domain)
	}

	// 按域名长度降序排序，用于最长匹配
	for i := 0; i < len(af.authorityZones); i++ {
		for j := i + 1; j < len(af.authorityZones); j++ {
			if len(af.authorityZones[i]) < len(af.authorityZones[j]) {
				af.authorityZones[i], af.authorityZones[j] = af.authorityZones[j], af.authorityZones[i]
			}
		}
	}

	return nil
}

// MatchAuthorityZone 匹配权威域
// 如果BIND插件禁用，直接返回false，不进行匹配
func (af *AuthorityForwarder) MatchAuthorityZone(queryDomain string) (bool, string) {
	// 检查BIND插件是否启用
	if !af.enabled {
		return false, ""
	}

	af.mu.RLock()
	defer af.mu.RUnlock()

	// 移除末尾的点
	queryDomain = strings.TrimSuffix(queryDomain, ".")

	// 尝试最长匹配
	for _, zone := range af.authorityZones {
		// 完全匹配
		if queryDomain == zone {
			return true, zone
		}
		// 前缀匹配
		if strings.HasSuffix(queryDomain, "."+zone) {
			return true, zone
		}
	}

	return false, ""
}

// GetBindAddress 获取BIND服务器地址
func (af *AuthorityForwarder) GetBindAddress() string {
	return af.bindAddress
}

// IsBindPluginEnabled 检查BIND插件是否启用
// 返回值: true表示启用，false表示禁用
func (af *AuthorityForwarder) IsBindPluginEnabled() bool {
	return af.enabled
}

// ReloadAuthorityZones 重新加载权威域列表
// 如果BIND插件禁用，直接返回nil，不进行加载
func (af *AuthorityForwarder) ReloadAuthorityZones() error {
	// 检查BIND插件是否启用
	if !af.enabled {
		return nil
	}
	return af.LoadAuthorityZones()
}
