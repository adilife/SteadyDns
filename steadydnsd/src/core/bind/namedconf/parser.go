// core/bind/namedconf/parser.go
// named.conf 文件解析模块

package namedconf

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConfigElement 配置元素结构
type ConfigElement struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Value         interface{}     `json:"value"`
	Comments      []string        `json:"comments"`      // 元素级注释
	ChildElements []ConfigElement `json:"childElements"` // 子元素
	LineComment   string          `json:"lineComment"`   // 行尾注释
}

// Parser 配置解析器
type Parser struct {
	filePath string
	basePath string
}

// NewParser 创建新的解析器实例
func NewParser(filePath string) *Parser {
	return &Parser{
		filePath: filePath,
		basePath: filepath.Dir(filePath),
	}
}

// Parse 解析 named.conf 文件
func (p *Parser) Parse() (*ConfigElement, error) {
	content, err := os.ReadFile(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	return p.parseContent(string(content), p.basePath)
}

// parseContent 解析配置内容
func (p *Parser) parseContent(content string, basePath string) (*ConfigElement, error) {
	// 根元素
	root := &ConfigElement{
		Name:          "root",
		Type:          "root",
		ChildElements: make([]ConfigElement, 0),
		Comments:      make([]string, 0),
	}

	// 按行解析
	lines := strings.Split(content, "\n")
	lineNum := 0
	currentComments := make([]string, 0)

	for lineNum < len(lines) {
		lineNum++
		line := strings.TrimSpace(lines[lineNum-1])

		// 空行
		if line == "" {
			continue
		}

		// 注释行
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			comment := strings.TrimPrefix(strings.TrimPrefix(line, "//"), "#")
			currentComments = append(currentComments, strings.TrimSpace(comment))
			continue
		}

		// 检查行尾注释
		lineWithoutComment := line
		lineComment := ""
		if strings.Contains(line, "//") {
			parts := strings.Split(line, "//")
			lineWithoutComment = strings.TrimSpace(parts[0])
			lineComment = strings.TrimSpace(parts[1])
		} else if strings.Contains(line, "#") {
			parts := strings.Split(line, "#")
			lineWithoutComment = strings.TrimSpace(parts[0])
			lineComment = strings.TrimSpace(parts[1])
		}

		// 处理 include 指令
		if strings.HasPrefix(lineWithoutComment, "include") {
			element, err := p.parseInclude(lineWithoutComment, lineComment, currentComments, basePath)
			if err != nil {
				return nil, fmt.Errorf("解析 include 指令失败 (行 %d): %v", lineNum, err)
			}
			root.ChildElements = append(root.ChildElements, *element)
			currentComments = make([]string, 0)
			continue
		}

		// 处理配置块
		blockStartRegex := regexp.MustCompile(`^[a-zA-Z0-9_\-]+(?:\s+[a-zA-Z0-9_\-]+)*(?:\s+"[^"]+")?(?:\s+IN)?\s*{`)
		if blockStartRegex.MatchString(lineWithoutComment) {
			// 解析块
			element, newLineNum, err := p.parseBlockWithLines(lineWithoutComment, lineComment, currentComments, lines, lineNum, basePath)
			if err != nil {
				return nil, fmt.Errorf("解析配置块失败 (行 %d): %v", lineNum, err)
			}
			root.ChildElements = append(root.ChildElements, *element)
			currentComments = make([]string, 0)
			lineNum = newLineNum
			continue
		}

		// 处理简单配置项
		element, err := p.parseSimple(lineWithoutComment, lineComment, currentComments)
		if err != nil {
			return nil, fmt.Errorf("解析配置项失败 (行 %d): %v", lineNum, err)
		}
		root.ChildElements = append(root.ChildElements, *element)
		currentComments = make([]string, 0)
	}

	return root, nil
}

