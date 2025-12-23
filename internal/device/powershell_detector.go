package device

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// PowerShellVersion 表示检测到的PowerShell版本信息
type PowerShellVersion struct {
	Version    string // 完整版本号，如 "5.1.19041.1682", "7.5.4.0"
	Major      int    // 主版本号，如 5, 7
	Minor      int    // 次版本号，如 1, 5
	Path       string // 可执行文件路径，如 "powershell.exe", "pwsh.exe"
	IsCore     bool   // 是否为PowerShell Core (7+)
	Available  bool   // 是否可用
	LastCheck  time.Time // 最后检查时间
}

// PowerShellDetector PowerShell版本检测器
type PowerShellDetector struct {
	log       *logger.Logger
	versions  []PowerShellVersion
	cacheTime time.Duration // 缓存时间
}

// NewPowerShellDetector 创建PowerShell版本检测器
func NewPowerShellDetector(log *logger.Logger) *PowerShellDetector {
	return &PowerShellDetector{
		log:       log,
		versions:  make([]PowerShellVersion, 0),
		cacheTime: 5 * time.Minute, // 缓存5分钟
	}
}

// DetectAll 检测所有可用的PowerShell版本
func (pd *PowerShellDetector) DetectAll() ([]PowerShellVersion, error) {
	// 检查缓存是否有效
	if len(pd.versions) > 0 && time.Since(pd.versions[0].LastCheck) < pd.cacheTime {
		pd.log.Debug("使用缓存的PowerShell版本信息")
		return pd.versions, nil
	}

	pd.log.Debug("开始检测PowerShell版本")
	pd.versions = make([]PowerShellVersion, 0)

	// 检测Windows PowerShell (powershell.exe)
	if version, err := pd.detectPowerShellVersion("powershell"); err == nil {
		pd.versions = append(pd.versions, version)
		pd.log.Debug("检测到Windows PowerShell: %s (%s)", version.Version, version.Path)
	}

	// 检测PowerShell Core (pwsh.exe)
	if version, err := pd.detectPowerShellVersion("pwsh"); err == nil {
		pd.versions = append(pd.versions, version)
		pd.log.Debug("检测到PowerShell Core: %s (%s)", version.Version, version.Path)
	}

	if len(pd.versions) == 0 {
		return nil, fmt.Errorf("未找到可用的PowerShell版本")
	}

	pd.log.Info("检测到 %d 个PowerShell版本", len(pd.versions))
	return pd.versions, nil
}

// GetPreferredVersion 根据配置获取首选版本
func (pd *PowerShellDetector) GetPreferredVersion(preferred string, fallbackOrder []string) (*PowerShellVersion, error) {
	versions, err := pd.DetectAll()
	if err != nil {
		return nil, err
	}

	// 如果指定了首选版本，优先返回匹配的版本
	if preferred != "auto" {
		for _, version := range versions {
			if pd.versionMatchesPreference(version, preferred) {
				pd.log.Debug("选择首选版本: %s (%s)", version.Version, version.Path)
				return &version, nil
			}
		}
		pd.log.Warn("首选版本 %s 不可用，将使用降级策略", preferred)
	}

	// 根据降级顺序尝试
	for _, exeName := range fallbackOrder {
		for _, version := range versions {
			if strings.EqualFold(version.Path, exeName) || strings.Contains(strings.ToLower(version.Path), strings.ToLower(exeName)) {
				pd.log.Debug("选择降级版本: %s (%s)", version.Version, version.Path)
				return &version, nil
			}
		}
	}

	// 如果降级顺序中没有找到，返回第一个可用版本
	if len(versions) > 0 {
		pd.log.Debug("使用默认版本: %s (%s)", versions[0].Version, versions[0].Path)
		return &versions[0], nil
	}

	return nil, fmt.Errorf("没有可用的PowerShell版本")
}

// detectPowerShellVersion 检测特定PowerShell可执行文件的版本
func (pd *PowerShellDetector) detectPowerShellVersion(exeName string) (PowerShellVersion, error) {
	// 构建命令获取版本信息
	cmd := exec.Command(exeName, "-Command", "$PSVersionTable.PSVersion.ToString()")
	output, err := cmd.Output()
	if err != nil {
		return PowerShellVersion{}, fmt.Errorf("无法执行 %s: %w", exeName, err)
	}

	versionStr := strings.TrimSpace(string(output))
	if versionStr == "" {
		return PowerShellVersion{}, fmt.Errorf("无法获取 %s 版本信息", exeName)
	}

	// 解析版本号
	major, minor, fullVersion, err := pd.parseVersion(versionStr)
	if err != nil {
		return PowerShellVersion{}, fmt.Errorf("解析版本号失败: %w", err)
	}

	// 检查是否可用
	available := pd.testAvailability(exeName)

	version := PowerShellVersion{
		Version:   fullVersion,
		Major:     major,
		Minor:     minor,
		Path:      exeName,
		IsCore:    major >= 7,
		Available: available,
		LastCheck: time.Now(),
	}

	return version, nil
}

// parseVersion 解析版本字符串
func (pd *PowerShellDetector) parseVersion(versionStr string) (major, minor int, fullVersion string, err error) {
	// 使用正则表达式提取版本号
	re := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?.*`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return 0, 0, "", fmt.Errorf("无效的版本格式: %s", versionStr)
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, "", fmt.Errorf("解析主版本号失败: %w", err)
	}

	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, "", fmt.Errorf("解析次版本号失败: %w", err)
	}

	// 构建完整版本字符串
	fullVersion = fmt.Sprintf("%d.%d", major, minor)
	if len(matches) > 3 && matches[3] != "" {
		fullVersion += "." + matches[3]
		if len(matches) > 4 && matches[4] != "" {
			fullVersion += "." + matches[4]
		}
	}

	return major, minor, fullVersion, nil
}

// testAvailability 测试PowerShell是否可用
func (pd *PowerShellDetector) testAvailability(exeName string) bool {
	// 尝试执行简单的命令
	cmd := exec.Command(exeName, "-Command", "Write-Host 'test'")
	err := cmd.Run()
	if err != nil {
		pd.log.Debug("PowerShell %s 不可用: %v", exeName, err)
		return false
	}
	return true
}

// versionMatchesPreference 检查版本是否匹配偏好设置
func (pd *PowerShellDetector) versionMatchesPreference(version PowerShellVersion, preference string) bool {
	switch strings.ToLower(preference) {
	case "5.1":
		return version.Major == 5 && version.Minor == 1
	case "7.x", "7", "core":
		return version.Major >= 7
	case "windows":
		return !version.IsCore
	default:
		// 如果是指定的具体版本号
		if strings.HasPrefix(preference, fmt.Sprintf("%d.%d", version.Major, version.Minor)) {
			return true
		}
	}
	return false
}

// IsAvailable 检查特定版本是否可用
func (pd *PowerShellDetector) IsAvailable(version string) bool {
	versions, err := pd.DetectAll()
	if err != nil {
		return false
	}

	for _, v := range versions {
		if strings.EqualFold(v.Path, version) || strings.Contains(v.Version, version) {
			return v.Available
		}
	}
	return false
}

// ClearCache 清除版本缓存
func (pd *PowerShellDetector) ClearCache() {
	pd.versions = make([]PowerShellVersion, 0)
	pd.log.Debug("清除PowerShell版本缓存")
}