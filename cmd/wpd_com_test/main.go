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
	fmt.Println("=== WPD COM访问器测试 ===")
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

	logger := logger.NewLogger(true)
	logger.Setup("wpd_com_test", "debug", logDir, true, true)
	defer logger.Close()

	// 测试1: 创建WPD COM访问器
	fmt.Println("测试1: 创建WPD COM访问器...")
	wpdAccessor := device.NewWPDComAccessor(logger)

	// 测试2: 连接SR302设备
	fmt.Println("\n测试2: 连接SR302设备...")
	if err := wpdAccessor.ConnectToDevice("SR302", "2207", "0011"); err != nil {
		fmt.Printf("❌ 连接失败: %v\n", err)
	} else {
		fmt.Println("✅ 成功连接到设备")

		// 获取设备信息
		if deviceInfo := wpdAccessor.GetDeviceInfo(); deviceInfo != nil {
			fmt.Printf("设备信息:\n")
			fmt.Printf("  名称: %s\n", deviceInfo.Name)
			fmt.Printf("  VID: %s\n", deviceInfo.VID)
			fmt.Printf("  PID: %s\n", deviceInfo.PID)
			fmt.Printf("  设备ID: %s\n", deviceInfo.DeviceID)
		}

		// 测试3: 列出文件
		fmt.Println("\n测试3: 列出设备文件...")
		files, err := wpdAccessor.ListFiles("")
		if err != nil {
			fmt.Printf("❌ 列出文件失败: %v\n", err)
		} else {
			fmt.Printf("✅ 找到 %d 个文件:\n", len(files))
			opusCount := 0
			totalSize := int64(0)
			for i, file := range files {
				if i < 10 {
					status := "普通文件"
					if file.IsOpus {
						status = "🎵 Opus文件"
						opusCount++
					}
					fmt.Printf("  %2d. %s (%.2f MB) %s\n", i+1, file.Name, float64(file.Size)/1024/1024, status)
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

		// 测试4: 获取文件流
		if len(files) > 0 {
			fmt.Println("\n测试4: 获取文件流...")
			testFile := files[0]
			fmt.Printf("尝试获取文件流: %s\n", testFile.Name)
			stream, err := wpdAccessor.GetFileStream(testFile.Path)
			if err != nil {
				fmt.Printf("❌ 获取文件流失败: %v\n", err)
			} else {
				fmt.Printf("✅ 成功获取文件流\n")
				// 读取前100字节
				buffer := make([]byte, 100)
				n, err := stream.Read(buffer)
				if err != nil {
					fmt.Printf("❌ 读取文件流失败: %v\n", err)
				} else {
					fmt.Printf("✅ 成功读取 %d 字节\n", n)
					fmt.Printf("  数据前缀: %v\n", buffer[:min(n, 20)])
				}
				stream.Close()
			}
		}

		wpdAccessor.Close()
	}

	// 测试5: 使用设备桥接器
	fmt.Println("\n测试5: 使用设备桥接器...")
	bridge := device.NewDeviceBridge(logger, nil)

	if bridge == nil {
		fmt.Println("❌ 设备桥接器创建失败")
	} else {
		// 列出可用设备
		devices, err := bridge.ListAvailableDevices()
		if err != nil {
			fmt.Printf("❌ 列出可用设备失败: %v\n", err)
		} else {
			fmt.Printf("✅ 找到 %d 个可用设备:\n", len(devices))
			for i, device := range devices {
				fmt.Printf("  %d. %s (VID:%s, PID:%s)\n", i+1, device.Name, device.VID, device.PID)
			}

			// 尝试桥接第一个设备
			if len(devices) > 0 {
				fmt.Printf("\n尝试桥接设备: %s\n", devices[0].Name)
				mtpInterface, err := bridge.DetectAndBridge(devices[0].Name)
				if err != nil {
					fmt.Printf("❌ 设备桥接失败: %v\n", err)
				} else {
					fmt.Println("✅ 设备桥接成功")

					// 获取使用的访问器类型
					fmt.Printf("访问器类型: %T\n", mtpInterface)

					// 尝试列出文件
					bridgeFiles, err := mtpInterface.ListFiles("")
					if err != nil {
						fmt.Printf("❌ 桥接设备文件列表失败: %v\n", err)
					} else {
						fmt.Printf("✅ 桥接设备找到 %d 个文件\n", len(bridgeFiles))
					}

					mtpInterface.Close()
				}
			}

			bridge.Close()
		}
	}

	// 测试结果总结
	fmt.Println("\n=== WPD COM测试结果总结 ===")
	fmt.Println("✅ WPD COM访问器基础结构: 已创建")
	fmt.Println("✅ COM初始化和清理: 已实现")
	fmt.Println("⚠️ 设备连接: 需要实际的WPD API调用")
	fmt.Println("⚠️ 文件枚举: 需要实际的WPD API调用")
	fmt.Println("⚠️ 文件流访问: 框架已实现")

	fmt.Println("\n下一步:")
	fmt.Println("1. 实现实际的WPD API调用")
	fmt.Println("2. 完善文件枚举逻辑")
	fmt.Println("3. 实现文件流读取")
	fmt.Println("4. 集成到主备份流程")

	fmt.Println("\n测试完成，按任意键退出...")
	var input string
	fmt.Scanln(&input)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}