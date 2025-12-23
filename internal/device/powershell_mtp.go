//go:build windows

package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// PowerShellMTPAccessor 使用PowerShell访问MTP设备
type PowerShellMTPAccessor struct {
	log *logger.Logger
}

// NewPowerShellMTPAccessor 创建PowerShell MTP访问器
func NewPowerShellMTPAccessor(log *logger.Logger) *PowerShellMTPAccessor {
	return &PowerShellMTPAccessor{
		log: log,
	}
}

// sanitizeDeviceName 对设备名称进行转义以防止PowerShell命令注入
// 转义PowerShell特殊字符：` $ ; & | > < " '
func sanitizeDeviceName(deviceName string) string {
	// PowerShell特殊字符转义映射
	dangerous := []string{"`", "$", ";", "&", "|", ">", "<", "\"", "'"}
	sanitized := deviceName
	for _, char := range dangerous {
		// 在PowerShell中使用反引号`作为转义字符
		sanitized = strings.ReplaceAll(sanitized, char, "`"+char)
	}
	return sanitized
}

// GetMTPDevicePath 通过PowerShell获取MTP设备路径
func (ps *PowerShellMTPAccessor) GetMTPDevicePath(deviceName string) (string, error) {
	ps.log.Debug("使用PowerShell查找MTP设备: %s", deviceName)

	// 方法1: 通过便携式设备命名空间
	if path := ps.getPortableDevicePath(sanitizeDeviceName(deviceName)); path != "" {
		ps.log.Info("通过便携式设备找到路径: %s", path)
		return path, nil
	}

	// 方法2: 通过桌面设备列表
	if path := ps.getDesktopDevicePath(sanitizeDeviceName(deviceName)); path != "" {
		ps.log.Info("通过桌面设备找到路径: %s", path)
		return path, nil
	}

	// 方法3: 通过WMI增强查询
	if path := ps.getWMIEnhancedPath(sanitizeDeviceName(deviceName)); path != "" {
		ps.log.Info("通过WMI增强查询找到路径: %s", path)
		return path, nil
	}

	return "", fmt.Errorf("未找到MTP设备 %s", deviceName)
}

