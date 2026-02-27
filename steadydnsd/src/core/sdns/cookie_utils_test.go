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
// core/sdns/cookie_utils_test.go
// Cookie工具函数单元测试

package sdns

import (
	"testing"

	"github.com/miekg/dns"
)

// createTestDNSMessageWithCookie 创建带有Cookie的测试DNS消息
func createTestDNSMessageWithCookie(clientCookie, serverCookie []byte) *dns.Msg {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	// 创建OPT记录
	opt := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
			Class:  dns.DefaultMsgSize,
		},
	}

	// 构建Cookie数据
	cookieData := make([]byte, 0, len(clientCookie)+len(serverCookie))
	cookieData = append(cookieData, clientCookie...)
	if serverCookie != nil {
		cookieData = append(cookieData, serverCookie...)
	}

	// 添加COOKIE选项
	cookieOpt := &dns.EDNS0_LOCAL{
		Code: EDNS0CookieOptionCode,
		Data: cookieData,
	}
	opt.Option = append(opt.Option, cookieOpt)
	msg.Extra = append(msg.Extra, opt)

	return msg
}

// TestExtractClientCookie 测试提取Client Cookie
func TestExtractClientCookie(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	msg := createTestDNSMessageWithCookie(clientCookie, serverCookie)

	extracted, err := ExtractClientCookie(msg)
	if err != nil {
		t.Fatalf("ExtractClientCookie() error = %v", err)
	}

	if len(extracted) != ClientCookieSize {
		t.Errorf("ExtractClientCookie() length = %d, want %d", len(extracted), ClientCookieSize)
	}

	if !bytesEqual(extracted, clientCookie) {
		t.Error("ExtractClientCookie() returned wrong client cookie")
	}
}

// TestExtractClientCookieNilMessage 测试提取Client Cookie - nil消息
func TestExtractClientCookieNilMessage(t *testing.T) {
	_, err := ExtractClientCookie(nil)
	if err == nil {
		t.Error("ExtractClientCookie(nil) should return error")
	}
}

// TestExtractClientCookieNoCookie 测试提取Client Cookie - 无Cookie
func TestExtractClientCookieNoCookie(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	// 添加没有Cookie的OPT记录
	opt := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
			Class:  dns.DefaultMsgSize,
		},
	}
	msg.Extra = append(msg.Extra, opt)

	_, err := ExtractClientCookie(msg)
	if err == nil {
		t.Error("ExtractClientCookie() should return error when no cookie")
	}
}

// TestExtractClientCookieTooShort 测试提取Client Cookie - 数据太短
func TestExtractClientCookieTooShort(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	// 创建带有太短Cookie数据的OPT记录
	opt := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
			Class:  dns.DefaultMsgSize,
		},
	}

	cookieOpt := &dns.EDNS0_LOCAL{
		Code: EDNS0CookieOptionCode,
		Data: []byte{1, 2, 3}, // 只有3字节，太短
	}
	opt.Option = append(opt.Option, cookieOpt)
	msg.Extra = append(msg.Extra, opt)

	_, err := ExtractClientCookie(msg)
	if err == nil {
		t.Error("ExtractClientCookie() should return error for too short cookie data")
	}
}

// TestExtractServerCookie 测试提取Server Cookie
func TestExtractServerCookie(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	msg := createTestDNSMessageWithCookie(clientCookie, serverCookie)

	extracted, err := ExtractServerCookie(msg)
	if err != nil {
		t.Fatalf("ExtractServerCookie() error = %v", err)
	}

	if len(extracted) != len(serverCookie) {
		t.Errorf("ExtractServerCookie() length = %d, want %d", len(extracted), len(serverCookie))
	}

	if !bytesEqual(extracted, serverCookie) {
		t.Error("ExtractServerCookie() returned wrong server cookie")
	}
}

// TestExtractServerCookieOnlyClient 测试提取Server Cookie - 只有Client Cookie
func TestExtractServerCookieOnlyClient(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	msg := createTestDNSMessageWithCookie(clientCookie, nil)

	_, err := ExtractServerCookie(msg)
	if err == nil {
		t.Error("ExtractServerCookie() should return error when only client cookie present")
	}
}

