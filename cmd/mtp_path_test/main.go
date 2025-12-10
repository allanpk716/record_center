package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== MTP设备路径精确测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 从日志中获取到的设备路径
	devicePath := `::{20D04FE0-3AEA-1069-A2D8-08002B30309D}\\?\usb#vid_2207&pid_0011&mi_00#7&117ed41b&0&0000#{6ac27878-a6fa-4155-ba85-f98f491d4f33}`

	fmt.Printf("测试设备路径: %s\n", devicePath)
	fmt.Println()

	// 测试1: 检查路径是否可访问
	fmt.Println("测试1: 检查设备路径可访问性...")
	if err := testPathAccessibility(devicePath); err != nil {
		fmt.Printf("❌ 路径不可访问: %v\n", err)
	} else {
		fmt.Printf("✅ 路径可访问\n")
	}

	// 测试2: 尝试列出根目录内容
	fmt.Println("\n测试2: 列出设备根目录内容...")
	files, err := listDeviceRoot(devicePath)
	if err != nil {
		fmt.Printf("❌ 无法列出根目录: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个项目:\n", len(files))
		for i, file := range files {
			if i < 10 {
				fmt.Printf("  - %s (%s)\n", file.Name, file.Type)
			}
		}
		if len(files) > 10 {
			fmt.Printf("  ... 还有 %d 个项目\n", len(files)-10)
		}
	}

	// 测试3: 深度搜索录音相关文件夹
	fmt.Println("\n测试3: 搜索录音相关文件夹...")
	recordDirs, err := findRecordingDirectories(devicePath)
	if err != nil {
		fmt.Printf("❌ 搜索录音目录失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个录音相关目录:\n", len(recordDirs))
		for _, dir := range recordDirs {
			fmt.Printf("  - %s\n", dir.Path)
		}
	}

	// 测试4: 在录音目录中查找.opus文件
	fmt.Println("\n测试4: 在录音目录中查找.opus文件...")
	if len(recordDirs) > 0 {
		for _, dir := range recordDirs {
			fmt.Printf("\n检查目录: %s\n", dir.Path)
			opusFiles, err := findOpusInDirectory(dir.Path)
			if err != nil {
				fmt.Printf("  ❌ 搜索失败: %v\n", err)
			} else {
				fmt.Printf("  ✅ 找到 %d 个.opus文件:\n", len(opusFiles))
				for j, file := range opusFiles {
					if j < 5 {
						fmt.Printf("    - %s (%.2f MB)\n", file.Name, float64(file.Size)/1024/1024)
					}
				}
				if len(opusFiles) > 5 {
					fmt.Printf("    ... 还有 %d 个文件\n", len(opusFiles)-5)
				}
			}
		}
	} else {
		fmt.Println("⚠️ 没有找到录音目录，跳过.opus文件搜索")
	}

	// 测试5: 尝试标准路径"内部共享存储空间\录音笔文件"
	fmt.Println("\n测试5: 尝试标准录音笔路径...")
	standardPath := devicePath + `\内部共享存储空间\录音笔文件`
	if err := testStandardRecordingPath(standardPath); err != nil {
		fmt.Printf("❌ 标准路径访问失败: %v\n", err)
	}

	fmt.Println("\n=== 测试完成 ===")
}

// DirItem 目录项
type DirItem struct {
	Name string
	Path string
	Type string // "文件夹" 或 "文件"
	Size int64
}

