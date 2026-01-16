# SteadyDNS UI 跨平台部署文档
# SteadyDNS UI Cross-Platform Deployment Guide

## 项目简介
## Project Introduction

SteadyDNS UI 是一个基于 React + Vite + Ant Design 开发的分级 DNS 服务器管理界面，支持 Windows/Linux (x86)/Linux (ARM) 及国产化系统（统信 UOS/麒麟）。

SteadyDNS UI is a hierarchical DNS server management interface developed based on React + Vite + Ant Design, supporting Windows/Linux (x86)/Linux (ARM) and domestic systems (UOS/Kylin).

## 技术栈
## Technology Stack

- React 19.x
- Vite 7.x
- Ant Design 6.x
- 纯静态资源构建，无平台专属依赖

- React 19.x
- Vite 7.x
- Ant Design 6.x
- Pure static resource build, no platform-specific dependencies

## 部署方式
## Deployment Methods

### 1. 静态部署（通用）
### 1. Static Deployment (Universal)

#### 步骤
#### Steps

1. **构建项目**
   ```bash
   # Linux/Mac
   ./scripts/build.sh
   
   # Windows
   .\scripts\build.bat
   ```

2. **部署静态文件**
   将 `dist` 目录下的所有文件复制到 Web 服务器的根目录。

3. **配置 Web 服务器**
   - Nginx 配置示例（见 nginx.conf）
   - Apache 配置示例：
     ```apache
     <VirtualHost *:80>
         ServerName steadydns.example.com
         DocumentRoot /var/www/steadydns-ui
         
         <Directory /var/www/steadydns-ui>
             AllowOverride All
             Require all granted
         </Directory>
         
         ErrorDocument 404 /index.html
     </VirtualHost>
     ```

### 2. Docker 部署（支持多架构）
### 2. Docker Deployment (Multi-architecture Support)

#### 步骤
#### Steps

1. **构建 Docker 镜像**
   - 构建 amd64 架构镜像：
     ```bash
     docker build -t steadydns-ui:amd64 .
     ```
   
   - 构建 arm64 架构镜像：
     ```bash
     docker build --platform linux/arm64 -t steadydns-ui:arm64 .
     ```
   
   - 构建多架构镜像并推送：
     ```bash
     docker buildx create --use
     docker buildx build --platform linux/amd64,linux/arm64 -t steadydns-ui:latest --push .
     ```

2. **运行 Docker 容器**
   ```bash
   docker run -d -p 80:80 --name steadydns-ui steadydns-ui:latest
   ```

3. **访问界面**
   打开浏览器访问 `http://localhost`

### 3. 开发环境启动
### 3. Development Environment Startup

#### 步骤
#### Steps

1. **启动开发服务器**
   ```bash
   # Linux/Mac
   ./scripts/start.sh
   
   # Windows
   .\scripts\start.bat
   ```

2. **访问界面**
   打开浏览器访问 `http://localhost:5173`

## 平台适配说明
## Platform Adaptation Instructions

### Windows 平台
### Windows Platform

- 使用批处理脚本 `scripts/start.bat` 和 `scripts/build.bat`
- 支持 Windows 10/11 64位系统
- 推荐使用 Node.js 16.x 或更高版本

- Use batch scripts `scripts/start.bat` and `scripts/build.bat`
- Support Windows 10/11 64-bit systems
- Recommended Node.js 16.x or higher

### Linux 平台
### Linux Platform

- 使用 Shell 脚本 `scripts/start.sh` 和 `scripts/build.sh`
- 支持 Ubuntu 18.04+, CentOS 7+, Debian 9+
- 支持 x86_64 和 ARM64 架构
- 推荐使用 Node.js 16.x 或更高版本

- Use Shell scripts `scripts/start.sh` and `scripts/build.sh`
- Support Ubuntu 18.04+, CentOS 7+, Debian 9+
- Support x86_64 and ARM64 architectures
- Recommended Node.js 16.x or higher

### 国产化系统
### Domestic Systems

#### 统信 UOS
#### UnionTech OS

- 字体适配：默认使用思源黑体
- 浏览器适配：支持 360 安全浏览器、UOS 自带浏览器
- 部署方式：同 Linux 平台
- 推荐使用 nginx 作为 Web 服务器

- Font adaptation: Default to Source Han Sans
- Browser adaptation: Support 360 Secure Browser, UOS built-in browser
- Deployment method: Same as Linux platform
- Recommended to use nginx as web server