// TestExtractServerCookieTooShort 测试提取Server Cookie - Server Cookie太短
func TestExtractServerCookieTooShort(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	opt := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
			Class:  dns.DefaultMsgSize,
		},
	}

	// Client Cookie + 太短的Server Cookie
	cookieData := append([]byte{1, 2, 3, 4, 5, 6, 7, 8}, []byte{9, 10}...)
	cookieOpt := &dns.EDNS0_LOCAL{
		Code: EDNS0CookieOptionCode,
		Data: cookieData,
	}
	opt.Option = append(opt.Option, cookieOpt)
	msg.Extra = append(msg.Extra, opt)

	_, err := ExtractServerCookie(msg)
	if err == nil {
		t.Error("ExtractServerCookie() should return error for too short server cookie")
	}
}

// TestExtractServerCookieTooLong 测试提取Server Cookie - Server Cookie太长
func TestExtractServerCookieTooLong(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	opt := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
			Class:  dns.DefaultMsgSize,
		},
	}

	// Client Cookie + 太长的Server Cookie (33字节)
	serverCookie := make([]byte, 33)
	cookieData := append([]byte{1, 2, 3, 4, 5, 6, 7, 8}, serverCookie...)
	cookieOpt := &dns.EDNS0_LOCAL{
		Code: EDNS0CookieOptionCode,
		Data: cookieData,
	}
	opt.Option = append(opt.Option, cookieOpt)
	msg.Extra = append(msg.Extra, opt)

	_, err := ExtractServerCookie(msg)
	if err == nil {
		t.Error("ExtractServerCookie() should return error for too long server cookie")
	}
}

// TestExtractCookie 测试提取完整Cookie
func TestExtractCookie(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	msg := createTestDNSMessageWithCookie(clientCookie, serverCookie)

	extracted, err := ExtractCookie(msg)
	if err != nil {
		t.Fatalf("ExtractCookie() error = %v", err)
	}

	expectedLen := len(clientCookie) + len(serverCookie)
	if len(extracted) != expectedLen {
		t.Errorf("ExtractCookie() length = %d, want %d", len(extracted), expectedLen)
	}
}

// TestInjectCookie 测试注入Cookie
func TestInjectCookie(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	err := InjectCookie(msg, clientCookie, serverCookie)
	if err != nil {
		t.Fatalf("InjectCookie() error = %v", err)
	}

	// 验证Cookie已注入
	if !HasCookieOption(msg) {
		t.Error("HasCookieOption() = false after InjectCookie, want true")
	}

	// 验证可以提取
	extractedClient, err := ExtractClientCookie(msg)
	if err != nil {
		t.Fatalf("ExtractClientCookie() error = %v", err)
	}
	if !bytesEqual(extractedClient, clientCookie) {
		t.Error("Extracted client cookie mismatch")
	}

	extractedServer, err := ExtractServerCookie(msg)
	if err != nil {
		t.Fatalf("ExtractServerCookie() error = %v", err)
	}
	if !bytesEqual(extractedServer, serverCookie) {
		t.Error("Extracted server cookie mismatch")
	}
}

// TestInjectCookieNilMessage 测试注入Cookie - nil消息
func TestInjectCookieNilMessage(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	err := InjectCookie(nil, clientCookie, nil)
	if err == nil {
		t.Error("InjectCookie(nil) should return error")
	}
}

// TestInjectCookieWrongClientSize 测试注入Cookie - Client Cookie长度错误
func TestInjectCookieWrongClientSize(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	// 7字节的Client Cookie (错误)
	wrongClientCookie := []byte{1, 2, 3, 4, 5, 6, 7}
	err := InjectCookie(msg, wrongClientCookie, nil)
	if err == nil {
		t.Error("InjectCookie() should return error for wrong client cookie size")
	}
}

// TestInjectCookieWrongServerSize 测试注入Cookie - Server Cookie长度错误
func TestInjectCookieWrongServerSize(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	// 7字节的Server Cookie (错误，应该是8-32字节)
	wrongServerCookie := []byte{9, 10, 11, 12, 13, 14, 15}

	err := InjectCookie(msg, clientCookie, wrongServerCookie)
	if err == nil {
		t.Error("InjectCookie() should return error for wrong server cookie size")
	}
}

