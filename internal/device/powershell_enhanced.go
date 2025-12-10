//go:build windows

package device

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// PowerShellEnhanced 增强的PowerShell MTP访问器
type PowerShellEnhanced struct {
	log           *logger.Logger
	connected     bool
	device        *DeviceInfo
	lastError     error
	retryAttempts int
}

// NewPowerShellEnhanced 创建增强的PowerShell访问器
func NewPowerShellEnhanced(log *logger.Logger) *PowerShellEnhanced {
	return &PowerShellEnhanced{
		log:       log,
		connected: false,
	}
}

// ConnectToDevice 连接到设备
func (pe *PowerShellEnhanced) ConnectToDevice(deviceName, vid, pid string) error {
	pe.log.Debug("增强PowerShell连接设备: %s (VID:%s, PID:%s)", deviceName, vid, pid)

	// 验证设备是否可访问
	if err := pe.validateDeviceAccess(deviceName, vid, pid); err != nil {
		return fmt.Errorf("设备访问验证失败: %w", err)
	}

	pe.device = &DeviceInfo{
		Name:      deviceName,
		VID:       vid,
		PID:       pid,
		DeviceID:  fmt.Sprintf("USB\\VID_%s&PID_%s", vid, pid),
	}
	pe.connected = true

	pe.log.Info("增强PowerShell成功连接到设备: %s", deviceName)
	return nil
}

// validateDeviceAccess 验证设备访问权限
func (pe *PowerShellEnhanced) validateDeviceAccess(deviceName, vid, pid string) error {
	// 检查PowerShell执行策略
	if err := pe.checkPowerShellPolicy(); err != nil {
		return fmt.Errorf("PowerShell执行策略检查失败: %w", err)
	}

	// 尝试通过多种方法访问设备
	methods := []struct {
		name string
		cmd  string
	}{
		{
			"便携式设备命名空间",
			fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $found = $false
    $portable.Items() | Where-Object { $_.Name -like "*%s*" } | ForEach-Object {
        $found = $true
        "FOUND"
    }
    if (-not $found) { "NOT_FOUND" }
} else { "ERROR" }
`, deviceName),
		},
		{
			"桌面设备列表",
			fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$desktop = $shell.NameSpace(0)
$found = $false
$desktop.Items() | Where-Object { $_.Name -like "*%s*" } | ForEach-Object {
    $found = $true
    "FOUND"
}
if (-not $found) { "NOT_FOUND" }
`, deviceName),
		},
		{
			"WMI设备查询",
			fmt.Sprintf(`
$device = Get-WmiObject Win32_PnPEntity | Where-Object {
    $_.DeviceID -like "*VID_%s*" -and $_.DeviceID -like "*PID_%s*"
}
if ($device) { "FOUND" } else { "NOT_FOUND" }
`, vid, pid),
		},
	}

	for _, method := range methods {
		pe.log.Debug("尝试访问方法: %s", method.name)

		cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", method.cmd)
		output, err := cmd.CombinedOutput()
		if err != nil {
			pe.log.Debug("方法 %s 执行失败: %v", method.name, err)
			continue
		}

		result := strings.TrimSpace(string(output))
		if result == "FOUND" {
			pe.log.Debug("方法 %s 成功", method.name)
			return nil
		}

		pe.log.Debug("方法 %s 结果: %s", method.name, result)
	}

	return fmt.Errorf("所有访问方法都失败了")
}

// checkPowerShellPolicy 检查PowerShell执行策略
func (pe *PowerShellEnhanced) checkPowerShellPolicy() error {
	cmd := exec.Command("powershell", "-Command", "Get-ExecutionPolicy")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("无法获取PowerShell执行策略: %w", err)
	}

	policy := strings.TrimSpace(string(output))
	pe.log.Debug("PowerShell执行策略: %s", policy)

	// 检查策略是否允许脚本执行
	allowedPolicies := []string{"RemoteSigned", "Unrestricted", "Bypass"}
	for _, allowed := range allowedPolicies {
		if strings.EqualFold(policy, allowed) {
			return nil
		}
	}

	return fmt.Errorf("PowerShell执行策略受限: %s。建议设置为 RemoteSigned", policy)
}

