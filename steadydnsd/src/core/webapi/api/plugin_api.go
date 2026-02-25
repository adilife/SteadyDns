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
// core/webapi/api/plugin_api.go

package api

import (
	"net/http"

	"SteadyDNS/core/plugin"

	"github.com/gin-gonic/gin"
)

// PluginAPIHandler 处理插件状态查询API请求
// 该函数处理 GET /api/plugins/status 请求，返回所有已注册插件的启用状态列表
// 响应格式包含插件名称、描述、版本、启用状态和功能特性列表
// 参数：
//   - c: Gin上下文对象，包含HTTP请求和响应信息
func PluginAPIHandler(c *gin.Context) {
	// 获取全局插件管理器实例
	pm := plugin.GetPluginManager()

	// 获取所有插件的信息列表
	pluginInfos := pm.GetAllPluginInfo()

	// 构建响应数据
	response := gin.H{
		"success": true,
		"data": gin.H{
			"plugins": pluginInfos,
		},
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, response)
}
