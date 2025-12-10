package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== MTP设备简化测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 测试1: 检查设备连接状态
	fmt.Println("测试1: 检查SR302设备连接状态...")
	deviceStatus, err := checkDeviceStatus()
	if err != nil {
		fmt.Printf("❌ 设备检查失败: %v\n", err)
	} else {
		fmt.Printf("✅ 设备状态:\n%s\n", deviceStatus)
	}

	// 测试2: 尝试通过便携式设备访问
	fmt.Println("\n测试2: 通过便携式设备访问...")
	opusFiles, err := findOpusFilesSimple()
	if err != nil {
		fmt.Printf("❌ 便携式设备访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个.opus文件:\n", len(opusFiles))
		for i, file := range opusFiles {
			if i < 10 {
				fmt.Printf("  - %s (%.2f MB)\n", file.Name, float64(file.Size)/1024/1024)
			}
		}
		if len(opusFiles) > 10 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(opusFiles)-10)
		}
	}

	// 测试3: 尝试WMI路径访问
	fmt.Println("\n测试3: 尝试WMI路径访问...")
	wmiFiles, err := testWMIPathAccess()
	if err != nil {
		fmt.Printf("❌ WMI路径访问失败: %v\n", err)
	} else {
		fmt.Printf("✅ WMI访问找到 %d 个项目\n", len(wmiFiles))
	}

	fmt.Println("\n=== 测试完成 ===")

	// 总结测试结果
	fmt.Println("\n=== 总结 ===")
	if len(opusFiles) > 0 {
		fmt.Printf("✅ 成功：找到 %d 个.opus文件，可以继续开发\n", len(opusFiles))
	} else {
		fmt.Println("❌ 失败：未找到.opus文件，需要探索其他方案")
		fmt.Println("建议：")
		fmt.Println("1. 检查设备是否正确连接并出现在文件管理器中")
		fmt.Println("2. 尝试使用go-ole库实现COM接口")
		fmt.Println("3. 考虑使用第三方MTP库")
	}
}

// FileInfo 文件信息
type FileInfo struct {
	Name string
	Size int64
	Path string
}

// checkDeviceStatus 检查设备状态
func checkDeviceStatus() (string, error) {
	script := "$device = Get-WmiObject Win32_PnPEntity | Where-Object { $_.DeviceID -like '*VID_2207*' -and $_.DeviceID -like '*PID_0011*' }; if ($device) { '设备已连接: ' + $device.Name + '`n设备ID: ' + $device.DeviceID } else { '设备未连接'; exit 1 }"

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PowerShell执行失败: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// findOpusFilesSimple 简单查找.opus文件
func findOpusFilesSimple() ([]*FileInfo, error) {
	script := `
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
$count = 0

if ($portable) {
    function Search-Opus($folder, $depth) {
        if ($depth -le 0) { return }
        try {
            $items = $folder.Items()
            foreach ($item in $items) {
                if ($item.Name.ToLower().EndsWith(".opus")) {
                    "OPUS|" + $item.Name + "|" + $item.Path + "|" + $item.Size
                    $count++
                }
                if ($item.IsFolder -and $depth -gt 1) {
                    try {
                        $sub = $folder.ParseName($item.Name)
                        Search-Opus $sub ($depth - 1)
                    } catch { }
                }
                if ($count -gt 50) { return }
            }
        } catch { }
    }

    $devices = $portable.Items()
    foreach ($device in $devices) {
        if ($device.IsFolder) {
            Search-Opus $device 4
        }
        if ($count -gt 50) { break }
    }
}
`

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("搜索.opus文件失败: %v", err)
	}

	return parseOpusOutput(string(output))
}

// testWMIPathAccess 测试WMI路径访问
func testWMIPathAccess() ([]*FileInfo, error) {
	script := "$devices = Get-WmiObject Win32_PnPEntity | Where-Object { $_.DeviceID -like '*VID_2207*' -and $_.DeviceID -like '*PID_0011*' }; $count = 0; foreach ($device in $devices) { 'DEVICE|' + $device.Name + '|' + $device.DeviceID; $count++ }; 'COUNT|' + $count"

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("WMI路径测试失败: %v", err)
	}

	return parseWMIOutput(string(output))
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
					Size: size,
				}
				files = append(files, file)
			}
		}
	}

	return files, nil
}

// parseWMIOutput 解析WMI输出
func parseWMIOutput(output string) ([]*FileInfo, error) {
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
					Size: 0,
				}
				devices = append(devices, device)
			}
		}
	}

	return devices, nil
}