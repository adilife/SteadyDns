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
// cmd/main.go

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SteadyDNS/core/bind"
	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/plugin"
	"SteadyDNS/core/plugin/plugins"
	"SteadyDNS/core/webapi/api"

	"github.com/gin-gonic/gin"
)

const (
	// Version 应用程序版本
	Version = "1.0.0"
	// DefaultConfigPath 默认配置文件路径
	DefaultConfigPath = "config/steadydns.conf"
	// DefaultPIDFile 默认PID文件路径
	DefaultPIDFile = "steadydns.pid"
	// DefaultLogDir 默认日志目录
	DefaultLogDir = "log"
	// StartArgsFile 启动参数保存文件
	StartArgsFile = "steadydns.startargs"
)

// CLIConfig 命令行配置
type CLIConfig struct {
	Command    string
	Daemon     bool
	Foreground bool
	ConfigPath string
	PIDFile    string
	LogDir     string
	LogStdout  bool
	LogFile    bool
	ShowHelp   bool
	ShowVer    bool
}

var (
	cliConfig CLIConfig
	logger    LoggerInterface
)

// LoggerInterface 定义日志接口
type LoggerInterface interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	LogError(format string, err error, args ...interface{})
	Printf(format string, args ...interface{})
}

func init() {
	// 自定义Usage函数
	flag.Usage = printHelp
}

