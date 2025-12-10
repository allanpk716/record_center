//go:build windows

package device

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/allanpk716/record_center/internal/logger"
)

// WindowsShellResolver Windows Shell COM路径解析器
type WindowsShellResolver struct {
	log     *logger.Logger
	priority int
}

// NewWindowsShellResolver 创建Windows Shell解析器
func NewWindowsShellResolver(log *logger.Logger) *WindowsShellResolver {
	return &WindowsShellResolver{
		log:     log,
		priority: 100, // 最高优先级
	}
}

// Resolve 解析设备路径
func (wsr *WindowsShellResolver) Resolve(deviceName, vid, pid string) (string, error) {
	wsr.log.Debug("使用Windows Shell COM解析器: %s", deviceName)

	// 暂时禁用CGO，返回错误
	return "", fmt.Errorf("Windows Shell COM暂时禁用")
}

// GetPriority 获取优先级
func (wsr *WindowsShellResolver) GetPriority() int {
	return wsr.priority
}

// IsAvailable 检查是否可用
func (wsr *WindowsShellResolver) IsAvailable() bool {
	// 检查CGO是否可用
	// 这里可以添加更详细的检查
	return true
}

// PowerShellResolver PowerShell路径解析器
type PowerShellResolver struct {
	log     *logger.Logger
	priority int
}

// NewPowerShellResolver 创建PowerShell解析器
func NewPowerShellResolver(log *logger.Logger) *PowerShellResolver {
	return &PowerShellResolver{
		log:     log,
		priority: 80, // 中等优先级
	}
}

// Resolve 解析设备路径
func (psr *PowerShellResolver) Resolve(deviceName, vid, pid string) (string, error) {
	psr.log.Debug("使用PowerShell解析器: %s", deviceName)

	// 创建PowerShell MTP访问器
	psAccessor := NewPowerShellMTPAccessor(psr.log)
	if psAccessor == nil {
		return "", fmt.Errorf("PowerShell访问器创建失败")
	}

	// 获取设备路径
	devicePath, err := psAccessor.GetMTPDevicePath(deviceName)
	if err != nil {
		psr.log.Debug("PowerShell路径获取失败: %v", err)
		return "", err
	}

	return devicePath, nil
}

// GetPriority 获取优先级
func (psr *PowerShellResolver) GetPriority() int {
	return psr.priority
}

// IsAvailable 检查是否可用
func (psr *PowerShellResolver) IsAvailable() bool {
	// 检查PowerShell是否可用
	cmd := exec.Command("powershell", "-Command", "Get-Host")
	err := cmd.Run()
	return err == nil
}

// WMIResolver WMI路径解析器
type WMIResolver struct {
	log     *logger.Logger
	priority int
}

// NewWMIResolver 创建WMI解析器
func NewWMIResolver(log *logger.Logger) *WMIResolver {
	return &WMIResolver{
		log:     log,
		priority: 60, // 较低优先级
	}
}

// Resolve 解析设备路径
func (wmir *WMIResolver) Resolve(deviceName, vid, pid string) (string, error) {
	wmir.log.Debug("使用WMI解析器: %s", deviceName)

	// 构建WMI查询脚本
	script := fmt.Sprintf(`
Get-WmiObject Win32_PnPEntity |
Where-Object { $_.DeviceID -like "*VID_%s*" -and $_.DeviceID -like "*PID_%s*" } |
Select-Object -First 1 |
ForEach-Object {
    $devicePath = "\\?\Volume{" + $_.DeviceID.Split("\")[2].Split("&")[0] + "}"
    if (Test-Path $devicePath) {
        $devicePath
    } else {
        "NOT_FOUND"
    }
}
`, vid, pid)

	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		wmir.log.Debug("WMI查询失败: %v", err)
		return "", err
	}

	result := strings.TrimSpace(string(output))
	if result == "NOT_FOUND" || result == "" {
		return "", fmt.Errorf("WMI未找到设备路径")
	}

	return result, nil
}

// GetPriority 获取优先级
func (wmir *WMIResolver) GetPriority() int {
	return wmir.priority
}

// IsAvailable 检查是否可用
func (wmir *WMIResolver) IsAvailable() bool {
	// 检查WMI是否可用
	cmd := exec.Command("wmic", "os", "get", "version")
	err := cmd.Run()
	return err == nil
}

