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

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheckHandler 健康检查处理函数
// 检查系统、数据库和DNS服务的健康状态
func HealthCheckHandler(c *gin.Context) {
	// 执行健康检查
	healthStatus := PerformHealthCheck()

	// 发送响应
	c.JSON(http.StatusOK, gin.H{"success": true, "data": healthStatus, "message": "健康检查完成"})
}
