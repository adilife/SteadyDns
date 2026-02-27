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

// core/database/database.go

package database

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"SteadyDNS/core/common"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库实例
var DB *gorm.DB

// LogManager 日志管理器
type LogManager struct {
	logger              *common.Logger
	gormLogger          logger.Interface
	mutex               sync.RWMutex
	currentGORMLogLevel logger.LogLevel
}

// 全局日志管理器实例
var logManager *LogManager

// 初始化日志管理器
func init() {
	logManager = &LogManager{
		logger:              common.NewLogger(),
		currentGORMLogLevel: logger.Silent,
	}
}

// GetLogManager 获取日志管理器实例
func GetLogManager() *LogManager {
	return logManager
}

// SetGORMLogLevel 设置 GORM 日志级别
func (lm *LogManager) SetGORMLogLevel(level logger.LogLevel) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	lm.currentGORMLogLevel = level
	lm.gormLogger = logger.Default.LogMode(level)

	// 如果 DB 已初始化，更新其日志级别
	if DB != nil {
		// 重新设置 GORM 实例的日志
		DB = DB.Session(&gorm.Session{
			Logger: lm.gormLogger,
		})
	}
}

// GetGORMLogLevel 获取当前 GORM 日志级别
func (lm *LogManager) GetGORMLogLevel() logger.LogLevel {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	return lm.currentGORMLogLevel
}

// UpdateGORMLogLevelFromConfig 从配置更新 GORM 日志级别
func (lm *LogManager) UpdateGORMLogLevelFromConfig() {
	logLevelStr := common.GetConfig("Logging", "DNS_LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "INFO" // 默认INFO级别
	}

	lm.SetGORMLogLevel(lm.mapToGORMLogLevel(logLevelStr))
}

// SetLogLevel 设置包内日志级别并同步更新 GORM 日志级别
func (lm *LogManager) SetLogLevel(level string) {
	// 设置包内日志级别
	lm.logger.SetLevel(common.ParseLogLevel(level))
	// 同步更新 GORM 日志级别
	lm.SetGORMLogLevel(lm.mapToGORMLogLevel(level))
}

// mapToGORMLogLevel 将字符串日志级别映射到 GORM 日志级别
func (lm *LogManager) mapToGORMLogLevel(levelStr string) logger.LogLevel {
	levelUpper := strings.ToUpper(levelStr)

	switch levelUpper {
	case "DEBUG":
		return logger.Info
	case "INFO":
		return logger.Silent
	case "WARN", "WARNING":
		return logger.Warn
	case "ERROR":
		return logger.Error
	case "FATAL":
		return logger.Error
	default:
		return logger.Silent
	}
}

// InitDB 初始化数据库连接
// 从环境变量获取数据库文件路径，如果没有则使用默认路径
// 设置GORM日志级别和连接池参数
func InitDB() {
	// 从配置文件获取数据库文件路径，如果没有则使用默认路径
	dbPath := common.GetConfig("Database", "DB_PATH")
	if dbPath == "" {
		dbPath = "steadydns.db" // 默认SQLite数据库文件
	}

	// 使用 LogManager 设置 GORM 日志
	logManager.UpdateGORMLogLevelFromConfig()

	// 连接SQLite数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logManager.gormLogger,
	})
	if err != nil {
		logManager.logger.Fatal("连接数据库失败: %v", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		logManager.logger.Fatal("获取数据库连接池失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(1)                   // sqlite 推荐设置为1
	sqlDB.SetMaxOpenConns(1)                   // sqlite 推荐设置为1
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 设置连接可复用的最大时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 设置连接最大空闲时间

	DB = db
	logManager.logger.Info("SQLite数据库连接成功")
}

// CheckConnection 检查数据库连接是否正常
// 执行一个简单的查询来验证数据库连接状态
func CheckConnection() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// 执行一个简单的查询
	var count int64
	err := DB.Model(&User{}).Count(&count).Error
	return err
}

// InitializeDatabase 初始化数据库，创建必要的表
// 创建用户表、转发组表和DNS服务器表
// 创建默认管理员用户和默认转发组
func InitializeDatabase() error {
	// 创建表 - 按依赖关系排序
	tables := []interface{}{
		&User{},            // 用户表
		&ForwardGroup{},    // 转发组表
		&DNSServer{},       // DNS服务器表
		&QPSHistory{},      // QPS历史记录表
		&ResourceHistory{}, // 资源使用历史记录表
		&NetworkHistory{},  // 网络流量历史记录表
	}

	for _, table := range tables {
		if err := DB.AutoMigrate(table); err != nil {
			return err
		}
	}

	// 创建默认管理员用户
	defaultUser := &User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "admin123", // 在生产环境中应使用哈希密码
	}

	// 检查是否已存在用户，如果不存在则创建默认用户
	var userCount int64
	if err := DB.Model(&User{}).Count(&userCount).Error; err != nil {
		return err
	}

	if userCount == 0 {
		// 使用CreateUser函数，会自动处理密码哈希
		if err := CreateUser(defaultUser); err != nil {
			return fmt.Errorf("创建默认管理员用户失败: %v", err)
		}
		common.NewLogger().Info("创建默认管理员用户: admin / admin123")
	}

	// 创建默认转发组
	defaultGroup := &ForwardGroup{
		Domain:      "Default",
		Description: "Default Domain",
	}

	// 检查是否已存在转发组，如果不存在则创建
	var groupCount int64
	if err := DB.Model(&ForwardGroup{}).Count(&groupCount).Error; err != nil {
		return err
	}

	if groupCount == 0 {
		if err := DB.Create(defaultGroup).Error; err != nil {
			return err
		}
		common.NewLogger().Info("创建默认转发域: Default")
	}

	return nil
}
