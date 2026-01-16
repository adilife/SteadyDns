#!/bin/bash
# Build project script for Linux
# 构建项目脚本 - Linux版本

# Change to project root directory
cd "$(dirname "$0")/.." || exit 1

echo "Building SteadyDNS UI project..."
echo "正在构建 SteadyDNS UI 项目..."

# Check if npm is installed
# 检查npm是否安装
if ! command -v npm &> /dev/null; then
    echo "Error: npm is not installed. Please install Node.js first."
    echo "错误: npm 未安装。请先安装 Node.js。"
    exit 1
fi

# Check if node_modules exists, if not install dependencies
# 检查node_modules是否存在，不存在则安装依赖
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    echo "正在安装依赖..."
    npm install
    if [ $? -ne 0 ]; then
        echo "Error: Failed to install dependencies."
        echo "错误: 依赖安装失败。"
        exit 1
    fi
fi

# Build project
# 构建项目
echo "Building project..."
echo "正在构建项目..."
npm run build

if [ $? -eq 0 ]; then
    echo "Build completed successfully!"
    echo "构建成功完成！"
    echo "Output directory: dist/"
    echo "输出目录: dist/"
else
    echo "Error: Build failed."
    echo "错误: 构建失败。"
    exit 1
fi

# Exit with success code
# 以成功代码退出
exit 0