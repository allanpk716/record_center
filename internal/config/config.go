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
	Source     SourceConfig     `mapstructure:"source" yaml:"source" json:"source"`
	Target     TargetConfig     `mapstructure:"target" yaml:"target" json:"target"`
	Backup     BackupConfig     `mapstructure:"backup" yaml:"backup" json:"backup"`
	Logging    LoggingConfig    `mapstructure:"logging" yaml:"logging" json:"logging"`
	PowerShell PowerShellConfig `mapstructure:"powershell" yaml:"powershell" json:"powershell"`
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
	FileExtensions    []string `mapstructure:"file_extensions" yaml:"file_extensions" json:"file_extensions"`
	SkipExisting      bool     `mapstructure:"skip_existing" yaml:"skip_existing" json:"skip_existing"`
	PreserveStructure bool     `mapstructure:"preserve_structure" yaml:"preserve_structure" json:"preserve_structure"`
	MaxConcurrent     int      `mapstructure:"max_concurrent" yaml:"max_concurrent" json:"max_concurrent"`
	// 新增完整性验证配置
	IntegrityCheck    bool     `mapstructure:"integrity_check" yaml:"integrity_check" json:"integrity_check" default:"true"`
	HashAlgorithm     string   `mapstructure:"hash_algorithm" yaml:"hash_algorithm" json:"hash_algorithm" default:"sha256"`
	// 新增断点续传配置
	EnableResume      bool     `mapstructure:"enable_resume" yaml:"enable_resume" json:"enable_resume" default:"true"`
	ChunkSize         string   `mapstructure:"chunk_size" yaml:"chunk_size" json:"chunk_size" default:"5MB"`
	ResumeInterval    string   `mapstructure:"resume_interval" yaml:"resume_interval" json:"resume_interval" default:"5MB"`
	TempDir           string   `mapstructure:"temp_dir" yaml:"temp_dir" json:"temp_dir" default:"./temp"`
	ResumeMaxAge      string   `mapstructure:"resume_max_age" yaml:"resume_max_age" json:"resume_max_age" default:"24h"`
	// 新增清理空文件夹配置
	CleanEmptyFolders bool     `mapstructure:"clean_empty_folders" yaml:"clean_empty_folders" json:"clean_empty_folders" default:"true"`
}

// 日志配置
type LoggingConfig struct {
	Level       string `mapstructure:"level" yaml:"level" json:"level"`
	File        string `mapstructure:"file" yaml:"file" json:"file"`
	Console     bool   `mapstructure:"console" yaml:"console" json:"console"`
	RotateHours int    `mapstructure:"rotate_hours" yaml:"rotate_hours" json:"rotate_hours"`
	MaxDays     int    `mapstructure:"max_days" yaml:"max_days" json:"max_days"`
}

// PowerShell配置
type PowerShellConfig struct {
	PreferredVersion   string   `mapstructure:"preferred_version" yaml:"preferred_version" json:"preferred_version"`         // "auto", "5.1", "7.x"
	FallbackOrder      []string `mapstructure:"fallback_order" yaml:"fallback_order" json:"fallback_order"`                 // 优先尝试的PowerShell可执行文件
	ExecutionPolicy    string   `mapstructure:"execution_policy" yaml:"execution_policy" json:"execution_policy"`             // "Bypass", "RemoteSigned"
	TimeoutSeconds     int      `mapstructure:"timeout_seconds" yaml:"timeout_seconds" json:"timeout_seconds"`               // 命令超时时间
	CompatibilityMode  string   `mapstructure:"compatibility_mode" yaml:"compatibility_mode" json:"compatibility_mode"`       // "strict"严格模式, "loose"宽松模式
	MaxRetries         int      `mapstructure:"max_retries" yaml:"max_retries" json:"max_retries"`                           // 最大重试次数
	RetryDelaySeconds  int      `mapstructure:"retry_delay_seconds" yaml:"retry_delay_seconds" json:"retry_delay_seconds"`   // 重试延迟
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
		PowerShell: PowerShellConfig{
			PreferredVersion:  "auto",
			FallbackOrder:     []string{"powershell", "pwsh"},
			ExecutionPolicy:   "Bypass",
			TimeoutSeconds:    30,
			CompatibilityMode: "strict",
			MaxRetries:        3,
			RetryDelaySeconds: 1,
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

	// PowerShell配置默认值
	viper.SetDefault("powershell.preferred_version", defaultConfig.PowerShell.PreferredVersion)
	viper.SetDefault("powershell.fallback_order", defaultConfig.PowerShell.FallbackOrder)
	viper.SetDefault("powershell.execution_policy", defaultConfig.PowerShell.ExecutionPolicy)
	viper.SetDefault("powershell.timeout_seconds", defaultConfig.PowerShell.TimeoutSeconds)
	viper.SetDefault("powershell.compatibility_mode", defaultConfig.PowerShell.CompatibilityMode)
	viper.SetDefault("powershell.max_retries", defaultConfig.PowerShell.MaxRetries)
	viper.SetDefault("powershell.retry_delay_seconds", defaultConfig.PowerShell.RetryDelaySeconds)

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

	// 验证PowerShell配置
	if err := validatePowerShellConfig(&config.PowerShell); err != nil {
		return fmt.Errorf("PowerShell配置验证失败: %w", err)
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

// 验证PowerShell配置
func validatePowerShellConfig(config *PowerShellConfig) error {
	// 验证首选版本
	validVersions := []string{"auto", "5.1", "7.x", "5", "7"}
	versionValid := false
	for _, version := range validVersions {
		if config.PreferredVersion == version {
			versionValid = true
			break
		}
	}
	if !versionValid {
		return fmt.Errorf("无效的首选版本: %s，有效值: auto, 5.1, 7.x, 5, 7", config.PreferredVersion)
	}

	// 验证执行策略
	validPolicies := []string{"Bypass", "RemoteSigned", "AllSigned", "Restricted", "Default"}
	policyValid := false
	for _, policy := range validPolicies {
		if config.ExecutionPolicy == policy {
			policyValid = true
			break
		}
	}
	if !policyValid {
		return fmt.Errorf("无效的执行策略: %s，有效值: Bypass, RemoteSigned, AllSigned, Restricted, Default", config.ExecutionPolicy)
	}

	// 验证兼容性模式
	validModes := []string{"strict", "loose"}
	modeValid := false
	for _, mode := range validModes {
		if config.CompatibilityMode == mode {
			modeValid = true
			break
		}
	}
	if !modeValid {
		return fmt.Errorf("无效的兼容性模式: %s，有效值: strict, loose", config.CompatibilityMode)
	}

	// 验证降级顺序
	if len(config.FallbackOrder) == 0 {
		config.FallbackOrder = []string{"powershell", "pwsh"}
	}

	// 验证超时设置
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = 30
	}

	// 验证重试设置
	if config.MaxRetries < 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelaySeconds <= 0 {
		config.RetryDelaySeconds = 1
	}

	return nil
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