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

// core/bind/bindserver.go
// BIND服务器操作

package bind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"SteadyDNS/core/bind/namedconf"
)

// ReloadBind 刷新BIND服务器配置
func (bm *BindManager) ReloadBind() error {
	// 记录方法调用
	bm.logger.Debug("执行BIND服务器刷新操作")

	// 获取互斥锁，实现事务性操作
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 1. 检查服务状态（在获取互斥锁之后）
	status, err := bm.GetBindStatus()
	if err != nil {
		bm.logger.Warn("检查BIND服务器状态失败: %v，继续执行刷新操作", err)
	} else if status != "running" {
		bm.logger.Info("BIND服务器未运行，无法执行刷新操作")
		return fmt.Errorf("BIND服务器未运行，无法执行刷新操作")
	}

	// 2. 优先使用systemctl命令
	bm.logger.Debug("使用systemctl命令刷新BIND服务器")

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, "systemctl", "reload", "named")

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("systemctl reload named命令输出: %s", outputStr)

	if err != nil {
		// 检查是否是因为服务不存在
		outputLower := strings.ToLower(outputStr)
		bm.logger.Debug("systemctl错误输出(小写): %s", outputLower)

		// 检查多种可能的服务不存在错误信息
		if strings.Contains(outputLower, "could not find unit") ||
			strings.Contains(outputLower, "unit not found") ||
			strings.Contains(outputLower, "named.service not found") ||
			strings.Contains(outputLower, "failed to reload named.service") ||
			strings.Contains(outputLower, "unit named.service not loaded") {
			bm.logger.Warn("systemctl服务不存在，尝试使用配置的BIND_EXEC_RELOAD命令")

			// 检查配置中的BindExecReload命令
			if bm.config.BindExecReload != "" {
				// 替换命令中的环境变量占位符
				cmdStr := bm.config.BindExecReload
				cmdStr = strings.ReplaceAll(cmdStr, "$RNDC_KEY", bm.config.RNDCKey)
				cmdStr = strings.ReplaceAll(cmdStr, "${RNDC_KEY}", bm.config.RNDCKey)
				cmdStr = strings.ReplaceAll(cmdStr, "$RNDC_PORT", bm.config.RNDPort)
				cmdStr = strings.ReplaceAll(cmdStr, "${RNDC_PORT}", bm.config.RNDPort)

				// 使用配置文件中定义的命令
				bm.logger.Debug("使用配置的BIND_EXEC_RELOAD命令: %s", cmdStr)

				// 设置超时上下文
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// 构建命令
				cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)

				// 执行命令
				output, err := cmd.CombinedOutput()
				bm.logger.Debug("BIND_EXEC_RELOAD命令输出: %s", string(output))

				if err != nil {
					bm.logger.Error("执行BIND_EXEC_RELOAD命令失败: %v, 输出: %s", err, string(output))
					return fmt.Errorf("执行BIND_EXEC_RELOAD命令失败: %s, 错误: %v", string(output), err)
				}

				bm.logger.Info("使用BIND_EXEC_RELOAD命令刷新BIND服务器成功")
				return nil
			} else {
				bm.logger.Error("BIND_EXEC_RELOAD命令未配置")
				return fmt.Errorf("BIND_EXEC_RELOAD命令未配置")
			}
		} else {
			// 如果systemctl命令失败但不是因为服务不存在，尝试使用rndc命令
			bm.logger.Warn("systemctl reload命令失败，尝试使用rndc命令")
		}

		// 回退到使用rndc命令
		// 设置命令路径，默认使用环境变量或系统路径
		rndcCmd := "rndc"

		// 设置超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 构建命令参数
		cmdArgs := []string{"reload"}

		// 如果配置了RNDCKey，添加相关参数
		if bm.config.RNDCKey != "" {
			cmdArgs = append([]string{"-k", bm.config.RNDCKey}, cmdArgs...)
			bm.logger.Debug("使用RNDC_KEY: %s", bm.config.RNDCKey)
		}

		// 使用CommandContext创建带有上下文的命令
		cmd := exec.CommandContext(ctx, rndcCmd, cmdArgs...)

		// 执行命令
		output, err := cmd.CombinedOutput()
		bm.logger.Debug("rndc命令输出: %s", string(output))

		if err != nil {
			bm.logger.Error("执行rndc命令失败: %v, 输出: %s", err, string(output))
			return fmt.Errorf("BIND服务器刷新失败: %s, 错误: %v", string(output), err)
		}

		bm.logger.Info("使用rndc命令刷新BIND服务器成功")
		return nil
	}

	bm.logger.Info("使用systemctl命令刷新BIND服务器成功")
	return nil
}

