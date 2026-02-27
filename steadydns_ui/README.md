# SteadyDNS UI

[![Version](https://img.shields.io/badge/version-v0.9.0--beta.1-blue.svg)](https://github.com/adilife/SteadyDns)
[![License](https://img.shields.io/badge/license-AGPLv3-green.svg)](LICENSE)

SteadyDNS UI 是一个基于 React + Vite 开发的 DNS 服务器管理界面，提供直观、易用的 DNS 服务器配置和监控功能，用于管理 SteadyDNS 服务器。

> ⚠️ **注意**: 当前版本为 v0.9.0-beta.1 测试版本，不建议直接用于生产环境。

## 功能特性

### 核心功能
- **用户认证** - JWT Token 认证，自动刷新，会话管理
- **仪表盘** - 系统概览、QPS 趋势、资源监控、热门域名/客户端统计
- **转发器管理** - 转发组配置、优先级管理、域名匹配测试
- **配置管理** - 服务器状态、BIND 管理、缓存管理、配置备份恢复
- **用户管理** - 用户 CRUD、密码修改、权限控制

### 插件功能
- **权威域管理** (BIND 插件) - DNS 区域管理、记录配置、SOA 管理、操作历史
- **DNS 规则配置** (DNS Rules 插件) - 规则管理、优先级配置
- **解析日志** (Log Analysis 插件) - 日志查询、筛选、下载

### 界面特性
- **多语言支持** - 中文、英文、阿拉伯语（RTL 布局支持）
- **响应式设计** - 桌面端和移动端适配
- **插件系统** - 动态菜单、插件状态检测

## 技术栈

- **React** 19.2.0
- **Vite** 7.2.4
- **Ant Design** 6.1.4
- **i18next** 25.8.3
- **Recharts** 3.6.0

## 项目结构

```
steadydns_ui/
├── VERSION              # 版本文件
├── CHANGELOG.md         # 变更日志
├── package.json         # 项目配置
├── vite.config.js       # Vite 配置
├── index.html           # 入口 HTML
├── src/
│   ├── main.jsx         # 应用入口
│   ├── App.jsx          # 主应用组件
│   ├── components/      # 组件目录
│   ├── pages/           # 页面组件
│   ├── i18n/            # 国际化
│   ├── utils/           # 工具函数
│   └── styles/          # 样式文件
├── scripts/             # 构建脚本
├── docs/                # 文档
└── public/              # 静态资源
```

## 开发指南

### 环境要求

- Node.js >= 16.0.0
- npm >= 8.0.0

### 安装依赖

```bash
npm install
```

### 启动开发服务器

```bash
npm run dev
```

开发服务器默认运行在 http://localhost:5173

### 构建生产版本

```bash
npm run build
```

构建产物输出到 `dist/` 目录。

### 运行代码检查

```bash
npm run lint
```

## 部署指南

### Go Embed 部署（推荐）

SteadyDNS UI 设计为通过 Go Embed 方式与后端集成部署：

1. **构建前端**
   ```bash
   npm run build
   ```

2. **后端集成**
   
   在 Go 后端项目中使用 `embed` 包嵌入 `dist/` 目录：
   ```go
   //go:embed dist
   var staticFiles embed.FS
   ```

3. **静态文件服务**
   
   配置 HTTP 服务器提供静态文件服务，API 请求代理到后端 API。

### Docker 部署

```bash
# 构建镜像
docker build -t steadydns-ui:latest .

# 运行容器
docker run -d -p 80:80 --name steadydns-ui steadydns-ui:latest
```

### Nginx 部署

1. 构建前端产物
2. 将 `dist/` 目录内容部署到 Nginx 静态目录
3. 配置 Nginx 反向代理 API 请求

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    root /usr/share/nginx/html;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    location /api {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 默认配置

- **默认用户**: admin
- **默认密码**: admin123
- **API 端口**: 8080
- **DNS 端口**: 53

## 许可证

本项目采用 GNU Affero General Public License v3.0 (AGPLv3) 许可证。

### 许可证说明

- **自由软件**：你可以自由使用、修改和分发本软件
- **源代码获取**：如果你运行修改后的版本，必须提供源代码访问
- **网络服务**：如果你通过网络提供本软件的服务，必须提供源代码访问

### 依赖兼容性

根据许可证兼容报告，本项目的所有依赖都使用与 AGPLv3 兼容的许可证：

- **核心依赖**：全部使用 MIT 许可证，与 AGPLv3 兼容
- **开发依赖**：大部分使用 MIT 许可证，terser 使用 BSD-2-Clause 许可证，两者都与 AGPLv3 兼容

详细的依赖许可证信息请参阅 [docs/DEPENDENCIES_LICENSES.md](docs/DEPENDENCIES_LICENSES.md)。

## 相关链接

- [GNU AGPLv3 许可证](LICENSE)
- [依赖许可证信息](docs/DEPENDENCIES_LICENSES.md)
- [变更日志](CHANGELOG.md)
- [后端项目](https://github.com/adilife/SteadyDns/tree/main/steadydnsd)

## 贡献

欢迎贡献代码、报告问题或提出建议。请确保你的贡献符合 AGPLv3 许可证的要求。

## 反馈与支持

如果您在使用过程中遇到问题或有改进建议，请通过以下方式反馈：

- 提交 Issue
- 发送邮件至技术支持
