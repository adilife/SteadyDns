// core/bind/bind.go
// BIND服务器操作核心模块

package bind

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"SteadyDNS/core/common"
)

// systemZones 定义系统区域列表，这些区域对前端完全屏蔽
var systemZones = map[string]bool{
	".":                    true, // 根区域
	"localhost":            true, // 本地回环正向解析
	"0.0.127.in-addr.arpa": true, // IPv4 本地回环反向解析
	"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa": true, // IPv6 本地回环反向解析
	"rpz.local": true, // 响应策略区（RPZ）
}

// isSystemZone 判断指定域名是否为系统区域
// 参数:
//   - domain: 域名
//
// 返回:
//   - true: 是系统区域
//   - false: 不是系统区域
func isSystemZone(domain string) bool {
	return systemZones[domain]
}

// NewBindManager 创建BIND管理器实例
func NewBindManager() *BindManager {
	logger := common.NewLogger()

	// 读取BIND配置
	config := BindConfig{
		Address:        common.GetConfig("BIND", "BIND_ADDRESS"),
		RNDCKey:        common.GetConfig("BIND", "RNDC_KEY"),
		ZoneFilePath:   common.GetConfig("BIND", "ZONE_FILE_PATH"),
		NamedConfPath:  common.GetConfig("BIND", "NAMED_CONF_PATH"),
		RNDPort:        common.GetConfig("BIND", "RNDC_PORT"),
		BindUser:       common.GetConfig("BIND", "BIND_USER"),
		BindGroup:      common.GetConfig("BIND", "BIND_GROUP"),
		BindExecStart:  common.GetConfig("BIND", "BIND_EXEC_START"),
		BindExecReload: common.GetConfig("BIND", "BIND_EXEC_RELOAD"),
		BindExecStop:   common.GetConfig("BIND", "BIND_EXEC_STOP"),
		NamedCheckConf: common.GetConfig("BIND", "NAMED_CHECKCONF"),
		NamedCheckZone: common.GetConfig("BIND", "NAMED_CHECKZONE"),
	}

	// 检查是否配置了BIND_CHECKCONF_PATH和BIND_CHECKZONE_PATH（旧配置项，兼容处理）
	if config.NamedCheckConf == "" {
		if checkConfPath := common.GetConfig("BIND", "BIND_CHECKCONF_PATH"); checkConfPath != "" {
			config.NamedCheckConf = checkConfPath
		}
	}

	if config.NamedCheckZone == "" {
		if checkZonePath := common.GetConfig("BIND", "BIND_CHECKZONE_PATH"); checkZonePath != "" {
			config.NamedCheckZone = checkZonePath
		}
	}

	// 创建历史记录管理器
	historyMgr := NewHistoryManager()

	bm := &BindManager{
		logger:     logger,
		config:     config,
		HistoryMgr: historyMgr,
	}

	// 设置HistoryManager的bindManager引用（用于恢复后刷新BIND）
	historyMgr.bindManager = bm

	return bm
}

