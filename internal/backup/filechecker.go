package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/config"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/internal/storage"
	"github.com/allanpk716/record_center/pkg/utils"
)

// FileChecker 文件检查器
type FileChecker struct {
	config    *config.Config
	log       *logger.Logger
	tracker   *storage.BackupTracker
}

// NewFileChecker 创建新的文件检查器
func NewFileChecker(cfg *config.Config, log *logger.Logger, tracker *storage.BackupTracker) *FileChecker {
	return &FileChecker{
		config:  cfg,
		log:     log,
		tracker: tracker,
	}
}

// ScanDeviceFiles 扫描设备中的文件
func (fc *FileChecker) ScanDeviceFiles(device *device.DeviceInfo) ([]*utils.FileInfo, error) {
	// 由于SR302是MTP设备，这里需要特殊处理
	// 暂时模拟扫描过程，实际实现需要使用MTP协议或Windows WPD API

	fc.log.Info("开始扫描设备文件: %s", device.Name)

	// 这里应该实现实际的MTP设备文件扫描
	// 由于MTP设备访问比较复杂，我们先创建一个模拟版本
	files, err := fc.mockScanDevice(device)
	if err != nil {
		return nil, fmt.Errorf("扫描设备文件失败: %w", err)
	}

	fc.log.Info("扫描完成，发现 %d 个.opus文件", len(files))
	return files, nil
}

// mockScanDevice 模拟设备扫描（实际项目中需要替换为真实的MTP实现）
func (fc *FileChecker) mockScanDevice(device *device.DeviceInfo) ([]*utils.FileInfo, error) {
	// 这里应该通过MTP协议访问设备文件系统
	// 由于没有实际的MTP库集成，我们创建一个模拟实现

	// 模拟文件结构
	mockFiles := []struct {
		path string
		size int64
	}{
		{"内部共享存储空间\\录音笔文件\\2025\\11月\\11月03日_1\\rec001.opus", 1024000},
		{"内部共享存储空间\\录音笔文件\\2025\\11月\\11月03日_1\\rec002.opus", 2048000},
		{"内部共享存储空间\\录音笔文件\\2025\\11月\\11月17日_1\\rec001.opus", 1536000},
		{"内部共享存储空间\\录音笔文件\\2025\\11月\\11月24日_2\\rec001.opus", 3072000},
		{"内部共享存储空间\\录音笔文件\\2025\\11月\\11月24日自动保存_1\\auto001.opus", 512000},
	}

	var files []*utils.FileInfo
	basePath := "内部共享存储空间\\录音笔文件"

	for _, mockFile := range mockFiles {
		// 创建模拟的FileInfo
		fileInfo := &utils.FileInfo{
			Path:         mockFile.path,
			RelativePath: strings.Replace(mockFile.path, basePath+"\\", "", 1),
			Name:         filepath.Base(mockFile.path),
			Size:         mockFile.size,
			ModTime:      time.Now().Add(-time.Duration(len(mockFiles)) * time.Hour), // 模拟不同的修改时间
			IsOpus:       utils.IsOpusFile(mockFile.path),
		}

		files = append(files, fileInfo)
		fc.log.Debug("发现文件: %s (%.2f MB)", fileInfo.RelativePath, float64(fileInfo.Size)/1024/1024)
	}

	return files, nil
}

// FilterFilesToBackup 过滤需要备份的文件
func (fc *FileChecker) FilterFilesToBackup(allFiles []*utils.FileInfo, deviceID string, force bool) ([]*utils.FileInfo, error) {
	if force {
		fc.log.Info("强制模式：备份所有文件")
		return allFiles, nil
	}

	// 使用备份跟踪器获取新文件
	newFiles, err := fc.tracker.GetNewFiles(allFiles, deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取新文件失败: %w", err)
	}

	// 按扩展名过滤
	var filteredFiles []*utils.FileInfo
	for _, file := range newFiles {
		if fc.shouldBackupFile(file) {
			filteredFiles = append(filteredFiles, file)
		} else {
			fc.log.Debug("跳过非.opus文件: %s", file.RelativePath)
		}
	}

	fc.log.Info("过滤完成，需要备份 %d 个文件", len(filteredFiles))
	return filteredFiles, nil
}

// shouldBackupFile 检查文件是否应该备份
func (fc *FileChecker) shouldBackupFile(file *utils.FileInfo) bool {
	// 检查文件扩展名
	for _, ext := range fc.config.Backup.FileExtensions {
		if strings.ToLower(filepath.Ext(file.Name)) == strings.ToLower(ext) {
			return true
		}
	}

	return false
}

