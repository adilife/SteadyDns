// core/sdns/domain_trie.go
// 域名匹配Trie树实现 - 使用反向域名存储优化最长后缀匹配

package sdns

import (
	"fmt"
	"strings"
	"sync"
)

// DomainTrieNode Trie树节点结构
type DomainTrieNode struct {
	children map[string]*DomainTrieNode // 子节点映射，键为域名标签
	group    *ForwardGroup              // 该节点对应的转发组（如果是完整域名）
	mu       sync.RWMutex               // 保护节点的读写锁
}

// DomainTrie 域名Trie树结构
type DomainTrie struct {
	root *DomainTrieNode // Trie树的根节点
	mu   sync.RWMutex    // 保护整个Trie树的读写锁
}

// NewDomainTrieNode 创建新的Trie树节点
func NewDomainTrieNode() *DomainTrieNode {
	return &DomainTrieNode{
		children: make(map[string]*DomainTrieNode),
		group:    nil,
	}
}

// NewDomainTrie 创建新的域名Trie树
func NewDomainTrie() *DomainTrie {
	return &DomainTrie{
		root: NewDomainTrieNode(),
	}
}

// reverseDomainLabels 反转域名标签
// 例如: www.example.com -> [com, example, www]
func reverseDomainLabels(domain string) []string {
	// 移除末尾的点
	domain = strings.TrimSuffix(domain, ".")
	// 分割域名
	labels := strings.Split(domain, ".")
	// 反转标签数组
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return labels
}

// Insert 插入域名和对应的转发组
// domain: 要插入的域名
// group: 域名对应的转发组
// 返回: 错误信息（如果有）
func (t *DomainTrie) Insert(domain string, group *ForwardGroup) error {
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}

	if group == nil {
		return fmt.Errorf("转发组不能为空")
	}

	// 反转域名标签
	labels := reverseDomainLabels(domain)

	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.root

	for _, label := range labels {
		if label == "" {
			return fmt.Errorf("域名标签不能为空")
		}

		current.mu.Lock()
		if _, exists := current.children[label]; !exists {
			current.children[label] = NewDomainTrieNode()
		}
		next := current.children[label]
		current.mu.Unlock()

		current = next
	}

	// 设置转发组
	current.mu.Lock()
	current.group = group
	current.mu.Unlock()

	return nil
}

// Search 搜索最长匹配的转发组
// domain: 要搜索的域名
// 返回: 最长匹配的转发组，如果没有匹配则返回nil
func (t *DomainTrie) Search(domain string) *ForwardGroup {
	if domain == "" {
		return nil
	}

	// 反转域名标签
	labels := reverseDomainLabels(domain)

	t.mu.RLock()
	defer t.mu.RUnlock()

	current := t.root
	var lastMatchedGroup *ForwardGroup

	for _, label := range labels {
		current.mu.RLock()
		child, exists := current.children[label]
		current.mu.RUnlock()

		if !exists {
			break
		}

		current = child

		// 检查当前节点是否有转发组（即完整域名匹配）
		current.mu.RLock()
		if current.group != nil {
			lastMatchedGroup = current.group
		}
		current.mu.RUnlock()
	}

	return lastMatchedGroup
}

// SearchWithZone 搜索最长匹配的转发组和对应的 zone
// domain: 要搜索的域名
// 返回: 最长匹配的转发组，匹配的zone，如果没有匹配则返回(nil, "")
func (t *DomainTrie) SearchWithZone(domain string) (*ForwardGroup, string) {
	if domain == "" {
		return nil, ""
	}

	// 反转域名标签
	labels := reverseDomainLabels(domain)

	t.mu.RLock()
	defer t.mu.RUnlock()

	current := t.root
	var lastMatchedGroup *ForwardGroup
	var lastMatchedZone string

	for i, label := range labels {
		current.mu.RLock()
		child, exists := current.children[label]
		current.mu.RUnlock()

		if !exists {
			break
		}

		current = child

		// 检查当前节点是否有转发组（即完整域名匹配）
		current.mu.RLock()
		if current.group != nil {
			lastMatchedGroup = current.group
			// 重建匹配的zone
			matchedLabels := labels[:i+1]
			// 反转回来
			for j, k := 0, len(matchedLabels)-1; j < k; j, k = j+1, k-1 {
				matchedLabels[j], matchedLabels[k] = matchedLabels[k], matchedLabels[j]
			}
			lastMatchedZone = strings.Join(matchedLabels, ".")
		}
		current.mu.RUnlock()
	}

	return lastMatchedGroup, lastMatchedZone
}

// Delete 删除域名
// domain: 要删除的域名
func (t *DomainTrie) Delete(domain string) {
	if domain == "" {
		return
	}

	// 反转域名标签
	labels := reverseDomainLabels(domain)

	t.mu.Lock()
	defer t.mu.Unlock()

	// 使用递归辅助函数删除节点
	t.deleteHelper(t.root, labels, 0)
}

// deleteHelper 递归删除辅助函数
// 返回: 是否可以删除当前节点（没有子节点且没有转发组）
func (t *DomainTrie) deleteHelper(node *DomainTrieNode, labels []string, index int) bool {
	// 如果已经处理完所有标签，清除该节点的转发组
	if index == len(labels) {
		node.mu.Lock()
		node.group = nil
		node.mu.Unlock()

		// 检查是否可以删除当前节点
		node.mu.RLock()
		canDelete := len(node.children) == 0 && node.group == nil
		node.mu.RUnlock()

		return canDelete
	}

	label := labels[index]

	node.mu.RLock()
	child, exists := node.children[label]
	node.mu.RUnlock()

	if !exists {
		return false // 域名不存在，不需要删除
	}

	// 递归删除子节点
	shouldDeleteChild := t.deleteHelper(child, labels, index+1)

	if shouldDeleteChild {
		node.mu.Lock()
		delete(node.children, label)
		// 检查当前节点是否也可以删除
		canDelete := len(node.children) == 0 && node.group == nil
		node.mu.Unlock()
		return canDelete
	}

	return false
}

// Clear 清空Trie树
func (t *DomainTrie) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.root = NewDomainTrieNode()
}
