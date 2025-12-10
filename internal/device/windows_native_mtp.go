//go:build windows

package device

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// WindowsNativeMTP Windows原生MTP访问器
type WindowsNativeMTP struct {
	log        *logger.Logger
	connected  bool
	deviceInfo *DeviceInfo
}

// NewWindowsNativeMTP 创建Windows原生MTP访问器
func NewWindowsNativeMTP(log *logger.Logger) *WindowsNativeMTP {
	return &WindowsNativeMTP{
		log: log,
	}
}

// ConnectToDevice 连接到设备
func (w *WindowsNativeMTP) ConnectToDevice(deviceName, vid, pid string) error {
	w.log.Info("Windows原生MTP连接设备: %s (VID:%s, PID:%s)", deviceName, vid, pid)

	// 使用PowerShell Shell COM访问
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $items = $portable.Items()
    foreach ($item in $items) {
        if ($item.Name -eq "%s") {
            Write-Output "DEVICE_FOUND|%s"
            exit 0
        }
    }
}
Write-Output "DEVICE_NOT_FOUND"
`, deviceName, deviceName)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("设备连接失败: %w", err)
	}

	if strings.Contains(string(output), "DEVICE_FOUND") {
		w.connected = true
		w.deviceInfo = &DeviceInfo{
			Name: deviceName,
			VID:  vid,
			PID:  pid,
		}
		w.log.Info("成功连接到设备: %s", deviceName)
		return nil
	}

	return fmt.Errorf("未找到设备: %s", deviceName)
}

// ListFiles 列出设备文件
func (w *WindowsNativeMTP) ListFiles(basePath string) ([]*FileInfo, error) {
	if !w.connected {
		return nil, fmt.Errorf("设备未连接")
	}

	w.log.Debug("Windows原生MTP列出文件: %s", basePath)

	// 使用深度递归搜索
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
$allFiles = @()

if ($portable) {
    $device = $portable.ParseName("%s")
    if ($device) {
        Write-Host "开始枚举设备文件..."

        function Enumerate-Files($folder, $depth = 0, $maxDepth = 6) {
            if ($depth -gt $maxDepth) { return }

            try {
                $items = $folder.Items()
                foreach ($item in $items) {
                    $name = $item.Name

                    if (-not $item.IsFolder) {
                        $ext = [System.IO.Path]::GetExtension($name).ToLower()
                        if ($ext -eq ".opus") {
                            $fileInfo = @{
                                Name = $name
                                Size = $item.Size
                                Path = $item.Path
                                ModTime = $item.ModifyDate
                            }
                            $script:allFiles += $fileInfo
                            Write-Host "找到Opus文件: $name"
                        }
                    } elseif ($depth -lt $maxDepth) {
                        try {
                            $subFolder = $folder.ParseName($name)
                            if ($subFolder) {
                                Enumerate-Files $subFolder ($depth + 1) $maxDepth
                            }
                        } catch {
                            Write-Host "无法访问文件夹: $name"
                        }
                    }
                }
            } catch {
                Write-Host "枚举文件夹失败: $($_.Exception.Message)"
            }
        }

        Enumerate-Files $device

        foreach ($file in $allFiles) {
            Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)|$($file.ModTime)"
        }
    } else {
        Write-Host "无法获取设备对象"
    }
} else {
    Write-Host "无法获取便携式设备命名空间"
}

Write-Output "DONE"
`, w.deviceInfo.Name)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		w.log.Error("PowerShell文件枚举失败: %v, 输出: %s", err, string(output))
		return nil, fmt.Errorf("文件枚举失败: %w", err)
	}

	w.log.Debug("PowerShell输出: %s", string(output))

	return w.parseFileOutput(string(output))
}

// parseFileOutput 解析文件输出
func (w *WindowsNativeMTP) parseFileOutput(output string) ([]*FileInfo, error) {
	lines := strings.Split(output, "\n")
	var files []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FILE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := int64(0)
				fmt.Sscanf(parts[2], "%d", &size)

				modTime := time.Now()
				if len(parts) >= 5 && parts[4] != "" {
					// 尝试解析修改时间
					if parsedTime, err := time.Parse("2006-01-02 15:04:05", parts[4]); err == nil {
						modTime = parsedTime
					}
				}

				file := &FileInfo{
					Name:    parts[1],
					Size:    size,
					Path:    parts[3],
					ModTime: modTime,
					IsOpus:  strings.ToLower(filepath.Ext(parts[1])) == ".opus",
				}
				files = append(files, file)
			}
		}
	}

	w.log.Info("Windows原生MTP找到 %d 个文件，其中 %d 个.opus文件", len(files), countOpusFiles(files))
	return files, nil
}

// GetFileStream 获取文件流
func (w *WindowsNativeMTP) GetFileStream(filePath string) (io.ReadCloser, error) {
	if !w.connected {
		return nil, fmt.Errorf("设备未连接")
	}

	w.log.Debug("Windows原生MTP获取文件流: %s", filePath)

	// 使用PowerShell复制文件到临时目录
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("mtp_%d.opus", time.Now().UnixNano()))

	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $device = $portable.ParseName("%s")
    if ($device) {
        # 通过路径查找文件
        function Find-And-Copy-File($folder, $targetPath, $destPath) {
            try {
                $items = $folder.Items()
                foreach ($item in $items) {
                    if ($item.Path -eq $targetPath) {
                        # 复制文件到目标路径
                        $item.InvokeVerb("copy")
                        $destFolder = $shell.NameSpace((Split-Path $destPath))
                        if ($destFolder) {
                            $destFile = $destFolder.ParseName((Split-Path $destPath -Leaf))
                            if ($destFile) {
                                # 这里需要实现复制逻辑
                                Write-Host "找到目标文件: $($item.Name)"
                                return $true
                            }
                        }
                    } elseif ($item.IsFolder) {
                        $subFolder = $folder.ParseName($item.Name)
                        if ($subFolder) {
                            if (Find-And-Copy-File $subFolder $targetPath $destPath) {
                                return $true
                            }
                        }
                    }
                }
            } catch {
                Write-Host "搜索失败: $($_.Exception.Message)"
            }
            return $false
        }

        if (Find-And-Copy-File $device "%s" "%s") {
            Write-Output "SUCCESS"
        } else {
            Write-Output "FAILED"
        }
    }
} else {
    Write-Output "NO_PORTABLE"
}
`, w.deviceInfo.Name, filePath, tempFile)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("文件复制失败: %w", err)
	}

	if strings.Contains(string(output), "SUCCESS") {
		file, err := os.Open(tempFile)
		if err != nil {
			return nil, fmt.Errorf("打开临时文件失败: %w", err)
		}
		return file, nil
	}

	return nil, fmt.Errorf("文件流访问尚未实现")
}

// Close 关闭连接
func (w *WindowsNativeMTP) Close() error {
	w.connected = false
	w.log.Debug("Windows原生MTP连接已关闭")
	return nil
}

// IsConnected 检查连接状态
func (w *WindowsNativeMTP) IsConnected() bool {
	return w.connected
}

// GetDeviceInfo 获取设备信息
func (w *WindowsNativeMTP) GetDeviceInfo() *DeviceInfo {
	return w.deviceInfo
}

// GetLastError 获取最后的错误
func (w *WindowsNativeMTP) GetLastError() error {
	return nil
}

// countOpusFiles 统计Opus文件数量
func countOpusFiles(files []*FileInfo) int {
	count := 0
	for _, file := range files {
		if file.IsOpus {
			count++
		}
	}
	return count
}