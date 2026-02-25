// core/plugin/plugin.go
//
// SteadyDNS Plugin Interface Definition
// Copyright (C) 2024 SteadyDNS Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package plugin

import (
	"github.com/gin-gonic/gin"
)

// Plugin 插件接口定义
// 所有插件必须实现此接口才能被系统加载和管理
type Plugin interface {
	// Name 返回插件的唯一标识名称
	// 该名称用于插件的注册、配置和日志记录
	// 返回值: 插件名称字符串
	Name() string

	// Description 返回插件的功能描述
	// 用于在管理界面展示插件用途
	// 返回值: 插件描述字符串
	Description() string

	// Version 返回插件的版本号
	// 采用语义化版本格式 (如: 1.0.0)
	// 返回值: 版本号字符串
	Version() string

	// Initialize 初始化插件
	// 在插件加载时调用，用于执行必要的初始化操作
	// 如：加载配置、建立数据库连接、初始化资源等
	// 返回值: 初始化错误信息，nil表示成功
	Initialize() error

	// Shutdown 关闭插件
	// 在插件卸载或系统关闭时调用，用于清理资源
	// 如：关闭数据库连接、保存状态、释放资源等
	// 返回值: 关闭错误信息，nil表示成功
	Shutdown() error

	// Routes 返回插件提供的HTTP路由定义列表
	// 用于将插件的API端点注册到Web服务器
	// 返回值: 路由定义切片
	Routes() []RouteDefinition
}

// RouteDefinition HTTP路由定义结构体
// 定义插件提供的API端点信息
type RouteDefinition struct {
	// Method HTTP请求方法 (GET, POST, PUT, DELETE, PATCH等)
	Method string

	// Path 路由路径 (如: /api/plugin/example)
	Path string

	// Handler GIN请求处理函数
	Handler gin.HandlerFunc

	// Description 路由功能描述
	Description string

	// AuthRequired 是否需要认证
	// true: 需要JWT认证才能访问
	// false: 公开访问
	AuthRequired bool

	// Middlewares 中间件列表
	// 按顺序执行的中间件函数
	Middlewares []gin.HandlerFunc
}

// PluginInfo 插件信息结构体
// 用于API响应，展示插件的基本信息
type PluginInfo struct {
	// Name 插件名称
	Name string `json:"name"`

	// Description 插件功能描述
	Description string `json:"description"`

	// Version 插件版本号
	Version string `json:"version"`

	// Enabled 插件是否启用
	Enabled bool `json:"enabled"`

	// Features 插件提供的功能特性列表
	Features []string `json:"features"`
}

// PluginStatus 插件状态常量
type PluginStatus int

const (
	// PluginStatusInactive 插件未激活
	PluginStatusInactive PluginStatus = iota

	// PluginStatusActive 插件已激活运行中
	PluginStatusActive

	// PluginStatusError 插件处于错误状态
	PluginStatusError

	// PluginStatusDisabled 插件已禁用
	PluginStatusDisabled
)

// 插件名称常量定义
const (
	// PluginNameBind BIND插件名称
	PluginNameBind = "bind"

	// PluginNameDNSRules DNS规则插件名称（预留）
	PluginNameDNSRules = "dns-rules"

	// PluginNameLogAnalysis 日志分析插件名称（预留）
	PluginNameLogAnalysis = "log-analysis"
)

// 预留插件信息定义
var (
	// ReservedPluginDNSRules DNS规则插件信息（预留）
	ReservedPluginDNSRules = PluginInfo{
		Name:        PluginNameDNSRules,
		Description: "DNS Rules Plugin - DNS query rules management",
		Version:     "0.1.0",
		Enabled:     false,
		Features:    []string{"rules", "filtering", "blocking"},
	}

	// ReservedPluginLogAnalysis 日志分析插件信息（预留）
	ReservedPluginLogAnalysis = PluginInfo{
		Name:        PluginNameLogAnalysis,
		Description: "Log Analysis Plugin - DNS query log analysis",
		Version:     "0.1.0",
		Enabled:     false,
		Features:    []string{"logs", "analytics", "reports", "statistics"},
	}
)

// String 返回插件状态的字符串表示
func (s PluginStatus) String() string {
	switch s {
	case PluginStatusInactive:
		return "inactive"
	case PluginStatusActive:
		return "active"
	case PluginStatusError:
		return "error"
	case PluginStatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// PluginMetadata 插件元数据结构体
// 包含插件的详细元信息
type PluginMetadata struct {
	// Name 插件名称
	Name string `json:"name"`

	// Description 插件描述
	Description string `json:"description"`

	// Version 插件版本
	Version string `json:"version"`

	// Author 插件作者
	Author string `json:"author"`

	// License 插件许可证
	License string `json:"license"`

	// Homepage 插件主页URL
	Homepage string `json:"homepage"`

	// Repository 插件代码仓库URL
	Repository string `json:"repository"`

	// Dependencies 插件依赖列表
	Dependencies []string `json:"dependencies"`

	// MinSteadyDNSVersion 所需的最低SteadyDNS版本
	MinSteadyDNSVersion string `json:"min_steadydns_version"`
}