// StartBind 启动BIND服务器
func (bm *BindManager) StartBind() error {
	// 记录方法调用
	bm.logger.Debug("执行BIND服务器启动操作")

	// 获取互斥锁，实现事务性操作
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 1. 检查服务状态（在获取互斥锁之后）
	status, err := bm.GetBindStatus()
	if err != nil {
		bm.logger.Warn("检查BIND服务器状态失败: %v，继续执行启动操作", err)
	}

	// 如果服务已经在运行，直接返回成功
	if status == "running" {
		bm.logger.Info("BIND服务器已经在运行中")
		return nil
	}

	// 2. 优先使用systemctl命令
	bm.logger.Debug("使用systemctl命令启动BIND服务器")

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, "systemctl", "start", "named")

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("systemctl start named命令输出: %s", outputStr)

	if err != nil {
		// 检查是否是因为服务不存在
		outputLower := strings.ToLower(outputStr)
		bm.logger.Debug("systemctl错误输出(小写): %s", outputLower)
		bm.logger.Debug("检查错误信息是否包含服务不存在的关键字")

		// 检查多种可能的服务不存在错误信息
		if strings.Contains(outputLower, "could not find unit") ||
			strings.Contains(outputLower, "unit not found") ||
			strings.Contains(outputLower, "named.service not found") ||
			strings.Contains(outputLower, "failed to start named.service") {
			bm.logger.Warn("systemctl服务不存在，尝试使用配置的BIND_EXEC_START命令")

			// 检查配置中的BindExecStart命令
			bm.logger.Debug("BIND_EXEC_START配置值: '%s'", bm.config.BindExecStart)
			if bm.config.BindExecStart != "" {
				// 替换命令中的环境变量占位符
				cmdStr := bm.config.BindExecStart
				cmdStr = strings.ReplaceAll(cmdStr, "$RNDC_KEY", bm.config.RNDCKey)
				cmdStr = strings.ReplaceAll(cmdStr, "${RNDC_KEY}", bm.config.RNDCKey)
				cmdStr = strings.ReplaceAll(cmdStr, "$RNDC_PORT", bm.config.RNDPort)
				cmdStr = strings.ReplaceAll(cmdStr, "${RNDC_PORT}", bm.config.RNDPort)

				// 使用配置文件中定义的命令
				bm.logger.Debug("使用配置的BIND_EXEC_START命令: %s", cmdStr)

				// 设置超时上下文
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// 构建命令
				cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)

				// 执行命令
				output, err := cmd.CombinedOutput()
				bm.logger.Debug("BIND_EXEC_START命令输出: %s", string(output))

				if err != nil {
					bm.logger.Error("执行BIND_EXEC_START命令失败: %v, 输出: %s", err, string(output))
					return fmt.Errorf("执行BIND_EXEC_START命令失败: %s, 错误: %v", string(output), err)
				}

				bm.logger.Info("使用BIND_EXEC_START命令启动BIND服务器成功")
				return nil
			} else {
				bm.logger.Error("BIND_EXEC_START命令未配置")
				return fmt.Errorf("BIND_EXEC_START命令未配置")
			}
		} else {
			bm.logger.Debug("错误信息不包含服务不存在的关键字，直接返回错误")
		}

		bm.logger.Error("执行systemctl start named命令失败: %v, 输出: %s", err, outputStr)
		return fmt.Errorf("BIND服务器启动失败: %s, 错误: %v", outputStr, err)
	}

	bm.logger.Info("使用systemctl命令启动BIND服务器成功")
	return nil
}

