// core/bind/zone.go
// 区域文件处理

package bind

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// generateUUID 生成UUID
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// 如果生成失败，使用简单的随机字符串作为备选
		return fmt.Sprintf("%x", b)
	}
	// 按照UUID格式格式化
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// parseZoneFile 解析zone文件
func (bm *BindManager) parseZoneFile(filePath, domain string) (*AuthZone, error) {
	// 读取zone文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取zone文件失败: %v", err)
	}

	zone := &AuthZone{
		Domain:  domain,
		Type:    "master",
		File:    strings.TrimPrefix(filePath, bm.config.ZoneFilePath),
		Records: make([]Record, 0), // 初始化通用记录切片
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// 解析SOA记录 - 添加多行模式修饰符
	soaRegex := regexp.MustCompile(`(?m)^@\s+IN\s+SOA\s+([^\s]+)\s+([^\s]+)\s*\(([\s\S]*?)\)\s*`)
	soaMatch := soaRegex.FindStringSubmatch(contentStr)
	if len(soaMatch) > 0 {
		primaryNS := soaMatch[1]
		adminEmail := soaMatch[2]
		soaContent := soaMatch[3]

		// 解析SOA记录的各个字段
		serialRegex := regexp.MustCompile(`\s*([0-9]+)\s*;\s*Serial`)
		refreshRegex := regexp.MustCompile(`\s*([0-9]+)\s*;\s*Refresh`)
		retryRegex := regexp.MustCompile(`\s*([0-9]+)\s*;\s*Retry`)
		expireRegex := regexp.MustCompile(`\s*([0-9]+)\s*;\s*Expire`)
		minimumRegex := regexp.MustCompile(`\s*([0-9]+)\s*;\s*Minimum`)

		zone.SOA = SOARecord{
			PrimaryNS:  primaryNS,
			AdminEmail: adminEmail,
			Serial:     getRegexMatch(serialRegex, soaContent, "2026010101"),
			Refresh:    getRegexMatch(refreshRegex, soaContent, "3600"),
			Retry:      getRegexMatch(retryRegex, soaContent, "1800"),
			Expire:     getRegexMatch(expireRegex, soaContent, "604800"),
			MinimumTTL: getRegexMatch(minimumRegex, soaContent, "86400"),
		}
	}

	// 解析其他记录
	inSOABlock := false // 标记当前是否在SOA记录块内
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "$") {
			continue
		}

		// 检查是否进入SOA记录块
		if strings.Contains(line, "SOA") && strings.Contains(line, "(") {
			inSOABlock = true
			continue
		}

		// 检查是否退出SOA记录块
		if inSOABlock {
			if strings.Contains(line, ")") {
				inSOABlock = false
			}
			continue // 跳过SOA记录块内的所有内容
		}

		// 跳过注释行
		if strings.HasPrefix(line, ";") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		var recordType, value string
		var priority int

		// 解析记录类型，跳过IN（如果存在）
		var typeIndex int = 1
		if fields[1] == "IN" {
			typeIndex = 2
		}

		if typeIndex >= len(fields) {
			continue
		}

		recordType = fields[typeIndex]

		// 处理MX记录的优先级
		if recordType == "MX" {
			if typeIndex+2 < len(fields) {
				priority, _ = strconv.Atoi(fields[typeIndex+1])
				value = fields[typeIndex+2]
			}
		} else {
			// 其他记录类型
			if typeIndex+1 < len(fields) {
				value = strings.Join(fields[typeIndex+1:], " ")
			}
		}

		// 创建通用记录
		record := Record{
			ID:       generateUUID(), // 生成唯一ID
			Name:     name,
			Type:     recordType,
			Value:    value,
			Priority: priority,
		}

		// 添加到通用记录切片
		zone.Records = append(zone.Records, record)
	}

	return zone, nil
}

// getRegexMatch 获取正则表达式匹配结果
func getRegexMatch(regex *regexp.Regexp, content, defaultValue string) string {
	match := regex.FindStringSubmatch(content)
	if len(match) > 1 {
		return match[1]
	}
	return defaultValue
}

// ensureDotSuffix 确保域名值只在需要时添加点
func ensureDotSuffix(value string) string {
	if value == "" {
		return value
	}
	// 如果已经以点结尾，直接返回
	if strings.HasSuffix(value, ".") {
		return value
	}
	// 否则添加点
	return value + "."
}