// containsUnquotedBrace 检查字符串是否包含未被引号包围的括号
func containsUnquotedBrace(s, brace string) bool {
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(s); i++ {
		char := s[i]

		// 处理转义字符
		if i > 0 && s[i-1] == '\\' {
			continue
		}

		// 处理单引号
		if char == '\'' {
			inSingleQuote = !inSingleQuote
		}

		// 处理双引号
		if char == '"' {
			inDoubleQuote = !inDoubleQuote
		}

		// 检查括号
		if char == brace[0] && !inSingleQuote && !inDoubleQuote {
			return true
		}
	}

	return false
}

// countUnquotedBraces 计算字符串中未被引号包围的括号数量
func countUnquotedBraces(s string) (int, int) {
	leftCount := 0
	rightCount := 0
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(s); i++ {
		char := s[i]

		// 处理转义字符
		if i > 0 && s[i-1] == '\\' {
			continue
		}

		// 处理单引号
		if char == '\'' {
			inSingleQuote = !inSingleQuote
		}

		// 处理双引号
		if char == '"' {
			inDoubleQuote = !inDoubleQuote
		}

		// 检查左括号
		if char == '{' && !inSingleQuote && !inDoubleQuote {
			leftCount++
		}

		// 检查右括号
		if char == '}' && !inSingleQuote && !inDoubleQuote {
			rightCount++
		}
	}

	return leftCount, rightCount
}

// parseInclude 解析 include 指令
func (p *Parser) parseInclude(line, lineComment string, comments []string, basePath string) (*ConfigElement, error) {
	// 匹配 include "file";
	includeRegex := regexp.MustCompile(`include\s+"([^"]+)"\s*;`)
	matches := includeRegex.FindStringSubmatch(line)
	if len(matches) != 2 {
		return nil, fmt.Errorf("无效的 include 指令: %s", line)
	}

	includePath := matches[1]
	// 处理相对路径
	if !filepath.IsAbs(includePath) {
		includePath = filepath.Join(basePath, includePath)
	}

	// 读取包含文件
	includeContent, err := os.ReadFile(includePath)
	if err != nil {
		return nil, fmt.Errorf("读取包含文件失败: %v", err)
	}

	// 解析包含文件内容
	includeRoot, err := p.parseContent(string(includeContent), filepath.Dir(includePath))
	if err != nil {
		return nil, fmt.Errorf("解析包含文件失败: %v", err)
	}

	return &ConfigElement{
		Name:          "include",
		Type:          "include",
		Value:         includePath,
		Comments:      comments,
		LineComment:   lineComment,
		ChildElements: includeRoot.ChildElements,
	}, nil
}

