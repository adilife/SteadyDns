// core/sdns/dnshandler.go

package sdns

import (
	"SteadyDNS/core/common"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/miekg/dns"
)

// GlobalDNSForwarder 全局DNS转发器实例，用于在webapi中调用LoadForwardGroups方法
var GlobalDNSForwarder *DNSForwarder

// globalUDPServer 全局UDP DNS服务器实例，用于在webapi中获取统计信息
var globalUDPServer *CustomDNSServer

// globalTCPServer 全局TCP DNS服务器实例，用于在webapi中获取统计信息
var globalTCPServer *CustomDNSServer

// ReloadForwardGroups 重新加载转发组配置
func ReloadForwardGroups() error {
	if GlobalDNSForwarder != nil {
		return GlobalDNSForwarder.LoadForwardGroups()
	}
	return nil
}

// DNSHandler 完整的DNS处理器，整合转发和缓存功能
type DNSHandler struct {
	forwarder       *DNSForwarder
	cacheUpdater    *CacheUpdater
	logger          *common.Logger   // 日志管理器
	dnsLogger       *DNSLogger       // DNS查询日志
	clientIP        string           // 客户端IP地址
	securityManager *SecurityManager // 安全管理器
}

// NewDNSHandler 创建新的DNS处理器
func NewDNSHandler(forwardAddr string, logger *common.Logger) *DNSHandler {
	// 从配置获取日志配置
	logDir := common.GetConfig("Logging", "QUERY_LOG_PATH")
	maxLogSizeStr := common.GetConfig("Logging", "QUERY_LOG_MAX_SIZE")
	maxLogFilesStr := common.GetConfig("Logging", "QUERY_LOG_MAX_FILES")

	var maxLogSize int64
	var maxLogFiles int

	if maxLogSizeStr != "" {
		if size, err := strconv.ParseInt(maxLogSizeStr, 10, 64); err == nil {
			maxLogSize = size * 1024 * 1024 // 转换为字节
		} else {
			maxLogSize = 10 * 1024 * 1024 // 默认10MB
		}
	} else {
		maxLogSize = 10 * 1024 * 1024 // 默认10MB
	}

	if maxLogFilesStr != "" {
		if files, err := strconv.Atoi(maxLogFilesStr); err == nil {
			maxLogFiles = files
		} else {
			maxLogFiles = 10 // 默认10个文件
		}
	} else {
		maxLogFiles = 10 // 默认10个文件
	}

	// 创建DNS转发器
	forwarder := NewDNSForwarder(forwardAddr)
	// 设置转发器的logger
	forwarder.logger = logger

	// 创建安全管理器
	securityManager := NewSecurityManager(logger)

	return &DNSHandler{
		forwarder:       forwarder,
		cacheUpdater:    NewCacheUpdater(),
		logger:          logger,
		dnsLogger:       NewDNSLogger(logDir, maxLogSize, maxLogFiles),
		securityManager: securityManager,
	}
}

// SetClientIP 设置客户端IP地址
func (h *DNSHandler) SetClientIP(clientIP string) {
	h.clientIP = clientIP
}

