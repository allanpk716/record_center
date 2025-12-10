//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
)

func main() {
	fmt.Println("=== WPD COM设备访问测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 设置信号处理，确保正确清理COM资源
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n收到中断信号，清理资源...")
		device.CleanupOle()
		os.Exit(0)
	}()

	// 创建日志器
	logDir := "logs"
	os.MkdirAll(logDir, 0755)

	logger, err := logger.NewLogger(&logger.Config{
		Level:      "debug",
		Console:    true,
		File:       true,
		FilePath:   "logs/wpd_test.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	})
	if err != nil {
		log.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	// 测试1: 初始化COM接口
	fmt.Println("测试1: 初始化COM接口...")
	com, err := device.NewCOMInterface(logger)
	if err != nil {
		fmt.Printf("❌ 创建COM接口失败: %v\n", err)
		return
	}
	defer com.Close()

	if err := com.Initialize(); err != nil {
		fmt.Printf("❌ 初始化COM接口失败: %v\n", err)
		return
	}
	fmt.Println("✅ COM接口初始化成功")

	// 测试2: 查找便携式设备
	fmt.Println("\n测试2: 查找便携式设备...")
	devices, err := com.FindPortableDevices()
	if err != nil {
		fmt.Printf("❌ 查找便携式设备失败: %v\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个便携式设备:\n", len(devices))
		for i, device := range devices {
			fmt.Printf("  %d. %s (ID: %s)\n", i+1, device.Name, device.DeviceID)
		}
	}

	// 测试3: 使用WPD访问器
	fmt.Println("\n测试3: 使用WPD访问器连接设备...")
	wpd := device.NewWPDAccessor(logger)

	if err := wpd.ConnectToDevice("SR302", "2207", "0011"); err != nil {
		fmt.Printf("❌ WPD连接设备失败: %v\n", err)
	} else {
		fmt.Println("✅ WPD成功连接到设备")

		// 获取设备信息
		if deviceInfo := wpd.GetDeviceInfo(); deviceInfo != nil {
			fmt.Printf("设备信息: %s\n", deviceInfo.Name)
		}

		// 测试4: 列出文件
		fmt.Println("\n测试4: 列出设备文件...")
		files, err := wpd.ListFiles("")
		if err != nil {
			fmt.Printf("❌ 列出文件失败: %v\n", err)
		} else {
			fmt.Printf("✅ 找到 %d 个文件:\n", len(files))
			opusCount := 0
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
			}
			if len(files) > 10 {
				fmt.Printf("  ... 还有 %d 个文件\n", len(files)-10)
			}
			fmt.Printf("\n🎵 找到 %d 个.opus文件\n", opusCount)
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

		wpd.Close()
	}

	// 测试6: 按模式搜索文件
	fmt.Println("\n测试6: 搜索.opus文件...")
	if devices != nil && len(devices) > 0 {
		// 重新连接以搜索特定文件
		if err := wpd.ConnectToDevice(devices[0].Name, "2207", "0011"); err == nil {
			patternFiles, err := wpd.GetFilesByPattern(".opus")
			if err != nil {
				fmt.Printf("❌ 搜索.opus文件失败: %v\n", err)
			} else {
				fmt.Printf("✅ 按模式搜索找到 %d 个.opus文件:\n", len(patternFiles))
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
		}
	}

	// 测试结果总结
	fmt.Println("\n=== 测试结果总结 ===")
	if devices != nil && len(devices) > 0 {
		fmt.Println("✅ COM接口工作正常")
		fmt.Println("✅ 能够检测到便携式设备")
		fmt.Println("⚠️ 需要进一步优化文件访问")
		fmt.Println("\n下一步:")
		fmt.Println("1. 完善文件流访问功能")
		fmt.Println("2. 优化设备路径解析")
		fmt.Println("3. 集成到主程序中")
	} else {
		fmt.Println("❌ COM接口无法检测到设备")
		fmt.Println("建议:")
		fmt.Println("1. 确认设备已正确连接")
		fmt.Println("2. 检查设备驱动程序")
		fmt.Println("3. 尝试管理员权限运行")
	}

	fmt.Println("\n测试完成，按任意键退出...")
	var input string
	fmt.Scanln(&input)

	// 清理COM资源
	device.CleanupOle()
}