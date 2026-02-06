# SteadyDNS服务器管理功能设计（基于现有API扩展）

## 1. 设计背景

基于对现有SteadyDNS项目代码的分析，当前项目已经实现了健康检查API、Dashboard API、BIND API等基础功能。为了提供更全面的服务器管理能力，需要在现有API基础上扩展实现服务器管理功能，包括动态管理、静态管理和BIND服务器管理三个主要模块。

## 2. 功能设计

### 2.1 动态管理功能

| 功能模块 | 详细描述 | API路径 | 实现方式 |
|---------|---------|---------|----------|
| DNS服务器状态管理 | 启动/停止/重启DNS服务器（UDP/TCP） | `/api/server/sdnsd/{action}` | 新增API端点，控制DNS服务器实例状态 |
| API服务器状态管理 | 启动/停止/重启API服务器（HTTP服务器） | `/api/server/httpd/{action}` | 新增API端点，控制API服务器实例状态 |
| 转发组配置重载 | 实时重载转发组配置，无需重启服务 | `/api/server/reload-forward-groups` | 调用现有`ReloadForwardGroups`函数 |
| 缓存管理 | 清空缓存、按域名清除缓存、获取缓存统计 | `/api/cache/{action}` | 实现现有`cacheapi.go`，利用`GlobalCacheUpdater` |
| 统计信息监控 | 获取DNS服务器统计信息（查询量、错误率等） | `/api/dashboard/stats` | 扩展现有Dashboard API |
| 健康状态检查 | 检查DNS服务和依赖组件的健康状态 | `/api/health` | 扩展现有健康检查API |
| 日志管理 | 调整日志级别、触发日志轮转 | `/api/server/logging/{action}` | 新增API端点，动态修改日志配置 |

### 2.2 静态管理功能

管理steadyDNS服务器的静态配置,包括DNS服务器配置、API服务器配置、转发组配置等。

| 功能模块 | 详细描述 | API路径 | 实现方式 |
|---------|---------|---------|----------|
| 配置文件管理 | 读取、更新、验证配置文件 | `/api/config/{section}/{key}` | 新增配置管理API，扩展现有配置功能 |
| 环境变量管理 | 查看和设置环境变量（运行时） | `/api/config/env` | 新增API端点，封装环境变量操作 |
| 配置版本控制 | 配置变更历史、回滚功能 | `/api/config/history` | 新增API端点，实现配置变更记录 |
| 配置备份与恢复 | 备份当前配置、从备份恢复 | `/api/config/backup` | 新增API端点，实现配置文件备份机制 |
| 默认配置管理 | 查看和重置默认配置 | `/api/config/defaults` | 新增API端点，基于现有默认配置模板 |

### 2.3 BIND服务器管理功能

| 功能模块 | 详细描述 | API路径 | 实现方式 |
|---------|---------|---------|----------|
| BIND服务状态管理 | 启动/停止/重启/重载BIND服务 | `/api/bind-server/{action}` | 新增API端点，执行BIND服务控制命令 |
| BIND配置文件管理 | 读取、更新、验证BIND配置文件 | `/api/bind-server/config` | 新增API端点，扩展现有BIND配置管理 |
| 权威域管理增强 | 批量操作、导入/导出、验证 | `/api/bind-zones/{action}` | 扩展现有BIND API |
| BIND服务器统计 | 获取BIND服务器运行统计信息 | `/api/bind-server/stats` | 新增API端点，解析BIND统计输出 |
| BIND服务健康检查 | 检查BIND服务状态和配置有效性 | `/api/bind-server/health` | 新增API端点，实现BIND服务健康检查 |
| BIND配置验证 | 验证BIND配置文件和区域文件 | `/api/bind-server/validate` | 新增API端点，使用named-checkconf和named-checkzone |

## 3. 架构设计

### 3.1 模块划分与代码文件

#### 3.1.1 需要新增的代码文件

