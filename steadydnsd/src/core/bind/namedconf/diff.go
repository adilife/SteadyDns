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

// core/bind/namedconf/diff.go
// named.conf 文件差异比较模块

package namedconf

import (
	"fmt"
	"os"
	"strings"
)

// DiffResult 差异结果
type DiffResult struct {
	Lines []DiffLine `json:"lines"`
	Stats DiffStats  `json:"stats"`
}

// DiffLine 差异行
type DiffLine struct {
	Type    string `json:"type"` // "unchanged", "added", "removed"
	LineNum int    `json:"lineNum"`
	Content string `json:"content"`
}

// DiffStats 差异统计
type DiffStats struct {
	Unchanged int `json:"unchanged"`
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Total     int `json:"total"`
}

// Diff 配置差异比较
func Diff(oldContent, newContent string) *DiffResult {
	// 按行分割
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// 简单的差异比较（基于行的比较）
	result := &DiffResult{
		Lines: make([]DiffLine, 0),
		Stats: DiffStats{},
	}

	// 计算最长长度
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	// 逐行比较
	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		var oldExists, newExists bool

		if i < len(oldLines) {
			oldLine = oldLines[i]
			oldExists = true
		}

		if i < len(newLines) {
			newLine = newLines[i]
			newExists = true
		}

		if !oldExists && newExists {
			// 新增行
			result.Lines = append(result.Lines, DiffLine{
				Type:    "added",
				LineNum: i + 1,
				Content: newLine,
			})
			result.Stats.Added++
		} else if oldExists && !newExists {
			// 删除行
			result.Lines = append(result.Lines, DiffLine{
				Type:    "removed",
				LineNum: i + 1,
				Content: oldLine,
			})
			result.Stats.Removed++
		} else if oldLine != newLine {
			// 修改行（显示为删除旧行，添加新行）
			result.Lines = append(result.Lines, DiffLine{
				Type:    "removed",
				LineNum: i + 1,
				Content: oldLine,
			})
			result.Stats.Removed++

			result.Lines = append(result.Lines, DiffLine{
				Type:    "added",
				LineNum: i + 1,
				Content: newLine,
			})
			result.Stats.Added++
		} else {
			// 未变化行
			result.Lines = append(result.Lines, DiffLine{
				Type:    "unchanged",
				LineNum: i + 1,
				Content: oldLine,
			})
			result.Stats.Unchanged++
		}
	}

	// 计算总行数
	result.Stats.Total = result.Stats.Unchanged + result.Stats.Added + result.Stats.Removed

	return result
}

// DiffFiles 比较两个文件的差异
func DiffFiles(oldFile, newFile string) (*DiffResult, error) {
	// 读取旧文件
	oldContent, err := readFileContent(oldFile)
	if err != nil {
		return nil, fmt.Errorf("读取旧文件失败: %v", err)
	}

	// 读取新文件
	newContent, err := readFileContent(newFile)
	if err != nil {
		return nil, fmt.Errorf("读取新文件失败: %v", err)
	}

	// 比较差异
	return Diff(oldContent, newContent), nil
}

// readFileContent 读取文件内容
func readFileContent(filePath string) (string, error) {
	content, err := readFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// readFile 读取文件
func readFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
