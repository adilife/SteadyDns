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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SteadyDNS/core/bind"
	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
	"SteadyDNS/core/webapi/api"
)

func main() {
	// 加载环境变量
	common.LoadEnv()

	// 检查数据库文件是否存在
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "steadydns.db"
	}

	// 检查数据库文件是否存在
	dbExists := checkDBFileExists(dbPath)

	// 初始化数据库
	database.InitDB()

	// 创建日志记录器
	logger := common.NewLogger()

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

	// 获取ServerManager实例
	serverManager := api.GetServerManager()
	if err := serverManager.StartDNSServer(); err != nil {
		log.Fatalf("DNS服务器启动失败: %v", err)
	}

	// 检查并启动BIND服务
	logger.Info("检查BIND服务状态...")
	if err := checkAndStartBindService(logger); err != nil {
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

	// 等待服务器运行
	select {
	// 这里可以添加一个通道来等待退出信号
	}
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
		common.NewLogger().Error("获取数据库路径失败: %v", err)
		absPath = dbPath
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		common.NewLogger().Error("数据库文件 %s 不存在", absPath)
		return false
	}

	common.NewLogger().Info("数据库文件 %s 存在", absPath)
	return true
}

// checkAndStartBindService 检查并启动BIND服务
// 参数:
//
//	logger: 日志记录器实例
//
// 返回值:
//
//	error: 操作过程中遇到的错误
func checkAndStartBindService(logger *common.Logger) error {
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
