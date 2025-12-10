//go:build windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
)

func main() {
	fmt.Println("=== OLE Shell设备访问测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n收到中断信号，退出程序...")
		os.Exit(0)
	}()

	// 创建日志器
	logDir := "logs"
	os.MkdirAll(logDir, 0755)

	logger, err := logger.NewLogger(&logger.Config{
		Level:      "debug",
		Console:    true,
		File:       true,
		FilePath:   "logs/ole_test.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	})
	if err != nil {
		fmt.Printf("创建日志器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 测试1: 初始化OLE Shell
	fmt.Println("测试1: 初始化OLE Shell...")
	oleShell := device.NewOLEShellAccessor(logger)

	if err := oleShell.Initialize(); err != nil {
		fmt.Printf("❌ 初始化OLE Shell失败: %v\n", err)
		return
	}
	defer oleShell.Close()
	fmt.Println("✅ OLE Shell初始化成功")

	// 测试2: 查找便携式设备
	fmt.Println("\n测试2: 查找便携式设备...")
	devices, err := oleShell.FindPortableDevices()
	if err != nil {
		fmt.Printf("❌ 查找便携式设备失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个便携式设备:\n", len(devices))
		for i, device := range devices {
			fmt.Printf("  %d. %s\n", i+1, device.Name)
			fmt.Printf("     ID: %s\n", device.DeviceID)
		}
	}

	// 测试3: 使用OLE WPD访问器
	fmt.Println("\n测试3: 使用OLE WPD访问器...")
	wpd := device.NewOLEWPDAccessor(logger)

	if err := wpd.ConnectToDevice("SR302", "2207", "0011"); err != nil {
		fmt.Printf("❌ OLE WPD连接设备失败: %v\n", err)

		// 尝试更宽松的匹配
		fmt.Println("\n尝试更宽松的设备匹配...")
		if devices != nil && len(devices) > 0 {
			fmt.Printf("尝试连接到第一个设备: %s\n", devices[0].Name)
			if err := wpd.ConnectToDevice(devices[0].Name, "", ""); err != nil {
				fmt.Printf("❌ 连接第一个设备也失败: %v\n", err)
			} else {
				fmt.Println("✅ 成功连接到第一个设备")
			}
		}
	}

	if wpd.IsConnected() {
		// 获取设备信息
		if deviceInfo := wpd.GetDeviceInfo(); deviceInfo != nil {
			fmt.Printf("✅ 连接的设备: %s\n", deviceInfo.Name)
			fmt.Printf("   设备ID: %s\n", deviceInfo.DeviceID)
		}

		// 测试4: 列出文件
		fmt.Println("\n测试4: 列出设备文件...")
		files, err := wpd.ListFiles("")
		if err != nil {
			fmt.Printf("❌ 列出文件失败: %v\n", err)
		} else {
			fmt.Printf("✅ 找到 %d 个文件:\n", len(files))
			opusCount := 0
			totalSize := int64(0)
			for i, file := range files {
				if i < 10 {
					fmt.Printf("  - %s (%.2f MB, Opus: %t)\n",
						file.Name,
						float64(file.Size)/1024/1024,
						file.IsOpus)
				}
				if file.IsOpus {
					opusCount++
				}
				totalSize += file.Size
			}
			if len(files) > 10 {
				fmt.Printf("  ... 还有 %d 个文件\n", len(files)-10)
			}
			fmt.Printf("\n📊 统计信息:\n")
			fmt.Printf("   总文件数: %d\n", len(files))
			fmt.Printf("   Opus文件数: %d\n", opusCount)
			fmt.Printf("   总大小: %.2f MB\n", float64(totalSize)/1024/1024)
		}

		// 测试5: 获取设备属性
		fmt.Println("\n测试5: 获取设备属性...")
		properties, err := wpd.GetDeviceProperties()
		if err != nil {
			fmt.Printf("❌ 获取设备属性失败: %v\n", err)
		} else {
			fmt.Println("✅ 设备属性:")
			for key, value := range properties {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		// 测试6: 按模式搜索文件
		fmt.Println("\n测试6: 搜索特定类型文件...")
		patternFiles, err := wpd.GetFilesByPattern(".opus")
		if err != nil {
			fmt.Printf("❌ 搜索.opus文件失败: %v\n", err)
		} else {
			fmt.Printf("✅ 找到 %d 个.opus文件:\n", len(patternFiles))
			for i, file := range patternFiles {
				if i < 5 {
					fmt.Printf("  - %s (%.2f MB)\n", file.Name, float64(file.Size)/1024/1024)
				}
			}
			if len(patternFiles) > 5 {
				fmt.Printf("  ... 还有 %d 个文件\n", len(patternFiles)-5)
			}
		}

		wpd.Close()
	} else {
		fmt.Println("❌ 无法连接到任何设备")
	}

	// 测试结果总结
	fmt.Println("\n=== 测试结果总结 ===")
	if devices != nil && len(devices) > 0 {
		fmt.Println("✅ OLE COM接口工作正常")
		fmt.Println("✅ 能够检测到便携式设备")

		if wpd.IsConnected() {
			fmt.Println("✅ 成功连接到设备")
			fmt.Println("\n✅ OLE方案可行！")
			fmt.Println("下一步:")
			fmt.Println("1. 完善文件流访问功能")
			fmt.Println("2. 实现文件复制到本地")
			fmt.Println("3. 集成到主程序MTP框架")
			fmt.Println("4. 添加进度显示和错误处理")
		} else {
			fmt.Println("⚠️ 检测到设备但连接失败")
			fmt.Println("需要:")
			fmt.Println("1. 调试设备匹配逻辑")
			fmt.Println("2. 改进设备路径解析")
		}
	} else {
		fmt.Println("❌ 无法检测到便携式设备")
		fmt.Println("可能的原因:")
		fmt.Println("1. 设备未正确连接")
		fmt.Println("2. 设备驱动程序问题")
		fmt.Println("3. 权限不足")
		fmt.Println("4. COM接口初始化失败")
		fmt.Println("\n建议:")
		fmt.Println("1. 确认设备在文件管理器中可见")
		fmt.Println("2. 尝试管理员权限运行")
		fmt.Println("3. 检查Windows是否安装了MTP驱动")
	}

	fmt.Println("\n测试完成，按任意键退出...")
	var input string
	fmt.Scanln(&input)
}