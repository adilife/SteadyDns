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

> ⚠️ **重要提示**：如果在本机部署 BIND，必须修改 BIND 的默认监听端口（53），因为 SteadyDNS 需要监听 53 端口对外提供 DNS 服务。BIND 将作为后端权威服务器运行在其他端口（如 5300），由 SteadyDNS 转发权威域查询请求。

**步骤 1：在配置文件中启用 BIND 插件**

编辑 `config/steadydns.conf` 文件，在 `[Plugins]` 节中设置：

```ini
[Plugins]
# BIND Plugin - Authoritative Domain Management
# 启用 BIND 插件以支持权威域管理功能
# 修改后需要重启服务才能生效
BIND_ENABLED=true
```

> **注意**：修改插件配置后需要重启 SteadyDNS 服务才能生效。

**步骤 2：安装 BIND 9.18+**
```bash
# CentOS/RHEL
yum install bind bind-utils

# Ubuntu/Debian
apt install bind9 bind9utils
```

**步骤 3：验证 BIND 版本**
```bash
named -v
# 输出应类似：BIND 9.18.x
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
    listen-on port 5300 { 127.0.0.1; };
    // 其他配置...
};
```

重启 BIND 服务：
```bash
systemctl restart named
```

**步骤 6：配置 RNDC**
```bash
# 生成 RNDC 密钥（如果尚未配置）
rndc-confgen -a

# 确保 SteadyDNS 有权限访问 BIND 配置目录
chmod -R 755 /var/named
```

**步骤 7：重启 SteadyDNS 服务**
```bash
./steadydns restart
# 或使用 systemd
systemctl restart steadydns
```

#### 不启用 BIND 插件

如果不需要权威域管理功能，可以在配置文件中禁用 BIND 插件：

```ini
[Plugins]
# 禁用 BIND 插件
BIND_ENABLED=false
```

禁用 BIND 插件后，SteadyDNS 将作为纯转发服务器运行：

- 仅提供 DNS 递归转发功能
- 支持上游 DNS 服务器配置
- 支持域名过滤和黑白名单
- 不支持权威域管理
- 无需安装 BIND 服务

> **注意**：修改插件配置后需要重启 SteadyDNS 服务才能生效。

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

# 3. 进入目录
cd steadydns-0.9.0-beta.1-linux-amd64

# 4. 启动服务
./steadydns start
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
mkdir -p /opt/steadydns
cp src/cmd/steadydns /opt/steadydns
```

### 方式三：使用 Makefile 安装

```bash
cd steadydnsd
make install  # 安装依赖
make build    # 编译
```

## Systemd 服务安装

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

### 配置文件

> **提示**: 配置文件 `config/steadydns.conf` 会在首次启动时自动创建。如需自定义配置，可在启动前手动创建。

首次启动服务时，系统会自动检测配置文件是否存在：

- **配置文件不存在**：自动创建默认配置文件 `config/steadydns.conf`
- **配置文件已存在**：直接加载现有配置

#### 主要配置项说明

| 配置节 | 主要配置项 | 说明 |
|--------|-----------|------|
| `[Database]` | `DB_PATH` | 数据库文件路径 |
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
./steadydns --version

# 查看帮助
./steadydns --help
```

### Systemd 服务（推荐）

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
cp steadydns /opt/steadydns

# 4. 启动服务
./steadydns start
# 或
systemctl start steadydns

# 5. 验证版本
./steadydns --version
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