// ValidateFile 验证文件是否可以备份
func (fc *FileChecker) ValidateFile(file *utils.FileInfo) error {
	// 检查文件是否存在（对于MTP设备，这个检查可能不适用）
	// 在实际实现中，需要通过MTP API检查文件状态

	// 检查文件大小
	if file.Size <= 0 {
		return fmt.Errorf("文件大小无效: %s", file.RelativePath)
	}

	// 检查文件名
	if file.Name == "" {
		return fmt.Errorf("文件名为空")
	}

	// 检查文件扩展名
	if !fc.shouldBackupFile(file) {
		return fmt.Errorf("文件类型不支持: %s", file.Name)
	}

	return nil
}

// GetTargetPath 获取文件的目标路径
func (fc *FileChecker) GetTargetPath(file *utils.FileInfo) (string, error) {
	if !fc.config.Backup.PreserveStructure {
		// 不保留目录结构，所有文件放在目标目录根目录
		return filepath.Join(fc.config.Target.BaseDirectory, file.Name), nil
	}

	// 保留目录结构
	// 将相对路径中的反斜杠转换为正斜杠，并构建目标路径
	relativePath := strings.ReplaceAll(file.RelativePath, "\\", string(filepath.Separator))
	targetPath := filepath.Join(fc.config.Target.BaseDirectory, relativePath)

	return targetPath, nil
}

// EnsureTargetDirectory 确保目标目录存在
func (fc *FileChecker) EnsureTargetDirectory(targetPath string) error {
	dir := filepath.Dir(targetPath)
	if fc.config.Target.CreateSubdirs {
		return utils.EnsureDir(dir)
	}
	return utils.EnsureDir(fc.config.Target.BaseDirectory)
}

// CheckDiskSpace 检查磁盘空间
func (fc *FileChecker) CheckDiskSpace(files []*utils.FileInfo) error {
	// 计算总大小
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	// 检查目标目录的磁盘空间
	// 在Windows上，可以使用syscall.GetDiskFreeSpaceEx
	// 这里先进行基本检查

	targetDir := fc.config.Target.BaseDirectory
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// 如果目录不存在，尝试创建
		if err := utils.EnsureDir(targetDir); err != nil {
			return fmt.Errorf("无法创建目标目录: %w", err)
		}
	}

	fc.log.Info("需要备份的文件总大小: %s", utils.FormatBytes(totalSize))
	return nil
}

// GetBackupStatistics 获取备份统计信息
func (fc *FileChecker) GetBackupStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 从备份跟踪器获取统计信息
	totalFiles, totalSize, lastBackup, err := fc.tracker.GetStatistics()
	if err != nil {
		return nil, fmt.Errorf("获取备份统计信息失败: %w", err)
	}

	stats["total_files_backed_up"] = totalFiles
	stats["total_size"] = totalSize
	stats["last_backup"] = lastBackup
	stats["config"] = map[string]interface{}{
		"target_directory": fc.config.Target.BaseDirectory,
		"preserve_structure": fc.config.Backup.PreserveStructure,
		"file_extensions": fc.config.Backup.FileExtensions,
		"skip_existing": fc.config.Backup.SkipExisting,
	}

	return stats, nil
}

// VerifyBackupIntegrity 验证备份完整性
func (fc *FileChecker) VerifyBackupIntegrity() error {
	fc.log.Info("开始验证备份完整性...")

	// 获取所有备份记录
	storage := fc.tracker.GetStorage()
	errorCount := 0

	for _, record := range storage.Records {
		if !record.Success {
			continue // 跳过失败的记录
		}

		// 检查目标文件是否存在
		if !utils.FileExists(record.TargetPath) {
			fc.log.Warn("备份文件缺失: %s", record.TargetPath)
			errorCount++
			continue
		}

		// 验证文件大小
		fileInfo, err := os.Stat(record.TargetPath)
		if err != nil {
			fc.log.Warn("无法获取备份文件信息: %s, %v", record.TargetPath, err)
			errorCount++
			continue
		}

		if fileInfo.Size() != record.FileSize {
			fc.log.Warn("备份文件大小不匹配: %s (期望: %d, 实际: %d)",
				record.TargetPath, record.FileSize, fileInfo.Size())
			errorCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("发现 %d 个完整性问题", errorCount)
	}

	fc.log.Info("备份完整性验证通过，检查了 %d 个文件", len(storage.Records))
	return nil
}