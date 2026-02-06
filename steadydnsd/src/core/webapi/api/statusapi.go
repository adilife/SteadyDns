// core/webapi/statusapi.go

package api

import (
	"SteadyDNS/core/common"
	"SteadyDNS/core/sdns"

	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerManager 服务器管理器
type ServerManager struct {
	logger                 *common.Logger
	dnsServerRunning       bool
	httpServerRunning      bool
	dnsServerStatusManager sdns.StatsManager
	mu                     sync.RWMutex
}

// 全局服务器管理器实例
var globalServerManager *ServerManager
var serverManagerOnce sync.Once

// GetServerManager 获取服务器管理器实例（单例模式）
func GetServerManager() *ServerManager {
	serverManagerOnce.Do(func() {
		globalServerManager = &ServerManager{
			logger: common.NewLogger(),
		}
	})
	return globalServerManager
}

// SetHTTPHandlerGin 设置HTTP处理器
func (sm *ServerManager) SetHTTPHandlerGin(handler *gin.Engine) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	httpServer.SetHandler(handler)
}

// IsDNSServerRunning 检查DNS服务器是否运行
func (sm *ServerManager) IsDNSServerRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.dnsServerRunning
}

// IsHTTPServerRunning 检查HTTP服务器是否运行
func (sm *ServerManager) IsHTTPServerRunning() bool {
	return httpServer.IsRunning()
}

// StartDNSServer 启动DNS服务器
func (sm *ServerManager) StartDNSServer() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.dnsServerRunning {
		sm.logger.Info("DNS服务器已经在运行中")
		return nil
	}

	sm.logger.Info("启动DNS服务器...")
	err := sdns.StartDNSServer(sm.logger)
	if err != nil {
		sm.logger.Error("DNS服务器启动失败: %v", err)
		return err
	}

	sm.dnsServerRunning = true
	sm.logger.Info("DNS服务器启动成功")
	return nil
}

// StopDNSServer 停止DNS服务器
func (sm *ServerManager) StopDNSServer() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.dnsServerRunning {
		sm.logger.Info("DNS服务器已经停止")
		return nil
	}

	sm.logger.Info("停止DNS服务器...")

	// 停止UDP服务器
	if sdns.GlobalUDPServer != nil {
		if err := sdns.GlobalUDPServer.Shutdown(); err != nil {
			sm.logger.Error("停止UDP DNS服务器失败: %v", err)
		} else {
			sm.logger.Info("UDP DNS服务器停止成功")
		}
		sdns.GlobalUDPServer = nil
	}

	// 停止TCP服务器
	if sdns.GlobalTCPServer != nil {
		if err := sdns.GlobalTCPServer.Shutdown(); err != nil {
			sm.logger.Error("停止TCP DNS服务器失败: %v", err)
		} else {
			sm.logger.Info("TCP DNS服务器停止成功")
		}
		sdns.GlobalTCPServer = nil
	}

	// 清理全局DNS转发器和缓存更新器
	sdns.GlobalDNSForwarder = nil
	sdns.GlobalCacheUpdater = nil

	// 标记服务器为停止状态
	sm.dnsServerRunning = false
	sm.logger.Info("DNS服务器停止成功")
	return nil
}

// RestartDNSServer 重启DNS服务器
func (sm *ServerManager) RestartDNSServer() error {
	sm.logger.Info("重启DNS服务器...")

	// 停止DNS服务器
	if err := sm.StopDNSServer(); err != nil {
		sm.logger.Error("停止DNS服务器失败: %v", err)
		return err
	}

	// 短暂延迟，确保服务器完全停止
	time.Sleep(100 * time.Millisecond)

	// 启动DNS服务器
	if err := sm.StartDNSServer(); err != nil {
		sm.logger.Error("启动DNS服务器失败: %v", err)
		return err
	}

	sm.logger.Info("DNS服务器重启成功")
	return nil
}

// StartHTTPServer 启动HTTP服务器（公共方法）
func (sm *ServerManager) StartHTTPServer() error {
	// 启动服务器
	return httpServer.Start()
}

// StopHTTPServer 停止HTTP服务器
func (sm *ServerManager) StopHTTPServer() error {
	return httpServer.Stop()
}

// RestartHTTPServer 重启HTTP服务器
func (sm *ServerManager) RestartHTTPServer() error {
	return httpServer.Restart()
}

// ReloadForwardGroups 重载转发组配置
func (sm *ServerManager) ReloadForwardGroups() error {
	sm.logger.Info("重载转发组配置...")

	if err := sdns.ReloadForwardGroups(); err != nil {
		sm.logger.Error("重载转发组配置失败: %v", err)
		return err
	}

	sm.logger.Info("重载转发组配置成功")
	return nil
}

// SetLogLevel 设置日志级别
func (sm *ServerManager) SetLogLevel(level string) error {
	sm.logger.Info("设置日志级别: %s", level)

	// 解析日志级别字符串为LogLevel类型
	logLevel := common.ParseLogLevel(level)

	// 设置当前logger的日志级别
	sm.logger.SetLevel(logLevel)

	// 同时更新全局logger的日志级别
	globalLogger := common.NewLogger()
	globalLogger.SetLevel(logLevel)

	sm.logger.Info("日志级别设置成功: %s", level)
	return nil
}

// GetServerStatus 获取服务器状态
func (sm *ServerManager) GetServerStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 读取日志级别配置
	apiLogLevel := common.GetConfig("API", "LOG_LEVEL")
	if apiLogLevel == "" {
		apiLogLevel = "info" // 默认值
	}

	dnsLogLevel := common.GetConfig("Logging", "DNS_LOG_LEVEL")
	if dnsLogLevel == "" {
		dnsLogLevel = "info" // 默认值
	}

	status := map[string]interface{}{
		"dns_server": map[string]interface{}{
			"running":    sm.dnsServerRunning,
			"udp_server": sdns.GlobalUDPServer != nil,
			"tcp_server": sdns.GlobalTCPServer != nil,
		},
		"http_server": map[string]interface{}{
			"running": sm.IsHTTPServerRunning(),
		},
		"cache": map[string]interface{}{
			"initialized": sdns.GlobalCacheUpdater != nil,
		},
		"forwarder": map[string]interface{}{
			"initialized": sdns.GlobalDNSForwarder != nil,
		},
		"logging": map[string]interface{}{
			"api_log_level": apiLogLevel,
			"dns_log_level": dnsLogLevel,
		},
		"timestamp": time.Now().Unix(),
	}

	// 添加DNS服务器统计信息
	if sdns.GlobalUDPServer != nil {
		statsManager := sdns.GlobalUDPServer.GetStatsManager()
		if statsManager != nil {
			status["dns_server"].(map[string]interface{})["stats"] = statsManager.GetNetworkStats()
		}
	}

	return status
}