// TestInjectCookieReplaceExisting 测试注入Cookie - 替换已存在的Cookie
func TestInjectCookieReplaceExisting(t *testing.T) {
	oldClientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	oldServerCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}
	msg := createTestDNSMessageWithCookie(oldClientCookie, oldServerCookie)

	newClientCookie := []byte{8, 7, 6, 5, 4, 3, 2, 1}
	newServerCookie := []byte{16, 15, 14, 13, 12, 11, 10, 9}

	err := InjectCookie(msg, newClientCookie, newServerCookie)
	if err != nil {
		t.Fatalf("InjectCookie() error = %v", err)
	}

	// 验证新Cookie已替换旧Cookie
	extractedClient, _ := ExtractClientCookie(msg)
	if !bytesEqual(extractedClient, newClientCookie) {
		t.Error("Client cookie was not replaced")
	}

	extractedServer, _ := ExtractServerCookie(msg)
	if !bytesEqual(extractedServer, newServerCookie) {
		t.Error("Server cookie was not replaced")
	}
}

// TestHasCookieOption 测试检查Cookie选项
func TestHasCookieOption(t *testing.T) {
	// 有Cookie的消息
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	msgWithCookie := createTestDNSMessageWithCookie(clientCookie, nil)

	if !HasCookieOption(msgWithCookie) {
		t.Error("HasCookieOption() = false for message with cookie, want true")
	}

	// 无Cookie的消息
	msgWithoutCookie := new(dns.Msg)
	msgWithoutCookie.SetQuestion("example.com.", dns.TypeA)

	if HasCookieOption(msgWithoutCookie) {
		t.Error("HasCookieOption() = true for message without cookie, want false")
	}
}

// TestHasCookieOptionNilMessage 测试检查Cookie选项 - nil消息
func TestHasCookieOptionNilMessage(t *testing.T) {
	if HasCookieOption(nil) {
		t.Error("HasCookieOption(nil) = true, want false")
	}
}

// TestRemoveCookie 测试移除Cookie
func TestRemoveCookie(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}
	msg := createTestDNSMessageWithCookie(clientCookie, serverCookie)

	// 验证Cookie存在
	if !HasCookieOption(msg) {
		t.Fatal("Cookie should exist before removal")
	}

	// 移除Cookie
	removed := RemoveCookie(msg)
	if !removed {
		t.Error("RemoveCookie() = false, want true")
	}

	// 验证Cookie已移除
	if HasCookieOption(msg) {
		t.Error("HasCookieOption() = true after RemoveCookie, want false")
	}
}

// TestRemoveCookieNoCookie 测试移除Cookie - 无Cookie
func TestRemoveCookieNoCookie(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	removed := RemoveCookie(msg)
	if removed {
		t.Error("RemoveCookie() = true for message without cookie, want false")
	}
}

// TestRemoveCookieNilMessage 测试移除Cookie - nil消息
func TestRemoveCookieNilMessage(t *testing.T) {
	removed := RemoveCookie(nil)
	if removed {
		t.Error("RemoveCookie(nil) = true, want false")
	}
}

// TestParseCookie 测试解析Cookie
func TestParseCookie(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 完整Cookie
	fullCookie := append(clientCookie, serverCookie...)
	gotClient, gotServer, err := ParseCookie(fullCookie)
	if err != nil {
		t.Fatalf("ParseCookie() error = %v", err)
	}
	if !bytesEqual(gotClient, clientCookie) {
		t.Error("ParseCookie() returned wrong client cookie")
	}
	if !bytesEqual(gotServer, serverCookie) {
		t.Error("ParseCookie() returned wrong server cookie")
	}

	// 只有Client Cookie
	gotClient, gotServer, err = ParseCookie(clientCookie)
	if err != nil {
		t.Fatalf("ParseCookie() error = %v", err)
	}
	if !bytesEqual(gotClient, clientCookie) {
		t.Error("ParseCookie() returned wrong client cookie")
	}
	if gotServer != nil {
		t.Error("ParseCookie() should return nil server cookie when only client cookie present")
	}
}

// TestParseCookieNil 测试解析Cookie - nil数据
func TestParseCookieNil(t *testing.T) {
	_, _, err := ParseCookie(nil)
	if err == nil {
		t.Error("ParseCookie(nil) should return error")
	}
}

// TestParseCookieTooShort 测试解析Cookie - 数据太短
func TestParseCookieTooShort(t *testing.T) {
	_, _, err := ParseCookie([]byte{1, 2, 3})
	if err == nil {
		t.Error("ParseCookie() should return error for too short data")
	}
}

// TestParseCookieTooLong 测试解析Cookie - 数据太长
func TestParseCookieTooLong(t *testing.T) {
	// 41字节 (8 + 33，超过最大限制)
	cookieData := make([]byte, 41)
	_, _, err := ParseCookie(cookieData)
	if err == nil {
		t.Error("ParseCookie() should return error for too long data")
	}
}

