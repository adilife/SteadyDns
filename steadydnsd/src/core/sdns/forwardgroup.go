// core/sdns/forwardgroup.go

package sdns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"SteadyDNS/core/common"
	"SteadyDNS/core/database"
)

// ForwardGroup 表示一个转发服务器组
type ForwardGroup struct {
	Name           string               `json:"name"`            // 组名，长度0-63
	Description    string               `json:"description"`     // 描述，长度0-65535
	PriorityQueues map[int][]*DNSServer `json:"priority_queues"` // 按优先级分组的DNS服务器列表
}

// DNSServer 表示单个DNS服务器
type DNSServer struct {
	Address     string `json:"address"`     // DNS服务器地址，支持IPv4/IPv6
	Port        int    `json:"port"`        // 端口，默认53
	Description string `json:"description"` // 描述，长度0-65535
	QueueIndex  int    `json:"queue_index"` // 队列序号
	Priority    int    `json:"priority"`    // 优先级 (1-3)
}

// GetAddress 获取服务器完整地址（IP:Port）
func (s *DNSServer) GetAddress() string {
	return net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
}

// DNSForwarder 主转发器结构
type DNSForwarder struct {
	groups             map[string]*ForwardGroup
	domainIndex        []string      // 域名索引，按长度降序排列，用于最长匹配
	defaultGroup       *ForwardGroup // 默认转发组
	serverStats        map[string]*ServerStats
	mu                 sync.RWMutex // 保护 groups 映射的锁
	statsMu            sync.RWMutex // 保护 serverStats 映射的锁
	cacheTTL           time.Duration
	priorityTimeout    time.Duration       // 优先级队列超时时间
	logger             *common.Logger      // 日志函数
	forwardPool        *ForwardWorkerPool  // 专用的DNS转发协程池
	authorityForwarder *AuthorityForwarder // 权威域转发管理器

	// 域名匹配缓存
	matchCache   map[string]*cacheEntry // 域名匹配结果缓存
	matchCacheMu sync.RWMutex           // 保护matchCache的锁

	// Cookie和TCP相关组件
	AdaptiveCookieManager   *AdaptiveCookieManager   // 自适应Cookie管理器
	TCPConnectionPool       *TCPConnectionPool       // TCP连接池
	ServerCapabilityProber  *ServerCapabilityProber  // 服务器能力探测器
}

// cacheEntry 域名匹配缓存项
type cacheEntry struct {
	group     *ForwardGroup // 匹配到的转发组
	expiresAt time.Time     // 过期时间
}

// ValidateForwardGroup 验证转发组配置
func ValidateForwardGroup(group *ForwardGroup) error {
	// 验证域名格式
	if err := validateDomain(group.Name); err != nil {
		return fmt.Errorf("域名格式错误: %v", err)
	}

	if len(group.Description) > 65535 {
		return fmt.Errorf("描述长度不能超过65535")
	}

	for priority, servers := range group.PriorityQueues {
		if priority < 1 || priority > 3 {
			return fmt.Errorf("优先级必须在1-3之间")
		}

		for _, server := range servers {
			if err := ValidateDNSServer(server); err != nil {
				return fmt.Errorf("队列 %d 中的服务器配置错误: %v", priority, err)
			}
		}
	}

	return nil
}

// validateDomain 验证域名格式是否符合DNS规范
func validateDomain(domain string) error {
	// 检查域名是否为空
	if len(domain) == 0 {
		return fmt.Errorf("域名不能为空")
	}

	// 检查域名总长度（包括点）是否超过255个字符
	if len(domain) > 255 {
		return fmt.Errorf("域名长度不能超过255个字符")
	}

	// 检查是否为默认转发组
	if domain == "Default" {
		return nil
	}

	// 分割域名标签
	tags := strings.Split(domain, ".")

	// 检查每个标签
	for _, tag := range tags {
		// 检查标签长度
		if len(tag) == 0 || len(tag) > 63 {
			return fmt.Errorf("域名标签长度必须在1-63之间")
		}

		// 检查标签是否以连字符开头或结尾
		if strings.HasPrefix(tag, "-") || strings.HasSuffix(tag, "-") {
			return fmt.Errorf("域名标签不能以连字符开头或结尾")
		}

		// 检查标签是否只包含字母、数字和连字符
		for _, c := range tag {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return fmt.Errorf("域名标签只能包含字母、数字和连字符")
			}
		}
	}

	return nil
}

