// core/bind/config.go
// 配置文件管理

package bind

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// addZoneToNamedConf 向named.conf添加zone配置
// 参数:
//   - filePath: named.conf 文件路径
//   - domain: 域名
//   - zoneFile: zone 文件名
//   - allowQuery: 允许查询的地址
//   - comment: zone 的前置注释（可选）
func (bm *BindManager) addZoneToNamedConf(filePath, domain, zoneFile, allowQuery, comment string) error {
	// 清理 allowQuery 中的分号，避免格式错误
	allowQuery = strings.TrimSuffix(allowQuery, ";")

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取named.conf文件失败: %v", err)
	}

	// 检查zone是否已存在
	zoneRegex := regexp.MustCompile(fmt.Sprintf(`zone\s+"%s"\s+IN`, regexp.QuoteMeta(domain)))
	if zoneRegex.Match(content) {
		return fmt.Errorf("权威域已存在")
	}

	// 构建zone配置
	var zoneConfig strings.Builder

	// 如果有注释，添加注释行
	if comment != "" {
		commentLines := strings.Split(comment, "\n")
		for _, line := range commentLines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" {
				zoneConfig.WriteString(fmt.Sprintf("// %s\n", trimmedLine))
			}
		}
	}

	// 添加zone配置块
	zoneConfig.WriteString(fmt.Sprintf("zone \"%s\" IN {\n", domain))
	zoneConfig.WriteString(fmt.Sprintf("    type master;\n"))
	zoneConfig.WriteString(fmt.Sprintf("    file \"%s\";\n", zoneFile))
	zoneConfig.WriteString(fmt.Sprintf("    allow-query { %s; };\n", allowQuery))
	zoneConfig.WriteString("};\n")

	// 检查文件末尾的换行符情况
	contentStr := string(content)
	if len(contentStr) == 0 {
		// 空文件，直接追加
		content = append(content, []byte(zoneConfig.String())...)
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
		content = append(content, []byte(zoneConfig.String())...)
	}

	// 写回文件
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("写入named.conf文件失败: %v", err)
	}

	return nil
}

// removeZoneFromNamedConf 从named.conf移除zone配置（包括其前置注释）
func (bm *BindManager) removeZoneFromNamedConf(filePath, domain string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取named.conf文件失败: %v", err)
	}

	contentStr := string(content)

	// 构建匹配模式（使用正则表达式来处理空格变化）
	zoneRegex := regexp.MustCompile(fmt.Sprintf(`zone\s+"%s"\s+IN\s*`, regexp.QuoteMeta(domain)))
	match := zoneRegex.FindStringIndex(contentStr)
	if match == nil {
		return nil // 没有找到该zone，直接返回
	}

	// 找到匹配的开始位置
	zoneStartIndex := match[0]

	// 向前查找注释的开始位置（如果有前置注释）
	commentStartIndex := zoneStartIndex
	if zoneStartIndex > 0 {
		// 从 zoneStartIndex-1 开始向前查找
		lines := strings.Split(contentStr[:zoneStartIndex], "\n")

		// 从后向前遍历，找到注释的起始位置
		commentLineCount := 0
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])

			// 跳过空行
			if line == "" {
				commentLineCount++
				continue
			}

			// 检查是否是注释行
			if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
				commentLineCount++
			} else {
				// 遇到非注释行，停止
				break
			}
		}

		// 如果有注释行，计算注释的起始位置
		if commentLineCount > 0 {
			// 找到注释开始的那一行
			lineStart := 0
			for i := 0; i < len(lines)-commentLineCount; i++ {
				lineStart += len(lines[i]) + 1 // +1 for newline
			}
			commentStartIndex = lineStart
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
	endIndex := commentStartIndex
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
	if endIndex > commentStartIndex {
		// 构建新内容
		newContentStr := contentStr[:commentStartIndex] + contentStr[endIndex:]

		// 清理可能产生的多余空行
		newContentStr = cleanupExtraNewlines(newContentStr)

		newContent := []byte(newContentStr)

		// 写回文件
		if err := os.WriteFile(filePath, newContent, 0644); err != nil {
			return fmt.Errorf("写入named.conf文件失败: %v", err)
		}
	}

	return nil
}

// updateZoneInNamedConf 更新 named.conf 中的 zone 配置（包括注释）
// 参数:
//   - filePath: named.conf 文件路径
//   - domain: 域名
//   - zoneFile: zone 文件名
//   - allowQuery: 允许查询的地址
//   - comment: zone 的前置注释（可选）
func (bm *BindManager) updateZoneInNamedConf(filePath, domain, zoneFile, allowQuery, comment string) error {
	// 先移除旧的 zone 配置（包括注释）
	if err := bm.removeZoneFromNamedConf(filePath, domain); err != nil {
		return fmt.Errorf("移除旧zone配置失败: %v", err)
	}

	// 添加新的 zone 配置（包含新注释）
	if err := bm.addZoneToNamedConf(filePath, domain, zoneFile, allowQuery, comment); err != nil {
		return fmt.Errorf("添加新zone配置失败: %v", err)
	}

	return nil
}

// cleanupExtraNewlines 清理多余的空行
func cleanupExtraNewlines(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	prevEmpty := false

	for _, line := range lines {
		isEmpty := strings.TrimSpace(line) == ""
		if isEmpty && prevEmpty {
			// 跳过连续的空行
			continue
		}
		result = append(result, line)
		prevEmpty = isEmpty
	}

	return strings.Join(result, "\n")
}
