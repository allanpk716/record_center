//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== 深度文件枚举测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 方法1: 直接访问便携式设备中的SR302
	fmt.Println("方法1: 直接访问便携式设备中的SR302...")
	files1, err := deepEnumPortableDevices()
	if err != nil {
		fmt.Printf("❌ 便携式设备深度枚举失败: %v\n", err)
	} else {
		fmt.Printf("✅ 便携式设备找到 %d 个文件:\n", len(files1))
		printFiles(files1)
	}

	// 方法2: 使用完整Shell递归搜索
	fmt.Println("\n方法2: 完整Shell递归搜索...")
	files2, err := fullShellRecursiveSearch()
	if err != nil {
		fmt.Printf("❌ Shell递归搜索失败: %v\n", err)
	} else {
		fmt.Printf("✅ Shell递归搜索找到 %d 个文件:\n", len(files2))
		printFiles(files2)
	}

	// 方法3: PowerShell WMI + Shell组合
	fmt.Println("\n方法3: PowerShell WMI + Shell组合...")
	files3, err := wmiShellCombined()
	if err != nil {
		fmt.Printf("❌ WMI+Shell组合失败: %v\n", err)
	} else {
		fmt.Printf("✅ WMI+Shell组合找到 %d 个文件:\n", len(files3))
		printFiles(files3)
	}

	// 方法4: 直接路径枚举（使用已知路径）
	fmt.Println("\n方法4: 直接路径枚举...")
	files4, err := directPathEnumeration()
	if err != nil {
		fmt.Printf("❌ 直接路径枚举失败: %v\n", err)
	} else {
		fmt.Printf("✅ 直接路径枚举找到 %d 个文件:\n", len(files4))
		printFiles(files4)
	}

	fmt.Println("\n=== 总结 ===")
	totalFiles := len(files1) + len(files2) + len(files3) + len(files4)
	if totalFiles > 0 {
		fmt.Printf("✅ 总共找到 %d 个.opus文件！\n", totalFiles)
		fmt.Println("🎉 MTP文件访问成功！")
	} else {
		fmt.Println("❌ 未找到任何.opus文件")
		fmt.Println("可能的原因:")
		fmt.Println("1. 设备中没有.opus文件")
		fmt.Println("2. 文件路径需要更深入的搜索")
		fmt.Println("3. 文件可能以其他格式存在")
		fmt.Println("4. 需要特定的访问权限")
	}

	fmt.Println("\n按任意键退出...")
	var input string
	fmt.Scanln(&input)
}

type FileInfo struct {
	Name string
	Size int64
	Path string
}

func printFiles(files []FileInfo) {
	if len(files) == 0 {
		fmt.Println("  (无文件)")
		return
	}

	totalSize := int64(0)
	for i, file := range files {
		if i < 10 {
			fmt.Printf("  %2d. %s (%.2f MB)\n", i+1, file.Name, float64(file.Size)/1024/1024)
		}
		totalSize += file.Size
	}

	if len(files) > 10 {
		fmt.Printf("  ... 还有 %d 个文件\n", len(files)-10)
	}
	fmt.Printf("  总大小: %.2f MB\n", float64(totalSize)/1024/1024)
}

// deepEnumPortableDevices 深度枚举便携式设备
func deepEnumPortableDevices() ([]FileInfo, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
$allFiles = @()

if ($portable) {
    $items = $portable.Items()
    foreach ($item in $items) {
        if ($item.Name -eq "SR302") {
            Write-Host "找到SR302设备，开始深度枚举..."

            function Deep-Enumerate($folder, $depth = 0, $maxDepth = 6) {
                $indent = "  " * $depth
                Write-Host "${indent}扫描: $($folder.Title)"

                try {
                    $items = $folder.Items()
                    foreach ($subItem in $items) {
                        $name = $subItem.Name
                        Write-Host "${indent}  项目: $name - 文件夹: $($subItem.IsFolder)"

                        if (-not $subItem.IsFolder) {
                            # 检查是否是音频文件
                            $ext = [System.IO.Path]::GetExtension($name).ToLower()
                            if ($ext -in @('.opus', '.mp3', '.wav', '.m4a', '.flac')) {
                                $fileInfo = @{
                                    Name = $name
                                    Size = $subItem.Size
                                    Path = $subItem.Path
                                }
                                $script:allFiles += $fileInfo
                                Write-Host "${indent}    🎵 找到音频: $name ($($subItem.Size) bytes)"
                            }
                        } elseif ($depth -lt $maxDepth) {
                            try {
                                $subFolder = $folder.ParseName($name)
                                if ($subFolder) {
                                    Deep-Enumerate $subFolder ($depth + 1) $maxDepth
                                }
                            } catch {
                                Write-Host "${indent}    ❌ 无法访问子文件夹: $($_.Exception.Message)"
                            }
                        }
                    }
                } catch {
                    Write-Host "${indent}❌ 枚举失败: $($_.Exception.Message)"
                }
            }

            try {
                $sr302Folder = $portable.ParseName("SR302")
                if ($sr302Folder) {
                    Deep-Enumerate $sr302Folder
                } else {
                    Write-Host "无法获取SR302文件夹对象"
                }
            } catch {
                Write-Host "访问SR302设备失败: $($_.Exception.Message)"
            }
            break
        }
    }
}

# 输出结果
foreach ($file in $allFiles) {
    Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)"
}

