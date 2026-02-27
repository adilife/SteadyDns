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
