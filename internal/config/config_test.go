package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("默认配置为空")
	}

	// 验证源设备配置
	if config.Source.DeviceName != "SR302" {
		t.Errorf("期望设备名称为 'SR302'，实际为 '%s'", config.Source.DeviceName)
	}
	if config.Source.BasePath != "内部共享存储空间\\录音笔文件" {
		t.Errorf("期望基础路径为 '内部共享存储空间\\录音笔文件'，实际为 '%s'", config.Source.BasePath)
	}
	if config.Source.VID != "2207" {
		t.Errorf("期望VID为 '2207'，实际为 '%s'", config.Source.VID)
	}
	if config.Source.PID != "0011" {
		t.Errorf("期望PID为 '0011'，实际为 '%s'", config.Source.PID)
	}

	// 验证目标配置
	if config.Target.BaseDirectory != "./backups" {
		t.Errorf("期望目标目录为 './backups'，实际为 '%s'", config.Target.BaseDirectory)
	}
	if !config.Target.CreateSubdirs {
		t.Error("期望创建子目录为 true")
	}

	// 验证备份配置
	if len(config.Backup.FileExtensions) != 1 {
		t.Fatalf("期望文件扩展名数量为 1，实际为 %d", len(config.Backup.FileExtensions))
	}
	if config.Backup.FileExtensions[0] != ".opus" {
		t.Errorf("期望文件扩展名为 '.opus'，实际为 '%s'", config.Backup.FileExtensions[0])
	}
	if !config.Backup.SkipExisting {
		t.Error("期望跳过已存在文件为 true")
	}
	if !config.Backup.PreserveStructure {
		t.Error("期望保留结构为 true")
	}
	if config.Backup.MaxConcurrent != 3 {
		t.Errorf("期望最大并发数为 3，实际为 %d", config.Backup.MaxConcurrent)
	}

	// 验证日志配置
	if config.Logging.Level != "info" {
		t.Errorf("期望日志级别为 'info'，实际为 '%s'", config.Logging.Level)
	}
	if config.Logging.File != "record_center.log" {
		t.Errorf("期望日志文件为 'record_center.log'，实际为 '%s'", config.Logging.File)
	}
	if !config.Logging.Console {
		t.Error("期望控制台输出为 true")
	}
	if config.Logging.RotateHours != 24 {
		t.Errorf("期望轮转时间为 24，实际为 %d", config.Logging.RotateHours)
	}
	if config.Logging.MaxDays != 7 {
		t.Errorf("期望保留天数为 7，实际为 %d", config.Logging.MaxDays)
	}
}

// TestLoadConfig_CreateDefault 测试创建默认配置文件
func TestLoadConfig_CreateDefault(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	// 确保文件不存在
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Fatal("配置文件不应该存在")
	}

	// 加载配置（应该创建默认文件）
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置文件已创建
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("配置文件未创建")
	}

	// 验证配置内容
	if config.Source.DeviceName != "SR302" {
		t.Errorf("期望设备名称为 'SR302'，实际为 '%s'", config.Source.DeviceName)
	}

	// 读取并验证YAML文件内容
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}

	var yamlConfig Config
	err = yaml.Unmarshal(data, &yamlConfig)
	if err != nil {
		t.Fatalf("解析YAML配置失败: %v", err)
	}

	if yamlConfig.Source.DeviceName != "SR302" {
		t.Errorf("YAML中设备名称为 '%s'，期望为 'SR302'", yamlConfig.Source.DeviceName)
	}
}

// TestLoadConfig_ExistingFile 测试加载已存在的配置文件
func TestLoadConfig_ExistingFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	// 创建自定义配置文件
	customConfig := Config{
		Source: SourceConfig{
			DeviceName: "CustomDevice",
			BasePath:   "/custom/path",
			VID:        "1234",
			PID:        "5678",
		},
		Target: TargetConfig{
			BaseDirectory: "/custom/backup",
			CreateSubdirs: false,
		},
		Backup: BackupConfig{
			FileExtensions: []string{".mp3", ".wav"},
			MaxConcurrent:  5,
		},
		Logging: LoggingConfig{
			Level: "debug",
			File:  "custom.log",
		},
	}

	// 写入配置文件
	data, err := yaml.Marshal(customConfig)
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 加载配置
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证加载的配置
	if config.Source.DeviceName != "CustomDevice" {
		t.Errorf("期望设备名称为 'CustomDevice'，实际为 '%s'", config.Source.DeviceName)
	}
	if config.Source.VID != "1234" {
		t.Errorf("期望VID为 '1234'，实际为 '%s'", config.Source.VID)
	}
	if config.Target.BaseDirectory != "/custom/backup" {
		t.Errorf("期望目标目录为 '/custom/backup'，实际为 '%s'", config.Target.BaseDirectory)
	}
	if config.Backup.MaxConcurrent != 5 {
		t.Errorf("期望最大并发数为 5，实际为 %d", config.Backup.MaxConcurrent)
	}
}