// ValidateDNSServer 验证DNS服务器配置
func ValidateDNSServer(server *DNSServer) error {
	if server.Address == "" {
		return fmt.Errorf("DNS服务器地址不能为空")
	}

	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}

	if len(server.Description) > 65535 {
		return fmt.Errorf("描述长度不能超过65535")
	}

	// 验证地址格式
	ip := net.ParseIP(server.Address)
	if ip == nil {
		return fmt.Errorf("无效的IP地址: %s", server.Address)
	}

	return nil
}

// LoadForwardGroups 从数据库加载转发组配置
func (f *DNSForwarder) LoadForwardGroups() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 清空现有转发组
	f.groups = make(map[string]*ForwardGroup)
	f.defaultGroup = nil

	// 从数据库获取所有转发组
	allGroups, err := database.GetForwardGroups()
	if err != nil {
		return fmt.Errorf("获取所有转发组失败: %v", err)
	}

	// 检查是否有ID=1的默认转发组
	var defaultGroup *database.ForwardGroup
	var defaultDNSServers []database.DNSServer
	var hasDefaultGroup bool

	for _, group := range allGroups {
		// 跳过禁用的转发组
		if !group.Enable {
			continue
		}

		// 处理默认转发组（ID=1）
		if group.ID == 1 {
			defaultGroup = &group
			defaultDNSServers = group.Servers
			hasDefaultGroup = true
		}

		// 创建DNS包中的ForwardGroup结构体
		dnsGroup := &ForwardGroup{
			Name:           group.Domain,
			Description:    group.Description,
			PriorityQueues: make(map[int][]*DNSServer),
		}

		// 按照优先级分组DNS服务器
		for _, server := range group.Servers {
			if _, exists := dnsGroup.PriorityQueues[server.Priority]; !exists {
				dnsGroup.PriorityQueues[server.Priority] = []*DNSServer{}
			}
			dnsServer := &DNSServer{
				Address:     server.Address,
				Port:        server.Port,
				Description: server.Description,
				QueueIndex:  server.QueueIndex,
				Priority:    server.Priority,
			}
			dnsGroup.PriorityQueues[server.Priority] = append(dnsGroup.PriorityQueues[server.Priority], dnsServer)
		}

		// 更新groups映射
		f.groups[group.Domain] = dnsGroup
		f.logger.Info("转发组配置加载成功，域名: %s, 服务器数量: %d", group.Domain, len(group.Servers))
	}

	// 如果没有ID=1的默认转发组，创建一个
	if !hasDefaultGroup {
		f.logger.Info("未找到ID=1的转发组，正在创建默认转发组")
		defaultGroup = &database.ForwardGroup{
			ID:          1,
			Domain:      "Default",
			Description: "Default Domain",
			Enable:      true,
			Servers:     []database.DNSServer{},
		}
		if err := database.CreateForwardGroup(defaultGroup); err != nil {
			return fmt.Errorf("创建默认转发组失败: %v", err)
		}
		f.logger.Info("默认转发组创建成功")

		// 创建DNS包中的ForwardGroup结构体
		dnsGroup := &ForwardGroup{
			Name:           defaultGroup.Domain,
			Description:    defaultGroup.Description,
			PriorityQueues: make(map[int][]*DNSServer),
		}

		// 更新groups映射
		f.groups[defaultGroup.Domain] = dnsGroup
		defaultDNSServers = defaultGroup.Servers
	}

	// 设置默认转发组
	f.defaultGroup = f.groups[defaultGroup.Domain]

	// 初始化域名索引
	f.initDomainIndex()

	// 清除域名匹配缓存，因为转发组配置已更新
	f.clearMatchCache()

	// 清理服务器统计信息，只保留当前活跃的服务器
	f.CleanupServerStats(defaultDNSServers)

	return nil
}

// AddForwardGroup 添加转发组
func (f *DNSForwarder) AddForwardGroup(group *ForwardGroup) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := ValidateForwardGroup(group); err != nil {
		return err
	}

	f.groups[group.Name] = group
	return nil
}