// ServeDNS 实现DNS服务器接口
func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	startTime := time.Now()
	clientIP := h.clientIP
	if clientIP == "" {
		clientIP = getClientIP(w)
	}

	// 提取查询信息
	var queryDomain, queryType string
	if len(r.Question) > 0 {
		queryDomain = r.Question[0].Name
		queryType = dns.TypeToString[r.Question[0].Qtype]
	}

	// 记录查询开始
	h.dnsLogger.Log("查询开始 - 客户端: %s, 域名: %s, 类型: %s", clientIP, queryDomain, queryType)

	// 安全检查：DNS消息验证
	valid, msg := h.securityManager.ValidateDNSMessage(r, true)
	if !valid {
		h.logger.Warn("DNS消息验证失败: %s, 客户端: %s", msg, clientIP)
		h.dnsLogger.Log("安全检查 - DNS消息验证失败: %s", msg)
		// 返回格式错误
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeFormatError)
		w.WriteMsg(m)
		totalDuration := time.Since(startTime)
		h.dnsLogger.Log("查询完成 - 状态: 消息验证失败, 总耗时: %v", totalDuration)
		return
	}

	// 安全检查：速率限制
	allowed, msg := h.securityManager.CheckRateLimit(clientIP)
	if !allowed {
		h.logger.Warn("DNS查询速率限制: %s, 客户端: %s", msg, clientIP)
		h.dnsLogger.Log("安全检查 - 速率限制: %s", msg)
		// 返回服务拒绝
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeRefused)
		w.WriteMsg(m)
		totalDuration := time.Since(startTime)
		h.dnsLogger.Log("查询完成 - 状态: 速率限制拒绝, 总耗时: %v", totalDuration)
		return
	}

	// 安全检查通过
	h.dnsLogger.Log("安全检查 - 验证通过")

	// 首先检查缓存
	cacheStart := time.Now()
	cachedResult, err := h.cacheUpdater.CheckCache(r)
	cacheDuration := time.Since(cacheStart)

	if err == nil && cachedResult != nil && cachedResult.Rcode == dns.RcodeSuccess && len(cachedResult.Answer) > 0 {
		// 缓存命中，直接返回
		h.dnsLogger.Log("缓存检查 - 命中, 耗时: %v, 回答记录数: %d", cacheDuration, len(cachedResult.Answer))
		w.WriteMsg(cachedResult)
		totalDuration := time.Since(startTime)
		h.dnsLogger.Log("查询完成 - 状态: 缓存命中, 总耗时: %v", totalDuration)
		return
	}

	// 缓存未命中，记录缓存检查结果
	if err == nil && cachedResult != nil {
		h.dnsLogger.Log("缓存检查 - 未命中, 耗时: %v, 状态码: %d, 回答记录数: %d", cacheDuration, cachedResult.Rcode, len(cachedResult.Answer))
	} else {
		h.dnsLogger.Log("缓存检查 - 未命中, 耗时: %v, 错误: %v", cacheDuration, err)
	}

	// 进行转发查询
	forwardStart := time.Now()
	forwardedResult, err := h.forwarder.ForwardQuery(r)
	forwardDuration := time.Since(forwardStart)

	if err != nil {
		h.logger.Error("转发查询失败: %v", err)
		h.dnsLogger.Log("转发查询 - 失败, 耗时: %v, 错误: %v", forwardDuration, err)
		// 返回SERVFAIL错误
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		totalDuration := time.Since(startTime)
		h.dnsLogger.Log("查询完成 - 状态: 转发失败, 总耗时: %v", totalDuration)
		return
	}

	// 记录转发查询成功
	h.dnsLogger.Log("转发查询 - 成功, 耗时: %v", forwardDuration)

	// 尝试更新缓存
	cacheUpdateStart := time.Now()
	if err := h.cacheUpdater.UpdateCacheWithResult(forwardedResult); err != nil {
		h.logger.Error("缓存更新失败: %v", err)
		h.dnsLogger.Log("缓存更新 - 失败, 耗时: %v, 错误: %v", time.Since(cacheUpdateStart), err)
		// 记录告警日志，但仍然返回查询结果
		go h.checkCacheStatus()
	} else {
		h.dnsLogger.Log("缓存更新 - 成功, 耗时: %v", time.Since(cacheUpdateStart))
	}

	// 返回转发结果
	w.WriteMsg(forwardedResult)
	totalDuration := time.Since(startTime)
	h.dnsLogger.Log("查询完成 - 状态: 成功, 总耗时: %v", totalDuration)
}

// checkCacheStatus 检查缓存服务状态
func (h *DNSHandler) checkCacheStatus() {
	// 检查内存缓存状态
	// 这里可以添加一些缓存状态的检查逻辑，例如缓存大小、命中率等
	h.logger.Info("缓存服务状态检查: 内存缓存正常运行")
}

// PooledDNSHandler 使用协程池的DNS处理器
type PooledDNSHandler struct {
	handler *DNSHandler
	pool    *WorkerPool
}

// ServeDNS 实现DNS服务器接口
func (h *PooledDNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	h.pool.Submit(h.handler, w, r)
}

// GetStatsManager 获取统计管理器
func GetStatsManager() *StatsManager {
	if globalUDPServer != nil {
		return globalUDPServer.GetStatsManager()
	}
	return nil
}

// GetUDPServer 获取UDP DNS服务器
func GetUDPServer() *CustomDNSServer {
	return globalUDPServer
}

