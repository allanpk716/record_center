# 录音笔备份工具 (Record Center)

一个专为 SR302 录音笔设计的自动化文件备份工具，支持文件完整性验证、断点续传等高级功能。

## 功能特点

- 🎯 **设备自动检测**：自动识别 SR302 录音笔（VID:2207, PID:0011）
- 📁 **智能文件管理**：仅备份 `.opus` 格式的录音文件，保持原有目录结构
- ✅ **文件完整性验证**：支持 SHA256/SHA1/MD5 哈希验证，确保备份文件完整性
- 🔄 **断点续传**：支持大文件的断点续传，网络中断或程序异常后可继续备份
- 🚀 **并发复制**：支持多文件并发复制，提高备份效率
- 📊 **进度显示**：实时显示备份进度和速度统计
- 🔍 **检查模式**：支持扫描但不实际复制文件，用于预览备份内容
- 📝 **详细日志**：完整的操作日志记录，支持多级别日志输出

## 系统要求

- **操作系统**：Windows 10/11
- **PowerShell**：支持 PowerShell 5.0 或更高版本
- **权限**：建议以管理员身份运行（某些MTP设备访问需要管理员权限）

## 安装和使用

### 1. 下载程序

从 Releases 页面下载对应的可执行文件，或者自行编译：

```bash
# 开发环境构建
go build -o record_center.exe cmd/record_center/main.go

# 或者直接运行
go run cmd/record_center/main.go
```

### 2. 配置文件

程序使用 `backup.yaml` 配置文件（位于程序根目录），首次运行会自动创建默认配置：

```yaml
# 录音笔备份工具配置文件

# 源设备配置
source:
  device_name: "SR302"                    # 录音笔设备名称
  base_path: "内部共享存储空间\\录音笔文件" # 设备内基础路径
  vid: "2207"                            # USB VID
  pid: "0011"                            # USB PID

# 目标备份配置
target:
  base_directory: "./backups"              # 备份目标目录
  create_subdirs: true                     # 是否创建子目录结构

# 备份配置
backup:
  file_extensions: [".opus"]               # 要备份的文件扩展名
  skip_existing: true                      # 跳过已存在的文件
  preserve_structure: true                 # 保持原有目录结构
  max_concurrent: 3                        # 最大并发复制数

  # 完整性验证配置
  integrity_check: true                    # 启用文件完整性验证
  hash_algorithm: "sha256"                 # 哈希算法 (md5, sha1, sha256)

  # 断点续传配置
  enable_resume: true                      # 启用断点续传功能
  chunk_size: "5MB"                        # 文件分块大小
  resume_interval: "5MB"                   # 保存进度的间隔
  temp_dir: "./temp"                       # 临时文件目录
  resume_max_age: "24h"                    # 断点信息保留时间

# 日志配置
logging:
  level: "info"                           # 日志级别: debug, info, warn, error
  file: "record_center.log"               # 日志文件名
  console: true                           # 是否输出到控制台
  rotate_hours: 24                        # 日志轮转时间（小时）
  max_days: 7                             # 日志保留天数
```

### 3. 基本使用

#### 首次使用 - 检测设备信息
```bash
record_center.exe detect
```
这个命令会自动检测您的录音笔并显示配置信息。

#### 检查设备连接和文件（不备份）
```bash
record_center.exe --check
```

#### 执行完整备份
```bash
record_center.exe
```

#### 强制重新备份所有文件
```bash
record_center.exe --force
```

#### 指定备份目标目录
```bash
record_center.exe --target "D:\录音笔备份"
```

#### 显示详细日志
```bash
record_center.exe --verbose
```

### 4. 设备自动检测（新功能）

如果您不确定录音笔的设备信息，可以使用自动检测命令：

```bash
record_center detect
```

这个命令会：
- 自动扫描所有连接的USB设备
- 识别可能的录音笔设备
- 显示设备名称、VID、PID信息
- 生成可直接使用的配置片段
- 特别标记SR302设备

检测输出示例：
```
🎤 检测到的录音笔设备：
============================================================

📱 设备 #1
   名称: SR302
   VID:  2207
   PID:  0011
   ID:   USB\VID_2207&PID_0011&MI_00\7&117ED41B&0&0000

   配置片段：
   source:
     device_name: "SR302"
     vid: "2207"
     pid: "0011"

✅ 检测到SR302设备！
```

### 5. 命令行参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `detect` | 自动检测录音笔设备信息 | `record_center detect` |
| `--config` | 指定配置文件路径 | `--config my_config.yaml` |
| `--check` | 仅扫描文件，不执行备份 | `--check` |
| `--force` | 强制重新备份所有文件 | `--force` |
| `--target` | 指定备份目标目录 | `--target "D:\backups"` |
| `--verbose` | 显示详细日志输出 | `--verbose` |
| `--help, -h` | 显示帮助信息 | `--help` |

## 工作原理

### MTP 设备访问

程序使用多种策略访问 MTP 设备：

1. **PowerShell 访问**（首选）
   - 通过 Windows Shell COM 接口
   - 支持便携式设备命名空间
   - 支持桌面设备列表访问
   - 支持 WMI 增强查询

2. **基本文件系统访问**（备选）
   - 通过标准文件系统 API
   - 适用于已挂载的设备

