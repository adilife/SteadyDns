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

// core/bind/validation.go
// 配置和区域验证

package bind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ValidateConfig 验证BIND配置
func (bm *BindManager) ValidateConfig() error {
	// 检查配置中的NamedCheckConf命令
	if bm.config.NamedCheckConf != "" {
		// 使用配置文件中定义的命令（可能是完整命令或绝对路径）
		bm.logger.Debug("使用配置的NAMED_CHECKCONF命令: %s", bm.config.NamedCheckConf)

		// 设置超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 检查是否是完整命令（包含空格）或仅绝对路径
		var cmd *exec.Cmd

		if strings.Contains(bm.config.NamedCheckConf, " ") {
			// 包含空格，作为完整命令使用shell执行
			bm.logger.Debug("执行命令: /bin/sh -c %s", bm.config.NamedCheckConf)
			cmd = exec.CommandContext(ctx, "/bin/sh", "-c", bm.config.NamedCheckConf)
		} else {
			// 仅路径，直接作为命令执行，添加配置文件路径参数
			// 构建正确的named.conf文件路径
			namedConfFile := filepath.Join(bm.config.NamedConfPath, "named.conf")
			bm.logger.Debug("执行命令: %s %s", bm.config.NamedCheckConf, namedConfFile)
			cmd = exec.CommandContext(ctx, bm.config.NamedCheckConf, namedConfFile)
		}

		// 执行命令
		output, err := cmd.CombinedOutput()
		if err != nil {
			bm.logger.Error("执行named-checkconf命令失败: %v, 输出: %s", err, string(output))
			return fmt.Errorf("BIND配置验证失败: %s, 错误: %v", string(output), err)
		}

		return nil
	}

	// 回退到使用默认的named-checkconf命令
	bm.logger.Debug("未配置NAMED_CHECKCONF和BIND_CHECKCONF_PATH，使用named-checkconf命令")

	// 设置命令路径，默认使用环境变量或系统路径
	checkConfCmd := "named-checkconf"
	// 构建正确的named.conf文件路径
	namedConfFile := filepath.Join(bm.config.NamedConfPath, "named.conf")
	bm.logger.Debug("执行默认命令: %s %s", checkConfCmd, namedConfFile)

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令参数，使用CommandContext创建带有上下文的命令
	cmd := exec.CommandContext(ctx, checkConfCmd, namedConfFile)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		bm.logger.Error("执行默认named-checkconf命令失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("BIND配置验证失败: %s, 错误: %v", string(output), err)
	}

	return nil
}

// ValidateZone 验证zone文件
func (bm *BindManager) ValidateZone(domain string) error {
	// 生成zone文件路径
	zoneFileName := fmt.Sprintf("%s.zone", domain)
	zoneFilePath := filepath.Join(bm.config.ZoneFilePath, zoneFileName)

	// 读取zone文件内容并记录日志，便于调试
	zoneContent, err := os.ReadFile(zoneFilePath)
	if err != nil {
		bm.logger.Debug("读取zone文件失败: %v", err)
	} else {
		bm.logger.Debug("本次修改的zone文件内容:\n%s", string(zoneContent))
	}

	// 检查配置中的NamedCheckZone命令
	if bm.config.NamedCheckZone != "" {
		// 使用配置文件中定义的命令（可能是完整命令或绝对路径）
		bm.logger.Debug("使用配置的NAMED_CHECKZONE命令: %s", bm.config.NamedCheckZone)

		// 设置超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 检查是否是完整命令（包含空格）或仅绝对路径
		var cmd *exec.Cmd

		if strings.Contains(bm.config.NamedCheckZone, " ") {
			// 包含空格，作为完整命令使用shell执行，替换占位符
			cmdStr := bm.config.NamedCheckZone
			cmdStr = strings.ReplaceAll(cmdStr, "$DOMAIN", domain)
			cmdStr = strings.ReplaceAll(cmdStr, "${DOMAIN}", domain)
			cmdStr = strings.ReplaceAll(cmdStr, "$ZONE_FILE", zoneFilePath)
			cmdStr = strings.ReplaceAll(cmdStr, "${ZONE_FILE}", zoneFilePath)

			bm.logger.Debug("执行命令: /bin/sh -c %s", cmdStr)
			cmd = exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
		} else {
			// 仅路径，直接作为命令执行，添加参数
			bm.logger.Debug("执行命令: %s %s %s", bm.config.NamedCheckZone, domain, zoneFilePath)
			cmd = exec.CommandContext(ctx, bm.config.NamedCheckZone, domain, zoneFilePath)
		}

		// 执行命令
		output, err := cmd.CombinedOutput()
		if err != nil {
			bm.logger.Error("执行named-checkzone命令失败: %v, 输出: %s", err, string(output))
			return fmt.Errorf("zone文件验证失败: %s, 错误: %v", string(output), err)
		}

		return nil
	}

	// 回退到使用默认的named-checkzone命令
	bm.logger.Debug("未配置NAMED_CHECKZONE和BIND_CHECKZONE_PATH，使用named-checkzone命令")

	// 设置命令路径，默认使用环境变量或系统路径
	checkZoneCmd := "named-checkzone"
	bm.logger.Debug("执行默认命令: %s %s %s", checkZoneCmd, domain, zoneFilePath)

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令参数，使用CommandContext创建带有上下文的命令
	cmd := exec.CommandContext(ctx, checkZoneCmd, domain, zoneFilePath)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		bm.logger.Error("执行默认named-checkzone命令失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("zone文件验证失败: %s, 错误: %v", string(output), err)
	}

	return nil
}