#### 麒麟操作系统
#### Kylin OS

- 字体适配：默认使用麒麟字体
- 浏览器适配：支持 360 安全浏览器、麒麟自带浏览器
- 部署方式：同 Linux 平台
- 推荐使用 nginx 作为 Web 服务器

- Font adaptation: Default to Kylin font
- Browser adaptation: Support 360 Secure Browser, Kylin built-in browser
- Deployment method: Same as Linux platform
- Recommended to use nginx as web server

## 依赖管理
## Dependency Management

- 所有 npm 包从阿里云镜像拉取，加速构建
- 静态资源本地打包，无海外 CDN 依赖
- 构建产物为纯静态资源，可直接部署

- All npm packages are pulled from Alibaba Cloud mirror to accelerate build
- Static resources are packaged locally, no overseas CDN dependencies
- Build artifacts are pure static resources, can be deployed directly

## 常见问题
## Common Issues

### 1. 构建失败
### 1. Build Failure

- 检查 Node.js 版本是否 >= 16.x
- 检查 npm 依赖是否正确安装
- 检查网络连接是否正常（拉取依赖需要网络）

- Check if Node.js version >= 16.x
- Check if npm dependencies are installed correctly
- Check if network connection is normal (network is required for pulling dependencies)

### 2. 页面无法访问
### 2. Page Cannot Be Accessed

- 检查 Web 服务器是否正常运行
- 检查防火墙是否开放对应端口
- 检查 Docker 容器是否正常启动
- 检查浏览器是否支持（推荐使用 Chrome、Firefox、Edge 等现代浏览器）

- Check if web server is running normally
- Check if firewall has opened the corresponding port
- Check if Docker container is started normally
- Check if browser is supported (recommended to use modern browsers such as Chrome, Firefox, Edge)

### 3. 国产化系统兼容性问题
### 3. Domestic System Compatibility Issues

- 字体显示异常：确保系统已安装思源黑体或麒麟字体
- 浏览器渲染问题：使用最新版本的 360 安全浏览器或系统自带浏览器
- 部署路径问题：使用 POSIX 风格路径（/），避免 Windows 反斜杠（\）

- Font display issue: Ensure Source Han Sans or Kylin font is installed on the system
- Browser rendering issue: Use the latest version of 360 Secure Browser or system built-in browser
- Deployment path issue: Use POSIX style path (/) instead of Windows backslash (\)

## 性能优化
## Performance Optimization

1. **代码分割**：已配置手动代码分割，将依赖库与业务代码分离
2. **静态资源压缩**：构建时自动压缩 JS、CSS、HTML 文件
3. **缓存策略**：配置了合理的静态资源缓存策略
4. **懒加载**：可根据需要实现路由懒加载，进一步优化首屏加载速度

1. **Code splitting**: Manual code splitting is configured to separate dependency libraries from business code
2. **Static resource compression**: Automatically compress JS, CSS, HTML files during build
3. **Cache strategy**: Reasonable static resource cache strategy is configured
4. **Lazy loading**: Route lazy loading can be implemented as needed to further optimize first-screen loading speed

## 安全注意事项
## Security Notes

1. **部署环境**：建议在防火墙后部署，限制访问 IP
2. **HTTPS**：生产环境建议配置 HTTPS
3. **权限控制**：可根据需要集成身份认证系统
4. **输入验证**：所有用户输入都应进行严格验证

1. **Deployment environment**: It is recommended to deploy behind a firewall and restrict access IP
2. **HTTPS**: HTTPS is recommended for production environment
3. **Permission control**: Identity authentication system can be integrated as needed
4. **Input validation**: All user inputs should be strictly validated

## 版本更新
## Version Update

1. **拉取最新代码**：`git pull`
2. **重新构建**：执行构建脚本
3. **部署新文件**：替换旧的静态文件
4. **重启服务**：如果使用 Docker，重启容器

1. **Pull latest code**: `git pull`
2. **Rebuild**: Execute build script
3. **Deploy new files**: Replace old static files
4. **Restart service**: If using Docker, restart container

## 联系方式
## Contact Information

- 项目地址：https://github.com/yourusername/steadydns-ui
- 问题反馈：https://github.com/yourusername/steadydns-ui/issues

- Project address: https://github.com/yourusername/steadydns-ui
- Issue feedback: https://github.com/yourusername/steadydns-ui/issues