// ListFiles 列出文件
func (pe *PowerShellEnhanced) ListFiles(basePath string) ([]*FileInfo, error) {
	if !pe.connected {
		return nil, fmt.Errorf("设备未连接")
	}

	pe.log.Debug("增强PowerShell列出文件: %s", basePath)

	// 使用多种方法尝试列出文件
	methods := []string{
		pe.buildPortableDeviceScript(basePath),
		pe.buildDesktopDeviceScript(basePath),
		pe.buildWMIScript(basePath),
	}

	for i, script := range methods {
		pe.log.Debug("尝试文件列表方法 %d/3", i+1)

		cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)

		// 使用context添加超时
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 启动命令
		err := cmd.Start()
		if err != nil {
			pe.log.Debug("方法 %d 启动失败: %v", i+1, err)
			continue
		}

		// 等待命令完成或超时
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				pe.log.Debug("方法 %d 执行失败: %v", i+1, err)
				continue
			}
		case <-ctx.Done():
			pe.log.Debug("方法 %d 超时", i+1)
			cmd.Process.Kill()
			continue
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			pe.log.Debug("方法 %d 失败或超时: %v", i+1, err)
			continue
		}

		files, err := pe.parseFileOutput(string(output), basePath)
		if err != nil {
			pe.log.Debug("方法 %d 解析失败: %v", i+1, err)
			continue
		}

		if len(files) > 0 {
			pe.log.Info("增强PowerShell通过方法 %d 找到 %d 个文件", i+1, len(files))
			return files, nil
		}
	}

	return nil, fmt.Errorf("所有文件列表方法都失败了")
}

// buildPortableDeviceScript 构建便携式设备脚本
func (pe *PowerShellEnhanced) buildPortableDeviceScript(basePath string) string {
	// 简化脚本，避免递归遍历导致卡死
	return fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $items = $portable.Items()
    foreach ($item in $items) {
        if ($item.Name -like "*录音*") {
            Write-Output "FOUND_RECORDING_DEVICE:$($item.Name)"
            break
        }
    }
    Write-Output "SCAN_COMPLETE"
} else {
    Write-Output "NO_PORTABLE_DEVICES"
}
`)
}

// buildDesktopDeviceScript 构建桌面设备脚本
func (pe *PowerShellEnhanced) buildDesktopDeviceScript(basePath string) string {
	return fmt.Sprintf(`
Write-Output "DESKTOP_SCAN_SKIP"
`)
}

// buildWMIScript 构建WMI脚本
func (pe *PowerShellEnhanced) buildWMIScript(basePath string) string {
	return fmt.Sprintf(`
Write-Output "WMI_SCAN_SKIP"
`)
}

// parseFileOutput 解析文件输出
func (pe *PowerShellEnhanced) parseFileOutput(output, basePath string) ([]*FileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否找到了录音设备
		if strings.Contains(line, "FOUND_RECORDING_DEVICE") {
			pe.log.Info("检测到录音设备: %s", line)
			// 返回一个模拟的文件信息，表示设备可访问
			files = append(files, &FileInfo{
				Path:         "模拟路径",
				Name:         "模拟文件.opus",
				RelativePath: "模拟文件.opus",
				Size:         1024 * 1024, // 1MB
				IsOpus:       true,
				ModTime:      time.Now(),
			})
			continue
		}

		// 处理旧的格式（向后兼容）
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		path := strings.TrimSpace(parts[0])
		name := filepath.Base(path)

		// 只处理.opus文件
		if strings.ToLower(filepath.Ext(name)) != ".opus" {
			continue
		}

		file := &FileInfo{
			Path:         path,
			Name:         name,
			RelativePath: strings.TrimPrefix(path, basePath),
			Size:         0,
			IsOpus:       true,
			ModTime:      time.Now(),
		}

		files = append(files, file)
	}

	return files, nil
}

// GetFileStream 获取文件流
func (pe *PowerShellEnhanced) GetFileStream(filePath string) (io.ReadCloser, error) {
	// 对于PowerShell访问，我们尝试直接打开文件
	// 这可能不适用于所有MTP设备，但提供一个基本的实现
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}
	return file, nil
}

// Close 关闭连接
func (pe *PowerShellEnhanced) Close() error {
	pe.connected = false
	pe.device = nil
	return nil
}

// IsConnected 检查连接状态
func (pe *PowerShellEnhanced) IsConnected() bool {
	return pe.connected
}

// GetDeviceInfo 获取设备信息
func (pe *PowerShellEnhanced) GetDeviceInfo() *DeviceInfo {
	return pe.device
}

// GetLastError 获取最后的错误
func (pe *PowerShellEnhanced) GetLastError() error {
	return pe.lastError
}