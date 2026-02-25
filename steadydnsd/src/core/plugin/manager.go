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
// core/plugin/manager.go

package plugin

import (
	"fmt"
	"sync"

	"SteadyDNS/core/common"
)

// PluginManager 插件管理器
// 负责插件的注册、状态管理和生命周期管理
type PluginManager struct {
	mu       sync.RWMutex      // 读写锁，保证并发安全
	plugins  map[string]Plugin // 已注册的插件映射表
	statuses map[string]bool   // 插件启用状态映射表
	logger   *common.Logger    // 日志记录器
}

// pluginManager 全局插件管理器实例
var pluginManager *PluginManager

// once 用于确保全局插件管理器只初始化一次
var once sync.Once

// GetPluginManager 获取全局插件管理器实例
// 使用单例模式确保全局只有一个插件管理器实例
// 返回值：
//   - *PluginManager: 插件管理器实例
func GetPluginManager() *PluginManager {
	once.Do(func() {
		pluginManager = &PluginManager{
			plugins:  make(map[string]Plugin),
			statuses: make(map[string]bool),
			logger:   common.NewLogger(),
		}
	})
	return pluginManager
}

// RegisterPlugin 注册插件到插件管理器
// 参数：
//   - plugin: 要注册的插件实例
//
// 返回值：
//   - error: 如果插件已存在或名称无效，返回错误
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("插件不能为空")
	}

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("插件名称不能为空")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查插件是否已注册
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("插件 %s 已注册", name)
	}

	// 注册插件
	pm.plugins[name] = plugin
	// 默认启用状态
	pm.statuses[name] = true

	pm.logger.Info("插件注册成功: %s (版本: %s)", name, plugin.Version())
	return nil
}

// GetAllPlugins 获取所有已注册的插件实例
// 返回值：
//   - []Plugin: 所有插件的切片
func (pm *PluginManager) GetAllPlugins() []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// GetPluginInfo 获取指定插件的详细信息
// 参数：
//   - name: 插件名称
//
// 返回值：
//   - *PluginInfo: 插件信息，如果插件不存在则返回nil
func (pm *PluginManager) GetPluginInfo(name string) *PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil
	}

	enabled := pm.statuses[name]

	return &PluginInfo{
		Name:        plugin.Name(),
		Description: plugin.Description(),
		Version:     plugin.Version(),
		Enabled:     enabled,
		Features:    pm.getPluginFeatures(plugin),
	}
}

// GetAllPluginInfo 获取所有插件的信息列表
// 包括已注册插件和预留插件
// 返回值：
//   - []PluginInfo: 所有插件信息的切片
func (pm *PluginManager) GetAllPluginInfo() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(pm.plugins)+2) // +2 for reserved plugins

	// 添加已注册插件信息
	for name, plugin := range pm.plugins {
		enabled := pm.statuses[name]
		infos = append(infos, PluginInfo{
			Name:        plugin.Name(),
			Description: plugin.Description(),
			Version:     plugin.Version(),
			Enabled:     enabled,
			Features:    pm.getPluginFeatures(plugin),
		})
	}

	// 添加预留插件信息（如果未注册）
	if _, exists := pm.plugins[PluginNameDNSRules]; !exists {
		dnsRulesEnabled := pm.statuses[PluginNameDNSRules]
		dnsRulesInfo := ReservedPluginDNSRules
		dnsRulesInfo.Enabled = dnsRulesEnabled
		infos = append(infos, dnsRulesInfo)
	}

	if _, exists := pm.plugins[PluginNameLogAnalysis]; !exists {
		logAnalysisEnabled := pm.statuses[PluginNameLogAnalysis]
		logAnalysisInfo := ReservedPluginLogAnalysis
		logAnalysisInfo.Enabled = logAnalysisEnabled
		infos = append(infos, logAnalysisInfo)
	}

	return infos
}

// IsPluginEnabled 检查指定插件是否启用
// 支持已注册插件和预留插件
// 参数：
//   - name: 插件名称
//
// 返回值：
//   - bool: true表示启用，false表示禁用或不存在
func (pm *PluginManager) IsPluginEnabled(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 检查是否是预留插件
	if name == PluginNameDNSRules || name == PluginNameLogAnalysis {
		return pm.statuses[name]
	}

	// 如果插件不存在，返回false
	if _, exists := pm.plugins[name]; !exists {
		return false
	}

	return pm.statuses[name]
}

