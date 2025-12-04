package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// 配置文件结构
type Config struct {
	Source  SourceConfig  `mapstructure:"source" yaml:"source" json:"source"`
	Target  TargetConfig  `mapstructure:"target" yaml:"target" json:"target"`
	Backup  BackupConfig  `mapstructure:"backup" yaml:"backup" json:"backup"`
	Logging LoggingConfig `mapstructure:"logging" yaml:"logging" json:"logging"`
}

// 源设备配置
type SourceConfig struct {
	DeviceName string `mapstructure:"device_name" yaml:"device_name" json:"device_name"`
	BasePath   string `mapstructure:"base_path" yaml:"base_path" json:"base_path"`
	VID        string `mapstructure:"vid" yaml:"vid" json:"vid"`
	PID        string `mapstructure:"pid" yaml:"pid" json:"pid"`
}

// 目标备份配置
type TargetConfig struct {
	BaseDirectory string `mapstructure:"base_directory" yaml:"base_directory" json:"base_directory"`
	CreateSubdirs bool   `mapstructure:"create_subdirs" yaml:"create_subdirs" json:"create_subdirs"`
}

// 备份配置
type BackupConfig struct {
	FileExtensions   []string `mapstructure:"file_extensions" yaml:"file_extensions" json:"file_extensions"`
	SkipExisting     bool     `mapstructure:"skip_existing" yaml:"skip_existing" json:"skip_existing"`
	PreserveStructure bool     `mapstructure:"preserve_structure" yaml:"preserve_structure" json:"preserve_structure"`
	MaxConcurrent    int      `mapstructure:"max_concurrent" yaml:"max_concurrent" json:"max_concurrent"`
}

// 日志配置
type LoggingConfig struct {
	Level       string `mapstructure:"level" yaml:"level" json:"level"`
	File        string `mapstructure:"file" yaml:"file" json:"file"`
	Console     bool   `mapstructure:"console" yaml:"console" json:"console"`
	RotateHours int    `mapstructure:"rotate_hours" yaml:"rotate_hours" json:"rotate_hours"`
	MaxDays     int    `mapstructure:"max_days" yaml:"max_days" json:"max_days"`
}

// 默认配置
func DefaultConfig() *Config {
	return &Config{
		Source: SourceConfig{
			DeviceName: "SR302",
			BasePath:   "内部共享存储空间\\录音笔文件",
			VID:        "2207",
			PID:        "0011",
		},
		Target: TargetConfig{
			BaseDirectory: "./backups",
			CreateSubdirs: true,
		},
		Backup: BackupConfig{
			FileExtensions:   []string{".opus"},
			SkipExisting:     true,
			PreserveStructure: true,
			MaxConcurrent:    3,
		},
		Logging: LoggingConfig{
			Level:       "info",
			File:        "record_center.log",
			Console:     true,
			RotateHours: 24,
			MaxDays:     7,
		},
	}
}

// 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = createDefaultConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("创建默认配置文件失败: %w", err)
		}
	}

	// 设置配置文件路径和格式
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	defaultConfig := DefaultConfig()
	viper.SetDefault("source.device_name", defaultConfig.Source.DeviceName)
	viper.SetDefault("source.base_path", defaultConfig.Source.BasePath)
	viper.SetDefault("source.vid", defaultConfig.Source.VID)
	viper.SetDefault("source.pid", defaultConfig.Source.PID)
	viper.SetDefault("target.base_directory", defaultConfig.Target.BaseDirectory)
	viper.SetDefault("target.create_subdirs", defaultConfig.Target.CreateSubdirs)
	viper.SetDefault("backup.file_extensions", defaultConfig.Backup.FileExtensions)
	viper.SetDefault("backup.skip_existing", defaultConfig.Backup.SkipExisting)
	viper.SetDefault("backup.preserve_structure", defaultConfig.Backup.PreserveStructure)
	viper.SetDefault("backup.max_concurrent", defaultConfig.Backup.MaxConcurrent)
	viper.SetDefault("logging.level", defaultConfig.Logging.Level)
	viper.SetDefault("logging.file", defaultConfig.Logging.File)
	viper.SetDefault("logging.console", defaultConfig.Logging.Console)
	viper.SetDefault("logging.rotate_hours", defaultConfig.Logging.RotateHours)
	viper.SetDefault("logging.max_days", defaultConfig.Logging.MaxDays)

	// 打印调试信息
	fmt.Printf("配置文件路径: %s\n", configPath)
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("配置文件存在: true\n")
	} else {
		fmt.Printf("配置文件存在: false\n")
	}

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 打印所有配置设置进行调试
	fmt.Printf("Viper读取的所有设置:\n")
	for key, value := range viper.AllSettings() {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// 解析配置到结构体
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 打印配置调试信息
	fmt.Printf("解析后的配置:\n")
	fmt.Printf("  Source.DeviceName: '%s'\n", config.Source.DeviceName)
	fmt.Printf("  Source.BasePath: '%s'\n", config.Source.BasePath)
	fmt.Printf("  Target.BaseDirectory: '%s'\n", config.Target.BaseDirectory)

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 处理相对路径
	config.Target.BaseDirectory = resolvePath(config.Target.BaseDirectory)

	return &config, nil
}

// 创建默认配置文件
func createDefaultConfig(configPath string) error {
	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 获取默认配置
	config := DefaultConfig()

	// 转换为YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// 验证配置
func validateConfig(config *Config) error {
	// 验证源设备配置
	if config.Source.DeviceName == "" {
		return fmt.Errorf("设备名称不能为空")
	}
	if config.Source.BasePath == "" {
		return fmt.Errorf("源路径不能为空")
	}

	// 验证目标目录配置
	if config.Target.BaseDirectory == "" {
		return fmt.Errorf("目标目录不能为空")
	}

	// 验证备份配置
	if len(config.Backup.FileExtensions) == 0 {
		return fmt.Errorf("文件扩展名列表不能为空")
	}
	if config.Backup.MaxConcurrent <= 0 {
		config.Backup.MaxConcurrent = 1
	}

	// 验证日志配置
	validLogLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLogLevels {
		if config.Logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("无效的日志级别: %s", config.Logging.Level)
	}

	if config.Logging.RotateHours <= 0 {
		config.Logging.RotateHours = 24
	}
	if config.Logging.MaxDays <= 0 {
		config.Logging.MaxDays = 7
	}

	return nil
}

// 解析路径（处理相对路径）
func resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path // 如果转换失败，返回原路径
	}

	return absPath
}

// 保存配置
func SaveConfig(config *Config, configPath string) error {
	// 序列化配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}