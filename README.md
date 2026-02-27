# SteadyDNS

[![Version](https://img.shields.io/badge/version-0.9.0--beta.1-blue.svg)](https://github.com/adilife/SteadyDNS)
[![License](https://img.shields.io/badge/license-AGPLv3-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)

A lightweight DNS solution tailored for small to medium-sized environments. Built with Golang, it boasts extreme concurrent processing capabilities and minimal resource consumption.

[![Language](https://img.shields.io/badge/Language-Golang%20%7C%20JavaScript-blue)](https://github.com/adilife/SteadyDns)

专为中小型环境设计的轻量级、高性能 DNS 解决方案，兼顾易用性和稳定性。

## 项目简介

SteadyDNS 由两个核心子项目组成，采用前后端分离架构：

| 子项目 | 技术栈 | 说明 |
|--------|--------|------|
| [steadydnsd](./steadydnsd) | Go | DNS 服务端核心，负责 DNS 请求解析、智能上游转发、配置持久化等核心逻辑 |
| [steadydns_ui](./steadydns_ui) | React | Web 管理面板，提供可视化配置、状态监控、规则管理等功能 |

## 核心特性

### 整体特性

- 🚀 **轻量级** - 单二进制文件部署，无额外依赖
- ⚡ **高性能** - 基于 Go 原生并发模型，支持每秒万级 DNS 请求处理
- 🎨 **可视化管理** - Web 面板一键配置，无需修改配置文件
- 🔄 **智能转发** - 支持多上游 DNS 服务器配置，按优先级/可用性自动切换
- ⚡ **实时生效** - 配置修改即时生效，无需重启 DNS 服务
- 📊 **状态监控** - 实时查看 DNS 请求量、响应耗时、上游可用性等指标
- 🛡️ **稳定性保障** - 自动屏蔽不可用的上游 DNS，避免解析失败

### 后端特性 (steadydnsd)

- 支持 A/AAAA/CNAME/MX 等主流 DNS 记录类型解析
- 自定义本地权威区域（基于 BIND9.18+ 服务）
- 支持 TCP/UDP 协议，兼容 IPv4/IPv6
- 配置文件自动备份与恢复
- 日志记录与审计功能
- RESTful API 接口
- JWT 认证

### 前端特性 (steadydns_ui)

- 简洁易用的操作界面
- 上游 DNS 服务器管理（添加/删除/优先级调整）
- 本地解析规则可视化配置
- DNS 服务状态实时监控面板

## 快速开始

### 环境要求

- 操作系统：Linux
- 架构支持：x86_64
- 必须组件：BIND9.18+
- 端口要求：53（DNS 服务）、8080（Web 面板，可自定义）

### 安装部署

```bash
# 1. 克隆仓库
git clone https://github.com/adilife/SteadyDNS.git
cd SteadyDNS

# 2. 构建后端（包含前端）
cd steadydnsd
make build-full

# 3. 启动服务
cd src/cmd
./steadydns start

# 4. 访问 Web 管理界面
# http://localhost:8080
# 默认用户名: admin
# 默认密码: admin123
```

## 项目结构

```
SteadyDNS/
├── README.md                 # 项目总览（本文件）
├── CHANGELOG.md              # 变更日志
├── LICENSE                   # 许可证 (AGPLv3)
│
├── steadydnsd/               # 后端项目
│   ├── src/                  # 源代码
│   ├── docs/                 # 文档
│   ├── Makefile              # 构建脚本
│   └── README.md             # 后端详细文档
│
└── steadydns_ui/             # 前端项目
    ├── src/                  # 源代码
    ├── public/               # 静态资源
    ├── package.json          # 依赖配置
    └── README.md             # 前端详细文档
```

## 开发指南

### 后端开发

```bash
cd steadydnsd
make help          # 查看可用命令
make build         # 编译
make test          # 运行测试
make run-dev       # 开发模式运行
```

详见 [steadydnsd/README.md](./steadydnsd/README.md)

### 前端开发

```bash
cd steadydns_ui
npm install        # 安装依赖
npm run dev        # 开发模式
npm run build      # 构建生产版本
```

详见 [steadydns_ui/README.md](./steadydns_ui/README.md)

## 许可证

本项目采用 GNU Affero General Public License v3.0 (AGPLv3) 许可证进行许可。

详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交 Issue 和 Pull Request。

## 联系方式

- GitHub: https://github.com/adilife/SteadyDNS