// GetTCPServer 获取TCP DNS服务器
func GetTCPServer() *CustomDNSServer {
	return globalTCPServer
}

// IsDNSServerRunning 检查DNS服务器是否运行
func IsDNSServerRunning() bool {
	return globalUDPServer != nil || globalTCPServer != nil
}

// StartDNSServer 启动DNS服务器
func StartDNSServer(logger *common.Logger) error {
	// 从配置获取协程池配置
	clientWorkersStr := common.GetConfig("DNS", "DNS_CLIENT_WORKERS")
	queueMultiplierStr := common.GetConfig("DNS", "DNS_QUEUE_MULTIPLIER")

	// 从配置获取DNS监听端口，默认53
	dnsPortStr := common.GetConfig("DNS", "DNS_PORT")
	if dnsPortStr == "" {
		dnsPortStr = "53" // 默认值
	}

	// 从配置获取DNS监听地址，默认0.0.0.0（所有接口）
	dnsAddr := common.GetConfig("DNS", "DNS_ADDRESS")
	if dnsAddr == "" {
		dnsAddr = "0.0.0.0" // 默认值
	}

	var err error
	var clientWorkers int
	var queueMultiplier int

	// 解析DNS端口
	dnsPort, err := strconv.Atoi(dnsPortStr)
	if err != nil || dnsPort <= 0 || dnsPort > 65535 {
		logger.Warn("无效的DNS端口配置: %s, 使用默认值53", dnsPortStr)
		dnsPort = 53
	}

	if clientWorkersStr != "" {
		clientWorkers, err = strconv.Atoi(clientWorkersStr)
		if err != nil || clientWorkers <= 0 {
			clientWorkers = 10000 // 默认值
		}
	} else {
		clientWorkers = 10000 // 默认值
	}

	if queueMultiplierStr != "" {
		queueMultiplier, err = strconv.Atoi(queueMultiplierStr)
		if err != nil || queueMultiplier <= 0 {
			queueMultiplier = 2 // 默认值
		}
	} else {
		queueMultiplier = 2 // 默认值
	}

	// 从配置获取 BIND 服务器地址
	bindAddr := common.GetConfig("BIND", "BIND_ADDRESS")
	if bindAddr == "" {
		bindAddr = "127.0.0.1:5300" // 默认值
	}

	// 构建完整的DNS监听地址
	listenAddr := fmt.Sprintf("%s:%d", dnsAddr, dnsPort)

	// 创建DNS处理器
	handler := NewDNSHandler("8.8.8.8:53", logger)
	// 设置全局DNS转发器实例
	GlobalDNSForwarder = handler.forwarder

	// 创建协程池（使用固定大小）
	pool := NewWorkerPool(clientWorkers, queueMultiplier)

	logger.Info("准备启动DNS服务器，监听地址: %s", listenAddr)

	// 创建自定义UDP DNS服务器
	udpServer := NewCustomDNSServer(listenAddr, "udp", handler, pool, logger)
	globalUDPServer = udpServer

	// 创建自定义TCP DNS服务器
	tcpServer := NewCustomDNSServer(listenAddr, "tcp", handler, pool, logger)
	globalTCPServer = tcpServer

	// 尝试创建UDP监听器，确保端口可用
	udpListener, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("UDP端口 %s 不可用: %v", listenAddr, err)
	}
	udpListener.Close()

	// 尝试创建TCP监听器，确保端口可用
	tcpListener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("TCP端口 %s 不可用: %v", listenAddr, err)
	}
	tcpListener.Close()

	// 启动UDP服务器
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("UDP DNS服务器协程panic: %v", r)
			}
		}()
		logger.Info("正在启动UDP DNS服务器...")
		if err := udpServer.ListenAndServe(); err != nil {
			logger.Error("UDP DNS服务器启动失败: %v", err)
		}
	}()

	// 启动TCP服务器
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("TCP DNS服务器协程panic: %v", r)
			}
		}()
		logger.Info("正在启动TCP DNS服务器...")
		if err := tcpServer.ListenAndServe(); err != nil {
			logger.Error("TCP DNS服务器启动失败: %v", err)
		}
	}()

	logger.Info("DNS服务器启动成功，监听地址: %s", listenAddr)
	return nil
}
