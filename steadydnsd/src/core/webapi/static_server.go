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
// core/webapi/static_server.go
// 静态文件服务器，支持开发模式和生产模式

package webapi

import (
	"io/fs"
	"net/http"
	"os"
	"strings"

	"SteadyDNS/core/webapi/static"
)

// DevModeEnvKey 开发模式环境变量键名
const DevModeEnvKey = "STEADYDNS_DEV_MODE"

// DefaultDevStaticDir 开发模式下默认的前端静态文件目录
const DefaultDevStaticDir = "../steadydns_ui/dist"

// StaticServer 静态文件服务器
type StaticServer struct {
	devMode   bool
	staticDir string
}

// NewStaticServer 创建静态文件服务器
// devMode: 是否为开发模式（从文件系统读取）
// staticDir: 开发模式下的静态文件目录
func NewStaticServer(devMode bool, staticDir string) *StaticServer {
	if staticDir == "" {
		staticDir = DefaultDevStaticDir
	}
	return &StaticServer{
		devMode:   devMode,
		staticDir: staticDir,
	}
}

// IsDevMode 检查是否为开发模式
func IsDevMode() bool {
	return os.Getenv(DevModeEnvKey) == "true"
}

// Handler 返回静态文件处理器
func (s *StaticServer) Handler() http.Handler {
	if s.devMode {
		return s.devHandler()
	}
	return s.prodHandler()
}

// devHandler 开发模式处理器（从文件系统读取）
func (s *StaticServer) devHandler() http.Handler {
	return http.FileServer(http.Dir(s.staticDir))
}

// prodHandler 生产模式处理器（从 Embed 读取）
func (s *StaticServer) prodHandler() http.Handler {
	subFS, err := fs.Sub(static.StaticFS, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Failed to load static files", http.StatusInternalServerError)
		})
	}
	return http.FileServer(http.FS(subFS))
}

// SPAHandler 返回 SPA 路由处理器
// 处理前端路由：对于非 API 路由和非静态文件请求，返回 index.html
func (s *StaticServer) SPAHandler(apiPrefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// API 路由不处理
		if strings.HasPrefix(path, apiPrefix) {
			http.NotFound(w, r)
			return
		}

		// 检查文件是否存在
		var fileExists bool
		if s.devMode {
			_, err := os.Stat(s.staticDir + path)
			fileExists = err == nil
		} else {
			subFS, _ := fs.Sub(static.StaticFS, "dist")
			_, err := fs.Stat(subFS, strings.TrimPrefix(path, "/"))
			fileExists = err == nil
		}

		// 如果文件存在，直接服务
		if fileExists {
			s.Handler().ServeHTTP(w, r)
			return
		}

		// 对于 SPA 路由，返回 index.html
		s.serveIndexHTML(w, r)
	})
}

// serveIndexHTML 服务 index.html 文件
func (s *StaticServer) serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	if s.devMode {
		http.ServeFile(w, r, s.staticDir+"/index.html")
		return
	}

	// 生产模式：从 Embed 读取
	subFS, err := fs.Sub(static.StaticFS, "dist")
	if err != nil {
		http.Error(w, "Failed to load index.html", http.StatusInternalServerError)
		return
	}

	content, err := fs.ReadFile(subFS, "index.html")
	if err != nil {
		http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// SetupStaticRoutes 设置静态文件路由
// router: Gin 路由器
// apiPrefix: API 路由前缀（如 "/api"）
// staticPrefix: 静态文件路由前缀（如 "/"）
// 注意：此函数已弃用，静态文件路由在 setuproute.go 中设置
func SetupStaticRoutes(router interface{}, apiPrefix string) {
	// 静态文件路由已在 setuproute.go 的 setupStaticRoutes 函数中设置
	// 此函数保留用于兼容性
}

// GetStaticFS 获取静态文件系统（用于外部访问）
func GetStaticFS() (fs.FS, error) {
	return fs.Sub(static.StaticFS, "dist")
}

// GetStaticDir 获取开发模式下的静态文件目录
func (s *StaticServer) GetStaticDir() string {
	return s.staticDir
}

// IsDevModeServer 获取当前服务器模式
func (s *StaticServer) IsDevModeServer() bool {
	return s.devMode
}
