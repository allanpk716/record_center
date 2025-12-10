//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== 纯Windows原生MTP访问测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 测试1: WMI设备检测
	fmt.Println("测试1: WMI设备检测...")
	devices, err := detectDevicesWithWMI()
	if err != nil {
		fmt.Printf("❌ WMI设备检测失败: %v\n", err)
	} else {
		fmt.Printf("✅ WMI找到 %d 个设备:\n", len(devices))
		for i, device := range devices {
			fmt.Printf("  %d. %s (VID:%s, PID:%s)\n", i+1, device.Name, device.VID, device.PID)
		}
	}

	// 测试2: PowerShell便携式设备枚举
	fmt.Println("\n测试2: PowerShell便携式设备枚举...")
	portableDevices, err := enumeratePortableDevices()
	if err != nil {
		fmt.Printf("❌ 便携式设备枚举失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个便携式设备:\n", len(portableDevices))
		for i, device := range portableDevices {
			fmt.Printf("  %d. %s\n", i+1, device)
		}
	}

	// 测试3: Windows Shell COM访问
	fmt.Println("\n测试3: Windows Shell COM访问...")
	shellDevices, err := accessWithShellCOM()
	if err != nil {
		fmt.Printf("❌ Shell COM访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ Shell COM找到 %d 个设备:\n", len(shellDevices))
		for i, device := range shellDevices {
			fmt.Printf("  %d. %s\n", i+1, device)
		}
	}

	// 测试4: 尝试直接路径访问
	fmt.Println("\n测试4: 直接路径访问...")
	if len(devices) > 0 {
		for _, device := range devices {
			if strings.Contains(device.Name, "SR302") ||
			   (device.VID == "2207" && device.PID == "0011") {
				fmt.Printf("尝试访问SR302设备: %s\n", device.Name)
				files, err := accessDeviceDirectly(device)
				if err != nil {
					fmt.Printf("❌ 直接访问失败: %v\n", err)
				} else {
					fmt.Printf("✅ 直接访问成功，找到 %d 个文件:\n", len(files))
					for j, file := range files {
						if j < 5 {
							fmt.Printf("  - %s (%.2f MB)\n", file.Name, float64(file.Size)/1024/1024)
						}
					}
					if len(files) > 5 {
						fmt.Printf("  ... 还有 %d 个文件\n", len(files)-5)
					}
				}
				break
			}
		}
	}

	// 测试5: 文件管理器式访问
	fmt.Println("\n测试5: 文件管理器式访问...")
	files, err := accessFileManagerStyle()
	if err != nil {
		fmt.Printf("❌ 文件管理器式访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ 文件管理器式访问成功，找到 %d 个文件:\n", len(files))
		for i, file := range files {
			if i < 5 {
				fmt.Printf("  - %s (%.2f MB)\n", file.Name, float64(file.Size)/1024/1024)
			}
		}
		if len(files) > 5 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(files)-5)
		}
	}

	// 测试结果总结
	fmt.Println("\n=== 测试结果总结 ===")
	fmt.Println("测试1 WMI设备检测:", func() string {
		if err != nil { return "❌ 失败" }
		return "✅ 成功"
	}())
	fmt.Println("测试2 PowerShell便携式设备:", func() string {
		if err != nil { return "❌ 失败" }
		return "✅ 成功"
	}())
	fmt.Println("测试3 Windows Shell COM:", func() string {
		if err != nil { return "❌ 失败" }
		return "✅ 成功"
	}())
	fmt.Println("测试4 直接路径访问:", "需要具体设备测试")
	fmt.Println("测试5 文件管理器式访问:", func() string {
		if err != nil { return "❌ 失败" }
		return "✅ 成功"
	}())

	fmt.Println("\n按任意键退出...")
	var input string
	fmt.Scanln(&input)
}

type WMDDevice struct {
	Name     string
	VID      string
	PID      string
	DeviceID string
}

type FileInfo struct {
	Name string
	Size int64
	Path string
}