// StopBind 停止BIND服务器
func (bm *BindManager) StopBind() error {
	// 记录方法调用
	bm.logger.Debug("执行BIND服务器停止操作")

	// 获取互斥锁，实现事务性操作
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 1. 检查服务状态（在获取互斥锁之后）
	status, err := bm.GetBindStatus()
	if err != nil {
		bm.logger.Warn("检查BIND服务器状态失败: %v，继续执行停止操作", err)
	}

	// 如果服务已经停止，直接返回成功
	if status == "stopped" {
		bm.logger.Info("BIND服务器已经停止")
		return nil
	}

	// 2. 优先使用systemctl命令
	bm.logger.Debug("使用systemctl命令停止BIND服务器")

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, "systemctl", "stop", "named")

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("systemctl stop named命令输出: %s", outputStr)

	if err != nil {
		// 检查是否是因为服务不存在
		outputLower := strings.ToLower(outputStr)
		bm.logger.Debug("systemctl错误输出(小写): %s", outputLower)

		// 检查多种可能的服务不存在错误信息
		if strings.Contains(outputLower, "could not find unit") ||
			strings.Contains(outputLower, "unit not found") ||
			strings.Contains(outputLower, "named.service not found") ||
			strings.Contains(outputLower, "failed to stop named.service") ||
			strings.Contains(outputLower, "unit named.service not loaded") {
			bm.logger.Warn("systemctl服务不存在，尝试使用配置的BIND_EXEC_STOP命令")

			// 检查配置中的BindExecStop命令
			if bm.config.BindExecStop != "" {
				// 替换命令中的环境变量占位符
				cmdStr := bm.config.BindExecStop
				cmdStr = strings.ReplaceAll(cmdStr, "$RNDC_KEY", bm.config.RNDCKey)
				cmdStr = strings.ReplaceAll(cmdStr, "${RNDC_KEY}", bm.config.RNDCKey)
				cmdStr = strings.ReplaceAll(cmdStr, "$RNDC_PORT", bm.config.RNDPort)
				cmdStr = strings.ReplaceAll(cmdStr, "${RNDC_PORT}", bm.config.RNDPort)

				// 使用配置文件中定义的命令
				bm.logger.Debug("使用配置的BIND_EXEC_STOP命令: %s", cmdStr)

				// 设置超时上下文
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// 构建命令
				cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)

				// 执行命令
				output, err := cmd.CombinedOutput()
				bm.logger.Debug("BIND_EXEC_STOP命令输出: %s", string(output))

				if err != nil {
					bm.logger.Error("执行BIND_EXEC_STOP命令失败: %v, 输出: %s", err, string(output))
					return fmt.Errorf("执行BIND_EXEC_STOP命令失败: %s, 错误: %v", string(output), err)
				}

				bm.logger.Info("使用BIND_EXEC_STOP命令停止BIND服务器成功")
				return nil
			} else {
				bm.logger.Error("BIND_EXEC_STOP命令未配置")
				return fmt.Errorf("BIND_EXEC_STOP命令未配置")
			}
		}

		bm.logger.Error("执行systemctl stop named命令失败: %v, 输出: %s", err, outputStr)
		return fmt.Errorf("BIND服务器停止失败: %s, 错误: %v", outputStr, err)
	}

	bm.logger.Info("使用systemctl命令停止BIND服务器成功")
	return nil
}

// RestartBind 重启BIND服务器
func (bm *BindManager) RestartBind() error {
	// 记录方法调用
	bm.logger.Debug("执行BIND服务器重启操作")

	// 先停止BIND服务器
	if err := bm.StopBind(); err != nil {
		return err
	}

	// 等待1秒，确保服务完全停止
	time.Sleep(1 * time.Second)

	// 再启动BIND服务器
	if err := bm.StartBind(); err != nil {
		return err
	}

	bm.logger.Info("BIND服务器重启成功")
	return nil
}

// GetBindStatus 获取BIND服务器状态
func (bm *BindManager) GetBindStatus() (string, error) {
	// 记录方法调用
	bm.logger.Debug("执行BIND服务器状态检查")

	// 1. 优先使用systemctl status named命令
	status, err := bm.checkBindStatusWithSystemctl()
	if err == nil {
		return status, nil
	}

	// 2. systemctl失败，使用ps命令检查named进程
	status, err = bm.checkBindStatusWithPS()
	if err == nil {
		return status, nil
	}

	// 3. ps命令失败，使用端口检查
	status, err = bm.checkBindStatusWithPort()
	if err == nil {
		return status, nil
	}

	// 4. 端口检查失败，使用rndc status命令
	status, err = bm.checkBindStatusWithRndc()
	if err == nil {
		return status, nil
	}

	bm.logger.Debug("BIND服务器状态: 未知")
	return "unknown", nil
}

