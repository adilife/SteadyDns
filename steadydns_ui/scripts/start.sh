#!/bin/bash
# Start development server script for Linux
# 启动开发服务器脚本 - Linux版本

# Change to project root directory
cd "$(dirname "$0")/.." || exit 1

echo "Starting SteadyDNS UI development server..."
echo "正在启动 SteadyDNS UI 开发服务器..."

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

# Start development server
# 启动开发服务器
echo "Starting development server..."
echo "正在启动开发服务器..."
npm run dev

# Exit with the same code as npm run dev
# 以与npm run dev相同的代码退出
exit $?