// DirectFileResolver 直接文件系统解析器
type DirectFileResolver struct {
	log      *logger.Logger
	priority int
}

// NewDirectFileResolver 创建直接文件解析器
func NewDirectFileResolver(log *logger.Logger) *DirectFileResolver {
	return &DirectFileResolver{
		log:      log,
		priority: 40, // 最低优先级
	}
}

// Resolve 解析设备路径
func (dfr *DirectFileResolver) Resolve(deviceName, vid, pid string) (string, error) {
	dfr.log.Debug("使用直接文件系统解析器: %s", deviceName)

	// 尝试常见的设备挂载路径
	pathPatterns := []string{
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Microsoft", "Windows", "Explorer", "shell", deviceName),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Packages\\Microsoft.Windows.Explorer_*\\LocalCache\\Microsoft\\Windows\\Explorer\\shell", deviceName),
		`\\?\USB#VID_` + vid + `&PID_` + pid + `#` + deviceName + `#{a5dcbf10-6530-11d2-901f-00c04fb951ed}`,
		`\\?\` + deviceName,
		deviceName, // 直接设备名称
	}

	for _, pattern := range pathPatterns {
		// 处理通配符路径
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if dfr.testPathAccessibility(match) {
				dfr.log.Debug("找到可访问的设备路径: %s", match)
				return match, nil
			}
		}

		// 如果没有匹配项，直接测试模式字符串
		if dfr.testPathAccessibility(pattern) {
			dfr.log.Debug("找到可访问的设备路径: %s", pattern)
			return pattern, nil
		}
	}

	return "", fmt.Errorf("未找到可访问的设备路径")
}

// GetPriority 获取优先级
func (dfr *DirectFileResolver) GetPriority() int {
	return dfr.priority
}

// IsAvailable 检查是否可用
func (dfr *DirectFileResolver) IsAvailable() bool {
	return true // 直接文件系统访问总是可用的
}

// PowerShellEnhancedResolver 增强的PowerShell路径解析器
type PowerShellEnhancedResolver struct {
	log     *logger.Logger
	priority int
}

// NewPowerShellEnhancedResolver 创建增强的PowerShell解析器
func NewPowerShellEnhancedResolver(log *logger.Logger) *PowerShellEnhancedResolver {
	return &PowerShellEnhancedResolver{
		log:     log,
		priority: 120, // 最高优先级
	}
}

// Resolve 解析设备路径
func (pser *PowerShellEnhancedResolver) Resolve(deviceName, vid, pid string) (string, error) {
	pser.log.Debug("使用增强PowerShell解析器: %s", deviceName)

	// 创建增强PowerShell访问器
	enhanced := NewPowerShellEnhanced(pser.log)
	if enhanced == nil {
		return "", fmt.Errorf("增强PowerShell访问器创建失败")
	}

	// 尝试连接设备
	err := enhanced.ConnectToDevice(deviceName, vid, pid)
	if err != nil {
		pser.log.Debug("增强PowerShell连接失败: %v", err)
		return "", err
	}
	defer enhanced.Close()

	// 使用便携式设备命名空间查找路径
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $portable.Items() | Where-Object { $_.Name -like "*%s*" } | ForEach-Object {
        $_.Path()
    }
}
`, deviceName)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		pser.log.Debug("增强PowerShell路径获取失败: %v", err)
		return "", err
	}

	devicePath := strings.TrimSpace(string(output))
	if devicePath == "" {
		return "", fmt.Errorf("增强PowerShell未找到设备路径")
	}

	return devicePath, nil
}

// GetPriority 获取优先级
func (pser *PowerShellEnhancedResolver) GetPriority() int {
	return pser.priority
}

// IsAvailable 检查是否可用
func (pser *PowerShellEnhancedResolver) IsAvailable() bool {
	// 检查PowerShell是否可用以及执行策略
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", "Get-Host")
	err := cmd.Run()
	if err != nil {
		return false
	}

	// 检查COM对象是否可用
	comCmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", "$shell = New-Object -ComObject Shell.Application; $shell.Name")
	comErr := comCmd.Run()
	return comErr == nil
}