// ListMTPFiles 列出MTP设备中的文件
func (ps *PowerShellMTPAccessor) ListMTPFiles(devicePath, basePath string) ([]*MTPFileEntry, error) {
	ps.log.Debug("列出MTP设备文件: %s\\%s", devicePath, basePath)

	// 构建PowerShell命令
	psScript := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$folder = $shell.Namespace('%s').Self
if ($folder) {
    function Get-Files {
        param($folder, $basePath)
        $relativePath = $folder.Path.Replace('%s\', '')
        foreach ($item in $folder.Items()) {
            if ($item.IsFolder) {
                Get-Files $item.GetFolder $basePath
            } else {
                $relPath = $item.Path.Replace('%s\', '')
                if ($relPath.StartsWith($basePath)) {
                    # 优先使用ExtendedProperty获取真实文件大小
                    $size = 0
                    $sizeSource = "Unknown"
                    try {
                        $extendedSize = $item.ExtendedProperty("System.Size")
                        if ($extendedSize -and $extendedSize -gt 0) {
                            $size = [long]$extendedSize
                            $sizeSource = "ExtendedProperty"
                        }
                    } catch {
                        $sizeSource = "ExtendedProperty_Failed"
                    }

                    # 降级方法1：使用Size属性
                    if ($size -eq 0) {
                        try {
                            if ($item.Size -and $item.Size -gt 0) {
                                $size = [long]$item.Size
                                $sizeSource = "SizeProperty"
                            }
                        } catch {
                            $sizeSource = "SizeProperty_Failed"
                        }
                    }

                    # 降级方法2：使用GetDetailsOf
                    if ($size -eq 0) {
                        try {
                            $details = $folder.GetDetailsOf($item, 1)
                            if ($details -and $details -match '(\d+(?:,\d+)*)\s*(KB|MB|GB|B)') {
                                $num = $matches[1] -replace ',', ''
                                $unit = $matches[2]
                                $size = switch ($unit) {
                                    "KB" { [long][double]$num * 1KB }
                                    "MB" { [long][double]$num * 1MB }
                                    "GB" { [long][double]$num * 1GB }
                                    "B"  { [long][double]$num }
                                    default { 0 }
                                }
                                if ($size -gt 0) {
                                    $sizeSource = "GetDetailsOf"
                                }
                            }
                        } catch {
                            $sizeSource = "GetDetailsOf_Failed"
                        }
                    }

                    $modified = $item.ExtendedProperty("System.DateModified")
                    Write-Output "$($relPath)|$($size)|$($modified)|$($sizeSource)"
                }
            }
        }
    }
    Get-Files $folder ''
}
`, devicePath, basePath, basePath)

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		ps.log.Error("PowerShell命令执行失败: %v", err)
		return nil, fmt.Errorf("执行PowerShell失败: %w", err)
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")
	var files []*MTPFileEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			file := &MTPFileEntry{
				Path:     strings.TrimSpace(parts[0]),
				Name:     strings.TrimSuffix(strings.TrimSpace(parts[0]), "\\"),
				RelativePath: strings.TrimSpace(parts[0]),
				Size:     parseInt64(strings.TrimSpace(parts[1])),
				SizeSource: "Unknown", // 默认值
				IsDir:    false,
			}

			// 解析修改时间
			if len(parts) >= 3 {
				if modTimeStr := strings.TrimSpace(parts[2]); modTimeStr != "" {
					if modTime, err := time.Parse("2006-01-02 15:04:05", modTimeStr); err == nil {
						file.ModTime = modTime
					}
				}
			}

			// 解析大小来源
			if len(parts) >= 4 {
				file.SizeSource = strings.TrimSpace(parts[3])
			}

			// 记录文件大小和来源信息
			if file.Size > 0 {
				ps.log.Debug("文件: %s, 大小: %d bytes, 来源: %s", file.Name, file.Size, file.SizeSource)
			}

			files = append(files, file)
		}
	}

	ps.log.Debug("找到 %d 个文件", len(files))
	return files, nil
}

// OpenFileStream 打开MTP设备文件流
func (ps *PowerShellMTPAccessor) OpenFileStream(filePath string) (*MTPFileStream, error) {
	ps.log.Debug("打开MTP文件流: %s", filePath)

	// 创建PowerShell脚本来复制文件到临时位置
	tempFile := fmt.Sprintf("%s\\mtp_temp_%d", os.TempDir(), time.Now().UnixNano())

	psScript := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$folder = $shell.Namespace('%s').Self
$file = $folder.ParseName('%s')
if ($file) {
    $file.CopyTo('%s')
    Write-Output "SUCCESS"
} else {
    Write-Output "ERROR"
}
`, filepath.Dir(filePath), filepath.Base(filePath), tempFile)

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("PowerShell复制失败: %w", err)
	}

	if strings.Contains(string(output), "SUCCESS") {
		// 打开临时文件
		file, err := os.Open(tempFile)
		if err != nil {
			os.Remove(tempFile)
			return nil, fmt.Errorf("打开临时文件失败: %w", err)
		}

		return &MTPFileStream{
			file:     file,
			tempPath: tempFile,
		}, nil
	}

	return nil, fmt.Errorf("PowerShell复制文件失败")
}

// Close 关闭PowerShell访问器
func (ps *PowerShellMTPAccessor) Close() error {
	ps.log.Debug("关闭PowerShell MTP访问器")
	return nil
}

// 私有方法

