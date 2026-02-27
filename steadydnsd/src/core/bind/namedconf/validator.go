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

// core/bind/namedconf/validator.go
// named.conf 文件验证模块

package namedconf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Validator 配置验证器
type Validator struct {
	namedCheckConfPath string
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool   `json:"valid"`
	Error  string `json:"error,omitempty"`
	Output string `json:"output,omitempty"`
}

// NewValidator 创建新的验证器实例
func NewValidator(namedCheckConfPath string) *Validator {
	return &Validator{
		namedCheckConfPath: namedCheckConfPath,
	}
}

// ValidateContent 验证配置内容
func (v *Validator) ValidateContent(content string) (*ValidationResult, error) {
	// 生成临时文件
	tempFile, err := v.generateTempFile(content)
	if err != nil {
		return nil, fmt.Errorf("生成临时文件失败: %v", err)
	}
	defer os.Remove(tempFile)

	// 验证配置文件
	return v.validateFile(tempFile)
}

// ValidateFile 验证配置文件
func (v *Validator) ValidateFile(filePath string) (*ValidationResult, error) {
	return v.validateFile(filePath)
}

// generateTempFile 生成临时配置文件
func (v *Validator) generateTempFile(content string) (string, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "named-conf-*.conf")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer tempFile.Close()

	// 写入内容
	if _, err := tempFile.WriteString(content); err != nil {
		return "", fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 设置文件权限
	if err := tempFile.Chmod(0644); err != nil {
		return "", fmt.Errorf("设置临时文件权限失败: %v", err)
	}

	return tempFile.Name(), nil
}

// validateFile 验证配置文件
func (v *Validator) validateFile(filePath string) (*ValidationResult, error) {
	// 检查 named-checkconf 工具是否存在
	if v.namedCheckConfPath == "" {
		v.namedCheckConfPath = "named-checkconf"
	}

	// 构建命令
	cmd := exec.Command(v.namedCheckConfPath, filePath)

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 解析结果
	result := &ValidationResult{}

	if err != nil {
		// 验证失败
		result.Valid = false
		result.Error = strings.TrimSpace(outputStr)
		result.Output = outputStr
	} else {
		// 验证成功
		result.Valid = true
		result.Output = outputStr
	}

	return result, nil
}