if ($allFiles.Count -eq 0) {
    Write-Output "NONE"
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("深度枚举失败: %w, 输出: %s", err, string(output))
	}

	return parseFileOutput(string(output))
}

// fullShellRecursiveSearch 完整Shell递归搜索
func fullShellRecursiveSearch() ([]FileInfo, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$allFiles = @()

# 搜索所有可能的命名空间
$namespaces = @(
    17,  # 便携式设备
    0,   # 桌面
    5,   # 我的文档
    23   # 其他
)

foreach ($ns in $namespaces) {
    try {
        Write-Host "尝试命名空间: $ns"
        $folder = $shell.NameSpace($ns)
        if ($folder) {
            Write-Host "命名空间 $ns 可访问"

            function Global-Search($folder, $depth = 0, $maxDepth = 5) {
                if ($depth -gt $maxDepth) { return }

                try {
                    $items = $folder.Items()
                    foreach ($item in $items) {
                        $name = $item.Name

                        # 优先搜索SR302相关
                        if ($name -like "*SR302*" -or $name -like "*录音*" -or
                            $name -like "*Record*" -or $name -like "*Voice*") {

                            Write-Host "找到相关设备/文件夹: $name"

                            if (-not $item.IsFolder) {
                                $ext = [System.IO.Path]::GetExtension($name).ToLower()
                                if ($ext -in @('.opus', '.mp3', '.wav', '.m4a', '.flac')) {
                                    $script:allFiles += @{
                                        Name = $name
                                        Size = $item.Size
                                        Path = $item.Path
                                    }
                                }
                            } else {
                                try {
                                    $subFolder = $folder.ParseName($name)
                                    if ($subFolder) {
                                        Deep-Search-Audio $subFolder ($depth + 1)
                                    }
                                } catch {
                                    Write-Host "无法访问 $name`: $($_.Exception.Message)"
                                }
                            }
                        }

                        # 递归搜索
                        if ($item.IsFolder -and $depth -lt $maxDepth) {
                            try {
                                $subFolder = $folder.ParseName($name)
                                if ($subFolder) {
                                    Global-Search $subFolder ($depth + 1) $maxDepth
                                }
                            } catch {
                                # 忽略访问错误
                            }
                        }
                    }
                } catch {
                    Write-Host "搜索命名空间 $ns 失败: $($_.Exception.Message)"
                }
            }

            function Deep-Search-Audio($folder, $depth = 0) {
                if ($depth -gt 4) { return }

                try {
                    $items = $folder.Items()
                    foreach ($item in $items) {
                        $name = $item.Name

                        if (-not $item.IsFolder) {
                            $ext = [System.IO.Path]::GetExtension($name).ToLower()
                            if ($ext -in @('.opus', '.mp3', '.wav', '.m4a', '.flac')) {
                                Write-Host "  🎵 找到音频文件: $name"
                                $script:allFiles += @{
                                    Name = $name
                                    Size = $item.Size
                                    Path = $item.Path
                                }
                            }
                        } elseif ($depth -lt 4) {
                            try {
                                $subFolder = $folder.ParseName($name)
                                if ($subFolder) {
                                    Deep-Search-Audio $subFolder ($depth + 1)
                                }
                            } catch {
                                Write-Host "  无法访问子文件夹: $name"
                            }
                        }
                    }
                } catch {
                    Write-Host "深度搜索失败: $($_.Exception.Message)"
                }
            }

            Global-Search $folder
        }
    } catch {
        Write-Host "无法访问命名空间 $ns`: $($_.Exception.Message)"
    }
}

# 输出结果
foreach ($file in $allFiles) {
    Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)"
}

if ($allFiles.Count -eq 0) {
    Write-Output "NONE"
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Shell递归搜索失败: %w, 输出: %s", err, string(output))
	}

	return parseFileOutput(string(output))
}

