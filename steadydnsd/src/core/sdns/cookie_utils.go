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
// core/sdns/cookie_utils.go
// DNS Cookie工具函数 - 实现RFC 7873 DNS Cookies

package sdns

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/miekg/dns"
)

// EDNS0 Cookie选项常量
const (
	// EDNS0CookieOptionCode EDNS0 COOKIE选项代码 (RFC 7873)
	EDNS0CookieOptionCode = 10

	// ClientCookieSize Client Cookie固定长度 (8字节)
	ClientCookieSize = 8

	// MinServerCookieSize Server Cookie最小长度 (8字节)
	MinServerCookieSize = 8

	// MaxServerCookieSize Server Cookie最大长度 (32字节)
	MaxServerCookieSize = 32

	// MinCookieSize 最小Cookie长度 (仅Client Cookie)
	MinCookieSize = ClientCookieSize

	// MaxCookieSize 最大Cookie长度 (Client + Server Cookie)
	MaxCookieSize = ClientCookieSize + MaxServerCookieSize
)

// CookieError Cookie相关错误类型
type CookieError struct {
	Op  string
	Err string
}

// Error 实现error接口
func (e *CookieError) Error() string {
	return fmt.Sprintf("cookie %s error: %s", e.Op, e.Err)
}

// ExtractClientCookie 从DNS消息中提取Client Cookie (8字节)
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - []byte: Client Cookie (8字节)
//   - error: 错误信息
func ExtractClientCookie(msg *dns.Msg) ([]byte, error) {
	if msg == nil {
		return nil, &CookieError{Op: "extract_client", Err: "nil DNS message"}
	}

	// 查找EDNS0 COOKIE选项
	cookieOpt := findCookieOption(msg)
	if cookieOpt == nil {
		return nil, &CookieError{Op: "extract_client", Err: "no COOKIE option found"}
	}

	// 获取Cookie数据
	cookieData := cookieOpt.Data

	// 检查Cookie长度
	if len(cookieData) < ClientCookieSize {
		return nil, &CookieError{Op: "extract_client", Err: "cookie data too short for client cookie"}
	}

	// 提取Client Cookie (前8字节)
	clientCookie := make([]byte, ClientCookieSize)
	copy(clientCookie, cookieData[:ClientCookieSize])

	return clientCookie, nil
}

// ExtractServerCookie 从DNS消息中提取Server Cookie (8-32字节)
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - []byte: Server Cookie (8-32字节)
//   - error: 错误信息
func ExtractServerCookie(msg *dns.Msg) ([]byte, error) {
	if msg == nil {
		return nil, &CookieError{Op: "extract_server", Err: "nil DNS message"}
	}

	// 查找EDNS0 COOKIE选项
	cookieOpt := findCookieOption(msg)
	if cookieOpt == nil {
		return nil, &CookieError{Op: "extract_server", Err: "no COOKIE option found"}
	}

	// 获取Cookie数据
	cookieData := cookieOpt.Data

	// 检查是否有Server Cookie (总长度 > 8字节)
	if len(cookieData) <= ClientCookieSize {
		return nil, &CookieError{Op: "extract_server", Err: "no server cookie present"}
	}

	// 提取Server Cookie (8字节之后)
	serverCookieData := cookieData[ClientCookieSize:]

	// 验证Server Cookie长度
	if len(serverCookieData) < MinServerCookieSize {
		return nil, &CookieError{Op: "extract_server", Err: "server cookie too short"}
	}

	if len(serverCookieData) > MaxServerCookieSize {
		return nil, &CookieError{Op: "extract_server", Err: "server cookie too long"}
	}

	serverCookie := make([]byte, len(serverCookieData))
	copy(serverCookie, serverCookieData)

	return serverCookie, nil
}

