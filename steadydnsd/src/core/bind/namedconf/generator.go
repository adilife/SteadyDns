// core/bind/namedconf/generator.go
// named.conf 文件生成模块

package namedconf

import (
	"fmt"
	"strings"
)

// Generator 配置生成器
type Generator struct {
	indentSize int
}

// NewGenerator 创建新的生成器实例
func NewGenerator() *Generator {
	return &Generator{
		indentSize: 4,
	}
}

// Generate 生成 named.conf 文件内容
func (g *Generator) Generate(root *ConfigElement) (string, error) {
	if root == nil {
		return "", fmt.Errorf("根元素不能为空")
	}

	var sb strings.Builder
	g.generateElement(&sb, root, 0)

	return sb.String(), nil
}

// generateElement 生成单个配置元素
func (g *Generator) generateElement(sb *strings.Builder, element *ConfigElement, indent int) {
	// 生成元素级注释
	for _, comment := range element.Comments {
		g.writeIndent(sb, indent)
		sb.WriteString(fmt.Sprintf("# %s\n", comment))
	}

	// 根据元素类型生成不同的内容
	switch element.Type {
	case "root":
		// 根元素，只生成子元素
		for _, child := range element.ChildElements {
			g.generateElement(sb, &child, indent)
		}

	case "block":
		// 配置块
		g.writeIndent(sb, indent)
		if element.Value != "" {
			sb.WriteString(fmt.Sprintf("%s \"%s\" {", element.Name, element.Value))
		} else {
			sb.WriteString(fmt.Sprintf("%s {", element.Name))
		}

		// 行尾注释
		if element.LineComment != "" {
			sb.WriteString(fmt.Sprintf(" # %s", element.LineComment))
		}
		sb.WriteString("\n")

		// 生成子元素
		for _, child := range element.ChildElements {
			g.generateElement(sb, &child, indent+g.indentSize)
		}

		// 闭合块
		g.writeIndent(sb, indent)
		sb.WriteString("}\n")

	case "simple":
		// 简单配置项
		g.writeIndent(sb, indent)
		if element.Value != "" {
			sb.WriteString(fmt.Sprintf("%s \"%s\";", element.Name, element.Value))
		} else {
			sb.WriteString(fmt.Sprintf("%s;", element.Name))
		}

		// 行尾注释
		if element.LineComment != "" {
			sb.WriteString(fmt.Sprintf(" # %s", element.LineComment))
		}
		sb.WriteString("\n")

	case "include":
		// include 指令
		g.writeIndent(sb, indent)
		sb.WriteString(fmt.Sprintf("include \"%s\";", element.Value))

		// 行尾注释
		if element.LineComment != "" {
			sb.WriteString(fmt.Sprintf(" # %s", element.LineComment))
		}
		sb.WriteString("\n")

		// include 指令不生成子元素，因为子元素已经在解析时被合并

	default:
		// 未知类型，跳过
		g.writeIndent(sb, indent)
		sb.WriteString(fmt.Sprintf("// 未知元素类型: %s\n", element.Type))
	}
}

// writeIndent 写入缩进
func (g *Generator) writeIndent(sb *strings.Builder, indent int) {
	sb.WriteString(strings.Repeat(" ", indent))
}