// checkBindStatusWithSystemctl 使用systemctl检查BIND服务器状态
func (bm *BindManager) checkBindStatusWithSystemctl() (string, error) {
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, "systemctl", "status", "named")

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("systemctl status named命令输出: %s", outputStr)

	if err != nil {
		// 检查输出是否包含"inactive"
		if strings.Contains(outputStr, "inactive") {
			bm.logger.Debug("BIND服务器状态: 已停止")
			return "stopped", nil
		}
		// 检查输出是否包含"activating (auto-restart)"
		if strings.Contains(outputStr, "activating (auto-restart)") {
			bm.logger.Debug("BIND服务器状态: 正在自动重启")
			return "stopped", nil
		}
		return "", fmt.Errorf("systemctl检查失败: %v", err)
	}

	// 检查输出是否包含"active (running)"
	if strings.Contains(outputStr, "active (running)") {
		bm.logger.Debug("BIND服务器状态: 运行中")
		return "running", nil
	}

	// 检查输出是否包含"inactive"
	if strings.Contains(outputStr, "inactive") {
		bm.logger.Debug("BIND服务器状态: 已停止")
		return "stopped", nil
	}

	return "", fmt.Errorf("systemctl状态未知")
}

// checkBindStatusWithPS 使用ps命令检查BIND服务器状态
func (bm *BindManager) checkBindStatusWithPS() (string, error) {
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, "ps", "aux")

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("ps aux命令输出: %s", outputStr)

	if err != nil {
		return "", fmt.Errorf("ps命令执行失败: %v", err)
	}

	// 检查输出是否包含named进程
	if strings.Contains(outputStr, "/usr/sbin/named") || strings.Contains(outputStr, "named -u") {
		bm.logger.Debug("BIND服务器状态: 运行中")
		return "running", nil
	}

	return "stopped", nil
}

// checkBindStatusWithPort 使用端口检查BIND服务器状态
func (bm *BindManager) checkBindStatusWithPort() (string, error) {
	// 从配置中获取BIND地址
	bindAddress := bm.config.Address
	port := "53" // 默认端口

	// 解析地址，提取端口
	if strings.Contains(bindAddress, ":") {
		parts := strings.Split(bindAddress, ":")
		if len(parts) == 2 && parts[1] != "" {
			port = parts[1]
		}
	}

	bm.logger.Debug("检查BIND服务器端口: %s", port)

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, "netstat", "-tuln")

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("netstat -tuln命令输出: %s", outputStr)

	if err != nil {
		// 尝试使用ss命令
		cmd = exec.CommandContext(ctx, "ss", "-tuln")
		output, err = cmd.CombinedOutput()
		outputStr = string(output)
		bm.logger.Debug("ss -tuln命令输出: %s", outputStr)

		if err != nil {
			return "", fmt.Errorf("端口检查命令执行失败: %v", err)
		}
	}

	// 检查输出是否包含BIND端口
	if strings.Contains(outputStr, ":"+port+":") {
		bm.logger.Debug("BIND服务器状态: 运行中")
		return "running", nil
	}

	return "stopped", nil
}