// testPathAccessibility 测试路径可访问性
func testPathAccessibility(path string) error {
	script := fmt.Sprintf(`
try {
    $result = Test-Path -Path '%s'
    if ($result) {
        Write-Output "PATH_EXISTS"
    } else {
        Write-Output "PATH_NOT_EXISTS"
    }
} catch {
    Write-Error "Path test failed: $($_.Exception.Message)"
    exit 1
}
`, strings.Replace(path, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PowerShell执行失败: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if result != "PATH_EXISTS" {
		return fmt.Errorf("路径不存在或不可访问")
	}

	return nil
}

// listDeviceRoot 列出设备根目录
func listDeviceRoot(devicePath string) ([]*DirItem, error) {
	script := fmt.Sprintf(`
try {
    $shell = New-Object -ComObject Shell.Application
    $folder = $shell.NameSpace('%s')
    if (-not $folder) {
        Write-Error "无法获取文件夹对象"
        exit 1
    }

    $items = $folder.Items()
    $count = 0
    foreach ($item in $items) {
        $name = $item.Name
        $path = $item.Path
        $type = "文件"
        if ($item.IsFolder) {
            $type = "文件夹"
        }
        Write-Output "ITEM|$name|$path|$type"
        $count++

        # 限制数量防止过多输出
        if ($count -ge 50) { break }
    }
} catch {
    Write-Error "列出目录失败: $($_.Exception.Message)"
    exit 1
}
`, strings.Replace(devicePath, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("列出根目录失败: %v", err)
	}

	return parseItemsOutput(string(output))
}

// findRecordingDirectories 查找录音相关目录
func findRecordingDirectories(devicePath string) ([]*DirItem, error) {
	script := fmt.Sprintf(`
try {
    $shell = New-Object -ComObject Shell.Application
    $folder = $shell.NameSpace('%s')
    if (-not $folder) {
        Write-Error "无法获取文件夹对象"
        exit 1
    }

    # 递归搜索函数
    function Find-RecordingDirs($currentFolder, $maxDepth = 3) {
        if ($maxDepth -le 0) { return }

        try {
            $items = $currentFolder.Items()
            foreach ($item in $items) {
                $name = $item.Name
                $path = $item.Path

                # 检查是否是录音相关目录
                if ($item.IsFolder -and ($name -like "*录音*" -or $name -like "*Record*" -or $name -like "*内部*" -or $name -like "*共享*" -or $name -like "*存储*")) {
                    Write-Output "RECORD_DIR|$name|$path"

                    # 在录音目录中进一步搜索
                    try {
                        $subFolder = $currentFolder.ParseName($name)
                        Find-RecordingDirs $subFolder ($maxDepth - 1)
                    } catch {
                        # 忽略访问错误
                    }
                } elseif ($item.IsFolder -and $maxDepth -gt 1) {
                    # 递归搜索其他文件夹
                    try {
                        $subFolder = $currentFolder.ParseName($name)
                        Find-RecordingDirs $subFolder ($maxDepth - 1)
                    } catch {
                        # 忽略访问错误
                    }
                }
            }
        } catch {
            # 忽略访问错误
        }
    }

    Find-RecordingDirs $folder
} catch {
    Write-Error "搜索录音目录失败: $($_.Exception.Message)"
    exit 1
}
`, strings.Replace(devicePath, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("搜索录音目录失败: %v", err)
	}

	return parseRecordDirsOutput(string(output))
}

// findOpusInDirectory 在指定目录中查找.opus文件
func findOpusInDirectory(dirPath string) ([]*DirItem, error) {
	script := fmt.Sprintf(`
try {
    $shell = New-Object -ComObject Shell.Application
    $folder = $shell.NameSpace('%s')
    if (-not $folder) {
        Write-Error "无法获取文件夹对象"
        exit 1
    }

    # 递归搜索.opus文件
    function Find-OpusFiles($currentFolder, $maxDepth = 3) {
        if ($maxDepth -le 0) { return }

        try {
            $items = $currentFolder.Items()
            foreach ($item in $items) {
                $name = $item.Name
                $path = $item.Path

                if ($name.ToLower().EndsWith(".opus")) {
                    $size = $item.Size
                    Write-Output "OPUS_FILE|$name|$path|$size"
                }

                if ($item.IsFolder -and $maxDepth -gt 1) {
                    try {
                        $subFolder = $currentFolder.ParseName($name)
                        Find-OpusFiles $subFolder ($maxDepth - 1)
                    } catch {
                        # 忽略访问错误
                    }
                }
            }
        } catch {
            # 忽略访问错误
        }
    }

    Find-OpusFiles $folder
} catch {
    Write-Error "搜索.opus文件失败: $($_.Exception.Message)"
    exit 1
}
`, strings.Replace(dirPath, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("搜索.opus文件失败: %v", err)
	}

	return parseOpusFilesOutput(string(output))
}

// testStandardRecordingPath 测试标准录音笔路径
func testStandardRecordingPath(path string) error {
	script := fmt.Sprintf(`
try {
    $exists = Test-Path -Path '%s'
    if ($exists) {
        Write-Output "STANDARD_PATH_EXISTS"

        # 尝试列出内容
        $shell = New-Object -ComObject Shell.Application
        $folder = $shell.NameSpace('%s')
        if ($folder) {
            $items = $folder.Items()
            $count = $items.Count
            Write-Output "STANDARD_PATH_ITEMS|$count"

            # 查找.opus文件
            foreach ($item in $items) {
                if ($item.Name.ToLower().EndsWith(".opus")) {
                    Write-Output "STANDARD_OPUS_FOUND|$($item.Name)|$($item.Size)"
                }
            }
        }
    } else {
        Write-Output "STANDARD_PATH_NOT_EXISTS"
    }
} catch {
    Write-Error "标准路径测试失败: $($_.Exception.Message)"
    exit 1
}
`, strings.Replace(path, "'", "''", -1), strings.Replace(path, "'", "''", -1))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PowerShell执行失败: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "STANDARD_PATH_EXISTS") {
			fmt.Println("  ✅ 标准路径存在")
		} else if strings.Contains(line, "STANDARD_PATH_NOT_EXISTS") {
			fmt.Println("  ❌ 标准路径不存在")
		} else if strings.HasPrefix(line, "STANDARD_PATH_ITEMS|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				fmt.Printf("  📁 标准路径包含 %s 个项目\n", parts[1])
			}
		} else if strings.HasPrefix(line, "STANDARD_OPUS_FOUND|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				fmt.Printf("  🎵 找到.opus文件: %s (%.2f MB)\n", parts[1], parseSize(parts[2]))
			}
		}
	}

	return nil
}

// parseItemsOutput 解析项目输出
func parseItemsOutput(output string) ([]*DirItem, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var items []*DirItem

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ITEM|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				item := &DirItem{
					Name: parts[1],
					Path: parts[2],
					Type: parts[3],
				}
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// parseRecordDirsOutput 解析录音目录输出
func parseRecordDirsOutput(output string) ([]*DirItem, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var dirs []*DirItem

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "RECORD_DIR|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				dir := &DirItem{
					Name: parts[1],
					Path: parts[2],
					Type: "文件夹",
				}
				dirs = append(dirs, dir)
			}
		}
	}

	return dirs, nil
}

// parseOpusFilesOutput 解析.opus文件输出
func parseOpusFilesOutput(output string) ([]*DirItem, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []*DirItem

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OPUS_FILE|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				size := parseSize(parts[3])
				file := &DirItem{
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

// parseSize 解析文件大小
func parseSize(sizeStr string) int64 {
	// 简单的大小解析，可以根据需要扩展
	var size int64
	fmt.Sscanf(sizeStr, "%d", &size)
	return size
}