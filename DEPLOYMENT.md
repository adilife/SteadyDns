# SteadyDNS 部署说明

本文档提供 SteadyDNS 的详细部署指南，包括环境要求、安装步骤、配置说明和常见问题。

## 目录

- [环境要求](#环境要求)
  - [操作系统](#操作系统)
  - [第三方组件](#第三方组件)
  - [BIND 插件说明](#bind-插件说明)
  - [端口要求](#端口要求)
  - [硬件要求](#硬件要求)
- [安装方式](#安装方式)
  - [方式一：预编译二进制包（推荐）](#方式一预编译二进制包推荐)
  - [方式二：源码编译（开发 / 定制场景）](#方式二源码编译开发定制场景)
  - [方式三：Makefile 快速安装](#方式三makefile快速安装)
  - [Systemd 服务注册（生产推荐）](#systemd服务注册生产推荐)
- [配置说明](#配置说明)
  - [目录结构](#目录结构)
  - [配置文件管理](#配置文件管理)
  - [环境变量](#环境变量)
- [启动与管理](#启动与管理)
  - [命令行管理](#命令行管理)
  - [Systemd 服务（推荐）](#Systemd服务推荐)
  - [Web 管理面板访问](#Web管理面板访问)
- [升级指南](#升级指南)
  - [升级步骤](#升级步骤)
  - [数据库迁移](#数据库迁移)
- [常见问题](#常见问题)

---

## 环境要求

### 操作系统

| 系统 | 版本 | 架构 |
|------|------|------|
| Linux | CentOS 7+, Ubuntu 18.04+, Debian 10+ | x86_64, arm64 |

### 第三方组件

| 组件 | 版本要求 | 说明 |
|------|----------|------|
| BIND | 9.18+ | DNS 权威服务器（BIND 插件启用时需要） |
| SQLite | 3.x | 数据库（内嵌，无需单独安装） |

### BIND 插件说明

SteadyDNS 支持以两种模式运行，取决于是否启用 BIND 插件：

| 模式 | BIND 插件 | 功能说明 |
|------|----------|----------|
| **权威服务器模式** | 启用 | 支持权威域管理，可作为主/从 DNS 服务器 |
| **转发服务器模式** | 禁用 | 仅提供 DNS 转发功能，无权威域管理 |

#### 启用 BIND 插件

如需启用权威域管理功能，请确保：

> ⚠️ **重要提示**：SteadyDNS 需占用 53 端口提供 DNS 服务，本机部署 BIND必须修改默认监听端口（推荐 5300）。

**步骤 1：在配置文件中启用 BIND 插件**

编辑 `config/steadydns.conf` 文件，在 `[Plugins]` 节中设置：

```ini
[Plugins]
# BIND Plugin - Authoritative Domain Management
# 启用 BIND 插件以支持权威域管理功能
# 修改后需要重启服务才能生效
BIND_ENABLED=true
```

**步骤 2：安装 BIND 9.18+**
```bash
# CentOS/RHEL
yum install bind bind-utils

# Ubuntu/Debian
apt install bind9 bind9utils
```

**步骤 3：验证 BIND 版本**
```bash
named -v # 预期输出：BIND 9.18.x
```

**步骤 4：配置 BIND 相关参数**

在 `config/steadydns.conf` 中配置 BIND 相关选项：
```ini
[BIND]
# BIND 服务地址（注意：不是默认的 53 端口）
BIND_ADDRESS=127.0.0.1:5300
# RNDC 密钥文件路径
RNDC_KEY=/etc/named/rndc.key
# 区域文件存储路径
ZONE_FILE_PATH=/usr/local/bind9/var/named
# named 配置文件路径
NAMED_CONF_PATH=/etc/named
# RNDC 端口
RNDC_PORT=9530
```

**步骤 5：修改 BIND 监听端口**

编辑 BIND 配置文件 `/etc/named.conf`，修改监听端口：
```bash
# 修改 options 部分
options {
    listen-on port 5300 { 127.0.0.1; };  # 仅本机监听，端口改为 5300
    allow-query { 127.0.0.1; };          # 仅允许本机访问
    // 其他配置...
};
```

**步骤 6：重启 BIND 服务：**
```bash
# 重启BIND服务
systemctl restart named
# 验证监听端口
ss -tulpn | grep named  # 应显示 127.0.0.1:5300
```

**步骤 7：配置 RNDC**
```bash
# 生成 RNDC 密钥（如果尚未配置）
rndc-confgen -a
# 确保 SteadyDNS 有权限访问 BIND 配置目录
chmod -R 755 /usr/local/bind9/var/named
```

**步骤 8：重启 SteadyDNS 服务**
```bash
./steadydns restart
# 或使用 systemd
systemctl restart steadydns
```

#### 禁用 BIND 插件（纯转发模式）

仅需修改配置文件禁用插件，无需安装 BIND：

```ini
[Plugins]
# 禁用权威域管理，仅保留转发功能
BIND_ENABLED=false
```

> **✨ 禁用后支持**：DNS 递归转发、上游 DNS 配置、域名过滤 / 黑白名单；不支持：权威域管理。
> 
> **⚠️注意**：修改插件配置后需要重启 SteadyDNS 服务才能生效。

### 端口要求

| 端口 | 协议 | 说明 |安全建议|
|------|------|------|------|
| 53 | UDP/TCP | DNS 服务端口 | 仅向业务网段开放 |
| 8080 | TCP | Web 管理面板（可自定义） | 仅向运维网段开放（可自定义） |

### 硬件要求

| 配置项 | 最低要求 | 推荐配置 |
|--------|----------|----------|
| CPU | 1 核 | 2 核+ |
| 内存 | 512 MB | 1 GB+ |
| 磁盘 | 100 MB | 1 GB+ |

---

## 安装方式

### 方式一：预编译二进制包（推荐）
适合生产环境，无需编译依赖，快速部署：

```bash
# 1. 下载最新版本（替换为实际版本号）
VERSION="v0.9.0-beta.1"
wget https://github.com/adilife/SteadyDNS/releases/download/${VERSION}/steadydns-linux-amd64.tar.gz -O steadydns.tar.gz

# 2. 解压到指定目录（标准化路径）
mkdir -p /opt/steadydns
tar -xzf steadydns.tar.gz -C /opt/steadydns --strip-components=1

# 3. 启动服务
/opt/steadydns/steadydns start
```

### 方式二：源码编译（开发 / 定制场景）
适合需要修改源码、自定义构建的场景：

```bash
# 1. 安装编译依赖
# CentOS/RHEL
yum install -y git golang npm make gcc
# Ubuntu/Debian
apt install -y git golang npm make gcc

# 2. 克隆仓库
git clone https://github.com/adilife/SteadyDNS.git
cd SteadyDNS

# 3. 构建前端（如需自定义前端）
cd steadydns_ui
npm install && npm run build
cd ../

# 4. 编译后端（完整构建含前端）
cd steadydnsd
make build-full

# 5. 部署到标准化目录
mkdir -p /opt/steadydns
cp src/cmd/steadydns /opt/steadydns/
chmod +x /opt/steadydns/steadydns
```

### 方式三：Makefile 快速安装
适合熟悉 Makefile、需简化编译流程的场景：

```bash
cd SteadyDNS/steadydnsd
make install  # 自动安装编译依赖
make build    # 编译二进制文件
cp src/cmd/steadydns /opt/steadydns/  # 部署到目标目录
```

## Systemd 服务注册（生产推荐）
通过 Systemd 管理服务，支持开机自启、进程守护：

```bash
# 复制二进制文件
sudo cp steadydns /opt/steadydns/

# 安装服务
sudo cp scripts/steadydns.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable steadydns
sudo systemctl start steadydns
```
---

## 配置说明

### 目录结构

```
/etc/steadydns/
├── steadydns              # 可执行文件
├── config/
│   └── steadydns.conf     # 主配置文件
├── log/                   # 日志目录
├── backup/                # 备份目录
└── steadydns.db           # SQLite 数据库（自动创建）
```

### 配置文件管理

> **提示**: 启动服务时，若 config/steadydns.conf 不存在，自动生成默认配置。

  - 手动定制：生产环境建议先创建配置文件，再启动服务（避免默认配置风险）；
  - 配置备份：手工修改配置前，建议备份 config/steadydns.conf；
  - 管理界面：可通过管理Web页面对配置文件内容进行管理。

#### 主要配置项说明

| 配置节 | 主要配置项 | 说明 |
|--------|-----------|------|
| `[Database]` | `DB_PATH` | 数据库文件路径（建议保持默认） |
| `[APIServer]` | `API_SERVER_PORT` | API 服务端口（默认 8080） |
| `[JWT]` | `JWT_SECRET_KEY` | JWT 密钥（**生产环境必须修改**） |
| `[BIND]` | `BIND_ADDRESS` | BIND 服务器地址 |
| `[DNS]` | `DNS_CLIENT_WORKERS` | DNS 客户端工作池大小 |
| `[Security]` | `DNS_RATE_LIMIT_PER_IP` | 单 IP 请求限制 |

#### 生产环境重要配置

```ini
[JWT]
# 生产环境必须修改为强密钥
JWT_SECRET_KEY=your-strong-secret-key-change-this

[APIServer]
# 生产环境建议设置为 release
GIN_MODE=release

[Security]
# 根据实际需求调整限流参数
DNS_RATE_LIMIT_PER_IP=300
DNS_RATE_LIMIT_GLOBAL=10000
```

完整配置项说明请参考自动生成的配置文件中的注释。

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `STEADYDNS_DEV_MODE` | 开发模式（从文件系统读取前端） | `false` |

---

## 启动与管理

### 命令行管理
适用于测试环境、临时操作：

```bash
# 进入程序目录
cd /opt/steadydns

# 启动服务
./steadydns start

# 停止服务
./steadydns stop

# 重启服务
./steadydns restart

# 查看状态
./steadydns status

# 查看版本（验证部署版本）
./steadydns --version

# 查看帮助
./steadydns --help
```

### Systemd 服务（推荐）
适用于生产环境，支持进程守护、日志集中管理：

```bash
# 重载 systemd 配置
systemctl daemon-reload

# 启动服务
systemctl start steadydns

# 停止服务
systemctl stop steadydns

# 重启服务
systemctl restart steadydns

# 查看状态
systemctl status steadydns

# 设置开机自启
systemctl enable steadydns

# 取消开机自启
systemctl disable steadydns

# 查看日志（实时跟踪）
journalctl -u steadydns -f --no-pager

# 查看历史日志（近 1 小时）
journalctl -u steadydns --since "1 hour ago"
```

### Web 管理面板访问

1. 确认服务正常运行（systemctl status steadydns）；
2. 浏览器访问：http://<服务器IP>:<API_SERVER_PORT>（如 http://192.168.1.100:8080）
3. 首次登录：
   - 默认用户名：`admin`
   - 默认密码：`admin123`
     > ⚠️ **安全提示**：首次登录后请立即修改默认密码！

---

## 升级指南

### 升级步骤

```bash
# 1. 备份核心数据（关键步骤，避免数据丢失）
mkdir -p /opt/steadydns/backup/$(date +%Y%m%d)
cp /opt/steadydns/steadydns.db /opt/steadydns/backup/$(date +%Y%m%d)/
cp /opt/steadydns/config/steadydns.conf /opt/steadydns/backup/$(date +%Y%m%d)/

# 2. 停止服务（生产环境建议先灰度/暂停流量）
systemctl stop steadydns

# 3. 下载并替换二进制文件（替换为新版本）
VERSION="v0.9.0-beta.2"
wget https://github.com/adilife/SteadyDNS/releases/download/${VERSION}/steadydns-linux-amd64.tar.gz -O new-steadydns.tar.gz
tar -xzf new-steadydns.tar.gz -C /tmp/
cp /tmp/steadydns-*/steadydns /opt/steadydns/

# 4. 启动服务并验证
systemctl start steadydns
sleep 5  # 等待服务启动
systemctl status steadydns -l

# 5. 验证版本和功能
/opt/steadydns/steadydns --version
# 验证 DNS 功能（示例：解析本机域名）
dig @127.0.0.1 www.baidu.com -p 53
```

### 数据库迁移

- SteadyDNS 升级时会自动检测数据库版本，完成表结构迁移；

- 若迁移失败：
  1. 查看日志：journalctl -u steadydns --since "5 minutes ago"；
  2. 恢复备份数据库：cp /opt/steadydns/backup/$(date +%Y%m%d)/steadydns.db /opt/steadydns/；
  3. 回滚二进制文件到旧版本，联系技术支持。

---

## 常见问题

### 1. 端口被占用（启动失败）

**问题**：日志报错 bind: address already in use，服务无法启动。

**解决方案**：
```bash
# 检查端口占用
netstat -tlnp | grep :53
netstat -tlnp | grep :8080

# 停止占用端口的进程，或修改配置文件中的端口
```

### 2. 权限不足（操作报错）

**问题**：启动时报错 `permission denied`

**解决方案**：
```bash
# 使用 root 用户运行，或授予相应权限
sudo ./steadydns start

# 或使用 systemd 服务管理
systemctl start steadydns
```

### 3. BIND 配置错误（权威模式）

**问题**：BIND 相关功能无法使用

**解决方案**：
```bash
# 1. 检查 BIND 是否安装
named -v

# 2. 验证 BIND 配置
named-checkconf /etc/named.conf

# 3. 验证 BIND 监听端口
ss -tulpn | grep 5300

# 4. 验证 RNDC 连接
rndc -p 9530 status

# 5. 确保 SteadyDNS 可访问 RNDC 密钥
chmod 644 /etc/named/rndc.key
```

### 4. 数据库锁定（操作超时）

**问题**：操作时报错 `database is locked`，DNS 查询 / 管理操作超时。

**解决方案**：
```bash
# 1. 停止服务
systemctl stop steadydns

# 2. 查找占用数据库的进程
lsof /opt/steadydns/steadydns.db

# 3. 杀死异常进程（替换为实际 PID）
kill -9 <PID>

# 4. 重启服务
systemctl start steadydns
```

### 5. 忘记管理员密码

**解决方案**：
```bash
# 1. 安装 sqlite3（若未安装）
yum install -y sqlite3  # CentOS
apt install -y sqlite3  # Ubuntu

# 2. 生成 bcrypt 哈希密码（示例：新密码为 newAdmin@123）
# 可通过在线工具/代码生成，或使用以下命令（需安装 python）
python3 -c "import bcrypt; print(bcrypt.hashpw('newAdmin@123'.encode(), bcrypt.gensalt(rounds=12)).decode())"

# 3. 重置密码（替换为生成的哈希值）
sqlite3 /opt/steadydns/steadydns.db "UPDATE users SET password='$2b$12$xxxxxx' WHERE username='admin';"

# 4. 重启服务（可选）
systemctl restart steadydns
```

---

## 技术支持

- **GitHub Issues**: https://github.com/adilife/SteadyDNS/issues
- **文档**: https://github.com/adilife/SteadyDNS/tree/main/docs

---

## 相关文档

- [README.md](./README.md) - 项目概述
- [CHANGELOG.md](./CHANGELOG.md) - 变更日志
- [steadydnsd/README.md](./steadydnsd/README.md) - 后端详细文档
- [steadydns_ui/README.md](./steadydns_ui/README.md) - 前端详细文档