func main() {
	// 解析命令行参数
	parseArgs()

	// 显示帮助
	if cliConfig.ShowHelp {
		printHelp()
		os.Exit(0)
	}

	// 显示版本
	if cliConfig.ShowVer {
		printVersion()
		os.Exit(0)
	}

	// 执行命令
	switch cliConfig.Command {
	case "start", "":
		if err := cmdStart(); err != nil {
			log.Fatalf("启动失败: %v", err)
		}
	case "stop":
		if err := cmdStop(); err != nil {
			log.Fatalf("停止失败: %v", err)
		}
	case "restart":
		if err := cmdRestart(); err != nil {
			log.Fatalf("重启失败: %v", err)
		}
	case "status":
		if err := cmdStatus(); err != nil {
			log.Fatalf("获取状态失败: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", cliConfig.Command)
		printHelp()
		os.Exit(1)
	}
}

// parseArgs 解析命令行参数
func parseArgs() {
	// 检查第一个参数是否为命令
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		cliConfig.Command = os.Args[1]
		// 移除命令参数，保留其他参数给flag解析
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}

	// 定义flag
	flag.BoolVar(&cliConfig.Daemon, "d", false, "Run as daemon")
	flag.BoolVar(&cliConfig.Daemon, "daemon", false, "Run as daemon")
	flag.BoolVar(&cliConfig.Foreground, "f", false, "Run in foreground")
	flag.BoolVar(&cliConfig.Foreground, "foreground", false, "Run in foreground")
	flag.StringVar(&cliConfig.ConfigPath, "c", DefaultConfigPath, "Configuration file path")
	flag.StringVar(&cliConfig.ConfigPath, "config", DefaultConfigPath, "Configuration file path")
	flag.StringVar(&cliConfig.PIDFile, "p", DefaultPIDFile, "PID file path")
	flag.StringVar(&cliConfig.PIDFile, "pidfile", DefaultPIDFile, "PID file path")
	flag.StringVar(&cliConfig.LogDir, "l", DefaultLogDir, "Log directory")
	flag.StringVar(&cliConfig.LogDir, "log-dir", DefaultLogDir, "Log directory")
	flag.BoolVar(&cliConfig.LogStdout, "log-stdout", false, "Output log to stdout")
	flag.BoolVar(&cliConfig.LogFile, "log-file", false, "Output log to file")
	flag.BoolVar(&cliConfig.ShowHelp, "h", false, "Show help information")
	flag.BoolVar(&cliConfig.ShowHelp, "help", false, "Show help information")
	flag.BoolVar(&cliConfig.ShowVer, "v", false, "Show version information")
	flag.BoolVar(&cliConfig.ShowVer, "version", false, "Show version information")

	flag.Parse()

	// 如果没有指定运行模式，默认前台运行
	if !cliConfig.Daemon && !cliConfig.Foreground {
		cliConfig.Foreground = true
	}

	// 根据运行模式设置默认日志输出方式
	if cliConfig.Foreground && !cliConfig.LogFile {
		cliConfig.LogStdout = true
	}
	if cliConfig.Daemon && !cliConfig.LogStdout {
		cliConfig.LogFile = true
	}
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("SteadyDNS - DNS Server")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  steadydns [command] [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  start       Start service (default)")
	fmt.Println("  stop        Stop service")
	fmt.Println("  restart     Restart service")
	fmt.Println("  status      Check service status")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -d, --daemon          Run as daemon (for systemd service)")
	fmt.Println("  -f, --foreground      Run in foreground (default)")
	fmt.Println("  -c, --config PATH     Specify configuration file path (default: config/steadydns.conf)")
	fmt.Println("  -p, --pidfile PATH    Specify PID file path (default: steadydns.pid)")
	fmt.Println("  -l, --log-dir PATH    Specify log directory (default: log)")
	fmt.Println("  --log-stdout          Output log to stdout")
	fmt.Println("  --log-file            Output log to file")
	fmt.Println("  -v, --version         Show version information")
	fmt.Println("  -h, --help            Show help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  steadydns                    Run service in foreground")
	fmt.Println("  steadydns start -f           Run service in foreground")
	fmt.Println("  steadydns start -d           Run service as daemon")
	fmt.Println("  steadydns stop               Stop service")
	fmt.Println("  steadydns status             Check service status")
	fmt.Println("  steadydns -c /etc/steadydns/steadydns.conf  Use specified configuration file")
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("SteadyDNS version %s\n", Version)
	fmt.Println("DNS server implementation")
	fmt.Println("License: AGPLv3")
}

// cmdStart 启动服务命令
func cmdStart() error {
	daemonManager := common.NewDaemonManager(cliConfig.PIDFile)

	// 如果是守护进程模式启动的子进程
	if os.Getenv("STEADYDNS_DAEMON") == "1" {
		// 子进程直接运行服务（使用cliConfig中的参数）
		// 此时cliConfig已经被正确解析（包含-c, -p等参数）
		return runService(daemonManager)
	}

	// 检查是否已经在运行
	if daemonManager.IsRunning() {
		status, pid := daemonManager.GetStatus()
		return fmt.Errorf("Service is already running (status: %s, PID: %d)", status, pid)
	}

	// 后台模式：启动子进程
	if cliConfig.Daemon {
		return startDaemon(daemonManager)
	}

	// 前台模式：当前进程运行服务
	return runService(daemonManager)
}

// startDaemon 启动守护进程
func startDaemon(daemonManager *common.DaemonManager) error {
	fmt.Println("Starting daemon process...")

	// 构建启动参数
	startArgs := buildStartArgsFromCLI()

	// 启动守护进程
	if err := daemonManager.StartDaemon(startArgs); err != nil {
		return err
	}

	fmt.Println("Daemon process started successfully")
	return nil
}

// runService 运行服务（前台和后台共用）
func runService(daemonManager *common.DaemonManager) error {
	// 加载环境变量
	common.LoadEnv()

	// 初始化日志
	if err := initLogger(); err != nil {
		return fmt.Errorf("初始化日志失败: %v", err)
	}

	// 设置信号处理
	daemonManager.SetupSignalHandlers(func() {
		cleanup()
	})

	// 写入PID文件
	if err := writePIDFile(os.Getpid()); err != nil {
		logger.Warn("写入PID文件失败: %v", err)
	}

	// 保存启动参数
	if err := saveStartArgs(); err != nil {
		logger.Warn("保存启动参数失败: %v", err)
	}

	logger.Info("SteadyDNS 服务启动中...")
	logger.Info("版本: %s", Version)
	logger.Info("配置文件: %s", cliConfig.ConfigPath)
	logger.Info("日志目录: %s", cliConfig.LogDir)

	// 检查数据库文件是否存在
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "steadydns.db"
	}

	// 检查数据库文件是否存在
	dbExists := checkDBFileExists(dbPath)

	// 初始化数据库
	database.InitDB()

	// 根据数据库文件是否存在执行相应操作
	if !dbExists {
		logger.Warn("数据库文件不存在，开始初始化...")
		// 执行初始化操作
		if err := database.InitializeDatabase(); err != nil {
			log.Fatalf("数据库初始化失败: %v", err)
		}
		logger.Warn("数据库初始化完成")
	} else {
		logger.Info("数据库文件已存在，使用现有数据库")
	}

	// 初始化插件系统
	logger.Info("初始化插件系统...")
	pm := plugin.GetPluginManager()

	// 注册BIND插件
	bindPlugin := plugins.NewBindPlugin()
	if err := pm.RegisterPlugin(bindPlugin); err != nil {
		logger.Warn("注册BIND插件失败: %v", err)
	} else {
		logger.Info("BIND插件注册成功")
	}

	// 根据配置设置插件状态
	bindEnabled := common.GetConfigBool("Plugins", "BIND_ENABLED", true)
	pm.SetPluginEnabled("bind", bindEnabled)
	if bindEnabled {
		logger.Info("BIND插件已启用")
	} else {
		logger.Info("BIND插件已禁用")
	}

	// 设置预留插件状态（功能暂未实现）
	dnsRulesEnabled := common.GetConfigBool("Plugins", "DNS_RULES_ENABLED", false)
	pm.SetPluginEnabled(plugin.PluginNameDNSRules, dnsRulesEnabled)
	logger.Info("DNS规则插件状态: %v (预留功能)", dnsRulesEnabled)

	logAnalysisEnabled := common.GetConfigBool("Plugins", "LOG_ANALYSIS_ENABLED", false)
	pm.SetPluginEnabled(plugin.PluginNameLogAnalysis, logAnalysisEnabled)
	logger.Info("日志分析插件状态: %v (预留功能)", logAnalysisEnabled)

	// 初始化所有启用的插件
	if err := pm.InitializeEnabledPlugins(); err != nil {
		logger.Warn("初始化插件失败: %v", err)
	} else {
		logger.Info("插件系统初始化完成")
	}

	// 获取ServerManager实例
	serverManager := api.GetServerManager()
	if err := serverManager.StartDNSServer(); err != nil {
		log.Fatalf("DNS服务器启动失败: %v", err)
	}

	// 检查并启动BIND服务
	logger.Info("检查BIND服务状态...")
	if err := checkAndStartBindService(); err != nil {
		logger.Warn("BIND服务检查和启动失败: %v，将继续启动steadydns服务", err)
	} else {
		logger.Info("BIND服务状态检查完成")
	}

	// 获取服务器管理器实例
	httpServerInstance := api.GetHTTPServer()

	// 启动HTTP服务器
	if err := httpServerInstance.Start(); err != nil {
		logger.Error("API服务器启动失败: %v", err)
	}

	logger.Info("SteadyDNS 服务启动完成")

	// 等待服务器运行
	select {}
}

