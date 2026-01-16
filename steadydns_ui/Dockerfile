# Dockerfile for SteadyDNS UI
# 支持 amd64/arm64 多架构构建
# 使用多阶段构建减小最终镜像体积

# 阶段1: 构建阶段
# 使用node官方镜像作为构建基础
FROM node:18-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY package*.json ./

# 从阿里云镜像拉取依赖，加速构建
# 使用阿里云npm镜像
RUN npm config set registry https://registry.npmmirror.com/ && \
    npm install --legacy-peer-deps

# 复制源代码
COPY . .

# 构建项目
RUN npm run build

# 阶段2: 运行阶段
# 使用nginx作为web服务器
FROM nginx:alpine

# 复制构建产物到nginx目录
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制nginx配置文件
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 暴露80端口
EXPOSE 80

# 启动nginx
CMD ["nginx", "-g", "daemon off;"]

# 构建说明:
# 1. 构建amd64架构镜像:
#    docker build -t steadydns-ui:amd64 .
#
# 2. 构建arm64架构镜像:
#    docker build --platform linux/arm64 -t steadydns-ui:arm64 .
#
# 3. 构建多架构镜像并推送:
#    docker buildx create --use
#    docker buildx build --platform linux/amd64,linux/arm64 -t steadydns-ui:latest --push .
#
# 4. 运行容器:
#    docker run -d -p 80:80 --name steadydns-ui steadydns-ui:latest