// LoadConfig 从配置加载
func (f *DNSForwarder) LoadConfig() {
	// 从配置读取优先级超时时间
	priorityTimeoutMsStr := common.GetConfig("DNS", "DNS_PRIORITY_TIMEOUT_MS")

	f.logger.Debug("加载配置 - DNS_PRIORITY_TIMEOUT_MS配置值: '%s'", priorityTimeoutMsStr)

	var priorityTimeoutMs int
	if priorityTimeoutMsStr != "" {
		if t, err := strconv.Atoi(priorityTimeoutMsStr); err == nil && t > 0 {
			priorityTimeoutMs = t
			f.logger.Debug("加载配置 - 从配置读取到超时时间: %dms", priorityTimeoutMs)
		} else {
			priorityTimeoutMs = 50 // 默认50ms
			f.logger.Info("加载配置 - 配置值无效，使用默认值: %dms", priorityTimeoutMs)
		}
	} else {
		priorityTimeoutMs = 50 // 默认50ms
		f.logger.Info("加载配置 - 配置未设置，使用默认值: %dms", priorityTimeoutMs)
	}

	f.priorityTimeout = time.Duration(priorityTimeoutMs) * time.Millisecond
	f.logger.Debug("加载配置 - 最终优先级超时时间: %v", f.priorityTimeout)
}

// NewDNSForwarder 创建新的DNS转发器
func NewDNSForwarder(forwardAddr string) *DNSForwarder {
	logger := common.NewLogger()

	// 先创建TCP连接池
	tcpPool := NewTCPConnectionPool(nil)

	forwarder := &DNSForwarder{
		groups:             make(map[string]*ForwardGroup),
		domainIndex:        make([]string, 0),
		serverStats:        make(map[string]*ServerStats),
		cacheTTL:           30 * time.Second,
		logger:             logger,
		matchCache:         make(map[string]*cacheEntry), // 初始化域名匹配缓存
		authorityForwarder: NewAuthorityForwarder(),      // 初始化权威域转发管理器

		// 初始化Cookie和TCP相关组件
		AdaptiveCookieManager:  NewAdaptiveCookieManager(),
		TCPConnectionPool:      tcpPool,
		ServerCapabilityProber: NewServerCapabilityProber(0, logger, tcpPool),
	}

	// 加载配置
	forwarder.LoadConfig()

	// 从配置获取客户端协程池配置
	clientWorkers := common.GetConfigInt("DNS", "DNS_CLIENT_WORKERS", 10000)

	// 创建专用的DNS转发协程池（客户端协程池的3倍）
	forwardPoolSize := clientWorkers * 3
	forwarder.forwardPool = NewForwardWorkerPool(forwardPoolSize)
	forwarder.logger.Debug("创建专用DNS转发协程池，大小: %d", forwardPoolSize)

	// 加载转发组配置
	if err := forwarder.LoadForwardGroups(); err != nil {
		forwarder.logger.Error("加载转发组配置失败: %v", err)
	}

	// 启动健康检查协程
	forwarder.StartHealthChecks()

	// 启动统计更新协程
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			forwarder.UpdateServerStats()
		}
	}()

	// 启动缓存清理协程，每5分钟清理一次过期缓存
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			forwarder.cleanupMatchCache()
		}
	}()

	// 启动权威域重新加载协程
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // 每5分钟重新加载一次
		defer ticker.Stop()

		for range ticker.C {
			if err := forwarder.authorityForwarder.ReloadAuthorityZones(); err != nil {
				forwarder.logger.Warn("重新加载权威域失败: %v", err)
			} else {
				forwarder.logger.Debug("权威域重新加载成功")
			}
		}
	}()

	return forwarder
}

// cleanupMatchCache 清理过期的域名匹配缓存
func (f *DNSForwarder) cleanupMatchCache() {
	f.matchCacheMu.Lock()
	defer f.matchCacheMu.Unlock()

	now := time.Now()
	removed := 0

	// 遍历缓存，删除过期条目
	for domain, entry := range f.matchCache {
		if now.After(entry.expiresAt) {
			delete(f.matchCache, domain)
			removed++
		}
	}

	if removed > 0 {
		f.logger.Debug("清理域名匹配缓存，删除了 %d 个过期条目", removed)
	}
}

// clearMatchCache 清除所有域名匹配缓存
func (f *DNSForwarder) clearMatchCache() {
	f.matchCacheMu.Lock()
	defer f.matchCacheMu.Unlock()

	f.matchCache = make(map[string]*cacheEntry)
	f.logger.Debug("清除了所有域名匹配缓存")
}

// SetLogLevel 设置日志级别
func (f *DNSForwarder) SetLogLevel(level string) {
	// 解析日志级别
	logLevel := common.ParseLogLevel(level)
	// 更新内部logger的级别
	f.logger.SetLevel(logLevel)
	f.logger.Info("DNSForwarder日志级别设置为: %s", level)
}

// GetAuthorityForwarder 获取权威域转发管理器
func (f *DNSForwarder) GetAuthorityForwarder() *AuthorityForwarder {
	return f.authorityForwarder
}
