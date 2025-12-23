package device

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// PowerShellConfig PowerShell配置 (临时定义，应该使用config包中的定义)
type PowerShellConfig struct {
	PreferredVersion   string   `mapstructure:"preferred_version" yaml:"preferred_version" json:"preferred_version"`         // "auto", "5.1", "7.x"
	FallbackOrder      []string `mapstructure:"fallback_order" yaml:"fallback_order" json:"fallback_order"`                 // 优先尝试的PowerShell可执行文件
	ExecutionPolicy    string   `mapstructure:"execution_policy" yaml:"execution_policy" json:"execution_policy"`             // "Bypass", "RemoteSigned"
	TimeoutSeconds     int      `mapstructure:"timeout_seconds" yaml:"timeout_seconds" json:"timeout_seconds"`               // 命令超时时间
	CompatibilityMode  string   `mapstructure:"compatibility_mode" yaml:"compatibility_mode" json:"compatibility_mode"`       // "strict"严格模式, "loose"宽松模式
	MaxRetries         int      `mapstructure:"max_retries" yaml:"max_retries" json:"max_retries"`                           // 最大重试次数
	RetryDelaySeconds  int      `mapstructure:"retry_delay_seconds" yaml:"retry_delay_seconds" json:"retry_delay_seconds"`   // 重试延迟
}


// ExecutionResult 执行结果
type ExecutionResult struct {
	Output    string
	Error     error
	ExitCode  int
	Version   string  // 使用的PowerShell版本
	ExePath   string  // 使用的可执行文件路径
	Duration  time.Duration // 执行耗时
}

// PowerShellManager PowerShell管理器
type PowerShellManager struct {
	detector   *PowerShellDetector
	config     *PowerShellConfig
	log        *logger.Logger
	lastUsed   *PowerShellVersion // 最后成功使用的版本
}

// NewPowerShellManager 创建PowerShell管理器
func NewPowerShellManager(log *logger.Logger, config *PowerShellConfig) *PowerShellManager {
	// 设置默认配置
	if config == nil {
		config = &PowerShellConfig{
			PreferredVersion:  "auto",
			FallbackOrder:     []string{"powershell", "pwsh"},
			ExecutionPolicy:   "Bypass",
			TimeoutSeconds:    30,
			CompatibilityMode: "strict",
			MaxRetries:        3,
			RetryDelaySeconds: 1,
		}
	}

	return &PowerShellManager{
		detector: NewPowerShellDetector(log),
		config:   config,
		log:      log,
	}
}

// GetAvailableVersions 获取所有可用版本
func (pm *PowerShellManager) GetAvailableVersions() ([]PowerShellVersion, error) {
	return pm.detector.DetectAll()
}

// GetPreferredVersion 获取首选版本
func (pm *PowerShellManager) GetPreferredVersion() (*PowerShellVersion, error) {
	preferred := pm.config.PreferredVersion
	fallbackOrder := pm.config.FallbackOrder

	// 如果有上次成功使用的版本且仍然可用，优先使用
	if pm.lastUsed != nil && pm.detector.IsAvailable(pm.lastUsed.Path) {
		pm.log.Debug("复用上次成功的PowerShell版本: %s (%s)", pm.lastUsed.Version, pm.lastUsed.Path)
		return pm.lastUsed, nil
	}

	version, err := pm.detector.GetPreferredVersion(preferred, fallbackOrder)
	if err != nil {
		return nil, fmt.Errorf("获取首选PowerShell版本失败: %w", err)
	}

	return version, nil
}

// ExecuteCommand 执行PowerShell命令
func (pm *PowerShellManager) ExecuteCommand(command string, args ...string) (*ExecutionResult, error) {
	version, err := pm.GetPreferredVersion()
	if err != nil {
		return nil, fmt.Errorf("无法获取PowerShell版本: %w", err)
	}

	return pm.executeWithVersion(version, command, args...)
}