// extractZoneComment 从 named.conf 内容中提取指定 zone 的前置注释
// 参数:
//   - content: named.conf 文件内容
//   - zoneStartIndex: zone 配置块在内容中的起始位置
//
// 返回:
//   - 前置注释内容（多行注释用换行符连接），如果没有注释则返回空字符串
func extractZoneComment(content string, zoneStartIndex int) string {
	if zoneStartIndex <= 0 {
		return ""
	}

	// 获取 zone 之前的文本
	beforeZone := content[:zoneStartIndex]
	lines := strings.Split(beforeZone, "\n")

	var comments []string
	// 从后向前遍历，收集连续的注释行
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// 跳过空行
		if line == "" {
			continue
		}

		// 检查是否是注释行
		if strings.HasPrefix(line, "//") {
			comment := strings.TrimPrefix(line, "//")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else if strings.HasPrefix(line, "#") {
			comment := strings.TrimPrefix(line, "#")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else {
			// 遇到非注释行，停止收集
			break
		}
	}

	if len(comments) == 0 {
		return ""
	}

	return strings.Join(comments, "\n")
}

// GetAuthZones 获取所有权威域
func (bm *BindManager) GetAuthZones() ([]AuthZone, error) {
	zones := make([]AuthZone, 0)

	// 读取named.conf文件
	namedConfPath := filepath.Join(bm.config.NamedConfPath, "named.conf")
	contentBytes, err := os.ReadFile(namedConfPath)
	if err != nil {
		bm.logger.Error("读取named.conf文件失败: %v", err)
		return nil, fmt.Errorf("读取named.conf文件失败: %v", err)
	}
	content := string(contentBytes)

	// 放宽正则表达式，允许缺少allow-query配置
	zoneRegex := regexp.MustCompile(`zone\s+"([^"]+)"\s+IN\s+\{[^\}]*type\s+master;[^\}]*file\s+"([^"]+)"[^\}]*\}`)
	matches := zoneRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		// 提取域名和文件路径
		domain := content[match[2]:match[3]]
		zoneFile := content[match[4]:match[5]]

		zone := AuthZone{
			Domain: domain,
			Type:   "master",
			File:   zoneFile,
		}

		// 提取前置注释
		zoneStartIndex := match[0]
		zone.Comment = extractZoneComment(content, zoneStartIndex)

		// 提取allow-query配置（如果存在）
		allowQueryRegex := regexp.MustCompile(fmt.Sprintf(`zone\s+"%s"\s+IN\s+\{[^\}]*allow-query\s+\{\s*([^\}]+)\s*\}[^\}]*\}`, regexp.QuoteMeta(domain)))
		allowQueryMatch := allowQueryRegex.FindStringSubmatch(content)
		if len(allowQueryMatch) > 1 {
			zone.AllowQuery = strings.TrimSpace(allowQueryMatch[1])
		} else {
			// 如果没有配置allow-query，从全局配置中获取
			globalAllowQuery := common.GetConfig("BIND", "ALLOW_QUERY")
			if globalAllowQuery != "" {
				zone.AllowQuery = globalAllowQuery
			} else {
				// 如果全局配置中也没有设置，使用默认值"any"
				zone.AllowQuery = "any"
			}
		}

		// 读取zone文件获取详细信息
		zoneFilePath := filepath.Join(bm.config.ZoneFilePath, filepath.Base(zoneFile))
		zoneDetail, err := bm.parseZoneFile(zoneFilePath, zone.Domain)
		if err != nil {
			bm.logger.Warn("解析zone文件失败: %v, 跳过该域", err)
			continue
		}

		// 合并信息
		zone.SOA = zoneDetail.SOA
		zone.Records = zoneDetail.Records

		// 跳过系统区域
		if isSystemZone(zone.Domain) {
			bm.logger.Debug("跳过系统区域: %s", zone.Domain)
			continue
		}

		zones = append(zones, zone)
	}

	return zones, nil
}

// GetAuthZone 获取单个权威域
func (bm *BindManager) GetAuthZone(domain string) (*AuthZone, error) {
	// 检查是否为系统区域
	if isSystemZone(domain) {
		return nil, fmt.Errorf("系统区域不允许查询: %s", domain)
	}

	zones, err := bm.GetAuthZones()
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if zone.Domain == domain {
			return &zone, nil
		}
	}

	return nil, fmt.Errorf("权威域不存在")
}

// getAllZoneFiles 获取所有zone文件路径
func (bm *BindManager) getAllZoneFiles() ([]string, error) {
	entries, err := os.ReadDir(bm.config.ZoneFilePath)
	if err != nil {
		return nil, err
	}

	var zoneFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zone") {
			zoneFiles = append(zoneFiles, filepath.Join(bm.config.ZoneFilePath, entry.Name()))
		}
	}
	return zoneFiles, nil
}