// generateZoneContent 生成zone文件内容
func (bm *BindManager) generateZoneContent(zone AuthZone) string {
	var buffer bytes.Buffer

	// 写入TTL
	buffer.WriteString("$TTL 86400\n")

	// 写入SOA记录
	buffer.WriteString(fmt.Sprintf("@\tIN SOA %s %s (\n", ensureDotSuffix(zone.SOA.PrimaryNS), ensureDotSuffix(zone.SOA.AdminEmail)))
	buffer.WriteString(fmt.Sprintf("\t\t%s ; Serial\n", zone.SOA.Serial))
	buffer.WriteString(fmt.Sprintf("\t\t%s ; Refresh\n", zone.SOA.Refresh))
	buffer.WriteString(fmt.Sprintf("\t\t%s ; Retry\n", zone.SOA.Retry))
	buffer.WriteString(fmt.Sprintf("\t\t%s ; Expire\n", zone.SOA.Expire))
	buffer.WriteString(fmt.Sprintf("\t\t%s ; Minimum TTL\n", zone.SOA.MinimumTTL))
	buffer.WriteString(")\n\n")

	// 按记录类型分组，便于生成zone文件
	recordGroups := make(map[string][]Record)
	for _, record := range zone.Records {
		recordGroups[record.Type] = append(recordGroups[record.Type], record)
	}

	// 记录排序函数：@符号排在最前面，其他按名称字母顺序
	recordSorter := func(records []Record) {
		sort.Slice(records, func(i, j int) bool {
			// @符号排在最前面
			if records[i].Name == "@" && records[j].Name != "@" {
				return true
			}
			if records[i].Name != "@" && records[j].Name == "@" {
				return false
			}
			// 其他按名称字母顺序
			return records[i].Name < records[j].Name
		})
	}

	// 写入NS记录
	if nsRecords, ok := recordGroups["NS"]; ok {
		// 按名称排序
		recordSorter(nsRecords)
		for _, record := range nsRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN NS\t%s\n", record.Name, ensureDotSuffix(record.Value)))
		}
		buffer.WriteString("\n")
	}

	// 写入A记录
	if aRecords, ok := recordGroups["A"]; ok {
		// 按名称排序
		recordSorter(aRecords)
		for _, record := range aRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN A\t%s\n", record.Name, record.Value))
		}
		buffer.WriteString("\n")
	}

	// 写入AAAA记录
	if aaaaRecords, ok := recordGroups["AAAA"]; ok {
		// 按名称排序
		recordSorter(aaaaRecords)
		for _, record := range aaaaRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN AAAA\t%s\n", record.Name, record.Value))
		}
		buffer.WriteString("\n")
	}

	// 写入CNAME记录
	if cnameRecords, ok := recordGroups["CNAME"]; ok {
		// 按名称排序
		recordSorter(cnameRecords)
		for _, record := range cnameRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN CNAME\t%s\n", record.Name, ensureDotSuffix(record.Value)))
		}
		buffer.WriteString("\n")
	}

	// 写入MX记录
	if mxRecords, ok := recordGroups["MX"]; ok {
		// 按名称排序
		recordSorter(mxRecords)
		for _, record := range mxRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN MX %d\t%s\n", record.Name, record.Priority, ensureDotSuffix(record.Value)))
		}
		buffer.WriteString("\n")
	}

	// 写入TXT记录
	if txtRecords, ok := recordGroups["TXT"]; ok {
		// 按名称排序
		recordSorter(txtRecords)
		for _, record := range txtRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN TXT\t%s\n", record.Name, record.Value))
		}
		buffer.WriteString("\n")
	}

	// 写入PTR记录
	if ptrRecords, ok := recordGroups["PTR"]; ok {
		// 按名称排序
		recordSorter(ptrRecords)
		for _, record := range ptrRecords {
			buffer.WriteString(fmt.Sprintf("%s\tIN PTR\t%s\n", record.Name, ensureDotSuffix(record.Value)))
		}
		buffer.WriteString("\n")
	}

	// 写入其他记录
	for recordType, records := range recordGroups {
		// 跳过已处理的记录类型
		if recordType == "NS" || recordType == "A" || recordType == "AAAA" || recordType == "CNAME" || recordType == "MX" || recordType == "TXT" || recordType == "PTR" {
			continue
		}

		// 按名称排序
		recordSorter(records)
		for _, record := range records {
			// 对于其他可能需要加点的记录类型，也使用ensureDotSuffix
			buffer.WriteString(fmt.Sprintf("%s\tIN %s\t%s\n", record.Name, record.Type, ensureDotSuffix(record.Value)))
		}
		buffer.WriteString("\n")
	}

	return buffer.String()
}
