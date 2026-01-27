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

	// 测试场景1: 完全匹配
	isAuthority, zone := forwarder.MatchAuthorityZone("jcgov.gov.cn")
	if !isAuthority {
		t.Error("Expected to match authority zone jcgov.gov.cn")
	}
	if zone != "jcgov.gov.cn" {
		t.Errorf("Expected zone to be jcgov.gov.cn, got %s", zone)
	}

	// 测试场景2: 前缀匹配
	isAuthority, zone = forwarder.MatchAuthorityZone("www.jcgov.gov.cn")
	if !isAuthority {
		t.Error("Expected to match authority zone jcgov.gov.cn for www.jcgov.gov.cn")
	}
	if zone != "jcgov.gov.cn" {
		t.Errorf("Expected zone to be jcgov.gov.cn, got %s", zone)
	}

	// 测试场景3: 不匹配
	isAuthority, zone = forwarder.MatchAuthorityZone("example.com")
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
	err := forwarder.ReloadAuthorityZones()
	if err != nil {
		t.Errorf("Expected no error when reloading authority zones, got %v", err)
	}
}
