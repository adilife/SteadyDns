## SteadyDNS v0.9.0-beta.1

这是 SteadyDNS 的第一个 Beta 测试版本，专为中小型环境设计的轻量级、高性能 DNS 解决方案。

### 🎉 新功能

#### 后端
- **DNS 转发服务** - 支持多转发组配置，实现默认转发与域名定向转发，组内支持多优先级负载均衡
- **BIND 服务器集成** - 完整的 BIND9.18+ 服务器管理和 Zone 文件操作
- **RESTful API 接口** - 基于 Gin 框架的 Web API 接口
- **JWT 认证** - 安全的用户认证和授权机制
- **用户管理** - 完整的用户 CRUD 操作
- **Go Embed 打包** - 前端文件嵌入二进制，单文件部署
- **TCP 连接池优化** - 高效的 DNS 查询处理
- **DNS Cookie 支持** - DNS 安全扩展

#### 前端 (steadydns_ui)
- **React Web 管理面板** - 简洁易用的操作界面
- **上游 DNS 服务器管理** - 添加/删除/优先级调整
- **本地解析规则配置** - 可视化配置
- **实时状态监控** - DNS 服务状态实时监控面板

### 📦 下载

| 平台 | 架构 | 文件 | 大小 |
|------|------|------|------|
| Linux | x86_64 | steadydns-0.9.0-beta.1-linux-amd64.tar.gz | ~14 MB |
| Linux | arm64 | steadydns-0.9.0-beta.1-linux-arm64.tar.gz | ~11 MB |

### 📖 快速安装

```bash
# 下载
wget https://github.com/adilife/SteadyDNS/releases/download/v0.9.0-beta.1/steadydns-0.9.0-beta.1-linux-amd64.tar.gz

# 解压
tar -xzf steadydns-0.9.0-beta.1-linux-amd64.tar.gz

# 进入目录
cd steadydns-0.9.0-beta.1-linux-amd64

# 启动服务
./steadydns start

# 访问 Web 管理界面
# http://localhost:8080
# 默认用户名: admin
# 默认密码: admin123
```

### 🔧 Systemd 服务安装

```bash
# 复制二进制文件
sudo cp steadydns /opt/steadydns/

# 安装服务
sudo cp scripts/steadydns.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable steadydns
sudo systemctl start steadydns
```

详细部署文档请参考 [DEPLOYMENT.md](./steadydnsd/docs/DEPLOYMENT.md)

### ⚠️ 注意事项

- **这是 Beta 测试版本**，不建议直接用于生产环境
- **首次登录后请立即修改默认密码**
- 配置文件 `config/steadydns.conf` 会在首次启动时自动创建
- 生产环境请修改 `JWT_SECRET_KEY` 为强密钥

### 🛡️ 安全建议

1. 修改默认管理员密码
2. 更改 `JWT_SECRET_KEY` 为强密钥
3. 根据实际需求调整 API 限流参数
4. 生产环境设置 `GIN_MODE=release`

### 📚 相关文档

- [项目概述](./README.md)
- [完整变更日志](./CHANGELOG.md)
- [部署文档](./steadydnsd/docs/DEPLOYMENT.md)
- [后端文档](./steadydnsd/README.md)
- [前端文档](./steadydns_ui/README.md)

### 🐛 反馈问题

如遇到问题或有改进建议，请提交 [Issue](https://github.com/adilife/SteadyDNS/issues)

---

**感谢使用 SteadyDNS！**
