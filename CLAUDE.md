# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个用 Go 语言开发的录音笔备份工具，专门用于 SR302 录音笔的文件自动备份。

## 重要提示

**本项目使用 Windows 系统，路径使用反斜杠（\\）作为分隔符。**

## 常用命令

### 构建项目
```bash
# 开发环境构建
go build -o bin/record_center.exe cmd/record_center/main.go

# 直接运行
go run cmd/record_center/main.go
```

### 运行程序
```bash
# 基本运行
./bin/record_center.exe

# 带参数运行
./bin/record_center.exe --config configs/custom.yaml --verbose
./bin/record_center.exe --check  # 只扫描不备份
./bin/record_center.exe --force  # 强制重新备份
./bin/record_center.exe --target "D:\backups"  # 指定目标目录
```

### 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/backup
go test ./internal/device
```

### 依赖管理
```bash
# 整理依赖
go mod tidy

# 下载依赖
go mod download
```

## 架构说明

### 核心模块
- **cmd/record_center**: 程序入口，使用 Cobra 框架处理命令行参数
- **internal/backup**: 备份核心功能，包括文件复制、检查和管理
- **internal/device**: SR302 设备检测，使用 WMI 接口
- **internal/config**: 基于 Viper 的配置管理
- **internal/logger**: 自定义日志系统
- **internal/progress**: 进度显示和跟踪
- **internal/storage**: 备份记录存储（JSON格式）

### 关键设计
- 使用 Windows WMI 接口检测 USB 设备（VID: 2207, PID: 0011）
- 增量备份机制，通过 JSON 记录避免重复备份
- 并发文件复制，提高备份效率
- 支持断点续传和文件校验

## 配置说明

主配置文件位于 `configs/backup.yaml`，包含：
- 设备检测配置（VID/PID、设备名称）
- 备份目标配置（目录、子目录结构）
- 备份策略（文件类型、并发数）
- 日志配置（级别、轮转）

## 开发注意事项

1. 项目专门针对 SR302 录音笔，硬编码了设备的 VID/PID（2207:0011）
2. 大量使用了 Windows 特有的 API（WMI），不支持跨平台
3. 配置文件中的路径使用 Windows 格式，注意反斜杠转义（如：`"内部共享存储空间\\录音笔文件"`）
4. 日志文件会自动轮转，默认保留 7 天
5. 备份记录存储在 `data/` 目录，用于增量备份判断
6. 目前没有单元测试，开发时建议添加测试用例
7. 默认只备份 `.opus` 格式的录音文件

## 日志和调试

- 日志文件位于 `logs/record_center.log`
- 使用 `--verbose` 参数查看详细输出
- 使用 `--check` 参数进行干运行，不实际复制文件