// ExtractCookie 从DNS消息中提取完整Cookie (Client + Server)
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - []byte: 完整Cookie数据
//   - error: 错误信息
func ExtractCookie(msg *dns.Msg) ([]byte, error) {
	if msg == nil {
		return nil, &CookieError{Op: "extract", Err: "nil DNS message"}
	}

	// 查找EDNS0 COOKIE选项
	cookieOpt := findCookieOption(msg)
	if cookieOpt == nil {
		return nil, &CookieError{Op: "extract", Err: "no COOKIE option found"}
	}

	// 获取Cookie数据
	cookieData := cookieOpt.Data

	// 验证最小长度
	if len(cookieData) < MinCookieSize {
		return nil, &CookieError{Op: "extract", Err: "cookie data too short"}
	}

	// 验证最大长度
	if len(cookieData) > MaxCookieSize {
		return nil, &CookieError{Op: "extract", Err: "cookie data too long"}
	}

	// 复制并返回完整Cookie
	fullCookie := make([]byte, len(cookieData))
	copy(fullCookie, cookieData)

	return fullCookie, nil
}

// InjectCookie 向DNS消息注入EDNS0 COOKIE选项
// 参数:
//   - msg: DNS消息
//   - clientCookie: Client Cookie (8字节)
//   - serverCookie: Server Cookie (8-32字节，可选，可为nil)
//
// 返回:
//   - error: 错误信息
func InjectCookie(msg *dns.Msg, clientCookie []byte, serverCookie []byte) error {
	if msg == nil {
		return &CookieError{Op: "inject", Err: "nil DNS message"}
	}

	// 验证Client Cookie长度
	if len(clientCookie) != ClientCookieSize {
		return &CookieError{Op: "inject", Err: fmt.Sprintf("client cookie must be %d bytes", ClientCookieSize)}
	}

	// 验证Server Cookie长度 (如果提供)
	if serverCookie != nil {
		if len(serverCookie) < MinServerCookieSize || len(serverCookie) > MaxServerCookieSize {
			return &CookieError{Op: "inject", Err: fmt.Sprintf("server cookie must be %d-%d bytes", MinServerCookieSize, MaxServerCookieSize)}
		}
	}

	// 移除已存在的COOKIE选项
	RemoveCookie(msg)

	// 构建Cookie数据
	var cookieData []byte
	cookieData = append(cookieData, clientCookie...)
	if serverCookie != nil {
		cookieData = append(cookieData, serverCookie...)
	}

	// 创建EDNS0 COOKIE选项
	cookieOpt := &dns.EDNS0_LOCAL{
		Code: EDNS0CookieOptionCode,
		Data: cookieData,
	}

	// 添加或更新OPT记录
	opt := msg.IsEdns0()
	if opt == nil {
		// 消息没有EDNS0，需要添加
		opt = &dns.OPT{
			Hdr: dns.RR_Header{
				Name:   ".",
				Rrtype: dns.TypeOPT,
				Class:  dns.DefaultMsgSize,
			},
		}
		msg.Extra = append(msg.Extra, opt)
	}

	// 添加COOKIE选项到OPT记录
	opt.Option = append(opt.Option, cookieOpt)

	return nil
}

// HasCookieOption 检查消息是否已有COOKIE选项
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - bool: 是否存在COOKIE选项
func HasCookieOption(msg *dns.Msg) bool {
	if msg == nil {
		return false
	}

	return findCookieOption(msg) != nil
}

// RemoveCookie 移除消息中的COOKIE选项
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - bool: 是否成功移除
func RemoveCookie(msg *dns.Msg) bool {
	if msg == nil {
		return false
	}

	opt := msg.IsEdns0()
	if opt == nil {
		return false
	}

	removed := false
	newOptions := make([]dns.EDNS0, 0, len(opt.Option))

	for _, option := range opt.Option {
		if localOpt, ok := option.(*dns.EDNS0_LOCAL); ok {
			if localOpt.Code == EDNS0CookieOptionCode {
				removed = true
				continue // 跳过COOKIE选项
			}
		}
		newOptions = append(newOptions, option)
	}

	opt.Option = newOptions

	// 如果没有其他选项，移除整个OPT记录
	if len(opt.Option) == 0 {
		newExtra := make([]dns.RR, 0, len(msg.Extra))
		for _, rr := range msg.Extra {
			if rr.Header().Rrtype != dns.TypeOPT {
				newExtra = append(newExtra, rr)
			}
		}
		msg.Extra = newExtra
	}

	return removed
}

