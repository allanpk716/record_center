package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MTPFileInfo MTP文件信息
type MTPFileInfo struct {
	Name     string
	Size     int64
	Path     string
	IsFolder bool
}

func main() {
	fmt.Println("=== MTP设备访问测试程序 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 1. 检查SR302设备是否连接
	fmt.Println("步骤1: 检查SR302设备连接状态...")
	deviceInfo, err := checkSR302Device()
	if err != nil {
		fmt.Printf("❌ 设备检查失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 设备检查成功: %s\n", deviceInfo)
	fmt.Println()

	// 2. 测试PowerShell COM访问方法
	fmt.Println("步骤2: 测试PowerShell COM访问方法...")

	// 方法1: 便携式设备命名空间
	fmt.Println("\n2.1 测试便携式设备命名空间访问...")
	files, err := testPortableDevicesAccess()
	if err != nil {
		fmt.Printf("❌ 便携式设备访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ 便携式设备访问成功，找到 %d 个项目\n", len(files))
		for i, file := range files {
			if i < 5 { // 只显示前5个
				fmt.Printf("  - %s (%s, %d bytes)\n", file.Name, file.Path, file.Size)
			}
		}
		if len(files) > 5 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(files)-5)
		}
	}

	// 方法2: Shell.Application COM对象
	fmt.Println("\n2.2 测试Shell.Application COM对象...")
	files2, err := testShellApplicationAccess()
	if err != nil {
		fmt.Printf("❌ Shell.Application访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ Shell.Application访问成功，找到 %d 个项目\n", len(files2))
		for i, file := range files2 {
			if i < 5 { // 只显示前5个
				fmt.Printf("  - %s (%s, %d bytes)\n", file.Name, file.Path, file.Size)
			}
		}
		if len(files2) > 5 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(files2)-5)
		}
	}

	// 方法3: 直接路径访问
	fmt.Println("\n2.3 测试直接路径访问...")
	files3, err := testDirectPathAccess()
	if err != nil {
		fmt.Printf("❌ 直接路径访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ 直接路径访问成功，找到 %d 个项目\n", len(files3))
		for i, file := range files3 {
			if i < 5 { // 只显示前5个
				fmt.Printf("  - %s (%s, %d bytes)\n", file.Name, file.Path, file.Size)
			}
		}
		if len(files3) > 5 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(files3)-5)
		}
	}

	// 3. 寻找.opus文件
	fmt.Println("\n步骤3: 专门寻找.opus文件...")
	opusFiles, err := findOpusFiles()
	if err != nil {
		fmt.Printf("❌ 寻找.opus文件失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个.opus文件:\n", len(opusFiles))
		for i, file := range opusFiles {
			if i < 10 { // 只显示前10个
				fmt.Printf("  - %s (%.2f MB)\n", file.Name, float64(file.Size)/1024/1024)
			}
		}
		if len(opusFiles) > 10 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(opusFiles)-10)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}

