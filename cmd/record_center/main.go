package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/allanpk716/record_center/internal/config"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/backup"
	"github.com/allanpk716/record_center/internal/logger"
	"github.com/spf13/cobra"
)

var (
	configFile  string
	verbose     bool
	quiet       bool
	check       bool
	force       bool
	targetDir   string
)

// rootCmd 代表基础命令，没有参数就执行
var rootCmd = &cobra.Command{
	Use:   "record_center",
	Short: "录音笔备份工具",
	Long: `一个专门为SR302录音笔设计的自动备份工具。
支持MTP设备检测、增量备份、实时进度显示等功能。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化日志
		log := logger.InitLogger(verbose)
		log.Info("录音笔备份工具启动")

		// 加载配置
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			log.Error("配置加载失败: %v", err)
			os.Exit(1)
		}

		// 如果命令行指定了目标目录，覆盖配置文件中的设置
		if targetDir != "" {
			cfg.Target.BaseDirectory = targetDir
			log.Info("使用命令行指定的目标目录: %s", targetDir)
		}

		// 检测设备
		log.Info("正在检测SR302录音笔设备...")
		sr302Device, err := device.DetectSR302()
		if err != nil {
			log.Error("设备检测失败: %v", err)
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}

		log.Info("找到设备: %s (ID: %s)", sr302Device.Name, sr302Device.DeviceID)
		log.Info("VID: %s, PID: %s", sr302Device.VID, sr302Device.PID)

		// 创建备份管理器
		manager := backup.NewManager(cfg, log, quiet, verbose)

		// 执行备份
		if check {
			log.Info("检查模式: 仅扫描文件，不执行备份")
			err = manager.Check(sr302Device)
		} else {
			err = manager.Run(sr302Device, force)
		}

		if err != nil {
			log.Error("操作失败: %v", err)
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}

		log.Info("操作完成")
	},
}

// detectCmd 设备检测命令
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "检测并列出所有可用的录音笔设备",
	Long: `自动扫描并列出所有连接的录音笔设备信息，包括设备名称、VID和PID。
支持的设备包括SR302以及其他类似的录音设备。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化日志
		log := logger.InitLogger(verbose)
		log.Info("开始检测录音笔设备...")

		// 检测所有录音笔相关设备
		devices := detectAllRecordingDevices(log)

		if len(devices) == 0 {
			fmt.Println("❌ 未找到任何录音笔设备")
			fmt.Println("请确保：")
			fmt.Println("1. 录音笔已连接到电脑")
			fmt.Println("2. 设备驱动程序已正确安装")
			fmt.Println("3. 设备处于可访问状态")
			os.Exit(1)
		}

		fmt.Println("\n🎤 检测到的录音笔设备：")
		fmt.Println("=" + strings.Repeat("=", 60))

		// 显示所有检测到的设备
		for i, dev := range devices {
			fmt.Printf("\n📱 设备 #%d\n", i+1)
			fmt.Printf("   名称: %s\n", dev.Name)
			fmt.Printf("   VID:  %s\n", dev.VID)
			fmt.Printf("   PID:  %s\n", dev.PID)
			fmt.Printf("   ID:   %s\n", dev.DeviceID)

			// 生成配置片段
			fmt.Printf("\n   配置片段：\n")
			fmt.Printf("   source:\n")
			fmt.Printf("     device_name: \"%s\"\n", dev.Name)
			fmt.Printf("     vid: \"%s\"\n", dev.VID)
			fmt.Printf("     pid: \"%s\"\n", dev.PID)
			fmt.Println()
		}

		// 检查是否有SR302设备
		sr302Found := false
		for _, dev := range devices {
			if strings.Contains(strings.ToUpper(dev.Name), "SR302") ||
			   (dev.VID == "2207" && dev.PID == "0011") {
				sr302Found = true
				fmt.Println("✅ 检测到SR302设备！")
				fmt.Println("   您可以使用以下配置：")
				fmt.Printf("   device_name: \"%s\"\n", dev.Name)
				fmt.Printf("   vid: \"%s\"\n", dev.VID)
				fmt.Printf("   pid: \"%s\"\n", dev.PID)
				break
			}
		}

		if !sr302Found {
			fmt.Println("⚠️  未检测到SR302设备，但找到了其他录音设备")
			fmt.Println("   您可以尝试使用上述设备配置")
		}

		fmt.Println("\n" + strings.Repeat("=", 64))
		fmt.Println("💡 提示：")
		fmt.Println("   - 复制上述配置片段到 configs/backup.yaml 文件中")
		fmt.Println("   - 然后运行 record_center --check 测试配置")
		fmt.Println("   - 使用 record_center --verbose 查看详细日志")
	},
}

