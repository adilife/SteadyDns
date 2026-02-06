// core/httpd/httpd.go

package api

import (
	"SteadyDNS/core/common"
	"SteadyDNS/core/webapi/middleware"

	"context"
	"net/http"
	"os"
	"sync"
	"time"
)

// HTTPServer HTTP服务器包装器，包含服务器实例和运行状态
type HTTPServer struct {
	server      *http.Server
	ipv6Server  *http.Server
	servermu    sync.Mutex
	httpHandler http.Handler
	handlermu   sync.Mutex
	running     bool
	runningmu   sync.Mutex
	logger      *common.Logger
}

var httpServer *HTTPServer

// GetHTTPServer 获取HTTP服务器实例（单例模式）
func GetHTTPServer() *HTTPServer {
	if httpServer == nil {
		httpServer = NewHTTPServer()
	}
	return httpServer
}

// NewHTTPServer 创建新的HTTP服务器实例
func NewHTTPServer() *HTTPServer {
	return &HTTPServer{
		logger: common.NewLogger(),
	}
}

// SetServer 设置HTTP服务器实例
func (hs *HTTPServer) SetServer(server *http.Server) {
	hs.servermu.Lock()
	defer hs.servermu.Unlock()

	hs.server = server
}

// SetHandler 设置HTTP处理器
func (hs *HTTPServer) SetHandler(handler http.Handler) {
	hs.handlermu.Lock()
	defer hs.handlermu.Unlock()

	hs.httpHandler = handler
}

// SetIPv6Server 设置IPv6 HTTP服务器实例
func (hs *HTTPServer) SetIPv6Server(server *http.Server) {
	hs.servermu.Lock()
	defer hs.servermu.Unlock()

	hs.ipv6Server = server
}

// IsRunning 检查HTTP服务器是否运行
func (hs *HTTPServer) IsRunning() bool {
	hs.runningmu.Lock()
	defer hs.runningmu.Unlock()

	return hs.running
}

// Start 启动HTTP服务器
func (hs *HTTPServer) Start() error {
	hs.runningmu.Lock()
	defer hs.runningmu.Unlock()

	if hs.running {
		hs.logger.Info("HTTP服务器已经在运行中")
		return nil
	}

	// 显式调用common.LoadConfig()重新加载配置
	hs.logger.Info("重新加载配置文件...")
	common.LoadConfig()

	// 重新加载JWT配置
	hs.logger.Info("重新加载JWT配置...")
	middleware.GetJWTManager().ReloadConfig()

	// 获取端口，优先从配置文件读取，其次是环境变量，最后使用默认值8080
	port := common.GetConfig("APIServer", "API_SERVER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
	}

	// 获取IP地址配置
	ipAddr := common.GetConfig("APIServer", "API_SERVER_IP_ADDR")
	if ipAddr == "" {
		ipAddr = "0.0.0.0"
	}

	// 获取IPv6地址配置
	ipv6Addr := common.GetConfig("APIServer", "API_SERVER_IPV6_ADDR")
	if ipv6Addr == "" {
		ipv6Addr = "::"
	}

	// 创建一个新的ServeMux并设置路由
	mux := http.NewServeMux()
	SetupRoutes(mux)

	// 智能创建服务器实例
	// 情况1：如果IPv6地址为"::"（默认值），则只启动IPv6服务器（双栈监听）
	if ipv6Addr == "::" {
		hs.logger.Info("IPv6地址为默认值[::]，启动双栈监听...")
		// 创建IPv6服务器实例（双栈监听）
		ipv6AddrWithPort := "[" + ipv6Addr + "]:" + port
		newIPv6Server := &http.Server{
			Addr:    ipv6AddrWithPort,
			Handler: mux,
		}
		// 更新服务器实例
		hs.SetServer(nil)
		hs.SetIPv6Server(newIPv6Server)
	} else {
		// 情况2：如果IPv6地址为特定地址，且IPv4地址为特定地址，则启动两个服务器
		if ipAddr != "0.0.0.0" {
			hs.logger.Info("IPv6地址为特定地址，且IPv4地址为特定地址，启动两个服务器...")
			// 创建IPv4服务器实例
			addr := ipAddr + ":" + port
			newServer := &http.Server{
				Addr:    addr,
				Handler: mux,
			}
			// 创建IPv6服务器实例
			ipv6AddrWithPort := "[" + ipv6Addr + "]:" + port
			newIPv6Server := &http.Server{
				Addr:    ipv6AddrWithPort,
				Handler: mux,
			}
			// 更新服务器实例
			hs.SetServer(newServer)
			hs.SetIPv6Server(newIPv6Server)
		} else {
			// 情况3：如果IPv6地址为特定地址，但IPv4地址为默认值0.0.0.0，则只启动IPv4服务器
			hs.logger.Info("IPv6地址为特定地址，但IPv4地址为默认值0.0.0.0，只启动IPv4服务器...")
			// 创建IPv4服务器实例
			addr := ipAddr + ":" + port
			newServer := &http.Server{
				Addr:    addr,
				Handler: mux,
			}
			// 更新服务器实例
			hs.SetServer(newServer)
			hs.SetIPv6Server(nil)
		}
	}

	// 获取监听地址
	listenAddr := ""
	if hs.server != nil && hs.server.Addr != "" {
		listenAddr = hs.server.Addr
	}

	ipv6ListenAddr := ""
	if hs.ipv6Server != nil && hs.ipv6Server.Addr != "" {
		ipv6ListenAddr = hs.ipv6Server.Addr
	}

	// 启动HTTP服务器（如果非nil）
	if hs.server != nil {
		hs.logger.Info("启动API服务器 (HTTP)，监听地址: %s...", listenAddr)
		go func() {
			if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				hs.logger.Error("API服务器启动失败: %v", err)
			}
		}()
	}

	// 启动IPv6 HTTP服务器（如果非nil）
	if hs.ipv6Server != nil {
		hs.logger.Info("启动API服务器 (IPv6)，监听地址: %s...", ipv6ListenAddr)
		go func() {
			if err := hs.ipv6Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				hs.logger.Error("IPv6 API服务器启动失败: %v", err)
			}
		}()
	}

	hs.running = true
	hs.logger.Info("API服务器启动成功")
	return nil
}