// checkSR302Device 检查SR302设备
func checkSR302Device() (string, error) {
	script := `
$device = Get-WmiObject Win32_PnPEntity | Where-Object {
    $_.DeviceID -like "*VID_2207*" -and $_.DeviceID -like "*PID_0011*"
}
if ($device) {
    Write-Output "SR302设备找到: $($device.Name)"
    Write-Output "设备ID: $($device.DeviceID)"
} else {
    Write-Output "未找到SR302设备"
    exit 1
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PowerShell执行失败: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if strings.Contains(result, "未找到") {
		return "", fmt.Errorf("SR302设备未连接")
	}

	return result, nil
}

// testPortableDevicesAccess 测试便携式设备访问
func testPortableDevicesAccess() ([]*MTPFileInfo, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $items = $portable.Items()
    $count = 0
    foreach ($item in $items) {
        $count++
        $path = $item.Path
        $name = $item.Name
        $size = 0
        if (-not $item.IsFolder) {
            $size = $item.Size
        }
        Write-Output "FILE|$name|$size|$path|$($item.IsFolder)"
    }
    Write-Output "COUNT|$count"
} else {
    Write-Error "无法访问便携式设备"
    exit 1
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("便携式设备访问失败: %v", err)
	}

	return parsePowerShellOutput(string(output))
}

// testShellApplicationAccess 测试Shell.Application访问
func testShellApplicationAccess() ([]*MTPFileInfo, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$desktop = $shell.NameSpace(0)
$found = $false
$count = 0

# 递归搜索函数
function Search-Items($folder, $maxDepth = 3) {
    if ($maxDepth -le 0) { return }

    try {
        $items = $folder.Items()
        foreach ($item in $items) {
            $count++
            $name = $item.Name
            $path = $item.Path
            $size = 0
            if (-not $item.IsFolder) {
                $size = $item.Size
            }
            Write-Output "FILE|$name|$size|$path|$($item.IsFolder)"

            # 如果是录音设备，尝试深入搜索
            if ($name -like "*录音*" -or $name -like "*SR302*" -and $item.IsFolder) {
                try {
                    $subfolder = $folder.ParseName($name)
                    Search-Items $subfolder ($maxDepth - 1)
                } catch {
                    # 忽略访问错误
                }
            }

            # 防止无限循环
            if ($count -gt 100) { break }
        }
    } catch {
        # 忽略访问错误
    }
}

# 搜索桌面
Search-Items $desktop
Write-Output "COUNT|$count"
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Shell.Application访问失败: %v", err)
	}

	return parsePowerShellOutput(string(output))
}

// testDirectPathAccess 测试直接路径访问
func testDirectPathAccess() ([]*MTPFileInfo, error) {
	script := `
# 尝试多种可能的路径
$paths = @(
    "::{20D04FE0-3AEA-1069-A2D8-08002B30309D}\\?\usb#vid_2207&pid_0011*",
    "\\?\usb#vid_2207&pid_0011*",
    "E:\",
    "F:\",
    "G:\"
)

$shell = New-Object -ComObject Shell.Application
$count = 0

foreach ($pattern in $paths) {
    try {
        # 尝试直接访问路径
        if (Test-Path $pattern) {
            $folder = $shell.NameSpace($pattern)
            if ($folder) {
                $items = $folder.Items()
                foreach ($item in $items) {
                    $count++
                    $name = $item.Name
                    $size = 0
                    if (-not $item.IsFolder) {
                        $size = $item.Size
                    }
                    $path = $item.Path
                    Write-Output "FILE|$name|$size|$path|$($item.IsFolder)"

                    if ($count -gt 50) { break }
                }
            }
        }
    } catch {
        # 忽略错误，继续下一个路径
    }
}

Write-Output "COUNT|$count"
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("直接路径访问失败: %v", err)
	}

	return parsePowerShellOutput(string(output))
}

// findOpusFiles 专门查找.opus文件
func findOpusFiles() ([]*MTPFileInfo, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
$count = 0

if ($portable) {
    # 递归搜索函数
    function Find-OpusFiles($folder, $maxDepth = 4) {
        if ($maxDepth -le 0) { return }

        try {
            $items = $folder.Items()
            foreach ($item in $items) {
                $name = $item.Name
                if ($name.ToLower().EndsWith(".opus")) {
                    $count++
                    $size = $item.Size
                    $path = $item.Path
                    Write-Output "OPUS|$name|$size|$path"
                }

                # 递归搜索文件夹
                if ($item.IsFolder -and $maxDepth -gt 1) {
                    try {
                        $subfolder = $folder.ParseName($name)
                        Find-OpusFiles $subfolder ($maxDepth - 1)
                    } catch {
                        # 忽略访问错误
                    }
                }

                # 防止无限循环
                if ($count -gt 100) { return }
            }
        } catch {
            # 忽略访问错误
        }
    }

    # 搜索所有便携式设备
    $devices = $portable.Items()
    foreach ($device in $devices) {
        if ($device.IsFolder) {
            Find-OpusFiles $device
        }
        if ($count -gt 100) { break }
    }
}

Write-Output "OPUS_COUNT|$count"
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("查找.opus文件失败: %v", err)
	}

	return parseOpusOutput(string(output))
}

// parsePowerShellOutput 解析PowerShell输出
func parsePowerShellOutput(output string) ([]*MTPFileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []*MTPFileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FILE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				size := int64(0)
				fmt.Sscanf(parts[2], "%d", &size)

				file := &MTPFileInfo{
					Name:     parts[1],
					Size:     size,
					Path:     parts[3],
					IsFolder: parts[4] == "True",
				}
				files = append(files, file)
			}
		}
	}

	return files, nil
}

// parseOpusOutput 解析.opus文件输出
func parseOpusOutput(output string) ([]*MTPFileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []*MTPFileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OPUS|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := int64(0)
				fmt.Sscanf(parts[2], "%d", &size)

				file := &MTPFileInfo{
					Name:     parts[1],
					Size:     size,
					Path:     parts[3],
					IsFolder: false,
				}
				files = append(files, file)
			}
		}
	}

	return files, nil
}