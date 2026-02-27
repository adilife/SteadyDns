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

// core/sdns/domain_trie_test.go
// 域名Trie树测试文件

package sdns

import (
	"testing"
)

// TestDomainTrie_Basic 测试基本的插入、搜索和删除功能
func TestDomainTrie_Basic(t *testing.T) {
	trie := NewDomainTrie()

	// 创建测试转发组
	groupCom := &ForwardGroup{Name: "com"}
	groupExampleCom := &ForwardGroup{Name: "example.com"}
	groupWwwExampleCom := &ForwardGroup{Name: "www.example.com"}

	// 插入域名
	if err := trie.Insert("com", groupCom); err != nil {
		t.Fatalf("插入 com 失败: %v", err)
	}
	if err := trie.Insert("example.com", groupExampleCom); err != nil {
		t.Fatalf("插入 example.com 失败: %v", err)
	}
	if err := trie.Insert("www.example.com", groupWwwExampleCom); err != nil {
		t.Fatalf("插入 www.example.com 失败: %v", err)
	}

	// 测试搜索
	tests := []struct {
		domain     string
		expected   string
	}{
		{"com", "com"},
		{"example.com", "example.com"},
		{"www.example.com", "www.example.com"},
		{"test.example.com", "example.com"},
		{"sub.test.example.com", "example.com"},
		{"test.com", "com"},
		{"unknown.org", ""},
	}

	for _, tt := range tests {
		result := trie.Search(tt.domain)
		var resultName string
		if result != nil {
			resultName = result.Name
		}
		if resultName != tt.expected {
			t.Errorf("搜索 %s 期望 %s, 实际 %s", tt.domain, tt.expected, resultName)
		}
	}

	// 测试删除
	trie.Delete("www.example.com")
	result := trie.Search("www.example.com")
	if result == nil || result.Name != "example.com" {
		t.Errorf("删除 www.example.com 后，搜索 www.example.com 应该返回 example.com")
	}

	// 测试清空
	trie.Clear()
	result = trie.Search("example.com")
	if result != nil {
		t.Errorf("清空 Trie 后，搜索 example.com 应该返回 nil")
	}
}

// TestDomainTrie_Empty 测试空域名处理
func TestDomainTrie_Empty(t *testing.T) {
	trie := NewDomainTrie()
	group := &ForwardGroup{Name: "test"}

	// 测试插入空域名
	if err := trie.Insert("", group); err == nil {
		t.Error("插入空域名应该返回错误")
	}

	// 测试插入空转发组
	if err := trie.Insert("test.com", nil); err == nil {
		t.Error("插入空转发组应该返回错误")
	}

	// 测试搜索空域名
	result := trie.Search("")
	if result != nil {
		t.Error("搜索空域名应该返回 nil")
	}

	// 测试删除空域名
	trie.Delete("") // 应该不会 panic
}

// TestDomainTrie_TrailingDot 测试带末尾点的域名
func TestDomainTrie_TrailingDot(t *testing.T) {
	trie := NewDomainTrie()
	group := &ForwardGroup{Name: "example.com"}

	// 插入带末尾点的域名
	if err := trie.Insert("example.com.", group); err != nil {
		t.Fatalf("插入 example.com. 失败: %v", err)
	}

	// 搜索不带末尾点的域名
	result := trie.Search("example.com")
	if result == nil || result.Name != "example.com" {
		t.Error("搜索 example.com 应该返回 example.com")
	}

	// 搜索带末尾点的域名
	result = trie.Search("www.example.com.")
	if result == nil || result.Name != "example.com" {
		t.Error("搜索 www.example.com. 应该返回 example.com")
	}
}