// Stop 停止HTTP服务器
func (hs *HTTPServer) Stop() error {
	hs.runningmu.Lock()

	if !hs.running {
		hs.runningmu.Unlock()
		hs.logger.Info("HTTP服务器已经停止")
		return nil
	}

	// 保存服务器实例
	server := hs.server
	ipv6Server := hs.ipv6Server

	// 立即标记服务器为停止状态，避免重复调用
	hs.running = false
	hs.runningmu.Unlock()

	hs.logger.Debug("停止HTTP服务器...")

	// 在后台异步执行服务器关闭操作
	go func() {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 停止IPv4服务器
		if server != nil {
			if err := server.Shutdown(ctx); err != nil {
				// if err := server.Close(); err != nil {
				// 区分上下文取消错误和其他实际错误
				if ctx.Err() == context.DeadlineExceeded {
					hs.logger.Error("停止HTTP服务器超时: %v", err)
				} else if ctx.Err() == context.Canceled {
					hs.logger.Warn("HTTP服务器关闭操作被取消: %v", err)
				} else {
					hs.logger.Error("停止HTTP服务器失败: %v", err)
				}
			} else {
				hs.logger.Debug("HTTP服务器停止成功")
			}
		}
	}()

	// 在后台异步执行IPv6服务器关闭操作
	go func() {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 停止IPv6服务器
		if ipv6Server != nil {
			if err := ipv6Server.Shutdown(ctx); err != nil {
				// 区分上下文取消错误和其他实际错误
				if ctx.Err() == context.DeadlineExceeded {
					hs.logger.Error("停止IPv6 HTTP服务器超时: %v", err)
				} else if ctx.Err() == context.Canceled {
					hs.logger.Warn("IPv6 HTTP服务器关闭操作被取消: %v", err)
				} else {
					hs.logger.Error("停止IPv6 HTTP服务器失败: %v", err)
				}
			} else {
				hs.logger.Debug("IPv6 HTTP服务器停止成功")
			}
		}
	}()

	return nil
}

// Restart 重启HTTP服务器
func (hs *HTTPServer) Restart() error {
	hs.logger.Info("重启HTTP服务器...")

	// 停止HTTP服务器
	if err := hs.Stop(); err != nil {
		hs.logger.Error("停止HTTP服务器失败: %v", err)
		return err
	}

	// 增加延迟时间，确保服务器完全停止
	time.Sleep(1 * time.Second)

	// 启动HTTP服务器
	if err := hs.Start(); err != nil {
		hs.logger.Error("启动HTTP服务器失败: %v", err)
		return err
	}

	hs.logger.Info("HTTP服务器重启成功")
	return nil
}

// createHTTPServer 创建新的HTTP服务器实例（从配置读取）
func (hs *HTTPServer) createHTTPServer() *http.Server {
	// 获取端口，优先从配置文件读取，其次是环境变量，最后使用默认值8080
	port := common.GetConfig("APIServer", "API_SERVER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
	}

	// 获取IP地址配置
	ipAddr := common.GetConfig("APIServer", "API_SERVER_IP_ADDR")
	if ipAddr == "" {
		ipAddr = "0.0.0.0"
	}

	// 构建服务器地址
	addr := ipAddr + ":" + port

	// 创建一个新的 ServeMux
	mux := http.NewServeMux()

	SetupRoutes(mux)

	// 创建HTTP服务器实例
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

// createIPv6HTTPServer 创建新的IPv6 HTTP服务器实例（从配置读取）
func (hs *HTTPServer) createIPv6HTTPServer() *http.Server {
	// 获取端口，优先从配置文件读取，其次是环境变量，最后使用默认值8080
	port := common.GetConfig("APIServer", "API_SERVER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
	}

	// 获取IPv6地址配置
	ipv6Addr := common.GetConfig("APIServer", "API_SERVER_IPV6_ADDR")
	if ipv6Addr == "" {
		ipv6Addr = "::"
	}

	// 构建服务器地址
	addr := "[" + ipv6Addr + "]:" + port

	// 创建一个新的 ServeMux
	mux := http.NewServeMux()

	SetupRoutes(mux)

	// 创建HTTP服务器实例
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