// checkBindStatusWithRndc 使用rndc status命令检查BIND服务器状态
func (bm *BindManager) checkBindStatusWithRndc() (string, error) {
	// 检查是否配置了RNDC
	if bm.config.RNDCKey == "" {
		return "", fmt.Errorf("未配置RNDC")
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建命令参数
	cmdArgs := []string{"status"}
	if bm.config.RNDCKey != "" {
		cmdArgs = append([]string{"-k", bm.config.RNDCKey}, cmdArgs...)
	}
	if bm.config.RNDPort != "" {
		cmdArgs = append([]string{"-p", bm.config.RNDPort}, cmdArgs...)
	}

	// 构建命令
	cmd := exec.CommandContext(ctx, "rndc", cmdArgs...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("rndc status命令输出: %s", outputStr)

	if err != nil {
		return "", fmt.Errorf("rndc命令执行失败: %v", err)
	}

	// 检查输出是否包含成功信息
	if strings.Contains(outputStr, "version") || strings.Contains(outputStr, "server is up") {
		bm.logger.Debug("BIND服务器状态: 运行中")
		return "running", nil
	}

	return "stopped", nil
}

// GetBindStats 获取BIND服务器统计信息
func (bm *BindManager) GetBindStats() (map[string]interface{}, error) {
	// 获取互斥锁，实现事务性操作
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 记录方法调用
	bm.logger.Debug("获取BIND服务器统计信息")

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 从BIND_EXEC_RELOAD中提取rndc命令的绝对路径
	rndcCmd := "rndc" // 默认值
	if bm.config.BindExecReload != "" {
		parts := strings.Fields(bm.config.BindExecReload)
		if len(parts) > 0 {
			rndcCmd = parts[0]
			bm.logger.Debug("从BIND_EXEC_RELOAD中提取rndc路径: %s", rndcCmd)
		}
	}

	// 构建rndc status命令
	cmdArgs := []string{"status"}

	// 添加服务器地址
	cmdArgs = append([]string{"-s", "127.0.0.1"}, cmdArgs...)

	// 添加RNDPort if configured
	if bm.config.RNDPort != "" {
		cmdArgs = append([]string{"-p", bm.config.RNDPort}, cmdArgs...)
		bm.logger.Debug("使用RNDC_PORT: %s", bm.config.RNDPort)
	}

	// 如果配置了RNDCKey，添加相关参数
	if bm.config.RNDCKey != "" {
		cmdArgs = append([]string{"-k", bm.config.RNDCKey}, cmdArgs...)
		bm.logger.Debug("使用RNDC_KEY: %s", bm.config.RNDCKey)
	}

	// 使用CommandContext创建带有上下文的命令
	cmd := exec.CommandContext(ctx, rndcCmd, cmdArgs...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	bm.logger.Debug("rndc status命令输出: %s", outputStr)

	if err != nil {
		bm.logger.Error("执行rndc status命令失败: %v, 输出: %s", err, outputStr)
		return nil, fmt.Errorf("获取BIND服务器统计信息失败: %s, 错误: %v", outputStr, err)
	}

	// 解析统计信息
	stats := make(map[string]interface{})

	// 简单解析输出，提取关键信息
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				stats[key] = value
			}
		}
	}

	bm.logger.Debug("BIND服务器统计信息获取成功")
	return stats, nil
}

// CheckBindHealth 检查BIND服务健康状态
func (bm *BindManager) CheckBindHealth() (map[string]interface{}, error) {
	// 获取互斥锁，实现事务性操作
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 记录方法调用
	bm.logger.Debug("检查BIND服务健康状态")

	health := make(map[string]interface{})

	// 1. 检查服务状态
	status, err := bm.GetBindStatus()
	if err != nil {
		health["status"] = "error"
		health["error"] = err.Error()
		return health, err
	}

	health["status"] = status

	// 2. 检查配置文件有效性
	configValid := true
	configError := ""
	if err := bm.ValidateConfig(); err != nil {
		configValid = false
		configError = err.Error()
	}
	health["config_valid"] = configValid
	if !configValid {
		health["config_error"] = configError
	}

	// 3. 检查端口可用性（从配置中获取端口）
	portAvailable := true
	portError := ""

	// 从配置中获取BIND地址和端口
	bindAddress := bm.config.Address
	port := "53" // 默认端口

	// 解析地址，提取端口
	if strings.Contains(bindAddress, ":") {
		parts := strings.Split(bindAddress, ":")
		if len(parts) == 2 && parts[1] != "" {
			port = parts[1]
		}
	}

	bm.logger.Debug("检查BIND服务器端口: %s", port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "netstat", "-tuln")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 尝试使用ss命令
		cmd = exec.CommandContext(ctx, "ss", "-tuln")
		output, err = cmd.CombinedOutput()
		if err != nil {
			portAvailable = false
			portError = err.Error()
		}
	}

	if portAvailable {
		if !strings.Contains(string(output), ":"+port) {
			portAvailable = false
			portError = "Port " + port + " not found"
		}
	}
	health["port_available"] = portAvailable
	if !portAvailable {
		health["port_error"] = portError
	}

	// 4. 综合健康状态评估
	overallHealth := "healthy"
	if status != "running" {
		overallHealth = "unhealthy"
	} else if !configValid {
		overallHealth = "degraded"
	} else if !portAvailable {
		overallHealth = "degraded"
	}
	health["overall_health"] = overallHealth

	bm.logger.Debug("BIND服务健康状态检查完成: %s", overallHealth)
	return health, nil
}