// SetPluginEnabled 设置插件的启用状态
// 支持已注册插件和预留插件
// 参数：
//   - name: 插件名称
//   - enabled: 是否启用
//
// 返回值：
//   - error: 如果插件不存在，返回错误
func (pm *PluginManager) SetPluginEnabled(name string, enabled bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查是否是预留插件
	if name == PluginNameDNSRules || name == PluginNameLogAnalysis {
		pm.statuses[name] = enabled
		pm.logger.Info("预留插件 %s 状态已设置为: %v", name, enabled)
		return nil
	}

	if _, exists := pm.plugins[name]; !exists {
		return fmt.Errorf("插件 %s 不存在", name)
	}

	pm.statuses[name] = enabled
	pm.logger.Info("插件 %s 状态已设置为: %v", name, enabled)
	return nil
}

// InitializeEnabledPlugins 初始化所有已启用的插件
// 遍历所有插件，对启用状态的插件调用Initialize方法
// 返回值：
//   - error: 如果任何插件初始化失败，返回错误
func (pm *PluginManager) InitializeEnabledPlugins() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var initErrors []error

	for name, plugin := range pm.plugins {
		if pm.statuses[name] {
			pm.logger.Info("正在初始化插件: %s", name)
			if err := plugin.Initialize(); err != nil {
				pm.logger.Error("插件 %s 初始化失败: %v", name, err)
				initErrors = append(initErrors, fmt.Errorf("插件 %s 初始化失败: %w", name, err))
			} else {
				pm.logger.Info("插件 %s 初始化成功", name)
			}
		} else {
			pm.logger.Info("插件 %s 已禁用，跳过初始化", name)
		}
	}

	if len(initErrors) > 0 {
		return fmt.Errorf("部分插件初始化失败: %v", initErrors)
	}
	return nil
}

// ShutdownAllPlugins 关闭所有插件
// 遍历所有已初始化的插件，调用Shutdown方法释放资源
// 返回值：
//   - error: 如果任何插件关闭失败，返回错误
func (pm *PluginManager) ShutdownAllPlugins() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var shutdownErrors []error

	for name, plugin := range pm.plugins {
		if pm.statuses[name] {
			pm.logger.Info("正在关闭插件: %s", name)
			if err := plugin.Shutdown(); err != nil {
				pm.logger.Error("插件 %s 关闭失败: %v", name, err)
				shutdownErrors = append(shutdownErrors, fmt.Errorf("插件 %s 关闭失败: %w", name, err))
			} else {
				pm.logger.Info("插件 %s 已关闭", name)
			}
		}
	}

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("部分插件关闭失败: %v", shutdownErrors)
	}
	return nil
}

// GetPluginRoutes 获取指定插件的路由定义
// 参数：
//   - name: 插件名称
//
// 返回值：
//   - []RouteDefinition: 路由定义列表，如果插件不存在或禁用则返回nil
func (pm *PluginManager) GetPluginRoutes(name string) []RouteDefinition {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists || !pm.statuses[name] {
		return nil
	}

	return plugin.Routes()
}

// GetAllEnabledRoutes 获取所有启用插件的路由定义
// 返回值：
//   - map[string][]RouteDefinition: 插件名称到路由列表的映射
func (pm *PluginManager) GetAllEnabledRoutes() map[string][]RouteDefinition {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	routes := make(map[string][]RouteDefinition)
	for name, plugin := range pm.plugins {
		if pm.statuses[name] {
			routes[name] = plugin.Routes()
		}
	}
	return routes
}

// getPluginFeatures 从插件路由中提取功能列表
// 这是一个内部辅助方法，不需要加锁
// 参数：
//   - plugin: 插件实例
//
// 返回值：
//   - []string: 功能列表
func (pm *PluginManager) getPluginFeatures(plugin Plugin) []string {
	routes := plugin.Routes()
	features := make([]string, 0)
	seen := make(map[string]bool)

	for _, route := range routes {
		// 从路由路径中提取功能名称
		// 例如：/api/bind/zones -> zones
		feature := extractFeature(route.Path)
		if feature != "" && !seen[feature] {
			features = append(features, feature)
			seen[feature] = true
		}
	}

	return features
}

// extractFeature 从路由路径中提取功能名称
// 参数：
//   - path: 路由路径
//
// 返回值：
//   - string: 功能名称
func extractFeature(path string) string {
	// 简单实现：取路径的最后一段作为功能名称
	// 例如：/api/bind/zones -> zones
	// 例如：/api/bind/server/status -> status
	parts := splitPath(path)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// splitPath 分割路径字符串
// 参数：
//   - path: 路径字符串
//
// 返回值：
//   - []string: 分割后的路径段
func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
