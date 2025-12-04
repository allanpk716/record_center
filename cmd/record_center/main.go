package main

import (
	"fmt"
	"os"

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
		manager := backup.NewManager(cfg, log, quiet)

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

// init 函数在main之前执行，用于初始化命令行参数
func init() {
	// 定义命令行参数
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "configs/backup.yaml", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细模式，显示更多信息")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "静默模式，不显示实时进度")
	rootCmd.PersistentFlags().BoolVarP(&check, "check", "k", false, "检查模式，只扫描不备份")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "强制重新备份，忽略已备份记录")
	rootCmd.PersistentFlags().StringVarP(&targetDir, "target", "t", "", "指定备份目标目录（覆盖配置文件）")
}

func main() {
	// 执行根命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}