// detectAllRecordingDevices 检测所有录音笔相关设备
func detectAllRecordingDevices(log *logger.Logger) []*device.DeviceInfo {
	var allDevices []*device.DeviceInfo

	// 方法1: 使用现有的SR302检测功能
	log.Debug("尝试检测SR302设备...")
	if sr302Device, err := device.DetectSR302(); err == nil {
		allDevices = append(allDevices, sr302Device)
		log.Debug("找到SR302设备: %s", sr302Device.Name)
	}

	// 方法2: 扫描其他可能的录音设备
	log.Debug("扫描其他录音设备...")
	otherDevices := scanForRecordingDevices(log)
	allDevices = append(allDevices, otherDevices...)

	// 去重
	return removeDuplicateDevices(allDevices)
}

// scanForRecordingDevices 扫描其他录音设备
func scanForRecordingDevices(log *logger.Logger) []*device.DeviceInfo {
	var devices []*device.DeviceInfo

	// 获取所有USB设备
	usbDevices, err := device.ScanAllUSBDevices()
	if err != nil {
		log.Warn("扫描USB设备失败: %v", err)
		return devices
	}

	// 筛选可能是录音笔的设备
	for _, usbDevice := range usbDevices {
		// 检查设备名称是否包含录音相关的关键词
		deviceName := strings.ToUpper(usbDevice.Name)

		// 录音设备关键词
		recordingKeywords := []string{
			"录音", "RECORD", "VOICE", "RECORDER", "音频", "AUDIO",
			"便携式", "PORTABLE", "存储", "STORAGE",
			"SR", "IC", "OLYMPUS", "SONY", "ZOOM",
			"MTP", "USB", "DEVICE",
		}

		isRecordingDevice := false
		for _, keyword := range recordingKeywords {
			if strings.Contains(deviceName, keyword) {
				isRecordingDevice = true
				break
			}
		}

		// 也检查VID是否来自已知的音频设备制造商
		audioVendors := []string{
			"2207", // SR302的制造商
			"046D", "0483", "0582", "0951", "10D6", "12BA", "13D3",
			"18D1", "1B71", "2006", "2109", "2207", "2341", "2478",
			"2717", "2886", "2A03", "2C99", "3282", "33F3", "3742",
			"3823", "4035", "4097", "4348", "49E3", "4A66", "5051",
			"5740", "59E3", "612B", "6868", "7881", "7904", "79A5",
			"8087", "8564", "9DA2", "A0A2", "A5C2", "AC8F", "B58E",
			"C251", "CA01", "CAFE", "DECA", "E401", "E856", "FCEA",
		}

		isAudioVendor := false
		for _, vendor := range audioVendors {
			if strings.EqualFold(usbDevice.VID, vendor) {
				isAudioVendor = true
				break
			}
		}

		// 如果是录音设备或音频制造商设备，加入列表
		if isRecordingDevice || isAudioVendor {
			// 检查是否已存在（避免重复）
			exists := false
			for _, existing := range devices {
				if existing.VID == usbDevice.VID && existing.PID == usbDevice.PID {
					exists = true
					break
				}
			}

			if !exists {
				log.Debug("发现潜在的录音设备: %s (VID:%s, PID:%s)",
					usbDevice.Name, usbDevice.VID, usbDevice.PID)
				devices = append(devices, usbDevice)
			}
		}
	}

	return devices
}

// removeDuplicateDevices 移除重复的设备
func removeDuplicateDevices(devices []*device.DeviceInfo) []*device.DeviceInfo {
	var unique []*device.DeviceInfo
	seen := make(map[string]bool)

	for _, dev := range devices {
		key := fmt.Sprintf("%s:%s", dev.VID, dev.PID)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, dev)
		}
	}

	return unique
}

// init 函数在main之前执行，用于初始化命令行参数
func init() {
	// 定义命令行参数
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "backup.yaml", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细模式，显示更多信息")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "静默模式，不显示实时进度")
	rootCmd.PersistentFlags().BoolVarP(&check, "check", "k", false, "检查模式，只扫描不备份")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "强制重新备份，忽略已备份记录")
	rootCmd.PersistentFlags().StringVarP(&targetDir, "target", "t", "", "指定备份目标目录（覆盖配置文件）")

	// 添加detect子命令
	rootCmd.AddCommand(detectCmd)
}

func main() {
	// 执行根命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}