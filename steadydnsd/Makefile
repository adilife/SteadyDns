# SteadyDNS Makefile
# 企业级DNS服务器构建脚本

# 版本信息
VERSION := $(shell cat VERSION 2>/dev/null || echo "unknown")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# 构建参数
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# 目录
SRC_DIR := src
CMD_DIR := $(SRC_DIR)/cmd
BIN_DIR := bin
PKG_DIR := pkg

# 前端相关目录
FRONTEND_DIR := ../steadydns_ui
STATIC_DIR := $(SRC_DIR)/core/webapi/static/dist

# 输出文件
BINARY_NAME := steadydns
BINARY := $(CMD_DIR)/$(BINARY_NAME)

# 默认目标
.PHONY: all
all: clean build

# 显示帮助信息
.PHONY: help
help:
	@echo "SteadyDNS v$(VERSION) - 企业级DNS服务器"
	@echo ""
	@echo "使用方法: make [目标]"
	@echo ""
	@echo "构建目标:"
	@echo "  build          编译应用程序"
	@echo "  build-full     完整构建（包含前端）"
	@echo "  build-linux    交叉编译Linux amd64版本"
	@echo "  build-arm64    交叉编译Linux arm64版本"
	@echo "  clean          清理构建产物"
	@echo ""
	@echo "前端相关:"
	@echo "  prepare-frontend  准备前端文件（从 steadydns_ui 复制）"
	@echo "  build-frontend    构建前端并复制到 Embed 目录"
	@echo ""
	@echo "测试目标:"
	@echo "  test           运行所有测试"
	@echo "  test-coverage  运行测试并生成覆盖率报告"
	@echo ""
	@echo "代码质量:"
	@echo "  lint           运行代码检查"
	@echo "  fmt            格式化代码"
	@echo "  vet            运行go vet检查"
	@echo ""
	@echo "安装与运行:"
	@echo "  install        安装依赖"
	@echo "  run            运行应用程序"
	@echo "  run-dev        以开发模式运行应用程序"
	@echo ""
	@echo "发布:"
	@echo "  release        准备发布版本"
	@echo "  release-full   准备完整发布版本（包含前端）"
	@echo "  version        显示版本信息"

# 编译应用程序
.PHONY: build
build:
	@echo ">>> 编译 SteadyDNS v$(VERSION)..."
	cd $(CMD_DIR) && $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo ">>> 编译完成: $(CMD_DIR)/$(BINARY_NAME)"

# 交叉编译 Linux amd64
.PHONY: build-linux
build-linux:
	@echo ">>> 交叉编译 Linux amd64..."
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	@echo ">>> 编译完成: $(BIN_DIR)/$(BINARY_NAME)-linux-amd64"

# 交叉编译 Linux arm64
.PHONY: build-arm64
build-arm64:
	@echo ">>> 交叉编译 Linux arm64..."
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@echo ">>> 编译完成: $(BIN_DIR)/$(BINARY_NAME)-linux-arm64"

# 清理构建产物
.PHONY: clean
clean:
	@echo ">>> 清理构建产物..."
	@rm -f $(CMD_DIR)/$(BINARY_NAME)
	@rm -rf $(BIN_DIR)
	@echo ">>> 清理完成"

# 运行所有测试
.PHONY: test
test:
	@echo ">>> 运行测试..."
	cd $(SRC_DIR) && $(GO) test ./... -v

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo ">>> 运行测试覆盖率分析..."
	cd $(SRC_DIR) && $(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=$(SRC_DIR)/coverage.out -o coverage.html
	@echo ">>> 覆盖率报告已生成: coverage.html"

# 运行代码检查
.PHONY: lint
lint:
	@echo ">>> 运行代码检查..."
	@which golangci-lint > /dev/null || (echo "请先安装 golangci-lint" && exit 1)
	cd $(SRC_DIR) && golangci-lint run ./...

# 格式化代码
.PHONY: fmt
fmt:
	@echo ">>> 格式化代码..."
	cd $(SRC_DIR) && $(GO) fmt ./...

# 运行 go vet 检查
.PHONY: vet
vet:
	@echo ">>> 运行 go vet..."
	cd $(SRC_DIR) && $(GO) vet ./...

# 安装依赖
.PHONY: install
install:
	@echo ">>> 安装依赖..."
	cd $(SRC_DIR) && $(GO) mod download
	@echo ">>> 依赖安装完成"

# 运行应用程序
.PHONY: run
run: build
	@echo ">>> 启动 SteadyDNS..."
	cd $(CMD_DIR) && ./$(BINARY_NAME) start

# 准备发布版本
.PHONY: release
release: clean test build-linux build-arm64
	@echo ">>> 准备发布版本 v$(VERSION)..."
	@echo ">>> 发布文件:"
	@ls -la $(BIN_DIR)/

# 显示版本信息
.PHONY: version
version:
	@echo "SteadyDNS v$(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# 检查代码规范
.PHONY: check
check: fmt vet test
	@echo ">>> 代码检查完成"

# ========================================
# 前端相关目标
# ========================================

# 准备前端文件（从 steadydns_ui 复制构建产物）
.PHONY: prepare-frontend
prepare-frontend:
	@echo ">>> 准备前端文件..."
	@if [ ! -d "$(FRONTEND_DIR)/dist" ]; then \
		echo "错误: 前端构建产物目录不存在: $(FRONTEND_DIR)/dist"; \
		echo "请先构建前端: cd $(FRONTEND_DIR) && npm run build"; \
		exit 1; \
	fi
	@rm -rf $(STATIC_DIR)
	@mkdir -p $(STATIC_DIR)
	@cp -r $(FRONTEND_DIR)/dist/* $(STATIC_DIR)/
	@echo ">>> 前端文件已复制到: $(STATIC_DIR)"

# 构建前端并复制到 Embed 目录
.PHONY: build-frontend
build-frontend:
	@echo ">>> 构建前端..."
	@if [ ! -d "$(FRONTEND_DIR)" ]; then \
		echo "错误: 前端目录不存在: $(FRONTEND_DIR)"; \
		exit 1; \
	fi
	cd $(FRONTEND_DIR) && npm run build
	$(MAKE) prepare-frontend

# 完整构建（包含前端）
.PHONY: build-full
build-full: prepare-frontend build
	@echo ">>> 完整构建完成（包含前端）"

# 以开发模式运行应用程序
.PHONY: run-dev
run-dev:
	@echo ">>> 以开发模式启动 SteadyDNS..."
	@echo ">>> 注意: 请确保已设置环境变量 STEADYDNS_DEV_MODE=true"
	cd $(CMD_DIR) && STEADYDNS_DEV_MODE=true ./$(BINARY_NAME) start

# 准备完整发布版本（包含前端）
.PHONY: release-full
release-full: clean prepare-frontend test build-linux build-arm64
	@echo ">>> 准备完整发布版本 v$(VERSION)（包含前端）..."
	@echo ">>> 发布文件:"
	@ls -la $(BIN_DIR)/
