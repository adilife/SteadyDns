# SteadyDNS UI

SteadyDNS UI 是一个基于 React + Vite 开发的 DNS 服务器管理界面，提供直观、易用的 DNS 服务器配置和监控功能。

## 功能特性

- DNS 规则管理
- 转发组配置
- 权威区域管理
- 缓存管理
- 服务器状态监控
- 日志查看
- 多语言支持

## 技术栈

- React 19+
- Vite
- Ant Design
- i18next
- Recharts

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
- **间接依赖**：所有间接依赖的许可证都与直接依赖相同或兼容

详细的依赖许可证信息请参阅 [docs/DEPENDENCIES_LICENSES.md](docs/DEPENDENCIES_LICENSES.md)。

## 开发指南

### 安装依赖

```bash
npm install
```

### 启动开发服务器

```bash
npm run dev
```

### 构建生产版本

```bash
npm run build
```

### 运行代码检查

```bash
npm run lint
```

## 贡献

欢迎贡献代码、报告问题或提出建议。请确保你的贡献符合 AGPLv3 许可证的要求。

## 相关链接

- [GNU AGPLv3 许可证](LICENSE)
- [依赖许可证信息](docs/DEPENDENCIES_LICENSES.md)