// GetBindConfig 获取BIND配置信息
func (bm *BindManager) GetBindConfig() (map[string]string, error) {
	// 记录方法调用
	bm.logger.Debug("获取BIND配置信息")

	config := make(map[string]string)

	// 从配置实例中提取BIND相关配置
	config["BIND_ADDRESS"] = bm.config.Address
	config["RNDC_KEY"] = bm.config.RNDCKey
	config["ZONE_FILE_PATH"] = bm.config.ZoneFilePath
	config["NAMED_CONF_PATH"] = bm.config.NamedConfPath
	config["RNDC_PORT"] = bm.config.RNDPort
	config["BIND_USER"] = bm.config.BindUser
	config["BIND_GROUP"] = bm.config.BindGroup
	config["BIND_EXEC_START"] = bm.config.BindExecStart
	config["BIND_EXEC_RELOAD"] = bm.config.BindExecReload
	config["BIND_EXEC_STOP"] = bm.config.BindExecStop
	config["NAMED_CHECKCONF"] = bm.config.NamedCheckConf
	config["NAMED_CHECKZONE"] = bm.config.NamedCheckZone

	bm.logger.Debug("BIND配置信息获取成功")
	return config, nil
}

// GetNamedConfPath 获取 named.conf 文件路径
func (bm *BindManager) GetNamedConfPath() string {
	return filepath.Join(bm.config.NamedConfPath, "named.conf")
}

// GetNamedConfContent 读取 named.conf 文件内容
func (bm *BindManager) GetNamedConfContent() (string, error) {
	// 记录方法调用
	bm.logger.Debug("读取 named.conf 文件内容")

	// 获取文件路径
	namedConfPath := bm.GetNamedConfPath()

	// 读取文件内容
	content, err := os.ReadFile(namedConfPath)
	if err != nil {
		bm.logger.Error("读取 named.conf 文件失败: %v", err)
		return "", fmt.Errorf("读取 named.conf 文件失败: %v", err)
	}

	bm.logger.Debug("读取 named.conf 文件内容成功")
	return string(content), nil
}

// UpdateNamedConfContent 更新 named.conf 文件内容
func (bm *BindManager) UpdateNamedConfContent(content string) error {
	// 记录方法调用
	bm.logger.Debug("更新 named.conf 文件内容")

	// 获取文件路径
	namedConfPath := bm.GetNamedConfPath()

	// 获取互斥锁，实现事务性操作
	bm.mu.Lock()

	// 1. 备份原始配置
	backupManager := namedconf.NewBackupManager("./backup", 10)
	if _, err := backupManager.BackupFile(namedConfPath); err != nil {
		bm.logger.Warn("备份 named.conf 文件失败: %v", err)
		// 继续执行，不因为备份失败而中断
	}

	// 2. 生成临时文件并验证
	validator := namedconf.NewValidator(bm.config.NamedCheckConf)
	validationResult, err := validator.ValidateContent(content)
	if err != nil {
		bm.mu.Unlock()
		bm.logger.Error("验证配置内容失败: %v", err)
		return fmt.Errorf("验证配置内容失败: %v", err)
	}

	if !validationResult.Valid {
		bm.mu.Unlock()
		bm.logger.Error("配置验证失败: %s", validationResult.Error)
		return fmt.Errorf("配置验证失败: %s", validationResult.Error)
	}

	// 3. 写入新配置
	if err := os.WriteFile(namedConfPath, []byte(content), 0644); err != nil {
		bm.mu.Unlock()
		bm.logger.Error("写入 named.conf 文件失败: %v", err)
		return fmt.Errorf("写入 named.conf 文件失败: %v", err)
	}

	// 释放锁，避免与 ReloadBind 中的锁产生死锁
	bm.mu.Unlock()

	// 4. 重载 BIND 服务
	if err := bm.ReloadBind(); err != nil {
		bm.logger.Error("重载 BIND 服务失败: %v", err)
		// 不回滚，因为配置本身是有效的，只是重载失败
	}

	bm.logger.Info("更新 named.conf 文件内容成功")
	return nil
}

