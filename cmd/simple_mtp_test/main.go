//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== 简单MTP测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 测试1: 直接访问SR302设备
	fmt.Println("测试1: 直接访问SR302设备...")
	files, err := accessSR302Direct()
	if err != nil {
		fmt.Printf("❌ 直接访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个文件:\n", len(files))
		printFiles(files)
	}

	// 测试2: 便携式设备枚举
	fmt.Println("\n测试2: 便携式设备枚举...")
	files2, err := enumeratePortableDevice()
	if err != nil {
		fmt.Printf("❌ 便携式设备枚举失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个文件:\n", len(files2))
		printFiles(files2)
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

// accessSR302Direct 直接访问SR302设备
func accessSR302Direct() ([]FileInfo, error) {
	script := `$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $items = $portable.Items()
    foreach ($item in $items) {
        if ($item.Name -eq "SR302") {
            Write-Host "找到SR302设备"
            try {
                $deviceFolder = $portable.ParseName("SR302")
                if ($deviceFolder) {
                    $subItems = $deviceFolder.Items()
                    Write-Host "找到 $($subItems.Count) 个子项目"
                    foreach ($subItem in $subItems) {
                        $name = $subItem.Name
                        $isFolder = $subItem.IsFolder
                        Write-Host "项目: $name, 文件夹: $isFolder"
                        if (-not $isFolder) {
                            $ext = [System.IO.Path]::GetExtension($name).ToLower()
                            if ($ext -eq ".opus") {
                                Write-Output "FILE|$name|$($subItem.Size)|$($subItem.Path)"
                            }
                        }
                    }
                } else {
                    Write-Host "无法获取SR302文件夹"
                }
            } catch {
                Write-Host "错误: $($_.Exception.Message)"
            }
            break
        }
    }
} else {
    Write-Host "无法获取便携式设备命名空间"
}
Write-Output "END"`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("PowerShell执行失败: %w, 输出: %s", err, string(output))
	}

	return parseOutput(string(output))
}

// enumeratePortableDevice 枚举便携式设备
func enumeratePortableDevice() ([]FileInfo, error) {
	script := `$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    Write-Host "便携式设备命名空间可访问"
    $items = $portable.Items()
    Write-Host "找到 $($items.Count) 个设备:"
    foreach ($item in $items) {
        Write-Host "  设备: $($item.Name)"
    }
} else {
    Write-Host "便携式设备命名空间不可访问"
}
Write-Output "END"`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("PowerShell执行失败: %w, 输出: %s", err, string(output))
	}

	fmt.Println("PowerShell输出:")
	fmt.Println(string(output))

	return []FileInfo{}, nil
}

func parseOutput(output string) ([]FileInfo, error) {
	lines := strings.Split(output, "\n")
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