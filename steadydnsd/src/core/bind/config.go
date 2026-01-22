// core/bind/config.go
// 配置文件管理

package bind

import (
	"fmt"
	"os"
	"regexp"
)

// addZoneToNamedConf 向named.conf添加zone配置
func (bm *BindManager) addZoneToNamedConf(filePath, domain, zoneFile, allowQuery string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取named.conf文件失败: %v", err)
	}

	// 检查zone是否已存在
	zoneRegex := regexp.MustCompile(fmt.Sprintf(`zone\s+"%s"\s+IN`, regexp.QuoteMeta(domain)))
	if zoneRegex.Match(content) {
		return fmt.Errorf("权威域已存在")
	}

	// 生成zone配置 - 使用双引号字符串，确保换行符正确处理
	zoneConfig := fmt.Sprintf("\nzone \"%s\" IN {\n    type master;\n    file \"%s\";\n    allow-query { %s; };\n};\n", domain, zoneFile, allowQuery)

	// 追加到文件末尾
	content = append(content, []byte(zoneConfig)...)

	// 写回文件
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("写入named.conf文件失败: %v", err)
	}

	return nil
}

// removeZoneFromNamedConf 从named.conf移除zone配置
func (bm *BindManager) removeZoneFromNamedConf(filePath, domain string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取named.conf文件失败: %v", err)
	}

	// 匹配整个zone配置块
	zoneRegex := regexp.MustCompile(fmt.Sprintf(`\nzone \"%s\" IN \{[^\}]*\};`, regexp.QuoteMeta(domain)))
	newContent := zoneRegex.ReplaceAll(content, []byte(""))

	// 写回文件
	if err := os.WriteFile(filePath, newContent, 0644); err != nil {
		return fmt.Errorf("写入named.conf文件失败: %v", err)
	}

	return nil
}
