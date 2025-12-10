package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/pkg/utils"
)

// ProgressDisplay 进度显示器
type ProgressDisplay struct {
	tracker       *ProgressTracker
	progressBar   *progressbar.ProgressBar
	ticker        *time.Ticker
	done          chan bool
	quiet         bool
	log           *logger.Logger
	lastDisplay   time.Time
}

// NewProgressDisplay 创建新的进度显示器
func NewProgressDisplay(tracker *ProgressTracker, quiet bool, log *logger.Logger) *ProgressDisplay {
	return &ProgressDisplay{
		tracker: tracker,
		quiet:   quiet,
		log:     log,
		done:    make(chan bool),
	}
}

// Start 开始显示进度
func (pd *ProgressDisplay) Start() error {
	if pd.quiet {
		pd.log.Info("静默模式：进度显示已禁用")
		return nil
	}

	// 创建进度条
	info := pd.tracker.GetProgressInfo()
	pd.progressBar = progressbar.NewOptions64(
		info.TotalSize,
		progressbar.OptionSetDescription("备份进度"),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetItsString("B"),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	// 启动定时更新
	pd.ticker = time.NewTicker(500 * time.Millisecond)
	go pd.updateDisplay()

	pd.log.Debug("进度显示器已启动")
	return nil
}

// StartDelayed 延迟启动进度显示，使用传入的参数
func (pd *ProgressDisplay) StartDelayed(totalFiles int, totalSize int64) error {
	if pd.quiet {
		pd.log.Info("静默模式：进度显示已禁用")
		return nil
	}

	// 创建进度条
	pd.progressBar = progressbar.NewOptions64(
		totalSize,
		progressbar.OptionSetDescription("备份进度"),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetItsString("B"),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	// 启动定时更新
	pd.ticker = time.NewTicker(500 * time.Millisecond)
	go pd.updateDisplay()

	pd.log.Debug("进度显示器已启动（延迟模式）")
	return nil
}

// Stop 停止显示进度
func (pd *ProgressDisplay) Stop() {
	if pd.quiet {
		return
	}

	if pd.ticker != nil {
		pd.ticker.Stop()
	}

	pd.done <- true

	// 显示最终完成信息
	if pd.progressBar != nil {
		pd.progressBar.Finish()
	}

	fmt.Println() // 换行
	pd.log.Debug("进度显示器已停止")
}

// updateDisplay 更新显示内容
func (pd *ProgressDisplay) updateDisplay() {
	for {
		select {
		case <-pd.ticker.C:
			pd.refreshDisplay()
		case <-pd.done:
			return
		}
	}
}

// refreshDisplay 刷新显示
func (pd *ProgressDisplay) refreshDisplay() {
	info := pd.tracker.GetProgressInfo()

	// 限制更新频率（至少间隔100ms）
	now := time.Now()
	if now.Sub(pd.lastDisplay) < 100*time.Millisecond {
		return
	}
	pd.lastDisplay = now

	// 更新进度条
	if pd.progressBar != nil && info.TotalSize > 0 {
		pd.progressBar.Set64(info.CopiedSize)
	}

	// 显示详细信息
	pd.displayDetailedInfo(info)
}

// displayDetailedInfo 显示详细信息
func (pd *ProgressDisplay) displayDetailedInfo(info *ProgressInfo) {
	if pd.quiet {
		return
	}

	// 清屏并移动到行首
	fmt.Print("\033[H\033[2J")

	// 显示标题
	fmt.Println(color.CyanString("录音笔备份工具 v1.0"))
	fmt.Println(color.WhiteString(strings.Repeat("=", 50)))

	// 显示总体进度
	fmt.Printf(color.YellowString("总文件数: %d 文件 (%s)\n"), info.TotalFiles, utils.FormatBytes(info.TotalSize))
	fmt.Printf(color.GreenString("已完成: %d/%d 文件 (%.1f%%)\n"), info.CompletedFiles, info.TotalFiles, info.ProgressPercent)

	fmt.Println() // 空行

	// 显示当前文件信息
	if info.CurrentFile != "" {
		fmt.Printf(color.CyanString("当前文件: %s\n"), info.CurrentFile)
		if info.TotalSize > 0 {
			fileProgress := float64(info.CopiedSize) / float64(info.TotalSize) * 100
			if fileProgress > 100 {
				fileProgress = 100
			}

			// 显示文件进度条
			barWidth := 50
			completed := int(fileProgress * float64(barWidth) / 100)
			bar := strings.Repeat("█", completed) + strings.Repeat("░", barWidth-completed)
			fmt.Printf(color.GreenString("文件进度: [%s] %.1f%% [%s/%s]\n"),
				bar, fileProgress,
				utils.FormatBytes(info.CopiedSize),
				utils.FormatBytes(info.TotalSize))
		}
	}

	fmt.Println() // 空行

	// 显示速度和时间信息
	fmt.Printf(color.GreenString("速度: %.2f MB/s | "), info.Speed)

	if info.EstimatedTime > 0 {
		fmt.Printf(color.YellowString("剩余时间: %s | "), utils.FormatDuration(info.EstimatedTime))
	}

	fmt.Printf(color.CyanString("已用时间: %s\n"), utils.FormatDuration(info.ElapsedTime))

	// 显示平均速度
	avgSpeed := pd.tracker.GetAverageSpeed()
	if avgSpeed > 0 {
		fmt.Printf(color.MagentaString("平均速度: %.2f MB/s\n"), avgSpeed)
	}

	// 每隔一定时间记录到日志
	if time.Since(pd.lastDisplay) > 5*time.Second {
		pd.tracker.LogProgress()
	}
}

// ShowCompletion 显示完成信息
func (pd *ProgressDisplay) ShowCompletion() {
	if pd.quiet {
		pd.log.Info("备份完成")
		return
	}

	info := pd.tracker.GetProgressInfo()
	totalTime := info.ElapsedTime

	fmt.Println() // 换行
	fmt.Println(color.GreenString(strings.Repeat("=", 50)))
	fmt.Println(color.GreenString("✅ 备份完成！"))
	fmt.Printf(color.CyanString("总计: %d 个文件, %s\n"), info.CompletedFiles, utils.FormatBytes(info.TotalSize))
	fmt.Printf(color.CyanString("耗时: %s\n"), utils.FormatDuration(totalTime))

	if totalTime > 0 {
		avgSpeed := float64(info.TotalSize) / totalTime.Seconds() / 1024 / 1024
		fmt.Printf(color.CyanString("平均速度: %.2f MB/s\n"), avgSpeed)
	}

	fmt.Println() // 空行
}

// ShowError 显示错误信息
func (pd *ProgressDisplay) ShowError(err error) {
	if pd.quiet {
		pd.log.Error("备份失败: %v", err)
		return
	}

	fmt.Println() // 换行
	fmt.Println(color.RedString(strings.Repeat("=", 50)))
	fmt.Println(color.RedString("❌ 备份失败！"))
	fmt.Printf(color.RedString("错误: %v\n"), err)
	fmt.Println() // 空行
}

// ShowWarning 显示警告信息
func (pd *ProgressDisplay) ShowWarning(message string) {
	if pd.quiet {
		pd.log.Warn("警告: %s", message)
		return
	}

	fmt.Printf(color.YellowString("⚠️  警告: %s\n"), message)
}

// ShowInfo 显示信息
func (pd *ProgressDisplay) ShowInfo(message string) {
	if pd.quiet {
		pd.log.Info("信息: %s", message)
		return
	}

	fmt.Printf(color.BlueString("ℹ️  信息: %s\n"), message)
}

// UpdateStatus 更新状态显示
func (pd *ProgressDisplay) UpdateStatus(status string) {
	if pd.quiet {
		pd.log.Debug("状态: %s", status)
		return
	}

	// 移动到状态行并更新
	fmt.Printf("\033[7;0H") // 移动到第7行开始
	fmt.Print(color.WhiteString(strings.Repeat(" ", 50))) // 清除行内容
	fmt.Printf("\033[7;0H") // 重新移动到第7行开始
	fmt.Printf(color.WhiteString("状态: %s"), status)
}

// ProgressIndicator 简单的进度指示器（用于不确定时长的操作）
func (pd *ProgressDisplay) ProgressIndicator(message string, done <-chan bool) {
	if pd.quiet {
		pd.log.Info(message)
		<-done
		return
	}

	indicators := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("\r%s %s", color.CyanString(indicators[i]), message)
			i = (i + 1) % len(indicators)
		case <-done:
			fmt.Printf("\r%s %s\n", color.GreenString("✓"), message)
			return
		}
	}
}