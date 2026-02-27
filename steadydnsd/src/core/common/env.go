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

// core/common/env.go
// 环境变量处理

package common

import (
	"os"
	"strconv"
)

// 环境变量键名常量
const (
	// DevModeEnvKey 开发模式环境变量键名
	DevModeEnvKey = "STEADYDNS_DEV_MODE"
)

// LoadEnv 加载环境变量（兼容旧版本）
func LoadEnv() {
	// 加载配置文件
	LoadConfig()
	NewLogger().Info("配置加载完成")
}

// IsDevMode 检查是否为开发模式
// 开发模式下，前端文件从文件系统读取，支持热更新
// 生产模式下，前端文件从 Embed 读取，单二进制部署
func IsDevMode() bool {
	return GetEnvBool(DevModeEnvKey, false)
}

// GetEnv 获取环境变量值，如果不存在则返回默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvBool 获取布尔类型环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return boolValue
	}
	return defaultValue
}

// GetEnvInt 获取整数类型环境变量
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}

// SetEnv 设置环境变量
func SetEnv(key, value string) error {
	return os.Setenv(key, value)
}
