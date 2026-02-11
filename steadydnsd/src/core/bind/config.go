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

	// 生成zone配置
	zoneConfig := fmt.Sprintf("zone \"%s\" IN {\n    type master;\n    file \"%s\";\n    allow-query { %s; };\n};\n", domain, zoneFile, allowQuery)

	// 检查文件末尾的换行符情况
	contentStr := string(content)
	if len(contentStr) == 0 {
		// 空文件，直接追加
		content = append(content, []byte(zoneConfig)...)
	} else {
		// 检查末尾有多少个换行符
		trailingNewlines := 0
		for i := len(contentStr) - 1; i >= 0 && contentStr[i] == '\n'; i-- {
			trailingNewlines++
		}

		if trailingNewlines == 0 {
			// 没有换行符，添加两个换行符（空行+分隔）
			content = append(content, []byte("\n\n")...)
		} else if trailingNewlines == 1 {
			// 只有一个换行符，再添加一个（形成空行）
			content = append(content, []byte("\n")...)
		}
		// 如果已有2个或更多换行符，不再添加

		// 追加zone配置
		content = append(content, []byte(zoneConfig)...)
	}

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

	contentStr := string(content)

	// 构建匹配模式（使用正则表达式来处理空格变化）
	zoneRegex := regexp.MustCompile(fmt.Sprintf(`zone\s+\"%s\"\s+IN\s*`, regexp.QuoteMeta(domain)))
	match := zoneRegex.FindStringIndex(contentStr)
	if match == nil {
		return nil // 没有找到该zone，直接返回
	}
	// 找到匹配的开始位置，确保包括前面的换行符
	startIndex := match[0]
	// 如果不是文件开头，向前查找换行符
	if startIndex > 0 {
		for i := startIndex - 1; i >= 0; i-- {
			if contentStr[i] == '\n' {
				startIndex = i
				break
			}
		}
		// 继续向前查找是否有额外的空行（连续的换行符）
		for startIndex > 0 && contentStr[startIndex-1] == '\n' {
			startIndex--
		}
	}
	// 从匹配的结束位置向后搜索，找到zone配置的第一个大括号
	searchStart := match[1]
	firstBraceIndex := -1
	for i := searchStart; i < len(contentStr); i++ {
		if contentStr[i] == '{' {
			firstBraceIndex = i
			break
		}
	}

	if firstBraceIndex == -1 {
		return fmt.Errorf("未找到zone配置的开始大括号")
	}

	// 从第一个大括号开始匹配
	braceCount := 0
	endIndex := startIndex
	foundOpeningBrace := false

	for i := firstBraceIndex; i < len(contentStr); i++ {
		if contentStr[i] == '{' {
			braceCount++
			foundOpeningBrace = true
		} else if contentStr[i] == '}' {
			braceCount--
			if foundOpeningBrace && braceCount == 0 {
				// 找到匹配的结束大括号，包含后面的分号
				endIndex = i + 1
				// 跳过可能的分号
				for endIndex < len(contentStr) && contentStr[endIndex] == ';' {
					endIndex++
				}
				break
			}
		}
	}

	// 确保找到完整的配置块
	if endIndex > startIndex {
		// 构建新内容
		newContentStr := contentStr[:startIndex] + contentStr[endIndex:]
		newContent := []byte(newContentStr)

		// 写回文件
		if err := os.WriteFile(filePath, newContent, 0644); err != nil {
			return fmt.Errorf("写入named.conf文件失败: %v", err)
		}
	}

	return nil
}