// ParseCookie 解析Cookie字符串为Client/Server部分
// 参数:
//   - cookieData: Cookie数据
//
// 返回:
//   - []byte: Client Cookie
//   - []byte: Server Cookie (可能为nil)
//   - error: 错误信息
func ParseCookie(cookieData []byte) ([]byte, []byte, error) {
	if cookieData == nil {
		return nil, nil, &CookieError{Op: "parse", Err: "nil cookie data"}
	}

	// 验证最小长度
	if len(cookieData) < MinCookieSize {
		return nil, nil, &CookieError{Op: "parse", Err: "cookie data too short"}
	}

	// 验证最大长度
	if len(cookieData) > MaxCookieSize {
		return nil, nil, &CookieError{Op: "parse", Err: "cookie data too long"}
	}

	// 提取Client Cookie
	clientCookie := make([]byte, ClientCookieSize)
	copy(clientCookie, cookieData[:ClientCookieSize])

	// 提取Server Cookie (如果存在)
	var serverCookie []byte
	if len(cookieData) > ClientCookieSize {
		serverCookieData := cookieData[ClientCookieSize:]
		if len(serverCookieData) >= MinServerCookieSize && len(serverCookieData) <= MaxServerCookieSize {
			serverCookie = make([]byte, len(serverCookieData))
			copy(serverCookie, serverCookieData)
		}
	}

	return clientCookie, serverCookie, nil
}

// IsEchoedCookie 检查是否为echoed Client Cookie (只有8字节，无Server Cookie)
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - bool: 是否只有Client Cookie (echoed)
//   - error: 错误信息
func IsEchoedCookie(msg *dns.Msg) (bool, error) {
	if msg == nil {
		return false, &CookieError{Op: "check_echoed", Err: "nil DNS message"}
	}

	// 查找EDNS0 COOKIE选项
	cookieOpt := findCookieOption(msg)
	if cookieOpt == nil {
		return false, &CookieError{Op: "check_echoed", Err: "no COOKIE option found"}
	}

	// 获取Cookie数据
	cookieData := cookieOpt.Data

	// 检查长度是否正好是8字节 (只有Client Cookie)
	if len(cookieData) == ClientCookieSize {
		return true, nil
	}

	return false, nil
}

// IsValidCookieSize 检查Cookie大小是否有效
// 参数:
//   - size: Cookie数据大小
//
// 返回:
//   - bool: 是否有效
func IsValidCookieSize(size int) bool {
	return size >= MinCookieSize && size <= MaxCookieSize
}

// GetCookieSize 获取DNS消息中Cookie的大小
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - int: Cookie大小 (0表示不存在)
//   - error: 错误信息
func GetCookieSize(msg *dns.Msg) (int, error) {
	if msg == nil {
		return 0, &CookieError{Op: "get_size", Err: "nil DNS message"}
	}

	cookieOpt := findCookieOption(msg)
	if cookieOpt == nil {
		return 0, nil
	}

	return len(cookieOpt.Data), nil
}

// CookieToHex 将Cookie数据转换为十六进制字符串
// 参数:
//   - cookie: Cookie数据
//
// 返回:
//   - string: 十六进制字符串
func CookieToHex(cookie []byte) string {
	if cookie == nil {
		return ""
	}
	return hex.EncodeToString(cookie)
}

// HexToCookie 将十六进制字符串转换为Cookie数据
// 参数:
//   - hexStr: 十六进制字符串
//
// 返回:
//   - []byte: Cookie数据
//   - error: 错误信息
func HexToCookie(hexStr string) ([]byte, error) {
	if hexStr == "" {
		return nil, errors.New("empty hex string")
	}

	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	return data, nil
}

