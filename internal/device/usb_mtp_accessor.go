//go:build windows

package device

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// USBMTPAccessor USB MTP访问器
type USBMTPAccessor struct {
	log           *logger.Logger
	connected     bool
	deviceInfo    *DeviceInfo
	windowsDriver bool
	mutex         sync.RWMutex
}

// NewUSBMTPAccessor 创建新的USB MTP访问器
func NewUSBMTPAccessor(log *logger.Logger) *USBMTPAccessor {
	return &USBMTPAccessor{
		log: log,
	}
}

// ConnectToDevice 连接到设备
func (u *USBMTPAccessor) ConnectToDevice(deviceName, vid, pid string) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.log.Debug("USB MTP连接设备: %s (VID:%s, PID:%s)", deviceName, vid, pid)

	// 在Windows上，直接使用Windows驱动方式访问MTP设备
	if err := u.connectWithWindowsDriver(deviceName, vid, pid); err != nil {
		return fmt.Errorf("连接设备失败: %w", err)
	}

	u.connected = true
	u.windowsDriver = true
	u.log.Info("USB MTP成功连接到设备: %s", u.deviceInfo.Name)
	return nil
}

// connectWithGousb 使用gousb连接设备 (已移除gousb依赖)
func (u *USBMTPAccessor) connectWithGousb(vid, pid string) error {
	// gousb依赖已移除，此方法不再使用
	return fmt.Errorf("gousb支持已移除，请使用Windows驱动方式")
}

// connectWithWindowsDriver 使用Windows驱动连接设备
func (u *USBMTPAccessor) connectWithWindowsDriver(deviceName, vid, pid string) error {
	u.log.Debug("尝试使用Windows驱动连接设备")

	// 使用WMI获取设备信息
	deviceInfo, err := u.getDeviceViaWMI(vid, pid)
	if err != nil {
		return fmt.Errorf("WMI设备检测失败: %w", err)
	}

	u.deviceInfo = deviceInfo

	// 验证设备是否真的可访问
	if err := u.verifyDeviceAccess(deviceInfo); err != nil {
		u.log.Warn("设备访问验证失败: %v", err)
		// 不返回错误，有些设备可能需要特殊处理
	}

	return nil
}

