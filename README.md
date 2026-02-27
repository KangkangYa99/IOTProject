# IOTProject - 物联网后端管理系统

> 基于 Go + Gin 构建的高性能物联网设备管理后端系统

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/status-stable-brightgreen.svg)](https://github.com/KangkangYa99/IOTProject)

---

## 📋 项目概述

**IOTProject** 是一个现代化的物联网设备管理后端系统，采用 Go 语言和 Gin 框架构建。项目专注于提供稳定、高效、安全的物联网设备管理解决方案。

**当前版本**: v1.0.1  
**最后更新**: 2026-01-31  
**开发状态**: 基础架构开发完成 / 设备模块规划中 🛠️

---

## 🚀 核心特性

### 💻 技术栈
- **Web 框架**: [Gin](https://github.com/gin-gonic/gin) - 高性能 HTTP 框架
- **数据库**: [PostgreSQL](https://www.postgresql.org/) - 使用 pgx 连接池
- **缓存**: [Redis](https://redis.io/) - 分布式锁、JWT 管理、限流
- **安全**: JWT 鉴权 + Bcrypt 密码哈希
- **部署**: Docker + Docker Compose

### 🛡️ 安全特性
- **JWT 鉴权**: 基于角色的访问控制（RBAC）
- **密码安全**: Bcrypt 加密存储
- **防重复注册**: Redis 分布式锁
- **Token 管理**: 支持单点登录 / 多设备登录（可配置）
- **Token 黑名单**: 登出后立即失效

### ⚡ 性能优化
- **Redis 缓存**: 用户信息、Token 管理
- **数据库连接池**: pgx 连接池优化
- **API 限流**: Redis + Lua 脚本实现
- **缓存策略**: 防止缓存击穿、穿透、雪崩

### 📱 功能模块
- **用户管理**: 注册、登录、信息更新、登出
- **权限管理**: 支持多角色（普通用户、管理员、经理）
- **管理员功能**: 支持管理员创建新用户
- **设备管理**: 设备绑定、解绑、状态监控（开发中）

## ✨ 已实现功能

### 📝 用户模块
- [x] 用户注册（用户名/手机号/邮箱）
- [x] 用户登录（JWT 鉴权）
- [x] 获取用户信息
- [x] 修改用户信息
- [x] 用户登出

### 🔐 安全防护
- [x] 注册防重复（Redis 分布式锁）
- [x] 单点登录 / 多设备登录（可配置）
- [x] Token 黑名单（登出立即失效）
- [x] 密码强度校验
- [x] API 限流保护

### 👑 管理员功能
- [x] 管理员创建用户
- [x] 角色权限管理
- [x] 用户状态管理

### 📊 开发体验
- [x] 统一 API 返回格式 `{code, data, message}`
- [x] 自动错误处理
- [x] 详细的日志记录
- [x] Docker 一键部署

---

## 🚀 快速开始

### 环境要求
- Go 1.21+
- PostgreSQL 12+
- Redis 6+
- Docker (可选)

### 1. 克隆项目

```bash
git clone https://github.com/KangkangYa99/IOTProject.git
cd IOTProject