// cmdStop 停止服务命令
func cmdStop() error {
	daemonManager := common.NewDaemonManager(cliConfig.PIDFile)

	status, pid := daemonManager.GetStatus()
	if status != "running" {
		return fmt.Errorf("Service is not running")
	}

	fmt.Printf("Stopping service (PID: %d)...\n", pid)

	if err := daemonManager.StopDaemon(); err != nil {
		return err
	}

	fmt.Println("Service stopped")
	return nil
}

// cmdRestart 重启服务命令
func cmdRestart() error {
	daemonManager := common.NewDaemonManager(cliConfig.PIDFile)

	fmt.Println("Restarting service...")

	// 构建启动参数
	var startArgs []string

	// 检查 restart 是否带了参数
	hasArgs := false
	for i := 1; i < len(os.Args); i++ {
		if strings.HasPrefix(os.Args[i], "-") {
			hasArgs = true
			break
		}
	}

	if hasArgs {
		// 使用 restart 带的参数
		startArgs = buildStartArgsFromCLI()
	} else {
		// 读取之前保存的参数
		savedArgs, err := loadStartArgs()
		if err != nil || len(savedArgs) == 0 {
			// 没有保存的参数，使用默认参数
			startArgs = []string{"-f"}
		} else {
			startArgs = savedArgs
		}
	}

	if err := daemonManager.RestartDaemon(startArgs); err != nil {
		return err
	}

	fmt.Println("Service restarted successfully")
	return nil
}

// cmdStatus 查看服务状态命令
func cmdStatus() error {
	daemonManager := common.NewDaemonManager(cliConfig.PIDFile)

	status, pid := daemonManager.GetStatus()

	fmt.Printf("Service status: %s\n", status)
	if pid > 0 {
		fmt.Printf("Process ID: %d\n", pid)
	}

	return nil
}

