package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/allanpk716/record_center/internal/config"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/internal/storage"
	"github.com/allanpk716/record_center/pkg/utils"
)

// CopyResult 复制结果
type CopyResult struct {
	File          *utils.FileInfo
	Success       bool
	Error         error
	BytesCopied   int64
	Duration      time.Duration
	TargetPath    string
	Skipped       bool
	SkipReason    string
}

// FileCopier 文件复制器
type FileCopier struct {
	config     *config.Config
	log        *logger.Logger
	tracker    *storage.BackupTracker
	device     *device.DeviceInfo
	semaphore  chan struct{} // 用于限制并发数
}

// NewFileCopier 创建新的文件复制器
func NewFileCopier(cfg *config.Config, log *logger.Logger, tracker *storage.BackupTracker, device *device.DeviceInfo) *FileCopier {
	maxConcurrent := cfg.Backup.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	return &FileCopier{
		config:    cfg,
		log:       log,
		tracker:   tracker,
		device:    device,
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// CopyFiles 复制多个文件
func (fc *FileCopier) CopyFiles(files []*utils.FileInfo, force bool) <-chan *CopyResult {
	resultChan := make(chan *CopyResult, len(files))

	go func() {
		var wg sync.WaitGroup
		wg.Add(len(files))

		for _, file := range files {
			go func(f *utils.FileInfo) {
				defer wg.Done()
				fc.semaphore <- struct{}{}
				defer func() { <-fc.semaphore }()

				result := fc.CopyFile(f, force)
				resultChan <- result
			}(file)
		}

		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

// CopyFile 复制单个文件
func (fc *FileCopier) CopyFile(file *utils.FileInfo, force bool) *CopyResult {
	startTime := time.Now()
	result := &CopyResult{
		File:        file,
		Success:     false,
		BytesCopied: 0,
		Duration:    0,
	}

	// 验证文件
	if err := fc.validateFile(file); err != nil {
		result.Error = fmt.Errorf("文件验证失败: %w", err)
		fc.log.Warn("文件验证失败: %s, %v", file.RelativePath, err)
		return result
	}

	// 检查是否需要跳过
	if !force {
		if skip, reason := fc.shouldSkipFile(file); skip {
			result.Skipped = true
			result.SkipReason = reason
			fc.log.Debug("跳过文件: %s, 原因: %s", file.RelativePath, reason)
			return result
		}
	}

	// 获取目标路径
	targetPath, err := fc.getTargetPath(file)
	if err != nil {
		result.Error = fmt.Errorf("获取目标路径失败: %w", err)
		return result
	}
	result.TargetPath = targetPath

	// 确保目标目录存在
	if err := fc.ensureTargetDirectory(targetPath); err != nil {
		result.Error = fmt.Errorf("创建目标目录失败: %w", err)
		return result
	}

	// 执行复制
	copiedBytes, err := fc.copyFileInternal(file, targetPath)
	result.BytesCopied = copiedBytes
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Errorf("文件复制失败: %w", err)
		fc.log.Error("复制文件失败: %s -> %s, %v", file.RelativePath, targetPath, err)
		return result
	}

	// 验证复制结果
	if err := fc.verifyCopy(file, targetPath, copiedBytes); err != nil {
		result.Error = fmt.Errorf("复制验证失败: %w", err)
		fc.log.Error("复制验证失败: %s, %v", file.RelativePath, err)
		return result
	}

	// 计算文件哈希（如果配置了）
	fileHash := ""
	if fc.config.Backup.SkipExisting {
		hash, err := utils.CalculateFileHash(targetPath)
		if err != nil {
			fc.log.Warn("计算文件哈希失败: %s, %v", targetPath, err)
		} else {
			fileHash = hash
		}
	}

	// 添加备份记录
	if err := fc.tracker.AddRecord(file.Path, targetPath, fc.device.DeviceID, file.Size, fileHash); err != nil {
		fc.log.Warn("添加备份记录失败: %s, %v", file.RelativePath, err)
	}

	result.Success = true
	result.BytesCopied = copiedBytes

	fc.log.Info("文件复制完成: %s -> %s (%s, 耗时: %s)",
		file.RelativePath, targetPath,
		utils.FormatBytes(copiedBytes),
		utils.FormatDuration(result.Duration))

	return result
}

// validateFile 验证文件
func (fc *FileCopier) validateFile(file *utils.FileInfo) error {
	if file == nil {
		return fmt.Errorf("文件信息为空")
	}

	if file.Path == "" {
		return fmt.Errorf("文件路径为空")
	}

	if file.Size <= 0 {
		return fmt.Errorf("文件大小无效: %d", file.Size)
	}

	if !fc.isSupportedFileType(file.Name) {
		return fmt.Errorf("不支持的文件类型: %s", file.Name)
	}

	return nil
}

// shouldSkipFile 检查是否应该跳过文件
func (fc *FileCopier) shouldSkipFile(file *utils.FileInfo) (bool, string) {
	// 检查是否已备份
	if fc.config.Backup.SkipExisting {
		backedUp, record, err := fc.tracker.IsFileBackedUp(file.Path)
		if err != nil {
			fc.log.Warn("检查备份状态失败: %s, %v", file.RelativePath, err)
			return false, ""
		}

		if backedUp && record != nil {
			return true, "文件已备份"
		}
	}

	return false, ""
}

// getTargetPath 获取目标路径
func (fc *FileCopier) getTargetPath(file *utils.FileInfo) (string, error) {
	if !fc.config.Backup.PreserveStructure {
		return filepath.Join(fc.config.Target.BaseDirectory, file.Name), nil
	}

	// 保留目录结构
	relativePath := strings.ReplaceAll(file.RelativePath, "\\", string(filepath.Separator))
	targetPath := filepath.Join(fc.config.Target.BaseDirectory, relativePath)
	return targetPath, nil
}

// ensureTargetDirectory 确保目标目录存在
func (fc *FileCopier) ensureTargetDirectory(targetPath string) error {
	if fc.config.Target.CreateSubdirs {
		dir := filepath.Dir(targetPath)
		return utils.EnsureDir(dir)
	}
	return utils.EnsureDir(fc.config.Target.BaseDirectory)
}

// copyFileInternal 内部复制方法
func (fc *FileCopier) copyFileInternal(file *utils.FileInfo, targetPath string) (int64, error) {
	// 对于MTP设备，这里需要特殊处理
	// 暂时使用模拟实现
	return fc.mockCopyFromDevice(file, targetPath)
}

// mockCopyFromDevice 模拟从设备复制文件（实际项目中需要替换为MTP实现）
func (fc *FileCopier) mockCopyFromDevice(file *utils.FileInfo, targetPath string) (int64, error) {
	// 创建一个临时源文件来模拟MTP设备的文件
	tempFile := filepath.Join(os.TempDir(), "rec_temp_"+file.Name)
	defer os.Remove(tempFile)

	// 创建模拟数据
	tempData := make([]byte, file.Size)
	for i := range tempData {
		tempData[i] = byte(i % 256)
	}

	if err := os.WriteFile(tempFile, tempData, 0644); err != nil {
		return 0, fmt.Errorf("创建临时文件失败: %w", err)
	}

	// 复制文件
	return fc.copyRegularFile(tempFile, targetPath)
}

// copyRegularFile 复制常规文件
func (fc *FileCopier) copyRegularFile(srcPath, dstPath string) (int64, error) {
	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return 0, fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return 0, fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制内容，同时更新进度
	var copied int64
	buffer := make([]byte, 64*1024) // 64KB缓冲区
	updateInterval := int64(1024 * 1024) // 每MB更新一次进度
	lastUpdate := int64(0)

	for {
		n, err := srcFile.Read(buffer)
		if n > 0 {
			written, writeErr := dstFile.Write(buffer[:n])
			copied += int64(written)

			if writeErr != nil {
				return copied, fmt.Errorf("写入目标文件失败: %w", writeErr)
			}

			// 定期更新进度（这里可以添加进度回调）
			if copied-lastUpdate >= updateInterval {
				lastUpdate = copied
				fc.log.Debug("复制进度: %s/%s (%.1f%%)",
					utils.FormatBytes(copied),
					utils.FormatBytes(fc.getFileSize(srcPath)),
					float64(copied)/float64(fc.getFileSize(srcPath))*100)
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return copied, fmt.Errorf("复制文件内容失败: %w", err)
		}
	}

	return copied, nil
}

// getFileSize 获取文件大小
func (fc *FileCopier) getFileSize(filePath string) int64 {
	if info, err := os.Stat(filePath); err == nil {
		return info.Size()
	}
	return 0
}

// verifyCopy 验证复制结果
func (fc *FileCopier) verifyCopy(file *utils.FileInfo, targetPath string, copiedBytes int64) error {
	// 检查目标文件是否存在
	if !utils.FileExists(targetPath) {
		return fmt.Errorf("目标文件不存在")
	}

	// 检查文件大小
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("获取目标文件信息失败: %w", err)
	}

	if targetInfo.Size() != file.Size {
		return fmt.Errorf("文件大小不匹配: 期望 %d, 实际 %d",
			file.Size, targetInfo.Size())
	}

	if copiedBytes != file.Size {
		return fmt.Errorf("复制字节数不匹配: 期望 %d, 实际 %d",
			file.Size, copiedBytes)
	}

	return nil
}

// isSupportedFileType 检查是否为支持的文件类型
func (fc *FileCopier) isSupportedFileType(filename string) bool {
	for _, ext := range fc.config.Backup.FileExtensions {
		if strings.ToLower(filepath.Ext(filename)) == strings.ToLower(ext) {
			return true
		}
	}
	return false
}

// GetCopyStatistics 获取复制统计信息
func (fc *FileCopier) GetCopyStatistics(results []*CopyResult) map[string]interface{} {
	stats := make(map[string]interface{})

	var totalFiles, successFiles, skippedFiles, errorFiles int
	var totalBytes, totalDuration int64
	var minDuration, maxDuration time.Duration

	minDuration = time.Hour // 设置一个很大的初始值

	for _, result := range results {
		totalFiles++
		totalBytes += result.BytesCopied
		totalDuration += result.Duration.Nanoseconds()

		if result.Success {
			successFiles++
		} else if result.Skipped {
			skippedFiles++
		} else {
			errorFiles++
		}

		if result.Duration < minDuration {
			minDuration = result.Duration
		}
		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}
	}

	stats["total_files"] = totalFiles
	stats["success_files"] = successFiles
	stats["skipped_files"] = skippedFiles
	stats["error_files"] = errorFiles
	stats["total_bytes"] = totalBytes
	stats["average_duration"] = time.Duration(totalDuration / int64(totalFiles))
	stats["min_duration"] = minDuration
	stats["max_duration"] = maxDuration

	if totalDuration > 0 {
		stats["average_speed"] = float64(totalBytes) / (float64(totalDuration) / 1e9) / 1024 / 1024 // MB/s
	}

	return stats
}