// CreateAuthZone 创建权威域
func (bm *BindManager) CreateAuthZone(zone AuthZone) error {
	// 检查是否为系统区域
	if isSystemZone(zone.Domain) {
		return fmt.Errorf("不能创建系统区域: %s", zone.Domain)
	}

	// 自动生成SOA序列号，忽略前端传入的值
	zone.SOA.Serial = generateSerial()

	// 为所有记录生成唯一ID（如果没有的话）
	for i, record := range zone.Records {
		if record.ID == "" {
			zone.Records[i].ID = generateUUID()
		}
	}

	// 处理AllowQuery为空的情况
	if zone.AllowQuery == "" {
		// 从全局配置中获取
		globalAllowQuery := common.GetConfig("BIND", "ALLOW_QUERY")
		if globalAllowQuery != "" {
			zone.AllowQuery = globalAllowQuery
		} else {
			// 如果全局配置中也没有设置，使用默认值"any"
			zone.AllowQuery = "any"
		}
	}

	// 检查NS记录是否为空，如果为空则使用SOA记录中的PrimaryNS作为默认值
	hasNSRecord := false
	for _, record := range zone.Records {
		if record.Type == "NS" {
			hasNSRecord = true
			break
		}
	}

	if !hasNSRecord {
		// 使用SOA记录中的PrimaryNS作为默认NS记录
		defaultNS := Record{
			Name:  "@",
			Type:  "NS",
			Value: zone.SOA.PrimaryNS,
		}
		zone.Records = append(zone.Records, defaultNS)
	}

	// 检查CNAME冲突
	if err := CheckCNAMEConflicts(zone); err != nil {
		return err
	}

	// 检查NS记录中使用的主机名是否有对应的A或AAAA记录
	for _, record := range zone.Records {
		if record.Type != "NS" {
			continue
		}

		ns := record
		// 检查NS记录的主机名是否是zone本身
		isSelfReference := false
		if ns.Name == "@" || ns.Name == zone.Domain || ns.Value == zone.Domain {
			isSelfReference = true
		}

		// 检查是否已经有对应的A或AAAA记录
		hasAddressRecord := false
		for _, addrRecord := range zone.Records {
			if (addrRecord.Type == "A" || addrRecord.Type == "AAAA") &&
				(addrRecord.Name == "@" || addrRecord.Name == zone.Domain) {
				hasAddressRecord = true
				break
			}
		}

		// 如果NS记录指向zone本身，并且没有对应的A或AAAA记录，添加一条默认的A记录
		if isSelfReference && !hasAddressRecord {
			defaultA := Record{
				Name:  "@",
				Type:  "A",
				Value: "127.0.0.1",
			}
			zone.Records = append(zone.Records, defaultA)
			break
		}
	}

	// 生成zone文件内容
	zoneContent := bm.generateZoneContent(zone)

	// 生成zone文件路径
	zoneFileName := fmt.Sprintf("%s.zone", zone.Domain)
	zoneFilePath := filepath.Join(bm.config.ZoneFilePath, zoneFileName)
	namedConfPath := filepath.Join(bm.config.NamedConfPath, "named.conf")

	// 设置zone.File字段，用于备份和恢复
	zone.File = zoneFilePath

	// 读取named.conf原始内容用于回滚
	originalNamedConf, err := os.ReadFile(namedConfPath)
	if err != nil {
		bm.logger.Error("读取named.conf文件失败: %v", err)
		return fmt.Errorf("读取named.conf文件失败: %v", err)
	}

	// 在操作前创建全量备份（named.conf + 所有zone文件）
	allZoneFiles, _ := bm.getAllZoneFiles()
	files := append([]string{namedConfPath}, allZoneFiles...)
	zoneJSON, _ := json.Marshal(zone)
	backupID, err := bm.HistoryMgr.CreateBackup(OperationCreate, zone.Domain, zoneJSON, files)
	if err != nil {
		bm.logger.Warn("创建备份失败: %v", err)
		backupID = 0
	}

	// 写入zone文件
	if err := os.WriteFile(zoneFilePath, []byte(zoneContent), 0644); err != nil {
		bm.logger.Error("创建zone文件失败: %v", err)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		return fmt.Errorf("创建zone文件失败: %v", err)
	}

	// 修改zone文件的所有者和组
	if bm.config.BindUser != "" && bm.config.BindGroup != "" {
		// 获取BIND用户和组的信息
		bindUser, err := user.Lookup(bm.config.BindUser)
		if err != nil {
			bm.logger.Warn("查找BIND用户失败: %v，跳过修改文件所有者", err)
		} else {
			// 将用户名转换为UID和GID
			uid, err := strconv.Atoi(bindUser.Uid)
			if err != nil {
				bm.logger.Warn("解析UID失败: %v，跳过修改文件所有者", err)
			} else {
				// 查找组
				bindGroup, err := user.LookupGroup(bm.config.BindGroup)
				if err != nil {
					bm.logger.Warn("查找BIND组失败: %v，跳过修改文件组", err)
				} else {
					gid, err := strconv.Atoi(bindGroup.Gid)
					if err != nil {
						bm.logger.Warn("解析GID失败: %v，跳过修改文件组", err)
					} else {
						// 修改文件所有者和组
						if err := os.Chown(zoneFilePath, uid, gid); err != nil {
							bm.logger.Warn("修改zone文件所有者和组失败: %v", err)
						} else {
							bm.logger.Debug("已将zone文件所有者和组修改为 %s:%s", bm.config.BindUser, bm.config.BindGroup)
						}
					}
				}
			}
		}
	}

	// 更新named.conf文件
	if err := bm.addZoneToNamedConf(namedConfPath, zone.Domain, zoneFileName, zone.AllowQuery, zone.Comment); err != nil {
		// 回滚：删除创建的zone文件
		os.Remove(zoneFilePath)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		bm.logger.Error("更新named.conf文件失败: %v", err)
		return fmt.Errorf("更新named.conf文件失败: %v", err)
	}

	// 验证named.conf配置
	if err := bm.ValidateConfig(); err != nil {
		// 回滚：恢复原始named.conf内容
		os.WriteFile(namedConfPath, originalNamedConf, 0644)
		// 回滚：删除创建的zone文件
		os.Remove(zoneFilePath)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		bm.logger.Error("验证named.conf配置失败: %v", err)
		return fmt.Errorf("验证named.conf配置失败: %v", err)
	}

	// 验证zone文件
	if err := bm.ValidateZone(zone.Domain); err != nil {
		// 回滚：恢复原始named.conf内容
		os.WriteFile(namedConfPath, originalNamedConf, 0644)
		// 回滚：删除创建的zone文件
		os.Remove(zoneFilePath)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		bm.logger.Error("验证zone文件失败: %v", err)
		return fmt.Errorf("验证zone文件失败: %v", err)
	}

	// 刷新BIND服务器
	if err := bm.ReloadBind(); err != nil {
		bm.logger.Error("刷新BIND服务器失败: %v", err)
		// 不回滚，因为配置本身是有效的，只是刷新失败
	}

	// 操作成功，保留备份记录
	return nil
}

