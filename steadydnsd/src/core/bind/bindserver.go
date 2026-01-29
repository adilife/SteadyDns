// core/bind/bindserver.go
// BIND服务器操作

package bind

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
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

// UpdateBindConfig 更新BIND配置
func (bm *BindManager) UpdateBindConfig(config map[string]string) error {
	// 记录方法调用
	bm.logger.Debug("更新BIND配置")

	// 更新配置实例中的BIND相关配置
	if value, ok := config["BIND_ADDRESS"]; ok {
		bm.config.Address = value
	}
	if value, ok := config["RNDC_KEY"]; ok {
		bm.config.RNDCKey = value
	}
	if value, ok := config["ZONE_FILE_PATH"]; ok {
		bm.config.ZoneFilePath = value
	}
	if value, ok := config["NAMED_CONF_PATH"]; ok {
		bm.config.NamedConfPath = value
	}
	if value, ok := config["RNDC_PORT"]; ok {
		bm.config.RNDPort = value
	}
	if value, ok := config["BIND_USER"]; ok {
		bm.config.BindUser = value
	}
	if value, ok := config["BIND_GROUP"]; ok {
		bm.config.BindGroup = value
	}
	if value, ok := config["BIND_EXEC_START"]; ok {
		bm.config.BindExecStart = value
	}
	if value, ok := config["BIND_EXEC_RELOAD"]; ok {
		bm.config.BindExecReload = value
	}
	if value, ok := config["BIND_EXEC_STOP"]; ok {
		bm.config.BindExecStop = value
	}
	if value, ok := config["NAMED_CHECKCONF"]; ok {
		bm.config.NamedCheckConf = value
	}
	if value, ok := config["NAMED_CHECKZONE"]; ok {
		bm.config.NamedCheckZone = value
	}

	bm.logger.Info("BIND配置更新成功")
	return nil
}