// parseBlockWithLines 解析配置块（使用行数组）
func (p *Parser) parseBlockWithLines(line, lineComment string, comments []string, lines []string, startLineNum int, basePath string) (*ConfigElement, int, error) {
	// 匹配配置块开始: name [value] {
	// 支持更复杂的格式，如 "listen-on port 5300 {" 和 "zone \"example.com\" IN {"

	blockRegex := regexp.MustCompile(`^([a-zA-Z0-9_\-]+(?:\s+[a-zA-Z0-9_\-]+)*)(?:\s+"([^"]+)")?(?:\s+IN)?\s*{`)
	matches := blockRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, startLineNum, fmt.Errorf("无效的配置块开始: %s", line)
	}

	name := matches[1]
	value := ""
	if len(matches) == 3 {
		value = matches[2]
	}

	element := &ConfigElement{
		Name:          name,
		Type:          "block",
		Value:         value,
		Comments:      comments,
		LineComment:   lineComment,
		ChildElements: make([]ConfigElement, 0),
	}

	// 解析块内容
	currentComments := make([]string, 0)
	lineNum := startLineNum

	for lineNum < len(lines) {
		lineNum++
		blockLine := strings.TrimSpace(lines[lineNum-1])

		// 空行
		if blockLine == "" {
			continue
		}

		// 注释行
		if strings.HasPrefix(blockLine, "//") || strings.HasPrefix(blockLine, "#") {
			comment := strings.TrimPrefix(strings.TrimPrefix(blockLine, "//"), "#")
			currentComments = append(currentComments, strings.TrimSpace(comment))
			continue
		}

		// 检查行尾注释
		lineWithoutComment := blockLine
		blockLineComment := ""
		if strings.Contains(blockLine, "//") {
			parts := strings.Split(blockLine, "//")
			lineWithoutComment = strings.TrimSpace(parts[0])
			blockLineComment = strings.TrimSpace(parts[1])
		} else if strings.Contains(blockLine, "#") {
			parts := strings.Split(blockLine, "#")
			lineWithoutComment = strings.TrimSpace(parts[0])
			blockLineComment = strings.TrimSpace(parts[1])
		}

		// 检查是否块结束
		trimmedLine := strings.TrimSpace(lineWithoutComment)
		if trimmedLine == "}" || trimmedLine == "};" {
			return element, lineNum, nil
		}

		// 处理包含块结束的配置项，如 "listen-on port 5300 { 127.0.0.1; };"
		if strings.Contains(lineWithoutComment, "{") && strings.Contains(lineWithoutComment, "}") {
			// 计算左右括号的数量
			leftCount := strings.Count(lineWithoutComment, "{")
			rightCount := strings.Count(lineWithoutComment, "}")

			// 如果左右括号数量相等，说明是单行块
			if leftCount == rightCount {
				// 直接作为简单配置项处理
				childElement, err := p.parseSimple(lineWithoutComment, lineComment, currentComments)
				if err != nil {
					return nil, startLineNum, fmt.Errorf("解析配置项失败: %v", err)
				}
				element.ChildElements = append(element.ChildElements, *childElement)
				currentComments = make([]string, 0)
				continue
			}
		}

		// include 指令
		if strings.HasPrefix(lineWithoutComment, "include") {
			includeElement, err := p.parseInclude(lineWithoutComment, blockLineComment, currentComments, basePath)
			if err != nil {
				return nil, startLineNum, fmt.Errorf("解析 include 指令失败: %v", err)
			}
			element.ChildElements = append(element.ChildElements, *includeElement)
			currentComments = make([]string, 0)
			continue
		}

		// 检查是否是嵌套块
		blockStartRegex := regexp.MustCompile(`^[a-zA-Z0-9_\-]+(?:\s+[a-zA-Z0-9_\-]+)*\s*{`)
		if blockStartRegex.MatchString(lineWithoutComment) {
			// 解析嵌套块
			childBlock, newLineNum, err := p.parseBlockWithLines(lineWithoutComment, blockLineComment, currentComments, lines, lineNum, basePath)
			if err != nil {
				return nil, startLineNum, fmt.Errorf("解析嵌套块失败: %v", err)
			}
			element.ChildElements = append(element.ChildElements, *childBlock)
			currentComments = make([]string, 0)
			lineNum = newLineNum
			continue
		}

		// 简单配置项
		childElement, err := p.parseSimple(lineWithoutComment, blockLineComment, currentComments)
		if err != nil {
			return nil, startLineNum, fmt.Errorf("解析配置项失败: %v", err)
		}
		element.ChildElements = append(element.ChildElements, *childElement)
		currentComments = make([]string, 0)
	}

	return nil, startLineNum, fmt.Errorf("配置块未闭合")
}

// parseBlock 解析配置块（保持向后兼容）
func (p *Parser) parseBlock(line, lineComment string, comments []string, scanner *bufio.Scanner, basePath string) (*ConfigElement, error) {
	// 读取所有剩余的行
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// 使用新的解析方法
	element, _, err := p.parseBlockWithLines(line, lineComment, comments, lines, 0, basePath)
	return element, err
}

// parseSimple 解析简单配置项
func (p *Parser) parseSimple(line, lineComment string, comments []string) (*ConfigElement, error) {
	// 移除分号
	line = strings.TrimSuffix(line, ";")

	// 分割名称和值
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 1 {
		return nil, fmt.Errorf("无效的配置项: %s", line)
	}

	name := parts[0]
	value := ""
	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
		// 移除引号
		value = strings.Trim(value, `"'`)
	}

	return &ConfigElement{
		Name:        name,
		Type:        "simple",
		Value:       value,
		Comments:    comments,
		LineComment: lineComment,
	}, nil
}