3. **模拟访问**（测试）
   - 创建临时文件模拟 MTP 内容
   - 用于程序功能测试

### 文件完整性验证

程序使用加密哈希算法验证文件完整性：

- **SHA256**（推荐）：安全性高，性能良好
- **SHA1**：兼容性好，安全性适中
- **MD5**：性能最佳，安全性较低

每个备份文件都会计算哈希值并存储在备份记录中。

### 断点续传机制

大文件备份时支持断点续传：

1. **分块处理**：文件被分成固定大小的块（默认5MB）
2. **进度保存**：每复制一定字节数后保存进度
3. **中断恢复**：程序重启后自动检测未完成的备份
4. **原子操作**：使用临时文件确保数据完整性

## 目录结构

```
record_center/
├── bin/                        # 编译后的可执行文件
├── backups/                    # 备份文件目录
│   └── 2025/                   # 按年组织的子目录
│       └── 11月/               # 按月组织的子目录
│           └── 具体录音文件
├── backup.yaml               # 配置文件（位于根目录）
├── data/                       # 数据目录
│   ├── backup_records.json    # 备份记录
│   └── resume/                # 断点续传信息
├── logs/                       # 日志目录
│   └── record_center.log      # 程序日志
└── temp/                       # 临时文件目录
```

## 故障排除

### 常见问题

#### 1. 设备检测失败
- **问题**：程序提示 "MTP设备未找到或无法访问"
- **解决方案**：
  - 确保录音笔已正确连接到电脑
  - 尝试重新插拔USB连接线
  - 以管理员身份运行程序
  - 检查设备管理器中是否识别设备

#### 2. PowerShell 访问失败
- **问题**：PowerShell 访问MTP设备时出错
- **解决方案**：
  - 检查PowerShell执行策略：`Get-ExecutionPolicy`
  - 如需要，设置执行策略：`Set-ExecutionPolicy RemoteSigned`
  - 确保Windows PowerShell服务正常运行

#### 3. 备份速度慢
- **问题**：文件复制速度较慢
- **解决方案**：
  - 调整 `max_concurrent` 配置增加并发数
  - 调整 `chunk_size` 配置优化块大小
  - 使用 USB 3.0 或更高版本的接口

#### 4. 断点续传失败
- **问题**：程序无法从中断处继续备份
- **解决方案**：
  - 检查 `temp_dir` 目录是否有足够空间
  - 确保没有杀毒软件阻止临时文件创建
  - 检查磁盘权限设置

### 日志分析

程序会生成详细的日志文件 `logs/record_center.log`：

```
2025/12/09 10:00:33 [INFO] 正在检测SR302录音笔设备...
2025/12/09 10:00:33 [DEBUG] 查找MTP设备: SR302
2025/12/09 10:00:33 [INFO] 文件复制完成（已验证）: ...
```

- **[INFO]**：一般信息
- **[DEBUG]**：调试信息（需要 --verbose 参数）
- **[WARN]**：警告信息
- **[ERROR]**：错误信息

### 配置优化建议

#### 大文件备份优化
```yaml
backup:
  chunk_size: "10MB"              # 增大块大小
  max_concurrent: 2               # 减少并发数避免带宽竞争
  enable_resume: true             # 确保启用断点续传
```

#### 网络存储备份优化
```yaml
backup:
  chunk_size: "2MB"               # 减小块大小
  max_concurrent: 1               # 单文件复制
  resume_interval: "1MB"          # 更频繁保存进度
```

## 开发信息

### 构建环境

- **Go 版本**：1.19 或更高版本
- **CGO**：需要启用（用于Windows API调用）
- **构建标签**：仅支持 Windows（`//go:build windows`）

### 项目结构

```
internal/
├── backup/           # 备份核心功能
├── config/          # 配置管理
├── device/          # 设备检测和访问
├── logger/          # 日志系统
├── progress/        # 进度显示
└── storage/         # 备份记录存储
```

### 依赖项

```go
require (
    github.com/spf13/cobra        # 命令行框架
    github.com/spf13/viper        # 配置管理
    golang.org/x/sys/windows      # Windows API
)
```

## 许可证

本项目采用 MIT 许可证。详见 LICENSE 文件。

## 更新日志

### v1.1.0 (2025-12-09)
- 🆕 **新增设备自动检测命令** `record_center detect`
  - 自动扫描所有USB设备
  - 智能识别录音笔设备
  - 显示设备详细信息（名称、VID、PID）
  - 生成可直接使用的配置片段
  - 特别标记SR302设备

### v1.0.0 (2025-12-09)
- ✅ 实现真实的MTP设备访问（PowerShell + Shell COM）
- ✅ 添加文件完整性验证（SHA256/SHA1/MD5）
- ✅ 实现断点续传功能
- ✅ 支持并发文件复制
- ✅ 添加详细的进度显示和日志
- ✅ 完善的错误处理和回退策略

## 支持和反馈

如果您遇到问题或有改进建议，请：

1. 查看本文档的故障排除部分
2. 检查日志文件获取详细错误信息
3. 在项目 Issues 页面提交问题
4. 提供详细的错误信息和环境描述

---

**注意**：本工具专为 SR302 录音笔设计，设备名称和VID/PID都是硬编码的。如果需要支持其他设备，请修改配置文件中的相关参数。