// TestLoadConfig_InvalidYAML 测试加载无效的YAML文件
func TestLoadConfig_InvalidYAML(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid_config.yaml")

	// 创建无效的YAML文件
	invalidYAML := `
source:
  device_name: "TestDevice"
  base_path: /test
  vid: "2207"
  pid: "0011"
invalid_yaml: [unclosed array
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("写入无效YAML文件失败: %v", err)
	}

	// 尝试加载配置
	_, err = LoadConfig(configFile)
	if err == nil {
		t.Fatal("加载无效YAML应该返回错误")
	}
}

// TestValidateConfig 测试配置验证
func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效配置",
			config: Config{
				Source: SourceConfig{
					DeviceName: "SR302",
					BasePath:   "/test/path",
				},
				Target: TargetConfig{
					BaseDirectory: "/backup",
				},
				Backup: BackupConfig{
					FileExtensions: []string{".opus"},
					MaxConcurrent:  3,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: false,
		},
		{
			name: "空设备名称",
			config: Config{
				Source: SourceConfig{
					DeviceName: "",
					BasePath:   "/test/path",
				},
				Target: TargetConfig{
					BaseDirectory: "/backup",
				},
				Backup: BackupConfig{
					FileExtensions: []string{".opus"},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "设备名称不能为空",
		},
		{
			name: "空源路径",
			config: Config{
				Source: SourceConfig{
					DeviceName: "SR302",
					BasePath:   "",
				},
				Target: TargetConfig{
					BaseDirectory: "/backup",
				},
				Backup: BackupConfig{
					FileExtensions: []string{".opus"},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "源路径不能为空",
		},
		{
			name: "空目标目录",
			config: Config{
				Source: SourceConfig{
					DeviceName: "SR302",
					BasePath:   "/test/path",
				},
				Target: TargetConfig{
					BaseDirectory: "",
				},
				Backup: BackupConfig{
					FileExtensions: []string{".opus"},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "目标目录不能为空",
		},
		{
			name: "空文件扩展名列表",
			config: Config{
				Source: SourceConfig{
					DeviceName: "SR302",
					BasePath:   "/test/path",
				},
				Target: TargetConfig{
					BaseDirectory: "/backup",
				},
				Backup: BackupConfig{
					FileExtensions: []string{},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorMsg:    "文件扩展名列表不能为空",
		},
		{
			name: "无效日志级别",
			config: Config{
				Source: SourceConfig{
					DeviceName: "SR302",
					BasePath:   "/test/path",
				},
				Target: TargetConfig{
					BaseDirectory: "/backup",
				},
				Backup: BackupConfig{
					FileExtensions: []string{".opus"},
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "无效的日志级别",
		},
		{
			name: "负数并发数",
			config: Config{
				Source: SourceConfig{
					DeviceName: "SR302",
					BasePath:   "/test/path",
				},
				Target: TargetConfig{
					BaseDirectory: "/backup",
				},
				Backup: BackupConfig{
					FileExtensions: []string{".opus"},
					MaxConcurrent:  -1,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: false, // 应该被修正为1
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(&tc.config)

			if tc.expectError {
				if err == nil {
					t.Errorf("期望返回错误: %s", tc.errorMsg)
				} else if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					// 检查错误消息是否包含期望的内容
					if !contains(err.Error(), tc.errorMsg) {
						t.Errorf("期望错误消息包含 '%s'，实际为 '%s'", tc.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望返回错误，但得到: %v", err)
				}
			}
		})
	}
}

// TestResolvePath 测试路径解析
func TestResolvePath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			input:    "relative/path",
			expected: "", // 相对路径会被转换为绝对路径，但具体值取决于测试环境
		},
		{
			input:    "./current/path",
			expected: "", // 相对路径会被转换为绝对路径
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := resolvePath(tc.input)

			// 对于绝对路径，应该保持不变
			if filepath.IsAbs(tc.input) {
				if result != tc.expected {
					t.Errorf("期望路径为 '%s'，实际为 '%s'", tc.expected, result)
				}
			} else {
				// 对于相对路径，应该转换为绝对路径
				if !filepath.IsAbs(result) && tc.input != "" {
					t.Errorf("相对路径应该转换为绝对路径，但得到: '%s'", result)
				}
			}
		})
	}
}

// TestSaveConfig 测试保存配置
func TestSaveConfig(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "save_test_config.yaml")

	// 创建配置
	config := &Config{
		Source: SourceConfig{
			DeviceName: "TestDevice",
			BasePath:   "/test/path",
			VID:        "2207",
			PID:        "0011",
		},
		Target: TargetConfig{
			BaseDirectory: "/test/backup",
			CreateSubdirs: true,
		},
		Backup: BackupConfig{
			FileExtensions: []string{".opus", ".mp3"},
			MaxConcurrent:  5,
		},
		Logging: LoggingConfig{
			Level:       "debug",
			File:        "test.log",
			Console:     false,
			RotateHours: 12,
			MaxDays:     30,
		},
	}

	// 保存配置
	err := SaveConfig(config, configFile)
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("配置文件未保存")
	}

	// 读取并验证内容
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}

	var savedConfig Config
	err = yaml.Unmarshal(data, &savedConfig)
	if err != nil {
		t.Fatalf("解析保存的配置失败: %v", err)
	}

	// 验证保存的内容
	if savedConfig.Source.DeviceName != "TestDevice" {
		t.Errorf("保存的设备名称为 '%s'，期望为 'TestDevice'", savedConfig.Source.DeviceName)
	}
	if savedConfig.Backup.MaxConcurrent != 5 {
		t.Errorf("保存的最大并发数为 %d，期望为 5", savedConfig.Backup.MaxConcurrent)
	}
	if savedConfig.Logging.MaxDays != 30 {
		t.Errorf("保存的最大天数为 %d，期望为 30", savedConfig.Logging.MaxDays)
	}
}

// TestSaveConfig_CreateDir 测试保存配置时创建目录
func TestSaveConfig_CreateDir(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "subdir", "nested_config.yaml")

	// 确保目录不存在
	subdir := filepath.Dir(configFile)
	if _, err := os.Stat(subdir); !os.IsNotExist(err) {
		t.Fatal("子目录不应该存在")
	}

	// 创建配置
	config := DefaultConfig()

	// 保存配置（应该创建目录）
	err := SaveConfig(config, configFile)
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 验证目录已创建
	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Fatal("子目录未创建")
	}

	// 验证文件存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("配置文件未保存")
	}
}

// TestLoadConfig_WithDefaults 测试配置加载时的默认值处理
func TestLoadConfig_WithDefaults(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "partial_config.yaml")

	// 创建部分配置文件（只包含部分字段）
	partialConfig := `
source:
  device_name: "PartialDevice"
target:
  base_directory: "/partial/backup"
`

	err := os.WriteFile(configFile, []byte(partialConfig), 0644)
	if err != nil {
		t.Fatalf("写入部分配置文件失败: %v", err)
	}

	// 加载配置
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证明确设置的值
	if config.Source.DeviceName != "PartialDevice" {
		t.Errorf("设备名称为 '%s'，期望为 'PartialDevice'", config.Source.DeviceName)
	}
	if config.Target.BaseDirectory != "/partial/backup" {
		t.Errorf("目标目录为 '%s'，期望为 '/partial/backup'", config.Target.BaseDirectory)
	}

	// 验证默认值
	if config.Source.VID != "2207" {
		t.Errorf("VID为 '%s'，期望使用默认值 '2207'", config.Source.VID)
	}
	if config.Source.PID != "0011" {
		t.Errorf("PID为 '%s'，期望使用默认值 '0011'", config.Source.PID)
	}
	if len(config.Backup.FileExtensions) == 0 {
		t.Error("文件扩展名列表应该有默认值")
	}
}

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}