// wmiShellCombined WMI + Shell组合方法
func wmiShellCombined() ([]FileInfo, error) {
	script := `
# 使用WMI找到设备，然后用Shell访问
$device = Get-WmiObject Win32_PnPEntity | Where-Object { $_.DeviceID -like "*VID_2207*" } | Select-Object -First 1

if ($device) {
    Write-Host "找到设备: $($device.Name)"
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)
    $allFiles = @()

    if ($portable) {
        $items = $portable.Items()
        foreach ($item in $items) {
            if ($item.Name -like "*SR302*") {
                Write-Host "在便携式设备中找到: $($item.Name)"
                try {
                    $deviceFolder = $portable.ParseName($item.Name)
                    if ($deviceFolder) {
                        $subItems = $deviceFolder.Items()
                        Write-Host "找到 $($subItems.Count) 个子项目"
                        foreach ($subItem in $subItems) {
                            Write-Host "  子项目: $($subItem.Name) - 文件夹: $($subItem.IsFolder)"
                            if (-not $subItem.IsFolder) {
                                $ext = [System.IO.Path]::GetExtension($subItem.Name).ToLower()
                                if ($ext -in @('.opus', '.mp3', '.wav', '.m4a', '.flac')) {
                                    $allFiles += @{ Name = $subItem.Name; Size = $subItem.Size; Path = $subItem.Path }
                                    Write-Host "    🎵 音频文件: $($subItem.Name)"
                                }
                            }
                        }
                    }
                } catch {
                    Write-Host "访问设备失败: $($_.Exception.Message)"
                }
                break
            }
        }
    }

    foreach ($file in $allFiles) {
        Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)"
    }
}

Write-Output "NONE"
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("WMI+Shell组合失败: %w, 输出: %s", err, string(output))
	}

	return parseFileOutput(string(output))
}

// directPathEnumeration 直接路径枚举
func directPathEnumeration() ([]FileInfo, error) {
	// 尝试多种可能的路径格式
	paths := []string{
		"shell:::{4234d49b-0245-4df3-b780-3893943456e1}\\SR302",  // 便携式设备直接路径
		"::{20D04FE0-3AEA-1069-A2D8-08002B30309D}\\::{645FF040-5081-101B-9F08-00AA002F954E}\\SR302", // 桌面路径
		"::{20D04FE0-3AEA-1069-A2D8-08002B30309D}\\SR302", // This PC路径
	}

	var allFiles []FileInfo

	for i, path := range paths {
		fmt.Printf("尝试路径 %d: %s\n", i+1, path)
		files, err := enumeratePath(path)
		if err != nil {
			fmt.Printf("  失败: %v\n", err)
			continue
		}
		fmt.Printf("  成功，找到 %d 个文件\n", len(files))
		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}

// enumeratePath 枚举指定路径
func enumeratePath(path string) ([]FileInfo, error) {
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$allFiles = @()

try {
    $folder = $shell.NameSpace('%s')
    if ($folder) {
        Write-Host "路径可访问"

        function Enumerate-All($folder, $depth = 0, $maxDepth = 6) {
            $indent = "  " * $depth
            Write-Host "${indent}枚举深度 $depth"

            try {
                $items = $folder.Items()
                Write-Host "${indent}找到 $($items.Count) 个项目"

                foreach ($item in $items) {
                    $name = $item.Name
                    Write-Host "${indent}项目: $name (文件夹: $($item.IsFolder))"

                    if (-not $item.IsFolder) {
                        $ext = [System.IO.Path]::GetExtension($name).ToLower()
                        if ($ext -in @('.opus', '.mp3', '.wav', '.m4a', '.flac', '.wma')) {
                            $fileInfo = @{
                                Name = $name
                                Size = $item.Size
                                Path = $item.Path
                            }
                            $script:allFiles += $fileInfo
                            Write-Host "${indent}  🎵 音频: $name ($($item.Size) bytes)"
                        }
                    } elseif ($depth -lt $maxDepth) {
                        try {
                            $subFolder = $folder.ParseName($name)
                            if ($subFolder) {
                                Enumerate-All $subFolder ($depth + 1) $maxDepth
                            }
                        } catch {
                            Write-Host "${indent}  ❌ 无法访问: $($_.Exception.Message)"
                        }
                    }
                }
            } catch {
                Write-Host "${indent}❌ 枚举失败: $($_.Exception.Message)"
            }
        }

        Enumerate-All $folder
    } else {
        Write-Host "路径不可访问"
    }
} catch {
    Write-Host "错误: $($_.Exception.Message)"
}

foreach ($file in $allFiles) {
    Write-Output "FILE|$($file.Name)|$($file.Size)|$($file.Path)"
}

if ($allFiles.Count -eq 0) {
    Write-Output "NONE"
}
`, strings.Replace(path, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("路径枚举失败: %w, 输出: %s", err, string(output))
	}

	return parseFileOutput(string(output))
}

// parseFileOutput 解析文件输出
func parseFileOutput(output string) ([]FileInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
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