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
// core/common/daemon.go

package common

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// DaemonManager 守护进程管理器
type DaemonManager struct {
	pidFile string
	logger  *Logger
}

// NewDaemonManager 创建新的守护进程管理器
// 参数:
//
//	pidFile: PID文件路径
//
// 返回值:
//
//	*DaemonManager: 守护进程管理器实例
func NewDaemonManager(pidFile string) *DaemonManager {
	return &DaemonManager{
		pidFile: pidFile,
		logger:  NewLogger(),
	}
}

// StartDaemon 启动守护进程
// 参数:
//
//	startArgs: 启动参数列表
//
// 返回值:
//
//	error: 启动过程中的错误
func (d *DaemonManager) StartDaemon(startArgs []string) error {
	// 检查是否已经有实例在运行
	if d.IsRunning() {
		return fmt.Errorf("服务已经在运行中")
	}

	// 构建子进程参数列表
	args := []string{os.Args[0], "start"}
	args = append(args, startArgs...)

	env := os.Environ()

	// 设置环境变量标记为守护进程模式
	env = append(env, "STEADYDNS_DAEMON=1")

	// 使用syscall创建守护进程
	attr := &os.ProcAttr{
		Dir:   ".",
		Env:   env,
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}

	// 启动新进程
	process, err := os.StartProcess(args[0], args, attr)
	if err != nil {
		return fmt.Errorf("启动守护进程失败: %v", err)
	}

	// 写入PID文件
	if err := d.writePIDFile(process.Pid); err != nil {
		process.Kill()
		return fmt.Errorf("写入PID文件失败: %v", err)
	}

	d.logger.Info("守护进程已启动，PID: %d", process.Pid)
	return nil
}

// StopDaemon 停止守护进程
// 返回值:
//
//	error: 停止过程中的错误
func (d *DaemonManager) StopDaemon() error {
	pid, err := d.readPIDFile()
	if err != nil {
		return fmt.Errorf("读取PID文件失败: %v", err)
	}

	if pid <= 0 {
		return fmt.Errorf("服务未运行")
	}

	// 发送终止信号
	process, err := os.FindProcess(pid)
	if err != nil {
		// 进程不存在，删除PID文件
		d.removePIDFile()
		return fmt.Errorf("找不到进程 %d", pid)
	}

	// 先尝试优雅终止
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// 如果优雅终止失败，强制终止
		if err := process.Kill(); err != nil {
			return fmt.Errorf("终止进程失败: %v", err)
		}
	}

	// 删除PID文件
	d.removePIDFile()
	d.logger.Info("服务已停止，PID: %d", pid)
	return nil
}

// RestartDaemon 重启守护进程
// 参数:
//
//	startArgs: 启动参数列表
//
// 返回值:
//
//	error: 重启过程中的错误
func (d *DaemonManager) RestartDaemon(startArgs []string) error {
	// 先停止
	if err := d.StopDaemon(); err != nil {
		d.logger.Warn("停止服务时出错: %v", err)
	}

	// 等待进程完全停止
	sleep(1)

	// 重新启动
	return d.StartDaemon(startArgs)
}

// IsRunning 检查服务是否正在运行
// 返回值:
//
//	bool: 服务是否正在运行
func (d *DaemonManager) IsRunning() bool {
	pid, err := d.readPIDFile()
	if err != nil || pid <= 0 {
		return false
	}

	// 检查进程是否存在
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 发送信号0检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetStatus 获取服务状态
// 返回值:
//
//	string: 状态描述
//	int: 进程ID，如果未运行则为0
func (d *DaemonManager) GetStatus() (string, int) {
	pid, err := d.readPIDFile()
	if err != nil || pid <= 0 {
		return "未运行", 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return "未运行", 0
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return "未运行", 0
	}

	return "运行中", pid
}

// writePIDFile 写入PID文件
// 参数:
//
//	pid: 进程ID
//
// 返回值:
//
//	error: 写入过程中的错误
func (d *DaemonManager) writePIDFile(pid int) error {
	// 确保目录存在
	dir := filepath.Dir(d.pidFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	content := strconv.Itoa(pid)
	return os.WriteFile(d.pidFile, []byte(content), 0644)
}

// readPIDFile 读取PID文件
// 返回值:
//
//	int: 进程ID
//	error: 读取过程中的错误
func (d *DaemonManager) readPIDFile() (int, error) {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

// removePIDFile 删除PID文件
func (d *DaemonManager) removePIDFile() {
	os.Remove(d.pidFile)
}

// Daemonize 将当前进程转换为守护进程
// 参数:
//
//	startArgs: 启动参数列表
//
// 返回值:
//
//	error: 转换过程中的错误
func (d *DaemonManager) Daemonize(startArgs []string) error {
	// 检查是否已经是守护进程
	if os.Getenv("STEADYDNS_DAEMON") == "1" {
		// 已经是守护进程，写入PID文件
		if err := d.writePIDFile(os.Getpid()); err != nil {
			return fmt.Errorf("写入PID文件失败: %v", err)
		}
		return nil
	}

	// 创建子进程作为守护进程
	return d.StartDaemon(startArgs)
}

// SetupSignalHandlers 设置信号处理
// 参数:
//
//	cleanup: 清理函数，在收到终止信号时调用
func (d *DaemonManager) SetupSignalHandlers(cleanup func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				d.logger.Info("收到终止信号: %v", sig)
				if cleanup != nil {
					cleanup()
				}
				d.removePIDFile()
				os.Exit(0)
			case syscall.SIGHUP:
				d.logger.Info("收到重载信号: %v", sig)
				// 可以在这里实现配置重载
			}
		}
	}()
}

// sleep 简单的睡眠函数
func sleep(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}
