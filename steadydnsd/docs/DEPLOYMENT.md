# SteadyDNS 部署说明

本文档提供 SteadyDNS 的详细部署指南，包括环境要求、安装步骤、配置说明和常见问题。

## 目录

- [环境要求](#环境要求)
- [安装方式](#安装方式)
- [配置说明](#配置说明)
- [启动与管理](#启动与管理)
- [升级指南](#升级指南)
- [常见问题](#常见问题)

---

## 环境要求

### 操作系统

| 系统 | 版本 | 架构 |
|------|------|------|
| Linux | CentOS 7+, Ubuntu 18.04+, Debian 10+ | x86_64, arm64 |

### 必须组件

| 组件 | 版本要求 | 说明 |
|------|----------|------|
| BIND | 9.18+ | DNS 权威服务器 |
| SQLite | 3.x | 数据库（内嵌，无需单独安装） |

### 端口要求

| 端口 | 协议 | 说明 |
|------|------|------|
| 53 | UDP/TCP | DNS 服务端口 |
| 8080 | TCP | Web 管理面板（可自定义） |

### 硬件要求

| 配置项 | 最低要求 | 推荐配置 |
|--------|----------|----------|
| CPU | 1 核 | 2 核+ |
| 内存 | 512 MB | 1 GB+ |
| 磁盘 | 100 MB | 1 GB+ |

---

## 安装方式

### 方式一：下载预编译二进制文件

```bash
# 1. 下载最新版本
wget https://github.com/adilife/SteadyDNS/releases/download/v0.9.0-beta.1/steadydns-linux-amd64.tar.gz

# 2. 解压
tar -xzf steadydns-linux-amd64.tar.gz

# 3. 移动到目标目录
mv steadydns /usr/local/bin/

# 4. 创建工作目录
mkdir -p /etc/steadydns
cd /etc/steadydns
```

### 方式二：从源码编译

```bash
# 1. 克隆仓库
git clone https://github.com/adilife/SteadyDNS.git
cd SteadyDNS

# 2. 构建前端（可选，已包含在完整构建中）
cd steadydns_ui
npm install
npm run build

# 3. 完整构建（包含前端）
cd ../steadydnsd
make build-full

# 4. 安装
cp src/cmd/steadydns /usr/local/bin/
mkdir -p /etc/steadydns
```

### 方式三：使用 Makefile 安装

```bash
cd steadydnsd
make install  # 安装依赖
make build    # 编译
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

### 配置文件示例

创建配置文件 `config/steadydns.conf`：

```ini
# SteadyDNS Configuration File
# Format: INI/Conf

[Database]
# Database file path (relative to working directory)
DB_PATH=steadydns.db

[APIServer]
# API Server port
API_SERVER_PORT=8080
# API Server IPv4 address
API_SERVER_IP_ADDR=0.0.0.0
# API Server IPv6 address
API_SERVER_IPV6_ADDR=::
# GIN running mode (debug/release)
GIN_MODE=release

[JWT]
# JWT secret key for authentication
JWT_SECRET_KEY=your-strong-secret-key-change-this
# Access token expiration (minutes)
ACCESS_TOKEN_EXPIRATION=300
# Refresh token expiration (days)
REFRESH_TOKEN_EXPIRATION=7
# JWT algorithm
JWT_ALGORITHM=HS256

[API]
# API rate limit enabled
RATE_LIMIT_ENABLED=true
# General API limit (requests per minute)
RATE_LIMIT_API=300
# Login API limit (requests per minute)
RATE_LIMIT_LOGIN=60
# API log enabled
LOG_ENABLED=true
# API log level
LOG_LEVEL=INFO

[BIND]
# BIND server address
BIND_ADDRESS=127.0.0.1:5300
# RNDC key file path
RNDC_KEY=/etc/named/rndc.key
# Zone file storage path
ZONE_FILE_PATH=/usr/local/bind9/var/named
# Named configuration path
NAMED_CONF_PATH=/etc/named
# RNDC port
RNDC_PORT=9530
# BIND user
BIND_USER=named
# BIND group
BIND_GROUP=named
# BIND start command
BIND_EXEC_START=/usr/local/bind9/sbin/named -u named
# named-checkconf executable path
BIND_CHECKCONF_PATH=/usr/local/bind9/bin/named-checkconf
# named-checkzone executable path
BIND_CHECKZONE_PATH=/usr/local/bind9/bin/named-checkzone

[DNS]
# Client processing worker pool size
DNS_CLIENT_WORKERS=10000
# Task queue multiplier
DNS_QUEUE_MULTIPLIER=2
# DNS server priority timeout (milliseconds)
DNS_PRIORITY_TIMEOUT_MS=50

[Cache]
# Cache size limit (MB)
DNS_CACHE_SIZE_MB=100
# Cache cleanup interval (seconds)
DNS_CACHE_CLEANUP_INTERVAL=60
# Error cache TTL (seconds)
DNS_CACHE_ERROR_TTL=3600

[Logging]
# Query log storage path (relative to working directory)
QUERY_LOG_PATH=log/
# Query log file size limit (MB)
QUERY_LOG_MAX_SIZE=10
# Query log file count limit
QUERY_LOG_MAX_FILES=10
# Log level
DNS_LOG_LEVEL=INFO

[Security]
# DNS query rate limit per IP (queries per minute)
DNS_RATE_LIMIT_PER_IP=50000
# Global DNS query rate limit (queries per minute)
DNS_RATE_LIMIT_GLOBAL=1000000
# DNS query ban duration (minutes)
DNS_BAN_DURATION=5
# DNS message size limit (bytes)
DNS_MESSAGE_SIZE_LIMIT=4096
# DNS query validation enabled
DNS_VALIDATION_ENABLED=true

[Plugins]
# BIND Plugin - Authoritative Domain Management
BIND_ENABLED=true
# DNS Rules Plugin (Reserved)
DNS_RULES_ENABLED=false
# Log Analysis Plugin (Reserved)
LOG_ANALYSIS_ENABLED=false
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `STEADYDNS_DEV_MODE` | 开发模式（从文件系统读取前端） | `false` |

---

## 启动与管理

### 命令行管理

```bash
# 启动服务
./steadydns start

# 停止服务
./steadydns stop

# 重启服务
./steadydns restart

# 查看状态
./steadydns status

# 查看版本
./steadydns version
```

### Systemd 服务（推荐）

创建服务文件 `/etc/systemd/system/steadydns.service`：

```ini
[Unit]
Description=SteadyDNS Server
After=network.target
Wants=network.target

[Service]
Type=forking
PIDFile=/etc/steadydns/steadydns.pid
WorkingDirectory=/etc/steadydns
ExecStart=/usr/local/bin/steadydns start
ExecStop=/usr/local/bin/steadydns stop
ExecReload=/usr/local/bin/steadydns restart
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

管理命令：

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

# 查看日志
journalctl -u steadydns -f
```

### 访问 Web 管理界面

启动服务后，通过浏览器访问：

```
http://<服务器IP>:8080
```

**默认凭据：**
- 用户名：`admin`
- 密码：`admin123`

> ⚠️ **安全提示**：首次登录后请立即修改默认密码！

---

## 升级指南

### 升级步骤

```bash
# 1. 备份数据
cp /etc/steadydns/steadydns.db /etc/steadydns/backup/steadydns.db.bak

# 2. 停止服务
./steadydns stop
# 或
systemctl stop steadydns

# 3. 替换二进制文件
cp steadydns /usr/local/bin/steadydns

# 4. 启动服务
./steadydns start
# 或
systemctl start steadydns

# 5. 验证版本
./steadydns version
```

### 数据库迁移

SteadyDNS 使用 SQLite 数据库，升级时会自动迁移表结构。如遇问题，请检查日志。

---

## 常见问题

### 1. 端口被占用

**问题**：启动时报错 `bind: address already in use`

**解决方案**：
```bash
# 检查端口占用
netstat -tlnp | grep :53
netstat -tlnp | grep :8080

# 停止占用端口的进程，或修改配置文件中的端口
```

### 2. 权限不足

**问题**：启动时报错 `permission denied`

**解决方案**：
```bash
# 使用 root 用户运行，或授予相应权限
sudo ./steadydns start

# 或使用 systemd 服务管理
systemctl start steadydns
```

### 3. BIND 配置错误

**问题**：BIND 相关功能无法使用

**解决方案**：
```bash
# 检查 BIND 是否安装
named -v

# 检查配置文件
named-checkconf /etc/named.conf

# 确保 SteadyDNS 有权限访问 BIND 配置目录
chmod -R 755 /var/named
```

### 4. 数据库锁定

**问题**：操作时报错 `database is locked`

**解决方案**：
```bash
# 停止服务
./steadydns stop

# 检查是否有其他进程访问数据库
lsof /etc/steadydns/steadydns.db

# 重启服务
./steadydns start
```

### 5. 忘记密码

**解决方案**：
```bash
# 重置管理员密码
sqlite3 /etc/steadydns/steadydns.db "UPDATE users SET password='$2a$12$新的bcrypt哈希值' WHERE username='admin';"
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