// getDeviceViaWMI 通过WMI获取设备信息
func (u *USBMTPAccessor) getDeviceViaWMI(vid, pid string) (*DeviceInfo, error) {
	script := fmt.Sprintf(`
$device = Get-WmiObject Win32_PnPEntity | Where-Object {
    $_.DeviceID -like "*VID_%s*" -and $_.DeviceID -like "*PID_%s*"
} | Select-Object -First 1

if ($device) {
    $name = $device.Name
    $deviceId = $device.DeviceID
    $description = $device.Description
    Write-Output "DEVICE_FOUND|$name|$deviceId|$description"
} else {
    Write-Output "DEVICE_NOT_FOUND"
}
`, vid, pid)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("WMI查询失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEVICE_FOUND|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				return &DeviceInfo{
					Name:     parts[1],
					DeviceID:  parts[2],
					VID:       vid,
					PID:       pid,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("WMI未找到设备")
}

// verifyDeviceAccess 验证设备访问
func (u *USBMTPAccessor) verifyDeviceAccess(deviceInfo *DeviceInfo) error {
	// 尝试列出设备文件来验证访问
	files, err := u.listFilesViaWindows(deviceInfo)
	if err != nil {
		return fmt.Errorf("设备文件列表验证失败: %w", err)
	}

	u.log.Debug("设备访问验证通过，找到 %d 个可访问项目", len(files))
	return nil
}

// listFilesViaWindows 通过Windows列出文件
func (u *USBMTPAccessor) listFilesViaWindows(deviceInfo *DeviceInfo) ([]*FileInfo, error) {
	// 使用Windows Shell COM对象尝试访问
	script := `
$shell = New-Object -ComObject Shell.Application
$found = $false

# 尝试多种访问方法
$methods = @(
    { Name = "Portable Devices"; Namespace = 17 },
    { Name = "This PC"; Namespace = 0 },
    { Name = "Desktop"; Namespace = 0 }
)

foreach ($method in $methods) {
    try {
        $folder = $shell.NameSpace($method.Namespace)
        if ($folder) {
            $items = $folder.Items()
            foreach ($item in $items) {
                $name = $item.Name
                if ($name -like "*录音*" -or $name -like "*SR302*" -or $name -like "*USB*") {
                    $found = $true
                    Write-Output "DEVICE_ITEM|$name|$($item.Path)"
                    break
                }
            }
        }
        if ($found) { break }
    } catch {
        # 忽略错误，继续下一个方法
    }
}

if (-not $found) {
    Write-Output "NO_DEVICE_ITEM"
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Windows Shell访问失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var items []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEVICE_ITEM|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				items = append(items, &FileInfo{
					Name: parts[1],
					Path: parts[2],
					// 其他字段可以根据需要填充
				})
			}
		}
	}

	return items, nil
}

// ListFiles 列出设备文件
func (u *USBMTPAccessor) ListFiles(basePath string) ([]*FileInfo, error) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	if !u.connected {
		return nil, fmt.Errorf("设备未连接")
	}

	u.log.Debug("USB MTP列出文件: %s", basePath)

	var files []*FileInfo
	var err error

	if u.windowsDriver {
		// 使用Windows驱动方式
		files, err = u.listFilesViaWindows(u.deviceInfo)
		if err != nil {
			u.log.Debug("Windows驱动文件列表失败: %v", err)
			return nil, fmt.Errorf("列出文件失败: %w", err)
		}

		// 如果找到了设备，尝试更深入的文件枚举
		if len(files) > 0 {
			moreFiles, err := u.enumerateDeviceFiles(files[0].Path)
			if err != nil {
				u.log.Debug("深入文件枚举失败: %v", err)
			} else {
				files = append(files, moreFiles...)
			}
		}
	} else {
		// 没有gousb支持，直接返回错误
		return nil, fmt.Errorf("不支持非Windows驱动方式的文件访问")
	}

	// 过滤.opus文件
	var opusFiles []*FileInfo
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name)) == ".opus" {
			file.IsOpus = true
			opusFiles = append(opusFiles, file)
		}
	}

	u.log.Info("USB MTP找到 %d 个文件，其中 %d 个.opus文件", len(opusFiles), len(opusFiles))
	return opusFiles, nil
}

// listFilesViaGousb 通过gousb列出文件（已移除）
func (u *USBMTPAccessor) listFilesViaGousb(basePath string) ([]*FileInfo, error) {
	// gousb依赖已移除
	return []*FileInfo{}, fmt.Errorf("gousb支持已移除")
}

// enumerateDeviceFiles 枚举设备文件
func (u *USBMTPAccessor) enumerateDeviceFiles(devicePath string) ([]*FileInfo, error) {
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
try {
    $folder = $shell.NameSpace('%s')
    if ($folder) {
        function Enumerate-Files($folder, $maxDepth = 4) {
            $items = $folder.Items()
            foreach ($item in $items) {
                $name = $item.Name
                $path = $item.Path

                if (-not $item.IsFolder -and $name.ToLower().EndsWith(".opus")) {
                    Write-Output "OPUS_FILE|$name|$path|$($item.Size)"
                }

                if ($item.IsFolder -and $maxDepth -gt 1) {
                    try {
                        $subFolder = $folder.ParseName($name)
                        Enumerate-Files $subFolder ($maxDepth - 1)
                    } catch {
                        # 忽略访问错误
                    }
                }
            }
        }

        Enumerate-Files $folder
    }
} catch {
    Write-Output "ERROR:$($($_.Exception.Message))"
}
`, strings.Replace(devicePath, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("设备文件枚举失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OPUS_FILE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := int64(0)
				fmt.Sscanf(parts[3], "%d", &size)

				file := &FileInfo{
					Name:         parts[1],
					Path:         parts[2],
					Size:         size,
					IsOpus:       true,
					ModTime:      time.Now(),
				}
				files = append(files, file)
			}
		} else if strings.HasPrefix(line, "ERROR:") {
			u.log.Debug("文件枚举错误: %s", line)
		}
	}

	return files, nil
}

// GetFileStream 获取文件流
func (u *USBMTPAccessor) GetFileStream(filePath string) (io.ReadCloser, error) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	if !u.connected {
		return nil, fmt.Errorf("设备未连接")
	}

	u.log.Debug("USB MTP获取文件流: %s", filePath)

	// 对于Windows驱动方式，尝试直接打开文件
	if u.windowsDriver {
		file, err := os.Open(filePath)
		if err == nil {
			return file, nil
		}
	}

	// 对于gousb方式，需要实现MTP协议的文件传输
	return nil, fmt.Errorf("文件流访问尚未实现")
}

// Close 关闭连接
func (u *USBMTPAccessor) Close() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.connected = false
	u.log.Debug("USB MTP连接已关闭")
	return nil
}

// IsConnected 检查连接状态
func (u *USBMTPAccessor) IsConnected() bool {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.connected
}

// GetDeviceInfo 获取设备信息
func (u *USBMTPAccessor) GetDeviceInfo() *DeviceInfo {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.deviceInfo
}

// GetLastError 获取最后的错误
func (u *USBMTPAccessor) GetLastError() error {
	return nil
}