// detectDevicesWithWMI 使用WMI检测设备
func detectDevicesWithWMI() ([]WMDDevice, error) {
	script := `
$devices = Get-WmiObject Win32_PnPEntity | Where-Object {
    $_.DeviceID -like "*VID_*" -and $_.DeviceID -like "*PID_*" -and
    ($_.Name -like "*录音*" -or $_.Name -like "*SR302*" -or $_.Name -like "*Portable*")
} | Select-Object Name, DeviceID, Description

if ($devices) {
    foreach ($device in $devices) {
        $deviceId = $device.DeviceID
        $vid = ""
        $pid = ""

        if ($deviceId -match "VID_([0-9A-Fa-f]+)") {
            $vid = $matches[1]
        }
        if ($deviceId -match "PID_([0-9A-Fa-f]+)") {
            $pid = $matches[1]
        }

        Write-Output "DEVICE|$($device.Name)|$vid|$pid|$deviceId"
    }
} else {
    Write-Output "NONE"
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("WMI查询失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var devices []WMDDevice

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEVICE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				devices = append(devices, WMDDevice{
					Name:     parts[1],
					VID:      parts[2],
					PID:      parts[3],
					DeviceID: parts[4],
				})
			}
		}
	}

	return devices, nil
}

// enumeratePortableDevices 枚举便携式设备
func enumeratePortableDevices() ([]string, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $items = $portable.Items()
    foreach ($item in $items) {
        Write-Output "DEVICE|$($item.Name)"
    }
} else {
    Write-Output "NONE"
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("便携式设备查询失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var devices []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEVICE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				devices = append(devices, parts[1])
			}
		}
	}

	return devices, nil
}

// accessWithShellCOM 使用Shell COM访问
func accessWithShellCOM() ([]string, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$methods = @(
    @{ Name = "Portable Devices"; Namespace = 17 },
    @{ Name = "This PC"; Namespace = 0 },
    @{ Name = "Desktop"; Namespace = 0 }
)

$foundDevices = @()
foreach ($method in $methods) {
    try {
        $folder = $shell.NameSpace($method.Namespace)
        if ($folder) {
            $items = $folder.Items()
            foreach ($item in $items) {
                $name = $item.Name
                if ($name -like "*录音*" -or $name -like "*SR302*" -or
                    $name -like "*Portable*" -or $name -like "*USB*") {
                    $foundDevices += $name
                }
            }
        }
    } catch {
        # 忽略错误
    }
}

if ($foundDevices.Count -gt 0) {
    $foundDevices | ForEach-Object { Write-Output "DEVICE|$_" }
} else {
    Write-Output "NONE"
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Shell COM访问失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var devices []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEVICE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				devices = append(devices, parts[1])
			}
		}
	}

	return devices, nil
}

// accessDeviceDirectly 直接访问设备
func accessDeviceDirectly(device WMDDevice) ([]FileInfo, error) {
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$found = $false

# 尝试多种路径
$paths = @(
    "::{20D04FE0-3AEA-1069-A2D8-08002B30309D}",
    "shell:::{4234d49b-0245-4df3-b780-3893943456e1}"
)

foreach ($basePath in $paths) {
    try {
        $folder = $shell.NameSpace($basePath)
        if ($folder) {
            $items = $folder.Items()
            foreach ($item in $items) {
                if ($item.Name -like "*%s*") {
                    $deviceFolder = $folder.ParseName($item.Name)
                    if ($deviceFolder) {
                        # 递归查找文件
                        function Find-OpusFiles($folder, $maxDepth = 3) {
                            $files = @()
                            try {
                                $items = $folder.Items()
                                foreach ($subItem in $items) {
                                    if (-not $subItem.IsFolder -and $subItem.Name.ToLower().EndsWith(".opus")) {
                                        $files += @{
                                            Name = $subItem.Name
                                            Size = $subItem.Size
                                            Path = $subItem.Path
                                        }
                                    } elseif ($subItem.IsFolder -and $maxDepth -gt 1) {
                                        try {
                                            $subFolder = $folder.ParseName($subItem.Name)
                                            $files += Find-OpusFiles $subFolder ($maxDepth - 1)
                                        } catch {
                                            # 忽略访问错误
                                        }
                                    }
                                }
                            } catch {
                                # 忽略访问错误
                            }
                            return $files
                        }

                        $opusFiles = Find-OpusFiles $deviceFolder
                        foreach ($file in $opusFiles) {
                            Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)"
                        }
                        $found = $true
                        break
                    }
                }
            }
        }
        if ($found) { break }
    } catch {
        # 忽略错误
    }
}

if (-not $found) {
    Write-Output "NONE"
}
`, device.Name)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("直接设备访问失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FILE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := int64(0)
				fmt.Sscanf(parts[2], "%d", &size)
				files = append(files, FileInfo{
					Name: parts[1],
					Size: size,
					Path: parts[3],
				})
			}
		}
	}

	return files, nil
}

