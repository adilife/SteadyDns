# SteadyDns

A lightweight DNS solution tailored for small to medium-sized environments. Built with Golang, it boasts extreme concurrent processing capabilities and minimal resource consumption. Core features include intelligent prioritized upstream forwarding, a real-time effective web management panel, and a zero-dependency rapid deployment experience.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Language](https://img.shields.io/badge/Language-Golang%20%7C%20JavaScript-blue)](https://github.com/adilife/SteadyDns)

专为中小型环境设计的轻量级、高性能 DNS 解决方案，兼顾易用性和稳定性，支持智能优先级转发、实时 Web 管理面板，零依赖快速部署。

## 项目简介
SteadyDns 由两个核心子项目组成，前后端分离架构：
- **steadydnsd**：Go 编写的 DNS 服务端核心，负责 DNS 请求解析、智能上游转发、配置持久化等核心逻辑，具备超高并发处理能力和极低资源消耗。
- **steadydns_ui**：基于 JavaScript/CSS 开发的 Web 管理面板，提供可视化配置、状态监控、规则管理等功能，配置实时生效无需重启服务。

## 核心特性
### 整体特性
- 轻量级：单二进制文件部署，无额外依赖
- 高性能：基于 Go 原生并发模型，支持每秒万级 DNS 请求处理
- 可视化管理：Web 面板一键配置，无需修改配置文件
- 智能转发：支持多上游 DNS 服务器配置，按优先级/可用性自动切换
- 实时生效：配置修改即时生效，无需重启 DNS 服务
- 状态监控：实时查看 DNS 请求量、响应耗时、上游可用性等指标
- 稳定性保障：自动屏蔽不可用的上游 DNS，避免解析失败

### 后端 (steadydnsd) 特性
- 支持 A/AAAA/CNAME/MX 等主流 DNS 记录类型解析
- 自定义本地解析规则（Hosts 映射）
- 支持 TCP/UDP 协议，兼容 IPv4/IPv6
- 配置文件自动备份与恢复
- 日志记录与审计功能

### 前端 (steadydns_ui) 特性
- 简洁易用的操作界面
- 上游 DNS 服务器管理（添加/删除/优先级调整）
- 本地解析规则可视化配置
- DNS 服务状态实时监控面板
- 配置导入/导出功能

## 快速开始
### 环境要求
- 操作系统：Linux
- 架构支持：x86_64
- 端口要求：需开放 53（DNS 服务）、8080（Web 面板，可自定义）端口

# Windows
./steadydnsd.exe --config ./config.yaml