// TestIsEchoedCookie 测试检查是否为echoed Client Cookie
func TestIsEchoedCookie(t *testing.T) {
	// 只有Client Cookie (8字节) - 应该是echoed
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	msg := createTestDNSMessageWithCookie(clientCookie, nil)

	isEchoed, err := IsEchoedCookie(msg)
	if err != nil {
		t.Fatalf("IsEchoedCookie() error = %v", err)
	}
	if !isEchoed {
		t.Error("IsEchoedCookie() = false for echoed cookie, want true")
	}

	// 有Server Cookie - 不是echoed
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}
	msg = createTestDNSMessageWithCookie(clientCookie, serverCookie)

	isEchoed, err = IsEchoedCookie(msg)
	if err != nil {
		t.Fatalf("IsEchoedCookie() error = %v", err)
	}
	if isEchoed {
		t.Error("IsEchoedCookie() = true for non-echoed cookie, want false")
	}
}

// TestIsEchoedCookieNilMessage 测试检查是否为echoed Client Cookie - nil消息
func TestIsEchoedCookieNilMessage(t *testing.T) {
	_, err := IsEchoedCookie(nil)
	if err == nil {
		t.Error("IsEchoedCookie(nil) should return error")
	}
}

// TestIsValidCookieSize 测试检查Cookie大小是否有效
func TestIsValidCookieSize(t *testing.T) {
	tests := []struct {
		size int
		want bool
	}{
		{0, false},   // 太小
		{7, false},   // 太小
		{8, true},    // 最小有效值 (只有Client Cookie)
		{16, true},   // 有效
		{40, true},   // 最大有效值
		{41, false},  // 太大
		{100, false}, // 太大
	}

	for _, tt := range tests {
		got := IsValidCookieSize(tt.size)
		if got != tt.want {
			t.Errorf("IsValidCookieSize(%d) = %v, want %v", tt.size, got, tt.want)
		}
	}
}

// TestGetCookieSize 测试获取Cookie大小
func TestGetCookieSize(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	// 有Cookie的消息
	msg := createTestDNSMessageWithCookie(clientCookie, serverCookie)
	size, err := GetCookieSize(msg)
	if err != nil {
		t.Fatalf("GetCookieSize() error = %v", err)
	}
	expectedSize := len(clientCookie) + len(serverCookie)
	if size != expectedSize {
		t.Errorf("GetCookieSize() = %d, want %d", size, expectedSize)
	}

	// 无Cookie的消息
	msgWithoutCookie := new(dns.Msg)
	msgWithoutCookie.SetQuestion("example.com.", dns.TypeA)
	size, err = GetCookieSize(msgWithoutCookie)
	if err != nil {
		t.Fatalf("GetCookieSize() error = %v", err)
	}
	if size != 0 {
		t.Errorf("GetCookieSize() = %d for message without cookie, want 0", size)
	}
}

// TestCookieToHex 测试Cookie转十六进制
func TestCookieToHex(t *testing.T) {
	tests := []struct {
		cookie []byte
		want   string
	}{
		{[]byte{0x01, 0x02, 0x03, 0x04}, "01020304"},
		{[]byte{0xFF, 0xFE, 0xFD, 0xFC}, "fffefdfc"},
		{[]byte{}, ""},
		{nil, ""},
	}

	for _, tt := range tests {
		got := CookieToHex(tt.cookie)
		if got != tt.want {
			t.Errorf("CookieToHex(%v) = %s, want %s", tt.cookie, got, tt.want)
		}
	}
}

