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
	}

	// 回退到使用rndc命令
	bm.logger.Debug("未配置BIND_EXEC_RELOAD，使用rndc命令")

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