// initLogger 初始化日志系统
func initLogger() error {
	// 确保日志目录存在
	if err := os.MkdirAll(cliConfig.LogDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建轮转日志记录器
	rotateLogger, err := common.NewRotateLogger(
		cliConfig.LogDir,
		"steadydns.log",
		10*1024*1024, // 10MB
		10,           // 10个历史文件
	)
	if err != nil {
		return err
	}

	// 设置是否输出到标准输出
	if cliConfig.LogStdout {
		rotateLogger.SetStdout(true)
	}

	// 重新加载日志级别（确保配置已加载）
	logLevel := common.GetLogLevelFromEnv()
	rotateLogger.SetLevel(logLevel)

	// 设置全局日志器，让所有 Logger 实例使用同一个输出
	common.SetGlobalLogger(rotateLogger)

	// 设置 GIN 的日志输出到同一个日志文件
	gin.DefaultWriter = rotateLogger
	// 禁用 GIN 的调试模式颜色输出（避免日志文件中出现颜色代码）
	gin.DisableConsoleColor()

	logger = rotateLogger
	return nil
}

// cleanup 清理资源
func cleanup() {
	logger.Info("正在关闭服务...")

	// 删除PID文件
	os.Remove(cliConfig.PIDFile)

	logger.Info("服务已关闭")

	// 最后关闭日志
	if rotateLogger, ok := logger.(*common.RotateLogger); ok {
		rotateLogger.Close()
	}
}

// writePIDFile 写入PID文件
func writePIDFile(pid int) error {
	return os.WriteFile(cliConfig.PIDFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// checkDBFileExists 检查数据库文件是否存在
// 参数:
//
//	dbPath: 数据库文件路径
//
// 返回值:
//
//	bool: 数据库文件是否存在
func checkDBFileExists(dbPath string) bool {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		logger.Error("获取数据库路径失败: %v", err)
		absPath = dbPath
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		logger.Error("数据库文件 %s 不存在", absPath)
		return false
	}

	logger.Info("数据库文件 %s 存在", absPath)
	return true
}

// checkAndStartBindService 检查并启动BIND服务
// 返回值:
//
//	error: 操作过程中遇到的错误
func checkAndStartBindService() error {
	// 获取插件管理器实例
	pm := plugin.GetPluginManager()

	// 检查BIND插件是否启用
	if !pm.IsPluginEnabled("bind") {
		logger.Info("BIND插件已禁用，跳过BIND服务检查和启动")
		return nil
	}

	// 创建BIND管理器实例
	bindManager := bind.NewBindManager()

	// 检查BIND服务状态
	status, err := bindManager.GetBindStatus()
	if err != nil {
		logger.Error("检查BIND服务状态失败: %v", err)
		return fmt.Errorf("检查BIND服务状态失败: %v", err)
	}

	logger.Info("当前BIND服务状态: %s", status)

	// 如果服务已经在运行，直接返回
	if status == "running" {
		logger.Info("BIND服务已经在运行中")
		return nil
	}

	// 服务未运行，尝试启动
	logger.Info("BIND服务未运行，尝试启动...")
	if err := bindManager.StartBind(); err != nil {
		logger.Error("启动BIND服务失败: %v", err)
		return fmt.Errorf("启动BIND服务失败: %v", err)
	}

	// 等待几秒钟，确保服务完全启动
	logger.Info("BIND服务启动命令执行完成，等待服务完全启动...")
	time.Sleep(3 * time.Second)

	// 再次检查服务状态
	status, err = bindManager.GetBindStatus()
	if err != nil {
		logger.Error("再次检查BIND服务状态失败: %v", err)
		return fmt.Errorf("再次检查BIND服务状态失败: %v", err)
	}

	logger.Info("启动后的BIND服务状态: %s", status)

	// 验证服务是否成功启动
	if status == "running" {
		logger.Info("BIND服务启动成功")
		return nil
	} else {
		logger.Warn("BIND服务启动命令执行完成，但状态检查显示服务未运行")
		return fmt.Errorf("BIND服务启动命令执行完成，但状态检查显示服务未运行，状态: %s", status)
	}
}

// saveStartArgs 保存启动参数到文件
// 返回值:
//
//	error: 保存过程中的错误
func saveStartArgs() error {
	var args []string

	// 根据当前配置构建参数列表
	if cliConfig.Daemon {
		args = append(args, "-d")
	} else {
		args = append(args, "-f")
	}

	if cliConfig.ConfigPath != DefaultConfigPath {
		args = append(args, "-c", cliConfig.ConfigPath)
	}

	if cliConfig.PIDFile != DefaultPIDFile {
		args = append(args, "-p", cliConfig.PIDFile)
	}

	if cliConfig.LogDir != DefaultLogDir {
		args = append(args, "-l", cliConfig.LogDir)
	}

	if cliConfig.LogStdout {
		args = append(args, "--log-stdout")
	}

	if cliConfig.LogFile {
		args = append(args, "--log-file")
	}

	// 写入文件
	content := strings.Join(args, "\n")
	return os.WriteFile(StartArgsFile, []byte(content), 0644)
}

// loadStartArgs 从文件读取启动参数
// 返回值:
//
//	[]string: 参数列表
//	error: 读取过程中的错误
func loadStartArgs() ([]string, error) {
	data, err := os.ReadFile(StartArgsFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var args []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			args = append(args, line)
		}
	}

	return args, nil
}

// buildStartArgsFromCLI 从当前命令行构建启动参数
// 返回值:
//
//	[]string: 参数列表
func buildStartArgsFromCLI() []string {
	var args []string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		args = append(args, arg)

		// 如果选项需要值，也包含值
		if (arg == "-c" || arg == "--config" || arg == "-p" || arg == "--pidfile" || arg == "-l" || arg == "--log-dir") && i+1 < len(os.Args) {
			args = append(args, os.Args[i+1])
			i++
		}
	}

	return args
}
