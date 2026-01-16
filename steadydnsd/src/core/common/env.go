// core/common/env.go
package common

// LoadEnv 加载环境变量（兼容旧版本）
func LoadEnv() {
	// 加载配置文件
	LoadConfig()
	NewLogger().Info("配置加载完成")
}
