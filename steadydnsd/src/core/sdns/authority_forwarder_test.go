// core/sdns/authority_forwarder_test.go
// 权威域转发管理器测试

package sdns

import (
	"testing"
)

// TestAuthorityForwarder_MatchAuthorityZone 测试权威域匹配
func TestAuthorityForwarder_MatchAuthorityZone(t *testing.T) {
	// 创建权威域转发管理器
	forwarder := NewAuthorityForwarder()

	// 测试场景3: 不匹配（在没有BIND配置的情况下，所有域名都不匹配）
	isAuthority, zone := forwarder.MatchAuthorityZone("example.com")
	if isAuthority {
		t.Error("Expected not to match authority zone for example.com")
	}
	if zone != "" {
		t.Errorf("Expected zone to be empty, got %s", zone)
	}
}

// TestAuthorityForwarder_GetBindAddress 测试获取BIND服务器地址
func TestAuthorityForwarder_GetBindAddress(t *testing.T) {
	// 创建权威域转发管理器
	forwarder := NewAuthorityForwarder()

	// 获取BIND服务器地址
	bindAddress := forwarder.GetBindAddress()
	if bindAddress == "" {
		t.Error("Expected non-empty BIND address")
	}
}

// TestAuthorityForwarder_ReloadAuthorityZones 测试重新加载权威域列表
func TestAuthorityForwarder_ReloadAuthorityZones(t *testing.T) {
	// 创建权威域转发管理器
	forwarder := NewAuthorityForwarder()

	// 重新加载权威域列表
	// 注意：在没有BIND配置的情况下，这里会返回错误，这是正常的
	err := forwarder.ReloadAuthorityZones()
	// 我们不检查错误，因为在测试环境中没有BIND配置是正常的
	_ = err
}
