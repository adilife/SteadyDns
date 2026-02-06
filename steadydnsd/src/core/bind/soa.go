// core/bind/soa.go
// SOA序列号管理

package bind

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// parseSerial 解析SOA序列号，提取日期和流水号
// 返回：日期字符串(YYYYMMDD)、流水号字符串(2位)、错误
func parseSerial(serial string) (string, string, error) {
	if len(serial) != 10 {
		return "", "", fmt.Errorf("序列号格式不正确，应为YYYYMMDD+2位流水号")
	}

	datePart := serial[:8]   // 前8位为日期
	serialPart := serial[8:] // 后2位为流水号

	return datePart, serialPart, nil
}

// generateSerial 生成SOA记录序列号
// 格式：YYYYMMDD+2位流水号（如2026012201）
func generateSerial() string {
	dateStr := time.Now().Format("20060102")
	return fmt.Sprintf("%s01", dateStr) // 初始流水号为01
}

// getCurrentSerial 获取当前SOA序列号
func (bm *BindManager) getCurrentSerial(domain string) (string, error) {
	zoneFileName := fmt.Sprintf("%s.zone", domain)
	zoneFilePath := filepath.Join(bm.config.ZoneFilePath, zoneFileName)

	// 读取zone文件内容
	content, err := os.ReadFile(zoneFilePath)
	if err != nil {
		return "", fmt.Errorf("读取zone文件失败: %v", err)
	}

	contentStr := string(content)

	// 查找SOA记录行
	lines := strings.Split(contentStr, "\n")
	var soaLine string
	for _, line := range lines {
		if strings.Contains(line, "SOA") {
			soaLine = line
			break
		}
	}

	if soaLine == "" {
		return "", fmt.Errorf("未找到SOA记录")
	}

	// 提取序列号
	// SOA记录格式：@ IN SOA ns1.example.com. admin.example.com. (2026012201 3600 1800 604800 86400)
	fields := strings.Fields(soaLine)
	if len(fields) < 6 {
		return "", fmt.Errorf("SOA记录格式不正确")
	}

	// 序列号通常是SOA记录括号后的第一个值
	var serial string
	for i, field := range fields {
		if field == "(" && i+1 < len(fields) {
			serial = fields[i+1]
			break
		}
	}

	if serial == "" {
		return "", fmt.Errorf("未找到SOA序列号")
	}

	return serial, nil
}