| 模块 | 文件名 | 职责 |
|------|--------|------|
| 服务器管理API | `/root/go/SteadyDns/steadydnsd/src/core/webapi/server_api.go` | 服务器管理相关的API接口处理 |
| 配置管理API | `/root/go/SteadyDns/steadydnsd/src/core/webapi/config_api.go` | 配置管理相关的API接口处理 |
| BIND服务器管理API | `/root/go/SteadyDns/steadydnsd/src/core/webapi/bind_server_api.go` | BIND服务器管理API接口处理 |
| 服务器管理核心 | `/root/go/SteadyDns/steadydnsd/src/core/sdns/server_manager.go` | 核心服务器管理功能，包括DNS服务器状态控制等 |
| 配置管理核心 | `/root/go/SteadyDns/steadydnsd/src/core/common/config_manager.go` | 配置管理核心功能，包括配置文件操作、版本控制等 |
| 服务器管理模型 | `/root/go/SteadyDns/steadydnsd/src/core/sdns/server_models.go` | 服务器管理相关的数据模型 |

#### 3.1.2 需要修改的现有代码文件

| 模块 | 文件名 | 职责 | 修改内容 |
|------|--------|------|----------|
| 缓存API实现 | `/root/go/SteadyDns/steadydnsd/src/core/webapi/cacheapi.go` | 缓存管理API | 实现缓存管理功能 |
| 健康检查API | `/root/go/SteadyDns/steadydnsd/src/core/webapi/health.go` | 健康检查API | 扩展健康检查功能，增加更多组件检查 |
| Dashboard API | `/root/go/SteadyDns/steadydnsd/src/core/webapi/dashboardapi.go` | 仪表盘API | 扩展统计信息获取功能 |
| 主入口文件 | `/root/go/SteadyDns/steadydnsd/src/cmd/main.go` | 应用入口 | 注册新的API路由 |
| 配置管理 | `/root/go/SteadyDns/steadydnsd/src/core/common/config.go` | 配置管理 | 扩展配置操作功能 |
| BIND管理 | `/root/go/SteadyDns/steadydnsd/src/core/bind/bind.go` | BIND管理 | 扩展BIND服务状态控制功能 |
| DNS处理器 | `/root/go/SteadyDns/steadydnsd/src/core/sdns/dnshandler.go` | DNS处理 | 增加服务器状态管理支持 |
| 响应工具 | `/root/go/SteadyDns/steadydnsd/src/core/webapi/response.go` | API响应 | 扩展响应结构和工具函数 |

### 3.2 核心组件

1. **ServerManager**：核心服务器管理组件，管理DNS服务器状态和配置
2. **ConfigManager**：配置管理组件，处理配置文件和环境变量
3. **BindServerManager**：BIND服务器管理组件，扩展现有BindManager
4. **ServerAPIHandler**：服务器管理API处理函数
5. **ConfigAPIHandler**：配置管理API处理函数
6. **BindServerAPIHandler**：BIND服务器管理API处理函数

### 3.3 数据流设计

1. **API请求流**：客户端 → API路由 → 相应APIHandler → 管理组件 → 操作执行 → 结果返回
2. **配置更新流**：配置修改 → 配置验证 → 配置持久化 → 服务重载（如需） → 状态更新
3. **状态监控流**：定期检查 → 状态收集 → 统计计算 → 状态存储 → API查询响应

## 4. 实现方案

### 4.1 核心实现

1. **ServerManager**：
   - 管理全局DNS服务器实例（UDP/TCP）
   - 实现服务器状态控制方法（启动/停止/重启）
   - 提供统计信息收集功能
   - 处理服务器事件和告警

2. **ConfigManager**：
   - 扩展现有配置管理功能
   - 实现配置文件读写和验证
   - 管理配置变更历史
   - 提供配置备份和恢复

3. **BindServerManager**：
   - 扩展现有BindManager
   - 实现BIND服务状态控制
   - 提供BIND配置管理功能
   - 实现BIND服务健康检查

### 4.2 API实现

1. **服务器管理API**：
   - `POST /api/server/sdnsd/start` - 启动DNS服务器
   - `POST /api/server/sdnsd/stop` - 停止DNS服务器
   - `POST /api/server/sdnsd/restart` - 重启DNS服务器
   - `POST /api/server/httpd/start` - 启动HTTPD服务器
   - `POST /api/server/httpd/stop` - 停止HTTPD服务器
   - `POST /api/server/httpd/restart` - 重启HTTPD服务器
   - `POST /api/server/reload-forward-groups` - 重载转发组
   - `POST /api/server/logging/level` - 设置日志级别

