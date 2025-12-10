package backup

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/allanpk716/record_center/internal/config"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/internal/progress"
	"github.com/allanpk716/record_center/internal/storage"
	"github.com/allanpk716/record_center/pkg/utils"
)

// BackupManager 备份管理器
type BackupManager struct {
	config    *config.Config
	log       *logger.Logger
	tracker   *storage.BackupTracker
	quiet     bool
	verbose   bool
}

// NewManager 创建新的备份管理器
func NewManager(cfg *config.Config, log *logger.Logger, quiet, verbose bool) *BackupManager {
	// 初始化备份跟踪器
	tracker := storage.NewBackupTracker("data/backup_records.json", log)
	if err := tracker.Load(); err != nil {
		log.Warn("加载备份记录失败，将创建新记录: %v", err)
	}

	return &BackupManager{
		config:  cfg,
		log:     log,
		tracker: tracker,
		quiet:   quiet,
		verbose: verbose,
	}
}

// Run 执行备份
func (bm *BackupManager) Run(device *device.DeviceInfo, force bool) error {
	startTime := time.Now()
	bm.log.Info("开始备份操作，设备: %s (VID:%s, PID:%s)", device.Name, device.VID, device.PID)

	// 创建文件检查器
	fileChecker := bm.createFileChecker(device)

	// 扫描设备文件
	bm.log.Info("正在扫描设备文件...")
	allFiles, err := fileChecker.ScanDeviceFiles(device)
	if err != nil {
		return fmt.Errorf("扫描设备文件失败: %w", err)
	}

	if len(allFiles) == 0 {
		bm.log.Info("没有发现.opus文件，备份完成")
		return nil
	}

	bm.log.Info("扫描完成，发现 %d 个文件", len(allFiles))

	// 过滤需要备份的文件
	filesToBackup, err := fileChecker.FilterFilesToBackup(allFiles, device.DeviceID, force)
	if err != nil {
		return fmt.Errorf("过滤备份文件失败: %w", err)
	}

	// 生成备份预览
	preview, err := bm.GeneratePreview(device, allFiles, filesToBackup)
	if err != nil {
		return fmt.Errorf("生成预览失败: %w", err)
	}

	// 显示预览信息
	bm.DisplayPreview(preview, bm.verbose)
	bm.DisplayPreviewSummary(preview)

	if len(filesToBackup) == 0 {
		bm.log.Info("没有需要备份的新文件")
		return nil
	}

	// 创建进度组件（在确定需要备份后才创建）
	progressTracker := progress.NewProgressTracker(bm.log)
	progressDisplay := progress.NewProgressDisplay(progressTracker, bm.quiet, bm.log)

	// 开始进度跟踪
	if err := progressTracker.StartWithParams(len(filesToBackup), utils.CalculateTotalSize(filesToBackup)); err != nil {
		return fmt.Errorf("启动进度跟踪失败: %w", err)
	}

	// 启动进度显示（使用延迟启动方式）
	if err := progressDisplay.StartDelayed(len(filesToBackup), utils.CalculateTotalSize(filesToBackup)); err != nil {
		bm.log.Warn("启动进度显示失败: %v", err)
	}
	defer progressDisplay.Stop()

	// 检查磁盘空间
	if err := fileChecker.CheckDiskSpace(filesToBackup); err != nil {
		bm.log.Warn("磁盘空间检查失败: %v", err)
	}

	// 创建文件复制器
	copier := bm.createFileCopier(device)

	// 执行文件复制
	bm.log.Info("开始复制 %d 个文件...", len(filesToBackup))
	results := bm.copyFilesWithProgress(copier, filesToBackup, progressTracker, progressDisplay, force)

	// 处理结果
	if err := bm.processCopyResults(results, progressDisplay); err != nil {
		return err
	}

	// 保存备份记录
	if err := bm.tracker.Save(); err != nil {
		bm.log.Warn("保存备份记录失败: %v", err)
	}

	// 显示统计信息
	bm.showBackupStatistics(startTime, len(allFiles), len(filesToBackup), results)

	progressDisplay.ShowCompletion()
	bm.log.Info("备份操作完成")
	return nil
}

// Check 检查设备文件（不执行备份）
func (bm *BackupManager) Check(device *device.DeviceInfo) error {
	bm.log.Info("检查模式: 仅扫描文件，不执行备份")

	fileChecker := bm.createFileChecker(device)

	// 扫描设备文件
	allFiles, err := fileChecker.ScanDeviceFiles(device)
	if err != nil {
		return fmt.Errorf("扫描设备文件失败: %w", err)
	}

	if len(allFiles) == 0 {
		bm.log.Info("没有发现.opus文件")
		return nil
	}

	// 过滤需要备份的文件
	filesToBackup, err := fileChecker.FilterFilesToBackup(allFiles, device.DeviceID, false)
	if err != nil {
		return fmt.Errorf("过滤备份文件失败: %w", err)
	}

	// 生成备份预览
	preview, err := bm.GeneratePreview(device, allFiles, filesToBackup)
	if err != nil {
		return fmt.Errorf("生成预览失败: %w", err)
	}

	// 显示预览信息
	bm.DisplayPreview(preview, bm.verbose)
	bm.DisplayPreviewSummary(preview)

	return nil
}