// UpdateAuthZone 更新权威域
func (bm *BindManager) UpdateAuthZone(zone AuthZone) error {
	// 检查是否为系统区域
	if isSystemZone(zone.Domain) {
		return fmt.Errorf("不能更新系统区域: %s", zone.Domain)
	}

	// 检查CNAME冲突
	if err := CheckCNAMEConflicts(zone); err != nil {
		return err
	}

	// 为所有记录生成唯一ID（如果没有的话）
	for i, record := range zone.Records {
		if record.ID == "" {
			zone.Records[i].ID = generateUUID()
		}
	}

	// 获取现有 zone 信息，用于比较是否需要更新 named.conf
	existingZones, err := bm.GetAuthZones()
	if err != nil {
		return fmt.Errorf("获取现有权威域失败: %v", err)
	}

	var existingZone *AuthZone
	for i := range existingZones {
		if existingZones[i].Domain == zone.Domain {
			existingZone = &existingZones[i]
			break
		}
	}

	if existingZone == nil {
		return fmt.Errorf("权威域不存在: %s", zone.Domain)
	}

	// 检查是否需要更新 named.conf（allow-query 或 comment 变更）
	needUpdateNamedConf := false
	if zone.AllowQuery != "" && zone.AllowQuery != existingZone.AllowQuery {
		needUpdateNamedConf = true
	}
	if zone.Comment != existingZone.Comment {
		needUpdateNamedConf = true
	}

	// 如果前端未提供 AllowQuery，使用现有值
	if zone.AllowQuery == "" {
		zone.AllowQuery = existingZone.AllowQuery
	}

	// 保持 file 字段一致
	zoneFileName := filepath.Base(existingZone.File)
	if zone.File == "" {
		zone.File = existingZone.File
	}

	// 生成zone文件路径
	zoneFilePath := filepath.Join(bm.config.ZoneFilePath, zoneFileName)

	// 读取操作前的zone文件内容用于备份
	originalZoneContent, err := os.ReadFile(zoneFilePath)
	if err != nil {
		bm.logger.Error("读取zone文件失败: %v", err)
		return fmt.Errorf("读取zone文件失败: %v", err)
	}

	// 解析现有zone文件，获取原有SOA信息
	originalZone, err := bm.parseZoneFile(zoneFilePath, zone.Domain)
	if err != nil {
		bm.logger.Error("解析现有zone文件失败: %v", err)
		return fmt.Errorf("解析现有zone文件失败: %v", err)
	}

	// 检查前端传入的SOA信息是否完整，如果不完整则使用原有的SOA信息
	if zone.SOA.PrimaryNS == "" || zone.SOA.AdminEmail == "" ||
		zone.SOA.Refresh == "" || zone.SOA.Retry == "" ||
		zone.SOA.Expire == "" || zone.SOA.MinimumTTL == "" {
		// 前端传入的SOA信息不完整，使用原有的SOA信息
		bm.logger.Debug("前端传入的SOA信息不完整，使用原有SOA信息")
		zone.SOA = originalZone.SOA
	}

	// 生成新的SOA序列号，忽略前端传入的值
	currentSerial, err := bm.getCurrentSerial(zone.Domain)
	if err != nil {
		// 如果获取当前序列号失败，生成新的序列号
		zone.SOA.Serial = generateSerial()
	} else {
		// 解析当前序列号
		currentDate, currentSeq, err := parseSerial(currentSerial)
		if err != nil {
			// 解析失败，生成新的序列号
			zone.SOA.Serial = generateSerial()
		} else {
			// 获取当前日期
			today := time.Now().Format("20060102")

			var newSeq int
			if currentDate == today {
				// 同一天，流水号+1
				currentSeqInt, _ := strconv.Atoi(currentSeq)
				newSeq = currentSeqInt + 1
				// 确保流水号不超过99
				if newSeq > 99 {
					newSeq = 1
				}
			} else {
				// 新的一天，重置为01
				newSeq = 1
			}

			// 生成新的序列号
			zone.SOA.Serial = fmt.Sprintf("%s%02d", today, newSeq)
		}
	}

	// 生成zone文件内容
	zoneContent := bm.generateZoneContent(zone)

	// 获取named.conf路径
	namedConfPath := bm.config.NamedConfPath

	// 在操作前创建全量备份（named.conf + 所有zone文件）
	allZoneFiles, _ := bm.getAllZoneFiles()
	files := append([]string{namedConfPath}, allZoneFiles...)
	zoneJSON, _ := json.Marshal(zone)
	backupID, err := bm.HistoryMgr.CreateBackup(OperationUpdate, zone.Domain, zoneJSON, files)
	if err != nil {
		bm.logger.Warn("创建备份失败: %v", err)
		backupID = 0
	}

	// 写入zone文件
	if err := os.WriteFile(zoneFilePath, []byte(zoneContent), 0644); err != nil {
		bm.logger.Error("更新zone文件失败: %v", err)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		return fmt.Errorf("更新zone文件失败: %v", err)
	}

	// 修改zone文件的所有者和组
	if bm.config.BindUser != "" && bm.config.BindGroup != "" {
		// 获取BIND用户和组的信息
		bindUser, err := user.Lookup(bm.config.BindUser)
		if err != nil {
			bm.logger.Warn("查找BIND用户失败: %v，跳过修改文件所有者", err)
		} else {
			// 将用户名转换为UID和GID
			uid, err := strconv.Atoi(bindUser.Uid)
			if err != nil {
				bm.logger.Warn("解析UID失败: %v，跳过修改文件所有者", err)
			} else {
				// 查找组
				bindGroup, err := user.LookupGroup(bm.config.BindGroup)
				if err != nil {
					bm.logger.Warn("查找BIND组失败: %v，跳过修改文件组", err)
				} else {
					gid, err := strconv.Atoi(bindGroup.Gid)
					if err != nil {
						bm.logger.Warn("解析GID失败: %v，跳过修改文件组", err)
					} else {
						// 修改文件所有者和组
						if err := os.Chown(zoneFilePath, uid, gid); err != nil {
							bm.logger.Warn("修改zone文件所有者和组失败: %v", err)
						} else {
							bm.logger.Debug("已将zone文件所有者和组修改为 %s:%s", bm.config.BindUser, bm.config.BindGroup)
						}
					}
				}
			}
		}
	}

	// 验证zone文件
	if err := bm.ValidateZone(zone.Domain); err != nil {
		// 回滚：恢复原来的zone文件内容
		os.WriteFile(zoneFilePath, originalZoneContent, 0644)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		bm.logger.Error("验证zone文件失败: %v", err)
		return fmt.Errorf("验证zone文件失败: %v", err)
	}

	// 如果需要更新 named.conf
	if needUpdateNamedConf {
		namedConfPath := filepath.Join(bm.config.NamedConfPath, "named.conf")
		if err := bm.updateZoneInNamedConf(namedConfPath, zone.Domain, zoneFileName, zone.AllowQuery, zone.Comment); err != nil {
			bm.logger.Error("更新named.conf文件失败: %v", err)
			return fmt.Errorf("更新named.conf文件失败: %v", err)
		}

		// 验证named.conf配置
		if err := bm.ValidateConfig(); err != nil {
			// 回滚：恢复原始named.conf
			bm.updateZoneInNamedConf(namedConfPath, existingZone.Domain,
				filepath.Base(existingZone.File), existingZone.AllowQuery, existingZone.Comment)
			bm.logger.Error("验证named.conf配置失败: %v", err)
			return fmt.Errorf("验证named.conf配置失败: %v", err)
		}
	}

	// 刷新BIND服务器
	if err := bm.ReloadBind(); err != nil {
		bm.logger.Error("刷新BIND服务器失败: %v", err)
		// 不回滚，因为配置本身是有效的，只是刷新失败
	}

	// 操作成功，保留备份记录
	return nil
}