// findCookieOption 在DNS消息中查找EDNS0 COOKIE选项 (内部函数)
// 参数:
//   - msg: DNS消息
//
// 返回:
//   - *dns.EDNS0_LOCAL: COOKIE选项
func findCookieOption(msg *dns.Msg) *dns.EDNS0_LOCAL {
	opt := msg.IsEdns0()
	if opt == nil {
		return nil
	}

	for _, option := range opt.Option {
		if localOpt, ok := option.(*dns.EDNS0_LOCAL); ok {
			if localOpt.Code == EDNS0CookieOptionCode {
				return localOpt
			}
		}
	}

	return nil
}

// ValidateCookieData 验证Cookie数据格式
// 参数:
//   - cookieData: Cookie数据
//
// 返回:
//   - error: 错误信息
func ValidateCookieData(cookieData []byte) error {
	if cookieData == nil {
		return &CookieError{Op: "validate", Err: "nil cookie data"}
	}

	if len(cookieData) < MinCookieSize {
		return &CookieError{Op: "validate", Err: fmt.Sprintf("cookie data too short, minimum %d bytes", MinCookieSize)}
	}

	if len(cookieData) > MaxCookieSize {
		return &CookieError{Op: "validate", Err: fmt.Sprintf("cookie data too long, maximum %d bytes", MaxCookieSize)}
	}

	// 如果包含Server Cookie，验证其长度
	if len(cookieData) > ClientCookieSize {
		serverCookieLen := len(cookieData) - ClientCookieSize
		if serverCookieLen < MinServerCookieSize {
			return &CookieError{Op: "validate", Err: fmt.Sprintf("server cookie too short, minimum %d bytes", MinServerCookieSize)}
		}
		if serverCookieLen > MaxServerCookieSize {
			return &CookieError{Op: "validate", Err: fmt.Sprintf("server cookie too long, maximum %d bytes", MaxServerCookieSize)}
		}
	}

	return nil
}

// CreateEchoedCookieResponse 创建echoed Client Cookie响应
// 参数:
//   - request: 原始请求消息
//   - clientCookie: Client Cookie
//
// 返回:
//   - *dns.Msg: 响应消息
//   - error: 错误信息
func CreateEchoedCookieResponse(request *dns.Msg, clientCookie []byte) (*dns.Msg, error) {
	if request == nil {
		return nil, &CookieError{Op: "create_echo", Err: "nil request message"}
	}

	if len(clientCookie) != ClientCookieSize {
		return nil, &CookieError{Op: "create_echo", Err: fmt.Sprintf("client cookie must be %d bytes", ClientCookieSize)}
	}

	// 创建响应消息
	response := new(dns.Msg)
	response.SetReply(request)

	// 注入echoed Client Cookie (只有Client Cookie，没有Server Cookie)
	err := InjectCookie(response, clientCookie, nil)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// CreateCookieResponse 创建包含完整Cookie的响应
// 参数:
//   - request: 原始请求消息
//   - clientCookie: Client Cookie
//   - serverCookie: Server Cookie
//
// 返回:
//   - *dns.Msg: 响应消息
//   - error: 错误信息
func CreateCookieResponse(request *dns.Msg, clientCookie []byte, serverCookie []byte) (*dns.Msg, error) {
	if request == nil {
		return nil, &CookieError{Op: "create_response", Err: "nil request message"}
	}

	if len(clientCookie) != ClientCookieSize {
		return nil, &CookieError{Op: "create_response", Err: fmt.Sprintf("client cookie must be %d bytes", ClientCookieSize)}
	}

	if serverCookie != nil {
		if len(serverCookie) < MinServerCookieSize || len(serverCookie) > MaxServerCookieSize {
			return nil, &CookieError{Op: "create_response", Err: fmt.Sprintf("server cookie must be %d-%d bytes", MinServerCookieSize, MaxServerCookieSize)}
		}
	}

	// 创建响应消息
	response := new(dns.Msg)
	response.SetReply(request)

	// 注入完整Cookie
	err := InjectCookie(response, clientCookie, serverCookie)
	if err != nil {
		return nil, err
	}

	return response, nil
}
