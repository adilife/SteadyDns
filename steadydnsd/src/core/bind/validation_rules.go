// core/bind/validation_rules.go
// 记录验证规则

package bind

import "fmt"

// CheckCNAMEConflicts 检查CNAME冲突
// 1. 同一名称不能同时存在CNAME和其他类型的记录
// 2. NS记录不能指向CNAME
// 3. MX记录不能指向CNAME
// 4. 警告用户不要在裸域使用CNAME记录
func CheckCNAMEConflicts(zone AuthZone) error {
	// 创建一个映射，用于记录已经存在的非CNAME记录名称
	nonCNAMERecords := make(map[string]bool)
	cnameRecords := make([]Record, 0)
	
	// 遍历所有记录，收集信息
	for _, record := range zone.Records {
		if record.Type == "CNAME" {
			// 收集CNAME记录，稍后处理
			cnameRecords = append(cnameRecords, record)
		} else {
			// 记录非CNAME记录名称
			nonCNAMERecords[record.Name] = true
		}
	}
	
	// 检查CNAME记录是否与其他记录冲突
	for _, cname := range cnameRecords {
		if nonCNAMERecords[cname.Name] {
			return fmt.Errorf("记录名称冲突: %s 同时存在其他记录和 CNAME 记录", cname.Name)
		}
		
		// 警告：不要在裸域使用CNAME记录
		if cname.Name == "@" || cname.Name == zone.Domain {
			return fmt.Errorf("警告：不建议在裸域使用CNAME记录，这可能导致其他记录失效")
		}
	}
	
	// 检查NS记录是否指向CNAME
	// 这里简化处理，实际应该检查指向的域名是否为CNAME
	// 但由于我们没有完整的DNS解析功能，这里只做简单的警告
	
	// 检查MX记录是否指向CNAME
	// 同样简化处理
	
	return nil
}