// getPortableDevicePath 通过便携式设备命名空间获取路径
func (ps *PowerShellMTPAccessor) getPortableDevicePath(deviceName string) string {
	// 便携式设备的命名空间常量是17
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $items = $portable.Items()
    foreach($item in $items) {
        if ($item.Name -like "*%s*") {
            $item.Path()
            break
        }
    }
}
`, deviceName))

	output, err := cmd.Output()
	if err != nil {
		ps.log.Debug("便携式设备查询失败: %v", err)
		return ""
	}

	path := strings.TrimSpace(string(output))
	if path != "" && ps.testPathAccessibility(path) {
		return path
	}

	return ""
}

// getDesktopDevicePath 通过桌面设备列表获取路径
func (ps *PowerShellMTPAccessor) getDesktopDevicePath(deviceName string) string {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$desktop = $shell.NameSpace(0)
$items = $desktop.Items()
foreach($item in $items) {
    if ($item.Name -like "*%s*") {
        $item.Path()
        break
    }
}
`, deviceName))

	output, err := cmd.Output()
	if err != nil {
		ps.log.Debug("桌面设备查询失败: %v", err)
		return ""
	}

	path := strings.TrimSpace(string(output))
	if path != "" && ps.testPathAccessibility(path) {
		return path
	}

	return ""
}

// getWMIEnhancedPath 通过WMI增强查询获取路径
func (ps *PowerShellMTPAccessor) getWMIEnhancedPath(deviceName string) string {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
Get-WmiObject Win32_PnPEntity |
Where-Object { $_.DeviceID -like "*USB*" -and ($_.Name -like "*%s*" -or $_.FriendlyName -like "*%s*")} |
Select-Object -First 1 |
ForEach-Object {
    # 尝试获取设备路径
    $devicePath = "\\?\Volume{" + $_.DeviceID.Split("\")[2].Split("&")[0] + "}"
    if (Test-Path $devicePath) {
        $devicePath
    } else {
        # 如果直接路径不可用，尝试通过Shell访问
        try {
            $shell = New-Object -ComObject Shell.Application
            $found = $false
            $desktop = $shell.NameSpace(0)
            $desktop.Items() | Where-Object { $_.Name -like "*%s*" } | ForEach-Object {
                $_.Path()
                $found = $true
            }
            if (-not $found) {
                "NOT_FOUND"
            }
        } catch {
            "ERROR"
        }
    }
}
`, deviceName, deviceName))

	output, err := cmd.Output()
	if err != nil {
		ps.log.Debug("WMI增强查询失败: %v", err)
		return ""
	}

	result := strings.TrimSpace(string(output))
	if result != "" && result != "NOT_FOUND" && result != "ERROR" {
		if ps.testPathAccessibility(result) {
			return result
		}
	}

	return ""
}

// testPathAccessibility 测试路径是否可访问
func (ps *PowerShellMTPAccessor) testPathAccessibility(path string) bool {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf("Test-Path '%s'", path))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "True"
}

// parseInt64 解析int64
func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

// MTPFileEntry MTP文件条目
type MTPFileEntry struct {
	Path         string
	Name         string
	RelativePath string
	Size         int64
	SizeSource   string  // 数据来源：ExtendedProperty, SizeProperty, GetDetailsOf, Failed
	ModTime      time.Time
	IsDir        bool
}

// MTPFileStream MTP文件流
type MTPFileStream struct {
	file     *os.File
	tempPath string
}

// Read 实现io.Reader接口
func (mfs *MTPFileStream) Read(p []byte) (n int, err error) {
	return mfs.file.Read(p)
}

// Close 关闭文件流
func (mfs *MTPFileStream) Close() error {
	var errs []error

	// 关闭文件
	if mfs.file != nil {
		if err := mfs.file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭文件失败: %w", err))
		}
	}

	// 删除临时文件
	if mfs.tempPath != "" {
		if err := os.Remove(mfs.tempPath); err != nil {
			errs = append(errs, fmt.Errorf("删除临时文件失败: %w", err))
		}
	}

	// 如果有错误，返回组合错误
	if len(errs) > 0 {
		return fmt.Errorf("关闭流时发生错误: %v", errs)
	}
	return nil
}