// createFileChecker 创建文件检查器
func (bm *BackupManager) createFileChecker(device *device.DeviceInfo) *FileChecker {
	return NewFileChecker(bm.config, bm.log, bm.tracker)
}

// createFileCopier 创建文件复制器
func (bm *BackupManager) createFileCopier(device *device.DeviceInfo) *FileCopier {
	return NewFileCopier(bm.config, bm.log, bm.tracker, device)
}

// copyFilesWithProgress 带进度显示的文件复制
func (bm *BackupManager) copyFilesWithProgress(copier *FileCopier, files []*utils.FileInfo,
	tracker *progress.ProgressTracker, display *progress.ProgressDisplay, force bool) []*CopyResult {

	resultChan := copier.CopyFiles(files, force)
	var results []*CopyResult

	// 处理复制结果
	for result := range resultChan {
		results = append(results, result)

		if result.Success {
			tracker.CompleteFile()
			if !bm.quiet {
				bm.log.Debug("文件复制完成: %s", result.File.RelativePath)
			}
		} else if result.Skipped {
			if !bm.quiet {
				bm.log.Debug("文件跳过: %s, 原因: %s", result.File.RelativePath, result.SkipReason)
			}
		} else {
			bm.log.Error("文件复制失败: %s, %v", result.File.RelativePath, result.Error)
		}
	}

	return results
}

// processCopyResults 处理复制结果
func (bm *BackupManager) processCopyResults(results []*CopyResult, display *progress.ProgressDisplay) error {
	var successCount, skipCount, errorCount int
	var totalSize int64

	for _, result := range results {
		if result.Success {
			successCount++
			totalSize += result.BytesCopied
		} else if result.Skipped {
			skipCount++
		} else {
			errorCount++
			display.ShowError(result.Error)
		}
	}

	bm.log.Info("复制结果: 成功 %d, 跳过 %d, 失败 %d", successCount, skipCount, errorCount)
	bm.log.Info("总复制大小: %s", utils.FormatBytes(totalSize))

	if errorCount > 0 {
		return fmt.Errorf("有 %d 个文件复制失败", errorCount)
	}

	return nil
}

// showBackupStatistics 显示备份统计信息
func (bm *BackupManager) showBackupStatistics(startTime time.Time, totalFiles, backupFiles int, results []*CopyResult) {
	duration := time.Since(startTime)

	bm.log.Info("备份统计:")
	bm.log.Info("  扫描文件数: %d", totalFiles)
	bm.log.Info("  备份文件数: %d", backupFiles)
	bm.log.Info("  耗时: %s", utils.FormatDuration(duration))

	// 获取备份记录统计
	totalBackedUp, totalSize, lastBackup, err := bm.tracker.GetStatistics()
	if err == nil {
		bm.log.Info("  历史统计: 已备份 %d 个文件, 总大小 %s", totalBackedUp, utils.FormatBytes(totalSize))
		bm.log.Info("  上次备份: %s", lastBackup.Format("2006-01-02 15:04:05"))
	}

	// 计算速度
	if backupFiles > 0 && duration > 0 {
		avgSpeed := float64(backupFiles) / duration.Seconds()
		bm.log.Info("  平均速度: %.2f 文件/秒", avgSpeed)
	}
}

// GetDeviceInfo 获取设备信息
func (bm *BackupManager) GetDeviceInfo() (*device.DeviceInfo, error) {
	return device.DetectSR302()
}

// GetBackupHistory 获取备份历史
func (bm *BackupManager) GetBackupHistory() ([]storage.BackupRecord, error) {
	storage := bm.tracker.GetStorage()
	return storage.Records, nil
}

// CleanOldRecords 清理旧的备份记录
func (bm *BackupManager) CleanOldRecords(keepDays int) error {
	bm.log.Info("清理 %d 天前的备份记录...", keepDays)
	return bm.tracker.CleanOldRecords(keepDays)
}

// VerifyBackupIntegrity 验证备份完整性
func (bm *BackupManager) VerifyBackupIntegrity() error {
	bm.log.Info("开始验证备份完整性...")

	fileChecker := NewFileChecker(bm.config, bm.log, bm.tracker)
	return fileChecker.VerifyBackupIntegrity()
}

// ExportBackupReport 导出备份报告
func (bm *BackupManager) ExportBackupReport(exportPath string) error {
	bm.log.Info("导出备份报告到: %s", exportPath)

	// 确保目录存在
	dir := filepath.Dir(exportPath)
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("创建导出目录失败: %w", err)
	}

	return bm.tracker.ExportRecords(exportPath)
}

// GetConfiguration 获取当前配置
func (bm *BackupManager) GetConfiguration() *config.Config {
	return bm.config
}

// Close 关闭备份管理器
func (bm *BackupManager) Close() error {
	bm.log.Info("关闭备份管理器...")

	// 保存备份记录
	if err := bm.tracker.Save(); err != nil {
		bm.log.Warn("保存备份记录失败: %v", err)
		return err
	}

	bm.log.Info("备份管理器已关闭")
	return nil
}