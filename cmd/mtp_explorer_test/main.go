package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== MTP资源管理器式访问测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 测试1: 通过便携式设备查找SR302
	fmt.Println("测试1: 通过便携式设备查找SR302...")
	sr302Path, err := findSR302InPortableDevices()
	if err != nil {
		fmt.Printf("❌ 在便携式设备中找不到SR302: %v\n", err)
	} else {
		fmt.Printf("✅ 找到SR302路径: %s\n", sr302Path)
	}

	// 测试2: 如果找到了，尝试访问其内容
	if sr302Path != "" {
		fmt.Println("\n测试2: 尝试访问SR302内容...")
		files, err := exploreDeviceContent(sr302Path)
		if err != nil {
			fmt.Printf("❌ 访问设备内容失败: %v\n", err)
		} else {
			fmt.Printf("✅ 设备内容访问成功，找到 %d 个项目:\n", len(files))
			for i, file := range files {
				if i < 15 {
					fmt.Printf("  - %s (%s, %.2f MB)\n", file.Name, file.Type, float64(file.Size)/1024/1024)
				}
			}
			if len(files) > 15 {
				fmt.Printf("  ... 还有 %d 个项目\n", len(files)-15)
			}

			// 查找.opus文件
			opusCount := 0
			for _, file := range files {
				if strings.HasSuffix(strings.ToLower(file.Name), ".opus") {
					opusCount++
				}
			}
			fmt.Printf("\n🎵 找到 %d 个.opus文件\n", opusCount)
		}
	}

	// 测试3: 直接搜索所有便携式设备中的.opus文件
	fmt.Println("\n测试3: 在所有便携式设备中搜索.opus文件...")
	allOpusFiles, err := findAllOpusInPortableDevices()
	if err != nil {
		fmt.Printf("❌ 搜索.opus文件失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个.opus文件:\n", len(allOpusFiles))
		for i, file := range allOpusFiles {
			if i < 10 {
				fmt.Printf("  - %s (%.2f MB, 路径: %s)\n", file.Name, float64(file.Size)/1024/1024, file.Path)
			}
		}
		if len(allOpusFiles) > 10 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(allOpusFiles)-10)
		}
	}

	// 测试4: 尝试通过This PC访问
	fmt.Println("\n测试4: 通过This PC访问设备...")
	thisPCDevices, err := exploreThisPC()
	if err != nil {
		fmt.Printf("❌ This PC访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ This PC访问成功，找到 %d 个设备:\n", len(thisPCDevices))
		for _, device := range thisPCDevices {
			fmt.Printf("  - %s (%s)\n", device.Name, device.Path)
		}
	}

	// 测试5: 获取设备详细信息
	fmt.Println("\n测试5: 获取SR302设备详细信息...")
	deviceInfo, err := getSR302DetailedInfo()
	if err != nil {
		fmt.Printf("❌ 获取设备信息失败: %v\n", err)
	} else {
		fmt.Printf("✅ 设备信息:\n%s\n", deviceInfo)
	}

	fmt.Println("\n=== 测试完成 ===")
}

// FileInfo 文件信息
type FileInfo struct {
	Name string
	Path string
	Type string // "文件夹" 或 "文件"
	Size int64
}

// findSR302InPortableDevices 在便携式设备中查找SR302
func findSR302InPortableDevices() (string, error) {
	script := `
try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        $items = $portable.Items()
        foreach ($item in $items) {
            if ($item.Name -like "*SR302*" -or $item.Name -like "*录音*") {
                Write-Output "FOUND|$($item.Name)|$($item.Path)"
                exit 0
            }
        }
        Write-Output "NOT_FOUND"
    } else {
        Write-Output "NO_PORTABLE"
    }
} catch {
    Write-Error "便携式设备访问失败: $($_.Exception.Message)"
    exit 1
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PowerShell执行失败: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if strings.HasPrefix(result, "FOUND|") {
		parts := strings.Split(result, "|")
		if len(parts) >= 3 {
			return parts[2], nil
		}
	}

	return "", fmt.Errorf("未找到SR302设备")
}

// exploreDeviceContent 探索设备内容
func exploreDeviceContent(devicePath string) ([]*FileInfo, error) {
	script := fmt.Sprintf(`
try {
    $shell = New-Object -ComObject Shell.Application
    $device = $shell.NameSpace('%s')

    if ($device) {
        $items = $device.Items()
        $count = 0

        # 递归搜索函数
        function Explore-Items($folder, $maxDepth = 4) {
            if ($maxDepth -le 0) { return }

            try {
                $folderItems = $folder.Items()
                foreach ($item in $folderItems) {
                    $name = $item.Name
                    $path = $item.Path
                    $size = 0
                    $type = "文件夹"

                    if (-not $item.IsFolder) {
                        $size = $item.Size
                        $type = "文件"
                    }

                    Write-Output "ITEM|$name|$path|$type|$size"
                    $count++

                    # 递归搜索文件夹
                    if ($item.IsFolder -and $maxDepth -gt 1) {
                        try {
                            $subFolder = $folder.ParseName($name)
                            Explore-Items $subFolder ($maxDepth - 1)
                        } catch {
                            # 忽略访问错误
                        }
                    }

                    # 限制项目数量
                    if ($count -gt 200) { return }
                }
            } catch {
                # 忽略访问错误
            }
        }

        Explore-Items $device
    } else {
        Write-Error "无法访问设备: $devicePath"
        exit 1
    }
} catch {
    Write-Error "探索设备内容失败: $($_.Exception.Message)"
    exit 1
}
`, strings.Replace(devicePath, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("探索设备内容失败: %v", err)
	}

	return parseFilesOutput(string(output))
}

// findAllOpusInPortableDevices 在所有便携式设备中查找.opus文件
func findAllOpusInPortableDevices() ([]*FileInfo, error) {
	script := `
try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)
    $count = 0

    if ($portable) {
        # 递归搜索.opus文件的函数
        function Find-Opus-Files($folder, $maxDepth = 5) {
            if ($maxDepth -le 0) { return }

            try {
                $items = $folder.Items()
                foreach ($item in $items) {
                    $name = $item.Name

                    if ($name.ToLower().EndsWith(".opus")) {
                        $path = $item.Path
                        $size = $item.Size
                        Write-Output "OPUS|$name|$path|$size"
                        $count++
                    }

                    # 递归搜索文件夹
                    if ($item.IsFolder -and $maxDepth -gt 1) {
                        try {
                            $subFolder = $folder.ParseName($name)
                            Find-Opus-Files $subFolder ($maxDepth - 1)
                        } catch {
                            # 忽略访问错误
                        }
                    }

                    # 限制搜索数量
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
                Find-Opus-Files $device
            }
            if ($count -gt 100) { break }
        }
    }
} catch {
    Write-Error "搜索.opus文件失败: $($_.Exception.Message)"
    exit 1
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("搜索.opus文件失败: %v", err)
	}

	return parseOpusOutput(string(output))
}

