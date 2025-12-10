package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/pkg/utils"
)

// BackupRecord 备份记录
type BackupRecord struct {
	SourcePath      string    `json:"source_path"`
	TargetPath      string    `json:"target_path"`
	FileSize        int64     `json:"file_size"`
	FileHash        string    `json:"file_hash"`
	BackupTime      time.Time `json:"backup_time"`
	LastModified    time.Time `json:"last_modified"`
	DeviceID        string    `json:"device_id"`
	Success         bool      `json:"success"`
	// 新增完整性验证字段
	IntegrityCheck  bool      `json:"integrity_check"`
	Verified        bool      `json:"verified"`
	VerifyTime      time.Time `json:"verify_time"`
	HashAlgorithm   string    `json:"hash_algorithm"`
}

// BackupStorage 备份存储结构
type BackupStorage struct {
	Version            string        `json:"version"`
	LastBackup         time.Time     `json:"last_backup"`
	TotalFilesBackedUp int           `json:"total_files_backed_up"`
	TotalSize          int64         `json:"total_size"`
	Records            []BackupRecord `json:"records"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// BackupTracker 备份跟踪器
type BackupTracker struct {
	storagePath string
	storage     *BackupStorage
	log         *logger.Logger
	mu          struct {
		storage chan struct{}
	}
}

// NewBackupTracker 创建新的备份跟踪器
func NewBackupTracker(storagePath string, log *logger.Logger) *BackupTracker {
	return &BackupTracker{
		storagePath: storagePath,
		log:         log,
		storage:     &BackupStorage{
			Version:   "1.0",
			Records:   make([]BackupRecord, 0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// Load 加载备份记录
func (bt *BackupTracker) Load() error {
	bt.mu.storage = make(chan struct{}, 1)
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	// 如果文件不存在，创建默认存储
	if _, err := os.Stat(bt.storagePath); os.IsNotExist(err) {
		bt.log.Info("备份记录文件不存在，创建新的记录")
		return bt.save()
	}

	// 读取文件
	data, err := os.ReadFile(bt.storagePath)
	if err != nil {
		return fmt.Errorf("读取备份记录文件失败: %w", err)
	}

	// 解析JSON
	var storage BackupStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		bt.log.Warn("解析备份记录失败，创建新的记录: %v", err)
		bt.storage = &BackupStorage{
			Version:   "1.0",
			Records:   make([]BackupRecord, 0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return bt.save()
	}

	// 验证版本
	if storage.Version != "1.0" {
		bt.log.Warn("备份记录版本不匹配: %s，当前版本: 1.0", storage.Version)
	}

	bt.storage = &storage
	bt.log.Info("已加载 %d 个备份记录", len(storage.Records))
	return nil
}

// Save 保存备份记录
func (bt *BackupTracker) Save() error {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	return bt.save()
}

// save 内部保存方法（不加锁）
func (bt *BackupTracker) save() error {
	// 确保目录存在
	dir := filepath.Dir(bt.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建备份记录目录失败: %w", err)
	}

	// 更新时间戳
	bt.storage.UpdatedAt = time.Now()

	// 序列化
	data, err := json.MarshalIndent(bt.storage, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化备份记录失败: %w", err)
	}

	// 写入临时文件然后重命名（确保原子性）
	tempPath := bt.storagePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("写入临时备份记录文件失败: %w", err)
	}

	// 重命名
	if err := os.Rename(tempPath, bt.storagePath); err != nil {
		os.Remove(tempPath) // 清理临时文件
		return fmt.Errorf("保存备份记录文件失败: %w", err)
	}

	bt.log.Debug("备份记录已保存到: %s", bt.storagePath)
	return nil
}

// AddRecord 添加备份记录（保持向后兼容）
func (bt *BackupTracker) AddRecord(sourcePath, targetPath, deviceID string, fileSize int64, fileHash string) error {
	return bt.AddRecordWithVerify(sourcePath, targetPath, deviceID, fileSize, fileHash, false, "")
}

// AddRecordWithVerify 添加带完整性验证的备份记录
func (bt *BackupTracker) AddRecordWithVerify(sourcePath, targetPath, deviceID string, fileSize int64, fileHash string, integrityCheck bool, hashAlgorithm string) error {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	// 获取文件修改时间（对于MTP设备，可能失败）
	var lastModified time.Time
	if fileInfo, err := os.Stat(sourcePath); err == nil {
		lastModified = fileInfo.ModTime()
	} else {
		bt.log.Warn("无法获取源文件修改时间: %s", sourcePath)
		lastModified = time.Now()
	}

	record := BackupRecord{
		SourcePath:      sourcePath,
		TargetPath:      targetPath,
		FileSize:        fileSize,
		FileHash:        fileHash,
		BackupTime:      time.Now(),
		LastModified:    lastModified,
		DeviceID:        deviceID,
		Success:         true,
		IntegrityCheck:  integrityCheck,
		Verified:        integrityCheck && fileHash != "", // 如果有哈希值，认为已验证
		VerifyTime:      time.Now(),
		HashAlgorithm:   hashAlgorithm,
	}

	bt.storage.Records = append(bt.storage.Records, record)
	bt.storage.LastBackup = time.Now()
	bt.storage.TotalFilesBackedUp++
	bt.storage.TotalSize += fileSize

	bt.log.Debug("添加备份记录: %s", sourcePath)
	return nil
}

// isFileBackedUpInternal 内部方法，假设已经获取了锁
func (bt *BackupTracker) isFileBackedUpInternal(sourcePath string) (bool, *BackupRecord) {
	// 对于MTP设备路径，我们不能直接使用os.Stat
	// 只检查是否存在相同路径的备份记录
	// TODO: 实现MTP设备文件信息获取后，再进行文件大小和修改时间比较

	// 查找匹配的记录
	for i := range bt.storage.Records {
		record := &bt.storage.Records[i]
		if record.SourcePath == sourcePath && record.Success {
			return true, record
		}
	}

	return false, nil
}

// IsFileBackedUp 检查文件是否已备份
func (bt *BackupTracker) IsFileBackedUp(sourcePath string) (bool, *BackupRecord, error) {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	backedUp, record := bt.isFileBackedUpInternal(sourcePath)
	return backedUp, record, nil
}

// GetRecordByPath 根据路径获取备份记录
func (bt *BackupTracker) GetRecordByPath(sourcePath string) (*BackupRecord, error) {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	for i := range bt.storage.Records {
		if bt.storage.Records[i].SourcePath == sourcePath {
			return &bt.storage.Records[i], nil
		}
	}

	return nil, fmt.Errorf("未找到备份记录: %s", sourcePath)
}

// GetNewFiles 获取需要备份的新文件
func (bt *BackupTracker) GetNewFiles(files []*utils.FileInfo, deviceID string) ([]*utils.FileInfo, error) {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	var newFiles []*utils.FileInfo
	newCount := 0

	for _, file := range files {
		// 检查是否已备份（使用内部方法避免重复获取锁）
		backedUp, _ := bt.isFileBackedUpInternal(file.Path)

		if !backedUp {
			newFiles = append(newFiles, file)
			newCount++
		}
	}

	bt.log.Info("发现 %d 个新文件需要备份", newCount)
	return newFiles, nil
}

// GetStatistics 获取备份统计信息
func (bt *BackupTracker) GetStatistics() (int, int64, time.Time, error) {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	return bt.storage.TotalFilesBackedUp, bt.storage.TotalSize, bt.storage.LastBackup, nil
}

// RemoveRecord 移除备份记录
func (bt *BackupTracker) RemoveRecord(sourcePath string) error {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	for i, record := range bt.storage.Records {
		if record.SourcePath == sourcePath {
			// 更新统计
			bt.storage.TotalFilesBackedUp--
			bt.storage.TotalSize -= record.FileSize

			// 移除记录
			bt.storage.Records = append(bt.storage.Records[:i], bt.storage.Records[i+1:]...)
			bt.log.Debug("移除备份记录: %s", sourcePath)
			return nil
		}
	}

	return fmt.Errorf("未找到要移除的备份记录: %s", sourcePath)
}

// ClearRecords 清空所有备份记录
func (bt *BackupTracker) ClearRecords() error {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	bt.storage.Records = make([]BackupRecord, 0)
	bt.storage.TotalFilesBackedUp = 0
	bt.storage.TotalSize = 0
	bt.storage.LastBackup = time.Time{}
	bt.storage.UpdatedAt = time.Now()

	bt.log.Info("已清空所有备份记录")
	return nil
}

// GetRecordsByDevice 获取指定设备的备份记录
func (bt *BackupTracker) GetRecordsByDevice(deviceID string) []BackupRecord {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	var records []BackupRecord
	for _, record := range bt.storage.Records {
		if record.DeviceID == deviceID {
			records = append(records, record)
		}
	}

	return records
}

// CleanOldRecords 清理旧的备份记录
func (bt *BackupTracker) CleanOldRecords(keepDays int) error {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	cutoff := time.Now().AddDate(0, 0, -keepDays)
	cleaned := 0

	var newRecords []BackupRecord
	for _, record := range bt.storage.Records {
		if record.BackupTime.After(cutoff) {
			newRecords = append(newRecords, record)
		} else {
			cleaned++
		}
	}

	bt.storage.Records = newRecords
	bt.log.Info("清理了 %d 个超过 %d 天的旧备份记录", cleaned, keepDays)
	return nil
}

// ExportRecords 导出备份记录
func (bt *BackupTracker) ExportRecords(exportPath string) error {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	data, err := json.MarshalIndent(bt.storage, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化备份记录失败: %w", err)
	}

	return os.WriteFile(exportPath, data, 0644)
}

// GetStorage 获取存储对象（只读）
func (bt *BackupTracker) GetStorage() *BackupStorage {
	bt.mu.storage <- struct{}{}
	defer func() { <-bt.mu.storage }()

	// 返回副本避免并发问题
	storageCopy := *bt.storage
	recordsCopy := make([]BackupRecord, len(bt.storage.Records))
	copy(recordsCopy, bt.storage.Records)
	storageCopy.Records = recordsCopy

	return &storageCopy
}