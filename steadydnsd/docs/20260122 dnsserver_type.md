# BIND9 权威区域服务器记录类型速查表

## 一、核心解析记录

| 记录类型 | 全称 | 主要用途 | 示例 | 重要限制 |
|---------|------|----------|------|----------|
| **A** | Address | IPv4地址解析 | `www IN A 192.0.2.1` | 最基础的IP映射记录 |
| **AAAA** | IPv6 Address | IPv6地址解析 | `www IN AAAA 2001:db8::1` | 双栈支持必须配置 |
| **CNAME** | Canonical Name | 别名指向规范名 | `www IN CNAME example.com.` | 不能与其他记录共存 |
| **NS** | Name Server | 区域授权服务器 | `@ IN NS ns1.example.com.` | 必须指向A/AAAA记录 |
| **SOA** | Start of Authority | 区域权威声明 | `@ IN SOA ns1 admin (2024010101 3600...)` | 每个区域必须有且只有一个 |

## 二、邮件服务记录

| 记录类型 | 全称 | 主要用途 | 示例 | 重要说明 |
|---------|------|----------|------|----------|
| **MX** | Mail Exchange | 邮件服务器路由 | `@ IN MX 10 mail.example.com.` | 优先级决定投递顺序 |
| **TXT** | Text | 文本信息（SPF/DKIM/DMARC） | `@ IN TXT "v=spf1 mx -all"` | 邮件认证必备 |
| **PTR** | Pointer | 反向DNS解析 | `1.2.0.192.in-addr.arpa. IN PTR mail.example.com.` | 邮件服务器必须配置 |

## 三、服务发现记录

| 记录类型 | 全称 | 主要用途 | 示例 | 格式说明 |
|---------|------|----------|------|----------|
| **SRV** | Service | 服务定位器 | `_xmpp._tcp IN SRV 10 0 5222 xmpp.example.com.` | `_service._proto.name priority weight port target` |
| **NAPTR** | Naming Authority Pointer | 协议重写/ENUM | `!^.*$!sip:info@example.com! IN NAPTR 100 10 "u" "E2U+sip" ""` | 复杂服务发现 |

## 四、安全认证记录

| 记录类型 | 全称 | 主要用途 | 示例 | 安全作用 |
|---------|------|----------|------|----------|
| **CAA** | Certificate Authority Authorization | SSL证书授权 | `@ IN CAA 0 issue "letsencrypt.org"` | 控制CA颁发证书 |
| **SSHFP** | SSH Fingerprint | SSH公钥指纹 | `server IN SSHFP 1 1 1234567890abcdef` | SSH主机认证 |
| **TLSA** | TLS Authentication | TLS证书绑定 | `_443._tcp.www IN TLSA 3 1 1 a1b2c3...` | DANE协议实现 |
| **DNSKEY** | DNSSEC Key | DNSSEC公钥 | `@ IN DNSKEY 256 3 13 (AwEAAc...)` | 区域签名公钥 |
| **DS** | Delegation Signer | 委托签名 | `example.com. IN DS 12345 13 2 ABCDEF...` | 父区域认证子区域 |
| **RRSIG** | Resource Record Signature | 记录签名 | `www IN RRSIG A 13 3 3600 (...) ABCDEF...` | 记录集数字签名 |
| **NSEC** | Next Secure | DNSSEC否定存在 | `example.com. IN NSEC next.example.com. A NS SOA` | 证明名称不存在 |
| **NSEC3** | Next Secure v3 | DNSSEC否定存在（安全增强） | `example.com. IN NSEC3 1 0 10 ABCDEF...` | 防止区域遍历 |

## 五、特殊用途记录

| 记录类型 | 全称 | 主要用途 | 示例 | 使用场景 |
|---------|------|----------|------|----------|
| **DNAME** | Delegation Name | 子树重定向 | `old.com. IN DNAME new.com.` | 批量域名重定向 |
| **URI** | Uniform Resource Identifier | URI映射 | `_http._tcp.www IN URI 10 1 "https://www.example.com/"` | 服务重定向 |
| **RP** | Responsible Person | 责任人信息 | `@ IN RP admin.example.com. helpdesk.example.com.` | 联系信息（过时） |
| **HINFO** | Hardware Information | 硬件信息 | `server IN HINFO "Xeon" "Linux"` | 硬件描述（不安全） |
| **AFSDB** | AFS Database | AFS数据库定位 | `@ IN AFSDB 1 afsdb.example.com.` | Andrew文件系统 |
| **LOC** | Location | 地理位置 | `server IN LOC 37 23 30.900 N 122 02 48.600 W` | 物理位置坐标 |

## 六、记录类型快速选择指南

| 需求场景 | 首选记录 | 备选记录 | 特殊要求 |
|----------|----------|----------|----------|
| **网站访问** | A/AAAA | CNAME | 裸域名避免CNAME |
| **邮件收发** | MX + A/AAAA | - | 必须配置反向PTR |
| **负载均衡** | 多个A/AAAA | SRV | TTL设置较短 |
| **子域委派** | NS | - | 需要父子区域协调 |
| **邮件安全** | TXT (SPF/DKIM/DMARC) | - | 必须正确配置 |
| **服务发现** | SRV | NAPTR | 客户端需支持 |
| **SSL控制** | CAA | - | 防止错误颁发 |
| **安全解析** | DNSSEC全套 | - | 需要完整链 |
| **IPv6部署** | AAAA | - | 双栈配置 |
| **别名服务** | CNAME | DNAME | 注意CNAME限制 |

## 七、记录共存规则表

| 记录类型 | 能否与A共存 | 能否与CNAME共存 | 能否与MX共存 | 能否与NS共存 |
|----------|-------------|----------------|-------------|-------------|
| **A** | - | ❌ | ✓ | ✓ |
| **AAAA** | ✓ | ❌ | ✓ | ✓ |
| **CNAME** | ❌ | - | ❌ | ❌ |
| **MX** | ✓ | ❌ | - | ✓ |
| **NS** | ✓ | ❌ | ✓ | - |
| **TXT** | ✓ | ❌ | ✓ | ✓ |
| **SOA** | ✓ | ❌ | ✓ | ❌ |
| **SRV** | ✓ | ❌ | ✓ | ✓ |

**键**: ✓ = 可以共存, ❌ = 不能共存, - = 相同类型比较

## 八、TTL建议值参考

| 记录类型 | 生产环境TTL | 测试环境TTL | 说明 |
|----------|------------|------------|------|
| **A/AAAA（稳定）** | 3600-86400 | 300 | 1小时-1天 |
| **A/AAAA（动态）** | 300-900 | 60 | 5-15分钟 |
| **MX/TXT** | 86400 | 3600 | 较少变更 |
| **NS/SOA** | 86400-604800 | 3600 | 极少变更 |
| **CNAME** | 3600 | 300 | 同目标记录 |
| **DNSSEC相关** | 172800-604800 | 86400 | 较长缓存 |

## 九、重要注意事项总结

1. **NS记录**必须指向A/AAAA记录，不能是CNAME
2. **MX记录**必须指向A/AAAA记录，不能是CNAME
3. **SOA记录**每个区域必须且只能有一个
4. **CNAME记录**不能与其他任何记录共存
5. **PTR记录**在反向区域中配置，对邮件服务器至关重要
6. **DNSSEC记录**需要完整链条：DNSKEY + RRSIG + NSEC/NSEC3 + DS
7. **TXT记录**用于多种协议，注意字符串长度限制（255字节分段）

此表格提供了BIND9权威区域中主要服务器记录类型的快速参考，实际配置时应根据具体需求和DNS标准进行适当调整。