// exploreThisPC 探索This PC
func exploreThisPC() ([]*FileInfo, error) {
	script := `
try {
    $shell = New-Object -ComObject Shell.Application
    $thisPC = $shell.NameSpace(17)  # 便携式设备
    $count = 0

    # 尝试多种方法
    $methods = @(
        { $shell.NameSpace("::{20D04FE0-3AEA-1069-A2D8-08002B30309D}") },  # This PC
        { $shell.NameSpace(0) },  # Desktop
        { $shell.NameSpace(17) }  # Portable Devices
    )

    foreach ($method in $methods) {
        try {
            $folder = & $method
            if ($folder) {
                $items = $folder.Items()
                foreach ($item in $items) {
                    $name = $item.Name
                    $path = $item.Path

                    # 查找可能是录音设备的条目
                    if ($name -like "*SR302*" -or $name -like "*录音*" -or $name -like "*USB*" -or $name -like "*Storage*") {
                        Write-Output "DEVICE|$name|$path"
                        $count++
                    }

                    if ($count -gt 20) { break }
                }
            }
        } catch {
            # 忽略错误，继续下一个方法
        }
    }
} catch {
    Write-Error "This PC访问失败: $($_.Exception.Message)"
    exit 1
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("This PC访问失败: %v", err)
	}

	return parseDeviceOutput(string(output))
}

// getSR302DetailedInfo 获取SR302详细信息
func getSR302DetailedInfo() (string, error) {
	script := `
try {
    # 通过WMI获取设备信息
    $device = Get-WmiObject Win32_PnPEntity | Where-Object {
        $_.DeviceID -like "*VID_2207*" -and $_.DeviceID -like "*PID_0011*"
    } | Select-Object -First 1

    if ($device) {
        $info = "设备名称: $($device.Name)`n"
        $info += "设备ID: $($device.DeviceID)`n"
        $info += "描述: $($device.Description)`n"
        $info += "制造商: $($device.Manufacturer)`n"

        # 获取PowerShell访问状态
        try {
            $shell = New-Object -ComObject Shell.Application
            $portable = $shell.NameSpace(17)
            if ($portable) {
                $found = $false
                foreach ($item in $portable.Items()) {
                    if ($item.Name -like "*SR302*" -or $item.Name -like "*录音*") {
                        $found = $true
                        $info += "Shell访问: 可访问`n"
                        $info += "Shell路径: $($item.Path)`n"
                        break
                    }
                }
                if (-not $found) {
                    $info += "Shell访问: 未在便携式设备中找到`n"
                }
            } else {
                $info += "Shell访问: 便携式设备不可访问`n"
            }
        } catch {
            $info += "Shell访问: COM对象创建失败`n"
        }

        Write-Output $info
    } else {
        Write-Output "未找到SR302设备"
        exit 1
    }
} catch {
    Write-Error "获取设备信息失败: $($_.Exception.Message)"
    exit 1
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("获取设备信息失败: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// parseFilesOutput 解析文件输出
func parseFilesOutput(output string) ([]*FileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ITEM|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				size := int64(0)
				fmt.Sscanf(parts[4], "%d", &size)

				file := &FileInfo{
					Name: parts[1],
					Path: parts[2],
					Type: parts[3],
					Size: size,
				}
				files = append(files, file)
			}
		}
	}

	return files, nil
}

// parseOpusOutput 解析.opus文件输出
func parseOpusOutput(output string) ([]*FileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OPUS|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := int64(0)
				fmt.Sscanf(parts[3], "%d", &size)

				file := &FileInfo{
					Name: parts[1],
					Path: parts[2],
					Type: "文件",
					Size: size,
				}
				files = append(files, file)
			}
		}
	}

	return files, nil
}

// parseDeviceOutput 解析设备输出
func parseDeviceOutput(output string) ([]*FileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var devices []*FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEVICE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				device := &FileInfo{
					Name: parts[1],
					Path: parts[2],
					Type: "设备",
					Size: 0,
				}
				devices = append(devices, device)
			}
		}
	}

	return devices, nil
}