// DeleteAuthZone 删除权威域
func (bm *BindManager) DeleteAuthZone(domain string) error {
	// 检查是否为系统区域
	if isSystemZone(domain) {
		return fmt.Errorf("不能删除系统区域: %s", domain)
	}

	// 生成zone文件路径
	zoneFileName := fmt.Sprintf("%s.zone", domain)
	zoneFilePath := filepath.Join(bm.config.ZoneFilePath, zoneFileName)
	namedConfPath := filepath.Join(bm.config.NamedConfPath, "named.conf")

	// 读取操作前的named.conf内容用于备份和回滚
	originalNamedConf, err := os.ReadFile(namedConfPath)
	if err != nil {
		bm.logger.Error("读取named.conf文件失败: %v", err)
		return fmt.Errorf("读取named.conf文件失败: %v", err)
	}

	// 在操作前创建全量备份（named.conf + 所有zone文件）
	allZoneFiles, _ := bm.getAllZoneFiles()
	files := append([]string{namedConfPath}, allZoneFiles...)
	deleteData, _ := json.Marshal(map[string]string{
		"domain": domain,
	})
	backupID, err := bm.HistoryMgr.CreateBackup(OperationDelete, domain, deleteData, files)
	if err != nil {
		bm.logger.Warn("创建备份失败: %v", err)
		backupID = 0
	}

	// 更新named.conf文件，移除zone配置
	if err := bm.removeZoneFromNamedConf(namedConfPath, domain); err != nil {
		bm.logger.Error("更新named.conf文件失败: %v", err)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		return fmt.Errorf("更新named.conf文件失败: %v", err)
	}

	// 验证named.conf配置
	if err := bm.ValidateConfig(); err != nil {
		// 回滚：恢复named.conf配置
		os.WriteFile(namedConfPath, originalNamedConf, 0644)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		bm.logger.Error("验证named.conf配置失败: %v", err)
		return fmt.Errorf("验证named.conf配置失败: %v", err)
	}

	// 删除zone文件
	if err := os.Remove(zoneFilePath); err != nil && !os.IsNotExist(err) {
		// 回滚：恢复named.conf配置
		os.WriteFile(namedConfPath, originalNamedConf, 0644)
		// 操作失败，删除备份记录
		if backupID > 0 {
			bm.HistoryMgr.DeleteBackupRecord(backupID)
		}
		bm.logger.Error("删除zone文件失败: %v", err)
		return fmt.Errorf("删除zone文件失败: %v", err)
	}

	// 刷新BIND服务器
	if err := bm.ReloadBind(); err != nil {
		bm.logger.Error("刷新BIND服务器失败: %v", err)
		// 不回滚，因为配置本身是有效的，只是刷新失败
	}

	// 操作成功，保留备份记录
	return nil
}
