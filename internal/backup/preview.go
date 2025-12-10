package backup

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/storage"
	"github.com/allanpk716/record_center/pkg/utils"
)

// BackupPreview 备份预览信息
type BackupPreview struct {
	DeviceInfo      *device.DeviceInfo `json:"device_info"`
	TotalFiles      int                `json:"total_files"`
	TotalSize       int64              `json:"total_size"`
	AlreadyBackedUp int                `json:"already_backed_up"`
	AlreadySize     int64              `json:"already_size"`
	NeedBackup      int                `json:"need_backup"`
	NeedBackupSize  int64              `json:"need_backup_size"`
	NewFiles        []*utils.FileInfo  `json:"new_files"`
	LastBackupTime  time.Time          `json:"last_backup_time"`
	Storage         *storage.BackupStorage `json:"storage"`
}

// GeneratePreview 生成备份预览
func (bm *BackupManager) GeneratePreview(deviceInfo *device.DeviceInfo, allFiles []*utils.FileInfo, filesToBackup []*utils.FileInfo) (*BackupPreview, error) {
	// 获取备份记录
	bt := bm.tracker
	backupStorage := bt.GetStorage()
	if backupStorage == nil {
		bm.log.Warn("获取备份记录失败，使用空记录")
		backupStorage = &storage.BackupStorage{}
	}

	// 计算已备份的文件和大小
	alreadyCount := 0
	alreadySize := int64(0)
	backedUpMap := make(map[string]bool)

	// 创建文件路径到记录的映射
	for _, record := range backupStorage.Records {
		if record.Success && record.DeviceID == deviceInfo.DeviceID {
			backedUpMap[record.SourcePath] = true
		}
	}

	// 统计已备份的文件
	for _, file := range allFiles {
		if backedUpMap[file.Path] {
			alreadyCount++
			alreadySize += file.Size
		}
	}

	// 计算需要备份的文件大小
	needSize := int64(0)
	for _, file := range filesToBackup {
		needSize += file.Size
	}

	preview := &BackupPreview{
		DeviceInfo:      deviceInfo,
		TotalFiles:      len(allFiles),
		TotalSize:       utils.CalculateTotalSize(allFiles),
		AlreadyBackedUp: alreadyCount,
		AlreadySize:     alreadySize,
		NeedBackup:      len(filesToBackup),
		NeedBackupSize:  needSize,
		NewFiles:        filesToBackup,
		LastBackupTime:  backupStorage.LastBackup,
		Storage:         backupStorage,
	}

	return preview, nil
}

// DisplayPreview 显示备份预览信息
func (bm *BackupManager) DisplayPreview(preview *BackupPreview, verbose bool) {
	if bm.quiet {
		// 静默模式只显示简要信息
		bm.log.Info("总文件数: %d, 需要备份: %d", preview.TotalFiles, preview.NeedBackup)
		return
	}

	fmt.Println()
	fmt.Println(color.CyanString("=== 备份预览 ==="))
	fmt.Println(color.WhiteString(fmt.Sprintf("设备: %s (VID:%s PID:%s)",
		preview.DeviceInfo.Name, preview.DeviceInfo.VID, preview.DeviceInfo.PID)))

	// 统计信息
	fmt.Println()
	fmt.Println(color.YellowString("统计信息:"))
	fmt.Printf("  总文件数: %d 个 (%s)\n",
		preview.TotalFiles, utils.FormatBytes(preview.TotalSize))
	fmt.Printf("  已备份: %d 个 (%s)\n",
		preview.AlreadyBackedUp, utils.FormatBytes(preview.AlreadySize))
	fmt.Printf("  新增备份: %d 个 (%s)\n",
		preview.NeedBackup, utils.FormatBytes(preview.NeedBackupSize))

	// 备份历史
	if !preview.LastBackupTime.IsZero() {
		fmt.Printf("  最近备份: %s\n",
			preview.LastBackupTime.Format("2006-01-02 15:04:05"))
	}

	// 详细模式
	if verbose && len(preview.NewFiles) > 0 {
		fmt.Println()
		fmt.Println(color.MagentaString("本次需要备份的文件:"))
		fmt.Println(color.WhiteString(strings.Repeat("-", 80)))

		maxNameLen := 50
		for _, file := range preview.NewFiles {
			// 截断过长的文件名
			displayName := file.RelativePath
			if len(displayName) > maxNameLen {
				displayName = "..." + displayName[len(displayName)-maxNameLen+3:]
			}
			fmt.Printf("  %-*s %s\n", maxNameLen, displayName,
				utils.FormatBytes(file.Size))
		}
	}

	// 备份记录统计
	if verbose && preview.Storage != nil {
		fmt.Println()
		fmt.Println(color.CyanString("备份记录统计:"))
		fmt.Printf("  历史备份: %d 个文件\n", preview.Storage.TotalFilesBackedUp)
		fmt.Printf("  历史大小: %s\n", utils.FormatBytes(preview.Storage.TotalSize))
	}

	fmt.Println()
}

// DisplayPreviewSummary 显示预览摘要
func (bm *BackupManager) DisplayPreviewSummary(preview *BackupPreview) {
	if preview.NeedBackup == 0 {
		fmt.Println(color.GreenString("✅ 所有文件都已备份，无需操作"))
	} else {
		fmt.Printf(color.BlueString("准备备份 %d 个新文件 (%s)\n"),
			preview.NeedBackup, utils.FormatBytes(preview.NeedBackupSize))
	}
}