// TestHexToCookie 测试十六进制转Cookie
func TestHexToCookie(t *testing.T) {
	tests := []struct {
		hexStr  string
		want    []byte
		wantErr bool
	}{
		{"01020304", []byte{0x01, 0x02, 0x03, 0x04}, false},
		{"FFFEFDFC", []byte{0xFF, 0xFE, 0xFD, 0xFC}, false},
		{"", nil, true},
		{"invalid", nil, true},
		{"010", nil, true}, // 奇数长度
	}

	for _, tt := range tests {
		got, err := HexToCookie(tt.hexStr)
		if (err != nil) != tt.wantErr {
			t.Errorf("HexToCookie(%s) error = %v, wantErr %v", tt.hexStr, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && !bytesEqual(got, tt.want) {
			t.Errorf("HexToCookie(%s) = %v, want %v", tt.hexStr, got, tt.want)
		}
	}
}

// TestValidateCookieData 测试验证Cookie数据
func TestValidateCookieData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Valid client only",
			data:    []byte{1, 2, 3, 4, 5, 6, 7, 8},
			wantErr: false,
		},
		{
			name:    "Valid with server cookie",
			data:    append([]byte{1, 2, 3, 4, 5, 6, 7, 8}, make([]byte, 8)...),
			wantErr: false,
		},
		{
			name:    "Nil data",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "Too short",
			data:    []byte{1, 2, 3},
			wantErr: true,
		},
		{
			name:    "Too long",
			data:    make([]byte, 41),
			wantErr: true,
		},
		{
			name:    "Server cookie too short",
			data:    append([]byte{1, 2, 3, 4, 5, 6, 7, 8}, []byte{1, 2, 3, 4, 5, 6, 7}...),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCookieData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCookieData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCreateEchoedCookieResponse 测试创建echoed Client Cookie响应
func TestCreateEchoedCookieResponse(t *testing.T) {
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)

	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	response, err := CreateEchoedCookieResponse(request, clientCookie)
	if err != nil {
		t.Fatalf("CreateEchoedCookieResponse() error = %v", err)
	}

	// 验证是响应消息
	if !response.Response {
		t.Error("Response flag should be set")
	}

	// 验证包含echoed Cookie
	isEchoed, err := IsEchoedCookie(response)
	if err != nil {
		t.Fatalf("IsEchoedCookie() error = %v", err)
	}
	if !isEchoed {
		t.Error("Response should contain echoed cookie")
	}
}

// TestCreateEchoedCookieResponseNilRequest 测试创建echoed Client Cookie响应 - nil请求
func TestCreateEchoedCookieResponseNilRequest(t *testing.T) {
	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	_, err := CreateEchoedCookieResponse(nil, clientCookie)
	if err == nil {
		t.Error("CreateEchoedCookieResponse(nil) should return error")
	}
}

// TestCreateEchoedCookieResponseWrongCookieSize 测试创建echoed Client Cookie响应 - Cookie长度错误
func TestCreateEchoedCookieResponseWrongCookieSize(t *testing.T) {
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)

	wrongCookie := []byte{1, 2, 3, 4, 5, 6, 7} // 7字节
	_, err := CreateEchoedCookieResponse(request, wrongCookie)
	if err == nil {
		t.Error("CreateEchoedCookieResponse() should return error for wrong cookie size")
	}
}

// TestCreateCookieResponse 测试创建完整Cookie响应
func TestCreateCookieResponse(t *testing.T) {
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)

	clientCookie := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	serverCookie := []byte{9, 10, 11, 12, 13, 14, 15, 16}

	response, err := CreateCookieResponse(request, clientCookie, serverCookie)
	if err != nil {
		t.Fatalf("CreateCookieResponse() error = %v", err)
	}

	// 验证是响应消息
	if !response.Response {
		t.Error("Response flag should be set")
	}

	// 验证包含完整Cookie
	if !HasCookieOption(response) {
		t.Error("Response should contain cookie")
	}

	// 验证不是echoed
	isEchoed, _ := IsEchoedCookie(response)
	if isEchoed {
		t.Error("Response should not be echoed cookie")
	}
}

// TestCookieError 测试Cookie错误类型
func TestCookieError(t *testing.T) {
	err := &CookieError{
		Op:  "test",
		Err: "test error",
	}

	expected := "cookie test error: test error"
	if err.Error() != expected {
		t.Errorf("CookieError.Error() = %s, want %s", err.Error(), expected)
	}
}

// TestConstants 测试常量定义
func TestConstants(t *testing.T) {
	if EDNS0CookieOptionCode != 10 {
		t.Errorf("EDNS0CookieOptionCode = %d, want 10", EDNS0CookieOptionCode)
	}

	if ClientCookieSize != 8 {
		t.Errorf("ClientCookieSize = %d, want 8", ClientCookieSize)
	}

	if MinServerCookieSize != 8 {
		t.Errorf("MinServerCookieSize = %d, want 8", MinServerCookieSize)
	}

	if MaxServerCookieSize != 32 {
		t.Errorf("MaxServerCookieSize = %d, want 32", MaxServerCookieSize)
	}

	if MinCookieSize != 8 {
		t.Errorf("MinCookieSize = %d, want 8", MinCookieSize)
	}

	if MaxCookieSize != 40 {
		t.Errorf("MaxCookieSize = %d, want 40", MaxCookieSize)
	}
}
