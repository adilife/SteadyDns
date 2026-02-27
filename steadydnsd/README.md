# SteadyDNS

[![Version](https://img.shields.io/badge/version-0.9.0--beta.1-blue.svg)](https://github.com/your-org/steadydns)
[![License](https://img.shields.io/badge/license-AGPLv3-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)

SteadyDNS 是一个面向企业的 DNS 服务器解决方案，支持多种 DNS 转发策略和 BIND 服务器集成。

> ⚠️ **注意**: 当前版本为 v0.9.0-beta.1 测试版本，不建议直接用于生产环境。

## 功能特性

### 核心功能
- **DNS 转发服务** - 支持多转发组配置，实现默认转发与域名定向转发，组内支持多优先级负载均衡
- **BIND 集成** - 完整的 BIND 服务器管理和 Zone 文件操作
- **RESTful API** - 基于 Gin 框架的 Web API 接口
- **JWT 认证** - 安全的用户认证和授权机制
- **用户管理** - 完整的用户 CRUD 操作

### 高级特性
- **DNS Cookie** - 支持 DNS Cookie 安全扩展
- **健康检查** - DNS 服务器健康状态监控
- **速率限制** - API 请求频率限制保护
- **日志轮转** - 自动日志轮转和归档
- **备份恢复** - BIND 配置备份和恢复功能

## 快速开始

### 环境要求
- Go 1.21 或更高版本
- SQLite3
- BIND9 (可选，用于权威 DNS 功能)

### 编译安装

```bash
# 克隆仓库
git clone https://github.com/your-org/steadydns.git
cd steadydns

# 编译
make build

# 或使用 go build
cd src/cmd && go build -o steadydns main.go
```

### 启动服务

```bash
# 启动服务
./steadydns start

# 停止服务
./steadydns stop

# 重启服务
./steadydns restart

# 查看状态
./steadydns status
```

### 默认配置

- **默认用户**: admin
- **默认密码**: admin123
- **API 端口**: 8080 (可在配置文件中修改)
- **DNS 端口**: 53 (可在配置文件中修改)

### Web UI 访问

启动服务后，可通过浏览器访问 Web 管理界面：

```
http://localhost:8080
```

> **说明**: 前端文件已通过 Go Embed 打包进二进制文件，无需额外部署前端服务。

### 开发模式与生产模式

**生产模式**（默认）：
- 前端文件从 Embed 读取，单二进制部署
- 无需额外配置

**开发模式**：
- 前端文件从文件系统读取，支持热更新
- 设置环境变量：`STEADYDNS_DEV_MODE=true`
- 或使用：`make run-dev`

## API 接口

### 登录认证

```bash
# 登录获取 Token
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### 其他接口

访问 API 需要在请求头中包含 Authorization 字段：

```bash
curl -X GET http://localhost:8080/api/forwardgroups \
  -H "Authorization: Bearer <your-token>"
```

## 项目结构

```
steadydns/
├── VERSION              # 版本文件
├── CHANGELOG.md         # 变更日志
├── Makefile             # 构建脚本
├── LICENSE              # 许可证文件
├── README.md            # 项目说明
├── docs/                # 文档目录
│   └── DEPENDENCIES_LICENSES.md
├── src/                 # 源代码
│   ├── cmd/             # 主程序入口
│   │   ├── main.go
│   │   ├── config/      # 配置文件目录
│   │   ├── log/         # 日志目录
│   │   └── backup/      # 备份目录
│   └── core/            # 核心功能模块
│       ├── common/      # 通用工具
│       ├── database/    # 数据库操作
│       ├── sdns/        # DNS 核心
│       ├── webapi/      # Web API
│       ├── bind/        # BIND 集成
│       └── plugin/      # 插件系统
└── bin/                 # 编译输出目录
```

## 开发指南

### 运行测试

```bash
# 运行所有测试
make test

# 运行测试覆盖率
make test-coverage
```

### 代码规范

```bash
# 格式化代码
make fmt

# 代码检查
make vet
```

## 许可证

本项目采用 GNU Affero General Public License v3.0 (AGPLv3) 许可证进行许可。

详见 [LICENSE](LICENSE) 文件。

## 依赖项许可证

项目使用了多种开源依赖项，所有依赖项的许可证信息详见 [依赖项许可证清单](docs/DEPENDENCIES_LICENSES.md)。

## 反馈与支持

如果您在使用过程中遇到问题或有改进建议，请通过以下方式反馈：

- 提交 Issue
- 发送邮件至技术支持

---

**SteadyDNS** - 企业级 DNS 解决方案