2. **缓存API实现**：
   - `GET /api/cache/stats` - 获取缓存统计信息
   - `POST /api/cache/clear` - 清空缓存
   - `POST /api/cache/clear/{domain}` - 按域名清除缓存

3. **配置管理API**：
   - `GET /api/config` - 获取所有配置
   - `GET /api/config/{section}` - 获取指定节的配置
   - `GET /api/config/{section}/{key}` - 获取指定配置项
   - `PUT /api/config/{section}/{key}` - 更新配置项
   - `POST /api/config/reload` - 重载配置
   - `POST /api/config/backup` - 备份配置
   - `POST /api/config/restore` - 恢复配置
   - `GET /api/config/history` - 获取配置历史

4. **BIND服务器管理API**：
   - `GET /api/bind-server/status` - 获取BIND服务器状态
   - `POST /api/bind-server/start` - 启动BIND服务器
   - `POST /api/bind-server/stop` - 停止BIND服务器
   - `POST /api/bind-server/restart` - 重启BIND服务器
   - `POST /api/bind-server/reload` - 重载BIND服务器
   - `GET /api/bind-server/stats` - 获取BIND服务器统计
   - `GET /api/bind-server/health` - 检查BIND服务器健康状态
   - `POST /api/bind-server/validate` - 验证BIND配置

### 4.3 技术要点

1. **API设计一致性**：
   - 遵循现有API的设计模式和响应格式
   - 使用相同的中间件（认证、日志、速率限制）
   - 保持错误处理和响应结构的一致性

2. **安全性**：
   - API访问控制（使用现有AuthMiddleware）
   - 操作权限验证
   - 敏感配置保护
   - 操作审计日志

3. **可靠性**：
   - 操作幂等性
   - 事务支持（配置更新）
   - 回滚机制
   - 健康检查和自动恢复

4. **性能**：
   - 异步操作（如重载配置）
   - 缓存频繁访问的数据
   - 优化配置读取和解析

## 5. 集成与部署

### 5.1 集成方案

1. **代码集成**：
   - 在`main.go`的`setupRoutes()`中注册新的API路由
   - 集成到现有日志和监控系统
   - 利用现有依赖库

2. **配置集成**：
   - 添加服务器管理相关配置项
   - 扩展现有配置模板
   - 支持环境变量配置

### 5.2 部署方案

1. **部署流程**：
   - 编译包含新功能的二进制文件
   - 复制配置文件（如需更新）
   - 重启服务

2. **升级方案**：
   - 向后兼容现有配置
   - 提供配置迁移工具
   - 文档更新

3. **监控与告警**：
   - 集成现有监控系统
   - 添加服务器管理相关指标
   - 配置告警规则

## 6. 测试计划

### 6.1 单元测试

1. **核心功能测试**：
   - 服务器状态管理
   - 配置管理
   - BIND服务器管理

2. **API测试**：
   - API接口响应
   - 错误处理
   - 权限控制

### 6.2 集成测试

1. **功能集成测试**：
   - DNS服务器控制
   - 配置变更与重载
   - BIND服务管理

2. **性能测试**：
   - API响应时间
   - 配置重载性能
   - 并发操作处理

### 6.3 验收测试

1. **功能验证**：
   - 所有API接口功能验证
   - 服务器状态控制验证
   - 配置管理功能验证
   - BIND服务器管理验证

2. **可靠性测试**：
   - 异常场景处理
   - 服务重启后状态恢复
   - 配置错误处理

## 7. 总结

本设计方案基于现有SteadyDNS项目的API结构，通过扩展现有API和新增必要的API端点，实现了完整的服务器管理功能。方案详细列出了需要新增和修改的代码文件，确保实施过程的清晰性和可操作性。

实施此方案后，SteadyDNS将具备更强大的服务器管理能力，提供更完善的运维体验，适合生产环境的大规模部署和管理。同时，方案保持了代码的可维护性和可扩展性，为未来的功能增强奠定了基础。