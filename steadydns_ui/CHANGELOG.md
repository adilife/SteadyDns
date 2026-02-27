# Changelog

All notable changes to the SteadyDNS UI project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.9.0-beta.1] - 2026-02-27

### Added
- **用户认证系统**
  - JWT Token 认证机制
  - 自动 Token 刷新
  - 会话超时管理
  - 登录/登出功能

- **仪表盘**
  - 系统概览统计
  - QPS 趋势图表
  - 系统资源监控
  - 热门域名/客户端 TOP10
  - 缓存状态显示

- **权威域管理**（BIND 插件）
  - 权威域 CRUD 操作
  - DNS 记录管理（A, AAAA, MX, CNAME, TXT, SRV, PTR, CAA 等）
  - SOA 记录配置
  - 操作历史记录
  - 配置回滚功能

- **DNS 规则配置**（DNS Rules 插件）
  - DNS 规则 CRUD 操作
  - 规则优先级管理
  - 域名匹配测试

- **解析日志**（Log Analysis 插件）
  - 日志查询和筛选
  - 日志下载功能
  - 结果分类统计

- **转发器管理**
  - 转发组 CRUD 操作
  - 服务器优先级配置
  - 默认转发组管理
  - 域名匹配测试

- **配置管理**
  - 服务器状态监控
  - BIND 服务器管理
  - 缓存管理
  - 配置编辑和验证
  - 配置备份和恢复

- **用户管理**
  - 用户 CRUD 操作
  - 密码修改
  - 管理员账户保护

- **多语言支持**
  - 中文（zh-CN）
  - 英文（en-US）
  - 阿拉伯语（ar-SA）
  - RTL 布局支持

- **插件系统适配**
  - 动态菜单显示
  - 插件状态检测
  - 友好的禁用提示

- **响应式设计**
  - 桌面端布局
  - 移动端适配

### Technical
- React 19.2.0
- Vite 7.2.4
- Ant Design 6.1.4
- i18next 25.8.3
- Recharts 3.6.0
- AGPL-3.0-or-later 许可证

### Deployment
- 支持 Go Embed 嵌入部署
- Docker 多架构支持（amd64/arm64）
- Nginx 静态部署

### Notice
⚠️ **注意**: 当前版本为 v0.9.0-beta.1 测试版本，不建议直接用于生产环境。

---

## 版本说明

- **主版本号 (Major)**: 不兼容的 API 修改
- **次版本号 (Minor)**: 向下兼容的功能性新增
- **修订号 (Patch)**: 向下兼容的问题修正
- **预发布版本**: 如 -alpha, -beta, -rc 等