// ExecuteScript 执行PowerShell脚本
func (pm *PowerShellManager) ExecuteScript(script string) (*ExecutionResult, error) {
	version, err := pm.GetPreferredVersion()
	if err != nil {
		return nil, fmt.Errorf("无法获取PowerShell版本: %w", err)
	}

	// 构建脚本执行命令
	args := []string{"-ExecutionPolicy", pm.config.ExecutionPolicy, "-Command", script}
	return pm.executeWithVersion(version, args[0], args[1:]...)
}

// executeWithVersion 使用指定版本执行
func (pm *PowerShellManager) executeWithVersion(version *PowerShellVersion, command string, args ...string) (*ExecutionResult, error) {
	startTime := time.Now()

	// 构建完整命令
	allArgs := append([]string{command}, args...)

	pm.log.Debug("执行PowerShell命令: %s %s", version.Path, strings.Join(allArgs, " "))

	// 执行命令（带重试机制）
	var result *ExecutionResult
	var lastErr error

	for attempt := 0; attempt <= pm.config.MaxRetries; attempt++ {
		if attempt > 0 {
			pm.log.Debug("PowerShell命令执行重试 %d/%d", attempt, pm.config.MaxRetries)
			time.Sleep(time.Duration(pm.config.RetryDelaySeconds) * time.Second)
		}

		// 每次重试时重新创建cmd对象以避免stdout重复设置
		cmd := exec.Command(version.Path, allArgs...)

		// 设置超时（每次重试都需要新的超时控制）
		var timer *time.Timer
		if pm.config.TimeoutSeconds > 0 {
			timer = time.AfterFunc(time.Duration(pm.config.TimeoutSeconds)*time.Second, func() {
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			})
		}

		output, err := cmd.Output()

		// 清理超时定时器
		if timer != nil {
			timer.Stop()
		}
		result = &ExecutionResult{
			Output:   string(output),
			Error:    err,
			Version:  version.Version,
			ExePath:  version.Path,
			Duration: time.Since(startTime),
		}

		if err == nil {
			// 成功执行，记录使用的版本
			pm.lastUsed = version
			pm.log.Debug("PowerShell命令执行成功，耗时: %v", result.Duration)
			return result, nil
		}

		lastErr = err
		pm.log.Debug("PowerShell命令执行失败 (尝试 %d/%d): %v", attempt+1, pm.config.MaxRetries+1, err)

		// 如果是版本兼容性问题，尝试降级
		if pm.isCompatibilityError(err) && attempt == pm.config.MaxRetries {
			if fallbackVersion, fallbackErr := pm.tryFallbackVersion(version); fallbackErr == nil {
				pm.log.Info("尝试使用降级版本 %s", fallbackVersion.Path)
				return pm.executeWithVersion(fallbackVersion, command, args...)
			}
		}
	}

	return result, fmt.Errorf("PowerShell命令执行失败，已重试 %d 次: %w", pm.config.MaxRetries, lastErr)
}

// tryFallbackVersion 尝试使用降级版本
func (pm *PowerShellManager) tryFallbackVersion(currentVersion *PowerShellVersion) (*PowerShellVersion, error) {
	versions, err := pm.detector.DetectAll()
	if err != nil {
		return nil, err
	}

	// 寻找其他可用版本
	for _, version := range versions {
		if version.Path != currentVersion.Path && version.Available {
			pm.log.Debug("找到降级版本: %s (%s)", version.Version, version.Path)
			return &version, nil
		}
	}

	return nil, fmt.Errorf("没有可用的降级版本")
}

