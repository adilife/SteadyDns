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

// core/bind/models.go
// BIND服务器数据结构定义

package bind

import (
	"sync"
	"time"

	"SteadyDNS/core/common"
)

// BindConfig BIND服务器配置
type BindConfig struct {
	Address        string
	RNDCKey        string
	ZoneFilePath   string
	NamedConfPath  string
	RNDPort        string
	BindUser       string
	BindGroup      string
	BindExecStart  string
	BindExecReload string
	BindExecStop   string
	NamedCheckConf string // named-checkconf命令路径或完整命令
	NamedCheckZone string // named-checkzone命令路径或完整命令
}

// Record 通用DNS记录结构体
type Record struct {
	ID       string `json:"id,omitempty"` // 记录唯一标识符，使用UUID
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"` // 可选，用于MX记录
	TTL      int    `json:"ttl,omitempty"`      // 可选，记录的生存时间
	Comment  string `json:"comment,omitempty"`  // 可选，记录后的注释信息
}

// AuthZone 权威域信息
type AuthZone struct {
	Domain     string    `json:"domain"`
	Type       string    `json:"type"`
	File       string    `json:"file"`
	AllowQuery string    `json:"allow_query"`
	Comment    string    `json:"comment,omitempty"` // 权威域注释信息，对应 named.conf 中 zone 配置块的前置注释
	SOA        SOARecord `json:"soa"`
	Records    []Record  `json:"records"` // 通用记录切片，包含除SOA外的所有记录
}

// SOARecord SOA记录
type SOARecord struct {
	PrimaryNS  string `json:"primary_ns"`
	AdminEmail string `json:"admin_email"`
	Serial     string `json:"serial"`
	Refresh    string `json:"refresh"`
	Retry      string `json:"retry"`
	Expire     string `json:"expire"`
	MinimumTTL string `json:"minimum_ttl"`
}

// NSRecord NS记录
type NSRecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ARecord A记录
type ARecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AAAARecord AAAA记录
type AAAARecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CNAMERecord CNAME记录
type CNAMERecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MXRecord MX记录
type MXRecord struct {
	ID       string `json:"id,omitempty"`
	Priority int    `json:"priority"`
	Value    string `json:"value"`
}

// TXTRecord TXT记录
type TXTRecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PTRRecord PTR记录
type PTRRecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// OtherRecord 其他记录
type OtherRecord struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HistoryRecord 操作历史记录
type HistoryRecord struct {
	ID        int       `json:"id"`
	Operation string    `json:"operation"` // create, update, delete
	Domain    string    `json:"domain"`
	Timestamp time.Time `json:"timestamp"`
	Files     []string  `json:"files"` // 涉及的文件列表
}

// HistoryRepository 历史记录仓库
type HistoryRepository struct {
	records []HistoryRecord
	maxSize int
}

// BindManager BIND服务器管理器
type BindManager struct {
	logger     *common.Logger
	config     BindConfig
	HistoryMgr *HistoryManager
	mu         sync.Mutex // 互斥锁，用于实现事务性操作，避免多用户冲突
}