// testPathAccessibility 测试路径是否可访问
func (dfr *DirectFileResolver) testPathAccessibility(path string) bool {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf("Test-Path '%s'", path))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "True"
}

// NewWMIMTPAccessor 创建WMI MTP访问器（占位符实现）
func NewWMIMTPAccessor(log *logger.Logger) MTPInterface {
	return &WMIMTPAccessor{
		log: log,
	}
}

// WMIMTPAccessor WMI MTP访问器
type WMIMTPAccessor struct {
	log      *logger.Logger
	connected bool
	device   *DeviceInfo
}

// ConnectToDevice 连接设备
func (wmi *WMIMTPAccessor) ConnectToDevice(deviceName, vid, pid string) error {
	wmi.log.Debug("WMI MTP连接设备: %s", deviceName)
	wmi.connected = true
	wmi.device = &DeviceInfo{
		Name:      deviceName,
		VID:       vid,
		PID:       pid,
		DeviceID:  fmt.Sprintf("USB\\VID_%s&PID_%s", vid, pid),
	}
	return nil
}

// ListFiles 列出文件
func (wmi *WMIMTPAccessor) ListFiles(basePath string) ([]*FileInfo, error) {
	wmi.log.Debug("WMI MTP列出文件: %s", basePath)
	// WMI主要用于设备管理，文件访问需要降级到其他方法
	return []*FileInfo{}, nil
}

// GetFileStream 获取文件流
func (wmi *WMIMTPAccessor) GetFileStream(filePath string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("WMI不支持文件流访问")
}

// Close 关闭连接
func (wmi *WMIMTPAccessor) Close() error {
	wmi.connected = false
	wmi.device = nil
	return nil
}

// IsConnected 检查连接状态
func (wmi *WMIMTPAccessor) IsConnected() bool {
	return wmi.connected
}

// GetDeviceInfo 获取设备信息
func (wmi *WMIMTPAccessor) GetDeviceInfo() *DeviceInfo {
	return wmi.device
}

// NewDirectFileAccessor 创建直接文件访问器
func NewDirectFileAccessor(log *logger.Logger, devicePath string) MTPInterface {
	return &DirectFileAccessor{
		log:        log,
		devicePath: devicePath,
		connected:  true,
	}
}

// DirectFileAccessor 直接文件访问器
type DirectFileAccessor struct {
	log        *logger.Logger
	devicePath string
	connected  bool
	device     *DeviceInfo
}

// ConnectToDevice 连接设备
func (dfa *DirectFileAccessor) ConnectToDevice(deviceName, vid, pid string) error {
	dfa.log.Debug("直接文件访问器连接设备: %s", deviceName)
	dfa.device = &DeviceInfo{
		Name:      deviceName,
		VID:       vid,
		PID:       pid,
		DeviceID:  fmt.Sprintf("USB\\VID_%s&PID_%s", vid, pid),
	}
	return nil
}

// ListFiles 列出文件
func (dfa *DirectFileAccessor) ListFiles(basePath string) ([]*FileInfo, error) {
	dfa.log.Debug("直接文件访问器列出文件: %s\\%s", dfa.devicePath, basePath)

	fullPath := filepath.Join(dfa.devicePath, basePath)
	var files []*FileInfo

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		files = append(files, &FileInfo{
			Path:         path,
			Name:         info.Name(),
			Size:         info.Size(),
			IsOpus:       strings.ToLower(filepath.Ext(info.Name())) == ".opus",
			ModTime:      info.ModTime(),
		})

		return nil
	})

	return files, err
}

// GetFileStream 获取文件流
func (dfa *DirectFileAccessor) GetFileStream(filePath string) (io.ReadCloser, error) {
	dfa.log.Debug("直接文件访问器获取文件流: %s", filePath)
	file, err := os.Open(filePath)
	return file, err
}

// Close 关闭连接
func (dfa *DirectFileAccessor) Close() error {
	dfa.connected = false
	return nil
}

// IsConnected 检查连接状态
func (dfa *DirectFileAccessor) IsConnected() bool {
	return dfa.connected
}

// GetDeviceInfo 获取设备信息
func (dfa *DirectFileAccessor) GetDeviceInfo() *DeviceInfo {
	return dfa.device
}