// isCompatibilityError 判断是否为兼容性错误
func (pm *PowerShellManager) isCompatibilityError(err error) bool {
	errStr := strings.ToLower(err.Error())

	// 检查常见的兼容性错误模式
	compatibilityErrors := []string{
		"executionpolicy",
		"cannot be loaded",
		"security error",
		"com object",
		"shell.application",
		"namespace",
		"mtp",
		"portable device",
	}

	for _, pattern := range compatibilityErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// TestExecution 测试PowerShell执行
func (pm *PowerShellManager) TestExecution() error {
	result, err := pm.ExecuteCommand("-Command", "Write-Host 'PowerShell Test'")
	if err != nil {
		return fmt.Errorf("PowerShell执行测试失败: %w", err)
	}

	if !strings.Contains(result.Output, "PowerShell Test") {
		return fmt.Errorf("PowerShell执行测试返回异常结果: %s", result.Output)
	}

	pm.log.Info("PowerShell执行测试成功，版本: %s", result.Version)
	return nil
}

// GetExecutionPolicy 获取执行策略
func (pm *PowerShellManager) GetExecutionPolicy() (string, error) {
	result, err := pm.ExecuteCommand("-Command", "Get-ExecutionPolicy")
	if err != nil {
		return "", fmt.Errorf("获取执行策略失败: %w", err)
	}

	policy := strings.TrimSpace(result.Output)
	pm.log.Debug("PowerShell执行策略: %s", policy)
	return policy, nil
}

// SetExecutionPolicy 设置执行策略（如果权限允许）
func (pm *PowerShellManager) SetExecutionPolicy(policy string) error {
	script := fmt.Sprintf("Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy %s -Force", policy)
	result, err := pm.ExecuteScript(script)
	if err != nil {
		return fmt.Errorf("设置执行策略失败: %w", err)
	}

	pm.log.Info("PowerShell执行策略已设置为: %s", policy)
	_ = result // 避免未使用变量警告
	return nil
}

// CheckCOMObjectAccess 检查COM对象访问权限
func (pm *PowerShellManager) CheckCOMObjectAccess() error {
	script := `
$shell = New-Object -ComObject Shell.Application
if ($shell) {
    "SUCCESS"
} else {
    "FAILED"
}
`
	result, err := pm.ExecuteScript(script)
	if err != nil {
		return fmt.Errorf("COM对象访问测试失败: %w", err)
	}

	if !strings.Contains(result.Output, "SUCCESS") {
		return fmt.Errorf("COM对象访问不可用")
	}

	pm.log.Debug("COM对象访问检查通过")
	return nil
}

// GetVersionInfo 获取详细的版本信息
func (pm *PowerShellManager) GetVersionInfo() (map[string]interface{}, error) {
	script := `
$version = $PSVersionTable.PSVersion
[PSCustomObject]@{
    Major = $version.Major
    Minor = $version.Minor
    Build = $version.Build
    Revision = $version.Revision
    PSVersion = $version.ToString()
    PSEdition = $PSVersionTable.PSEdition
    PSCompatibleVersions = $PSVersionTable.PSCompatibleVersions -join ", "
    CLRVersion = $PSVersionTable.CLRVersion
    GitCommitId = $PSVersionTable.GitCommitId
    OS = $PSVersionTable.OS
    Platform = $PSVersionTable.Platform
} | ConvertTo-Json -Depth 3
`
	result, err := pm.ExecuteScript(script)
	if err != nil {
		return nil, fmt.Errorf("获取PowerShell版本信息失败: %w", err)
	}

	pm.log.Debug("PowerShell版本信息: %s", result.Output)

	// 这里可以进一步解析JSON返回map[string]interface{}
	// 为了简化，直接返回基本信息
	info := map[string]interface{}{
		"raw_output": result.Output,
		"version":    result.Version,
		"exe_path":   result.ExePath,
	}

	return info, nil
}

// UpdateConfig 更新配置
func (pm *PowerShellManager) UpdateConfig(config *PowerShellConfig) {
	pm.config = config
	pm.log.Debug("PowerShell管理器配置已更新")
}

// ClearCache 清除缓存
func (pm *PowerShellManager) ClearCache() {
	pm.detector.ClearCache()
	pm.lastUsed = nil
	pm.log.Debug("PowerShell管理器缓存已清除")
}