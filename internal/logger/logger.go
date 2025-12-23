package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// LogFilePermissions 日志文件权限 (0644: 所有者读写，组和其他用户只读)
	LogFilePermissions = 0644
)

// Logger 简单的日志器实现
type Logger struct {
	verbose bool
	logFile *os.File
	logger  *log.Logger
}

// NewLogger 创建新的日志器实例
func NewLogger(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
		logger:  log.New(os.Stdout, "", log.LstdFlags),
	}
}

// InitLogger 初始化日志器
func InitLogger(verbose bool) *Logger {
	logInstance := NewLogger(verbose)
	logInstance.Setup("record_center", "info", "./logs", true, false)
	return logInstance
}

// Setup 设置日志器
func (l *Logger) Setup(name, level, logDir string, console bool, enableContext7 bool) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	// 创建日志文件
	logFileName := name + ".log"
	logFilePath := filepath.Join(logDir, logFileName)

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, LogFilePermissions)
	if err != nil {
		fmt.Printf("创建日志文件失败: %v\n", err)
		return
	}

	l.logFile = file

	// 设置日志器
	if console && file != nil {
		// 同时输出到控制台和文件
		l.logger = log.New(io.MultiWriter(os.Stdout, file), "", log.LstdFlags)
	} else if file != nil {
		// 仅输出到文件
		l.logger = log.New(file, "", log.LstdFlags)
	} else {
		// 仅输出到控制台
		l.logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	// 如果启用context7功能
	if enableContext7 {
		if l.verbose {
			fmt.Println("已启用context7功能")
		}
	}

	// 测试日志
	if l.verbose {
		l.Debug("日志器初始化完成")
		l.Info("日志器设置: 名称=%s, 级别=%s, 目录=%s", name, level, logDir)
	}
}

// Debug 记录调试信息
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		msg := fmt.Sprintf("[DEBUG] "+format, args...)
		l.logger.Println(msg)
	}
}

// Info 记录信息
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf("[INFO] "+format, args...)
	l.logger.Println(msg)
}

// Warn 记录警告信息
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf("[WARN] "+format, args...)
	l.logger.Println(msg)
}

// Error 记录错误信息
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf("[ERROR] "+format, args...)
	l.logger.Println(msg)
}

// Fatal 记录致命错误并退出程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	msg := fmt.Sprintf("[FATAL] "+format, args...)
	l.logger.Println(msg)
	os.Exit(1)
}

// WithContext 添加上下文信息（context7功能）
func (l *Logger) WithContext(key string, value interface{}) *Logger {
	if l.verbose {
		fmt.Printf("[Context] %s: %v\n", key, value)
	}
	return l
}

// WithContexts 添加多个上下文信息
func (l *Logger) WithContexts(contexts map[string]interface{}) *Logger {
	for key, value := range contexts {
		l.WithContext(key, value)
	}
	return l
}

// SetLevel 动态设置日志级别
func (l *Logger) SetLevel(level string) {
	// 在这个简单实现中，我们只通过verbose标志控制debug输出
	l.verbose = strings.ToLower(level) == "debug"
}

// GetLogFile 获取当前日志文件路径
func (l *Logger) GetLogFile() string {
	if l.logFile != nil {
		return l.logFile.Name()
	}
	return ""
}

// Close 关闭日志器
func (l *Logger) Close() {
	if l.logFile != nil {
		l.Info("日志器关闭")
		l.logFile.Close()
	}
}

// 日志级别常量
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// 日志级别映射（用于验证和转换）
var logLevelMap = map[string]int{
	LevelDebug: 0,
	LevelInfo:  1,
	LevelWarn:  2,
	LevelError: 3,
}

// IsValidLogLevel 检查日志级别是否有效
func IsValidLogLevel(level string) bool {
	_, exists := logLevelMap[level]
	return exists
}

// GetLogLevels 获取所有有效的日志级别
func GetLogLevels() []string {
	levels := make([]string, 0, len(logLevelMap))
	for level := range logLevelMap {
		levels = append(levels, level)
	}
	return levels
}