// accessFileManagerStyle 文件管理器式访问
func accessFileManagerStyle() ([]FileInfo, error) {
	// 尝试从常见的便携式设备路径访问
	paths := []string{
		"::{20D04FE0-3AEA-1069-A2D8-08002B30309D}\\::{645FF040-5081-101B-9F08-00AA002F954E}", // Desktop
		"::{20D04FE0-3AEA-1069-A2D8-08002B30309D}", // This PC
		"shell:::{4234d49b-0245-4df3-b780-3893943456e1}", // Portable Devices
	}

	var allFiles []FileInfo

	for _, path := range paths {
		files, err := accessPathAndEnumerate(path)
		if err != nil {
			continue // 尝试下一个路径
		}
		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}

// accessPathAndEnumerate 访问路径并枚举文件
func accessPathAndEnumerate(path string) ([]FileInfo, error) {
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
try {
    $folder = $shell.NameSpace('%s')
    if ($folder) {
        function Enumerate-AllFiles($folder, $maxDepth = 4) {
            $files = @()
            try {
                $items = $folder.Items()
                foreach ($item in $items) {
                    $name = $item.Name

                    # 查找录音设备相关的文件夹
                    if ($item.IsFolder -and
                        ($name -like "*录音*" -or $name -like "*SR302*" -or
                         $name -like "*Record*" -or $name -like "*Voice*")) {

                        # 进入该文件夹查找.opus文件
                        try {
                            $subFolder = $folder.ParseName($name)
                            $subFiles = Enumerate-OpusFiles $subFolder 3
                            $files += $subFiles
                        } catch {
                            # 忽略访问错误
                        }
                    }

                    # 递归查找
                    if ($item.IsFolder -and $maxDepth -gt 1 -and
                        !($name -like "*录音*" -or $name -like "*SR302*")) {
                        try {
                            $subFolder = $folder.ParseName($name)
                            $files += Enumerate-AllFiles $subFolder ($maxDepth - 1)
                        } catch {
                            # 忽略访问错误
                        }
                    }
                }
            } catch {
                # 忽略访问错误
            }
            return $files
        }

        function Enumerate-OpusFiles($folder, $maxDepth) {
            $files = @()
            try {
                $items = $folder.Items()
                foreach ($item in $items) {
                    if (-not $item.IsFolder -and $item.Name.ToLower().EndsWith(".opus")) {
                        $files += @{
                            Name = $item.Name
                            Size = $item.Size
                            Path = $item.Path
                        }
                    } elseif ($item.IsFolder -and $maxDepth -gt 1) {
                        try {
                            $subFolder = $folder.ParseName($item.Name)
                            $files += Enumerate-OpusFiles $subFolder ($maxDepth - 1)
                        } catch {
                            # 忽略访问错误
                        }
                    }
                }
            } catch {
                # 忽略访问错误
            }
            return $files
        }

        $result = Enumerate-AllFiles $folder
        foreach ($file in $result) {
            Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)"
        }
    }
} catch {
    Write-Output "ERROR:$($($_.Exception.Message))"
}
`, strings.Replace(path, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("路径访问失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FILE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := int64(0)
				fmt.Sscanf(parts[2], "%d", &size)
				files = append(files, FileInfo{
					Name: parts[1],
					Size: size,
					Path: parts[3],
				})
			}
		}
	}

	return files, nil
}