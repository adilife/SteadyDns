// core/database/database.go

package database

import (
	"fmt"
	"time"

	"SteadyDNS/core/common"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库实例
var DB *gorm.DB

// InitDB 初始化数据库连接
// 从环境变量获取数据库文件路径，如果没有则使用默认路径
// 设置GORM日志级别和连接池参数
func InitDB() {
	// 从配置文件获取数据库文件路径，如果没有则使用默认路径
	dbPath := common.GetConfig("Database", "DB_PATH")
	if dbPath == "" {
		dbPath = "steadydns.db" // 默认SQLite数据库文件
	}

	// 设置GORM日志级别
	logLevel := common.GetLogLevelFromEnv()
	newLogger := logger.Default.LogMode(logger.Silent)

	switch logLevel {
	case common.DEBUG:
		newLogger = logger.Default.LogMode(logger.Info)
	case common.INFO:
		newLogger = logger.Default.LogMode(logger.Silent)
	case common.WARN:
		newLogger = logger.Default.LogMode(logger.Warn)
	case common.ERROR:
		newLogger = logger.Default.LogMode(logger.Error)
	default:
		newLogger = logger.Default.LogMode(logger.Silent)
	}

	// 连接SQLite数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		common.NewLogger().Fatal("连接数据库失败: %v", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		common.NewLogger().Fatal("获取数据库连接池失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(1)                   // sqlite 推荐设置为1
	sqlDB.SetMaxOpenConns(1)                   // sqlite 推荐设置为1
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 设置连接可复用的最大时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 设置连接最大空闲时间

	DB = db
	common.NewLogger().Info("SQLite数据库连接成功")
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
		&User{},         // 用户表
		&ForwardGroup{}, // 转发组表
		&DNSServer{},    // DNS服务器表
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