// ValidateNamedConfContent 验证配置内容
func (bm *BindManager) ValidateNamedConfContent(content string) (*namedconf.ValidationResult, error) {
	// 记录方法调用
	bm.logger.Debug("验证 named.conf 配置内容")

	validator := namedconf.NewValidator(bm.config.NamedCheckConf)
	result, err := validator.ValidateContent(content)
	if err != nil {
		bm.logger.Error("验证配置内容失败: %v", err)
		return nil, fmt.Errorf("验证配置内容失败: %v", err)
	}

	bm.logger.Debug("验证 named.conf 配置内容完成")
	return result, nil
}

// BackupNamedConf 备份 named.conf 文件
func (bm *BindManager) BackupNamedConf() (*namedconf.BackupInfo, error) {
	// 记录方法调用
	bm.logger.Debug("备份 named.conf 文件")

	// 获取文件路径
	namedConfPath := bm.GetNamedConfPath()

	backupManager := namedconf.NewBackupManager("./backup", 10)
	backupInfo, err := backupManager.BackupFile(namedConfPath)
	if err != nil {
		bm.logger.Error("备份 named.conf 文件失败: %v", err)
		return nil, fmt.Errorf("备份 named.conf 文件失败: %v", err)
	}

	bm.logger.Info("备份 named.conf 文件成功: %s", backupInfo.FilePath)
	return backupInfo, nil
}

// GenerateTempNamedConf 生成临时配置文件
func (bm *BindManager) GenerateTempNamedConf(content string) (string, error) {
	// 记录方法调用
	bm.logger.Debug("生成临时 named.conf 文件")

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "named-conf-*.conf")
	if err != nil {
		bm.logger.Error("创建临时文件失败: %v", err)
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer tempFile.Close()

	// 写入内容
	if _, err := tempFile.WriteString(content); err != nil {
		bm.logger.Error("写入临时文件失败: %v", err)
		return "", fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 设置文件权限
	if err := tempFile.Chmod(0644); err != nil {
		bm.logger.Warn("设置临时文件权限失败: %v", err)
		// 继续执行，不因为权限设置失败而中断
	}

	bm.logger.Debug("生成临时 named.conf 文件成功: %s", tempFile.Name())
	return tempFile.Name(), nil
}

// DiffNamedConf 比较配置差异
func (bm *BindManager) DiffNamedConf(oldContent, newContent string) *namedconf.DiffResult {
	// 记录方法调用
	bm.logger.Debug("比较 named.conf 配置差异")

	result := namedconf.Diff(oldContent, newContent)

	bm.logger.Debug("比较 named.conf 配置差异完成")
	return result
}

// ParseNamedConf 解析 named.conf 文件
func (bm *BindManager) ParseNamedConf() (*namedconf.ConfigElement, error) {
	// 记录方法调用
	bm.logger.Debug("解析 named.conf 文件")

	// 获取文件路径
	namedConfPath := bm.GetNamedConfPath()

	parser := namedconf.NewParser(namedConfPath)
	root, err := parser.Parse()
	if err != nil {
		bm.logger.Error("解析 named.conf 文件失败: %v", err)
		return nil, fmt.Errorf("解析 named.conf 文件失败: %v", err)
	}

	bm.logger.Debug("解析 named.conf 文件成功")
	return root, nil
}

// GenerateNamedConf 生成 named.conf 文件内容
func (bm *BindManager) GenerateNamedConf(config interface{}) (string, error) {
	// 记录方法调用
	bm.logger.Debug("生成 named.conf 文件内容")

	generator := namedconf.NewGenerator()
	
	// 确保 config 是 *namedconf.ConfigElement 类型
	root, ok := config.(*namedconf.ConfigElement)
	if !ok {
		bm.logger.Error("无效的配置类型")
		return "", fmt.Errorf("无效的配置类型")
	}

	content, err := generator.Generate(root)
	if err != nil {
		bm.logger.Error("生成 named.conf 文件内容失败: %v", err)
		return "", fmt.Errorf("生成 named.conf 文件内容失败: %v", err)
	}

	bm.logger.Debug("生成 named.conf 文件内容成功")
	return content, nil
}
