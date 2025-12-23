package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/allanpk716/record_center/internal/config"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/internal/storage"
	"github.com/allanpk716/record_center/pkg/utils"
)

// MockTracker 模拟备份跟踪器
type MockTracker struct {
	records  map[string]*storage.BackupRecord
	backedUp map[string]bool
}

func NewMockTracker() *MockTracker {
	return &MockTracker{
		records:  make(map[string]*storage.BackupRecord),
		backedUp: make(map[string]bool),
	}
}

func (m *MockTracker) IsFileBackedUp(sourcePath string) (bool, *storage.BackupRecord, error) {
	backedUp, exists := m.backedUp[sourcePath]
	if !exists {
		return false, nil, nil
	}

	if backedUp {
		if record, ok := m.records[sourcePath]; ok {
			return true, record, nil
		}
	}

	return false, nil, nil
}

func (m *MockTracker) AddRecord(sourcePath, targetPath, deviceID string, fileSize int64, fileHash string) error {
	m.backedUp[sourcePath] = true
	m.records[sourcePath] = &storage.BackupRecord{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		FileSize:   fileSize,
		FileHash:   fileHash,
		DeviceID:   deviceID,
		Success:    true,
	}
	return nil
}

func (m *MockTracker) AddRecordWithVerify(sourcePath, targetPath, deviceID string, fileSize int64, fileHash string, integrityCheck bool, hashAlgorithm string) error {
	return m.AddRecord(sourcePath, targetPath, deviceID, fileSize, fileHash)
}

// TestFileCopier_NewFileCopier 测试创建文件复制器
func TestFileCopier_NewFileCopier(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建配置
	cfg := &config.Config{
		Backup: config.BackupConfig{
			MaxConcurrent:    3,
			EnableResume:     false,
			IntegrityCheck:   false,
			FileExtensions:   []string{".opus"},
		},
		Target: config.TargetConfig{
			BaseDirectory: filepath.Join(tempDir, "backups"),
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{
		DeviceID: "test_device",
		Name:     "Test Device",
		VID:      "2207",
		PID:      "0011",
	}

	// 创建文件复制器
	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	if copier == nil {
		t.Fatal("创建文件复制器失败")
	}

	if copier.config != cfg {
		t.Error("配置未正确设置")
	}

	if copier.log == nil {
		t.Error("日志未正确设置")
	}

	if copier.tracker == nil {
		t.Error("跟踪器未正确设置")
	}

	if copier.device != deviceInfo {
		t.Error("设备信息未正确设置")
	}

	if cap(copier.semaphore) != 3 {
		t.Errorf("信号量容量错误，期望 3，实际 %d", cap(copier.semaphore))
	}
}

// TestFileCopier_ValidateFile 测试文件验证
func TestFileCopier_ValidateFile(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions: []string{".opus"},
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{DeviceID: "test"}
	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	testCases := []struct {
		name        string
		file        *utils.FileInfo
		expectError bool
		errorMsg    string
	}{
		{
			name:        "空文件信息",
			file:        nil,
			expectError: true,
			errorMsg:    "文件信息为空",
		},
		{
			name: "空文件路径",
			file: &utils.FileInfo{
				Path: "",
				Name: "test.opus",
				Size: 1024,
			},
			expectError: true,
			errorMsg:    "文件路径为空",
		},
		{
			name: "无效文件大小",
			file: &utils.FileInfo{
				Path: "/test/file.opus",
				Name: "test.opus",
				Size: -1,
			},
			expectError: true,
			errorMsg:    "文件大小无效",
		},
		{
			name: "不支持的文件类型",
			file: &utils.FileInfo{
				Path: "/test/file.mp3",
				Name: "file.mp3",
				Size: 1024,
			},
			expectError: true,
			errorMsg:    "不支持的文件类型",
		},
		{
			name: "有效的opus文件",
			file: &utils.FileInfo{
				Path: "/test/file.opus",
				Name: "file.opus",
				Size: 1024,
			},
			expectError: false,
		},
		{
			name: "零字节文件（应该允许）",
			file: &utils.FileInfo{
				Path: "/test/empty.opus",
				Name: "empty.opus",
				Size: 0,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := copier.validateFile(tc.file)

			if tc.expectError {
				if err == nil {
					t.Errorf("期望返回错误: %s", tc.errorMsg)
				} else if tc.errorMsg != "" && !contains(err.Error(), tc.errorMsg) {
					t.Errorf("错误消息不匹配，期望包含 '%s'，实际为 '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望返回错误，但得到: %v", err)
				}
			}
		})
	}
}

// TestFileCopier_ShouldSkipFile 测试是否应该跳过文件
func TestFileCopier_ShouldSkipFile(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions: []string{".opus"},
			SkipExisting:   true,
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{DeviceID: "test"}
	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	// 添加一个已备份的文件
	tracker.AddRecord("/test/backed_up.opus", "/backup/backed_up.opus", "test", 1024, "hash123")

	testCases := []struct {
		name         string
		file         *utils.FileInfo
		expectSkip   bool
		skipReason   string
	}{
		{
			name: "已备份的文件",
			file: &utils.FileInfo{
				Path:         "/test/backed_up.opus",
				RelativePath: "backed_up.opus",
				Name:         "backed_up.opus",
				Size:         1024,
			},
			expectSkip: true,
			skipReason: "文件已备份",
		},
		{
			name: "未备份的文件",
			file: &utils.FileInfo{
				Path:         "/test/new_file.opus",
				RelativePath: "new_file.opus",
				Name:         "new_file.opus",
				Size:         2048,
			},
			expectSkip: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			skip, reason := copier.shouldSkipFile(tc.file)

			if skip != tc.expectSkip {
				t.Errorf("期望跳过状态为 %v，实际为 %v", tc.expectSkip, skip)
			}

			if tc.expectSkip && reason != tc.skipReason {
				t.Errorf("期望跳过原因为 '%s'，实际为 '%s'", tc.skipReason, reason)
			}
		})
	}
}

// TestFileCopier_GetTargetPath 测试获取目标路径
func TestFileCopier_GetTargetPath(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name              string
		preserveStructure bool
		baseDirectory     string
		file              *utils.FileInfo
		expectedPath      string
	}{
		{
			name:              "不保留结构",
			preserveStructure: false,
			baseDirectory:     filepath.Join(tempDir, "backups"),
			file: &utils.FileInfo{
				Path:         "/source/subdir/file.opus",
				RelativePath: "subdir/file.opus",
				Name:         "file.opus",
			},
			expectedPath: filepath.Join(tempDir, "backups", "file.opus"),
		},
		{
			name:              "保留结构",
			preserveStructure: true,
			baseDirectory:     filepath.Join(tempDir, "backups"),
			file: &utils.FileInfo{
				Path:         "/source/subdir/file.opus",
				RelativePath: "subdir/file.opus",
				Name:         "file.opus",
			},
			expectedPath: filepath.Join(tempDir, "backups", "subdir", "file.opus"),
		},
		{
			name:              "保留结构 - Windows路径",
			preserveStructure: true,
			baseDirectory:     filepath.Join(tempDir, "backups"),
			file: &utils.FileInfo{
				Path:         "/source/subdir\\file.opus",
				RelativePath: "subdir\\file.opus",
				Name:         "file.opus",
			},
			expectedPath: filepath.Join(tempDir, "backups", "subdir", "file.opus"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Backup: config.BackupConfig{
					PreserveStructure: tc.preserveStructure,
				},
				Target: config.TargetConfig{
					BaseDirectory: tc.baseDirectory,
				},
			}

			log := logger.NewLogger(true)
			tracker := NewMockTracker()
			deviceInfo := &device.DeviceInfo{DeviceID: "test"}
			copier := NewFileCopier(cfg, log, tracker, deviceInfo)

			targetPath, err := copier.getTargetPath(tc.file)
			if err != nil {
				t.Fatalf("获取目标路径失败: %v", err)
			}

			if targetPath != tc.expectedPath {
				t.Errorf("期望目标路径为 '%s'，实际为 '%s'", tc.expectedPath, targetPath)
			}
		})
	}
}

// TestFileCopier_CopyFile 测试复制文件
func TestFileCopier_CopyFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backups")

	// 创建源目录和测试文件
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	testData := []byte("test audio data")
	sourceFile := filepath.Join(sourceDir, "test.opus")
	if err := os.WriteFile(sourceFile, testData, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions:    []string{".opus"},
			SkipExisting:      false,
			PreserveStructure: false,
			EnableResume:      false,
			IntegrityCheck:    false,
		},
		Target: config.TargetConfig{
			BaseDirectory: backupDir,
			CreateSubdirs: true,
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{
		DeviceID: "test_device",
		Name:     "Test Device",
		VID:      "2207",
		PID:      "0011",
	}

	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	// 创建FileInfo
	fileInfo := &utils.FileInfo{
		Path:         sourceFile,
		RelativePath: "test.opus",
		Name:         "test.opus",
		Size:         int64(len(testData)),
	}

	// 执行复制
	result := copier.CopyFile(fileInfo, false)

	// 验证结果
	if !result.Success {
		t.Errorf("文件复制失败: %v", result.Error)
	}

	if result.BytesCopied != int64(len(testData)) {
		t.Errorf("期望复制 %d 字节，实际复制 %d 字节", len(testData), result.BytesCopied)
	}

	// 验证目标文件存在
	targetFile := filepath.Join(backupDir, "test.opus")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Error("目标文件不存在")
	}

	// 验证文件内容
	copiedData, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}

	if string(copiedData) != string(testData) {
		t.Error("文件内容不匹配")
	}

	// 验证备份记录已添加
	if len(tracker.records) != 1 {
		t.Errorf("期望有 1 个备份记录，实际有 %d 个", len(tracker.records))
	}
}

// TestFileCopier_CopyFile_WithForce 测试强制复制
func TestFileCopier_CopyFile_WithForce(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backups")

	// 创建源目录和测试文件
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	testData := []byte("test audio data")
	sourceFile := filepath.Join(sourceDir, "test.opus")
	if err := os.WriteFile(sourceFile, testData, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions:    []string{".opus"},
			SkipExisting:      true, // 启用跳过已存在
			PreserveStructure: false,
			EnableResume:      false,
			IntegrityCheck:    false,
		},
		Target: config.TargetConfig{
			BaseDirectory: backupDir,
			CreateSubdirs: true,
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{
		DeviceID: "test_device",
		Name:     "Test Device",
		VID:      "2207",
		PID:      "0011",
	}

	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	// 创建FileInfo
	fileInfo := &utils.FileInfo{
		Path:         sourceFile,
		RelativePath: "test.opus",
		Name:         "test.opus",
		Size:         int64(len(testData)),
	}

	// 添加已备份记录（模拟文件已备份）
	tracker.AddRecord(sourceFile, filepath.Join(backupDir, "test.opus"), "test_device", 1024, "oldhash")

	// 不强制复制（应该跳过）
	result1 := copier.CopyFile(fileInfo, false)
	if !result1.Skipped {
		t.Error("不强制复制时应该跳过已备份的文件")
	}

	// 强制复制（应该复制）
	result2 := copier.CopyFile(fileInfo, true)
	if result2.Skipped {
		t.Error("强制复制时不应该跳过文件")
	}
	if !result2.Success {
		t.Errorf("强制复制失败: %v", result2.Error)
	}
}

// TestFileCopier_CopyFiles 测试并发复制多个文件
func TestFileCopier_CopyFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backups")

	// 创建源目录
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建多个测试文件
	numFiles := 5
	files := make([]*utils.FileInfo, 0, numFiles)

	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("test%d.opus", i)
		filePath := filepath.Join(sourceDir, fileName)
		testData := []byte(fmt.Sprintf("test audio data %d", i))

		if err := os.WriteFile(filePath, testData, 0644); err != nil {
			t.Fatalf("创建测试文件 %s 失败: %v", fileName, err)
		}

		files = append(files, &utils.FileInfo{
			Path:         filePath,
			RelativePath: fileName,
			Name:         fileName,
			Size:         int64(len(testData)),
		})
	}

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions:    []string{".opus"},
			MaxConcurrent:     2,
			PreserveStructure: false,
			EnableResume:      false,
			IntegrityCheck:    false,
		},
		Target: config.TargetConfig{
			BaseDirectory: backupDir,
			CreateSubdirs: true,
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{
		DeviceID: "test_device",
		Name:     "Test Device",
		VID:      "2207",
		PID:      "0011",
	}

	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	// 执行并发复制
	resultChan := copier.CopyFiles(context.Background(), files, false)

	// 收集结果
	results := make([]*CopyResult, 0, numFiles)
	for result := range resultChan {
		results = append(results, result)
	}

	// 验证结果
	if len(results) != numFiles {
		t.Errorf("期望有 %d 个结果，实际有 %d 个", numFiles, len(results))
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++

			// 验证目标文件存在
			targetPath := filepath.Join(backupDir, result.File.Name)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				t.Errorf("目标文件不存在: %s", targetPath)
			}
		}
	}

	if successCount != numFiles {
		t.Errorf("期望所有文件都复制成功，实际成功 %d 个", successCount)
	}

	// 验证备份记录
	if len(tracker.records) != numFiles {
		t.Errorf("期望有 %d 个备份记录，实际有 %d 个", numFiles, len(tracker.records))
	}
}

// TestFileCopier_VerifyCopy 测试复制验证
func TestFileCopier_VerifyCopy(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions: []string{".opus"},
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{DeviceID: "test"}
	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	// 创建源文件
	sourceData := []byte("test data")
	sourceFile := filepath.Join(tempDir, "source.opus")
	if err := os.WriteFile(sourceFile, sourceData, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	// 创建目标文件（较小）
	targetFile := filepath.Join(tempDir, "target.opus")
	if err := os.WriteFile(targetFile, sourceData[:len(sourceData)/2], 0644); err != nil {
		t.Fatalf("创建目标文件失败: %v", err)
	}

	fileInfo := &utils.FileInfo{
		Path: sourceFile,
		Name: "source.opus",
		Size: int64(len(sourceData)),
	}

	// 验证复制（应该失败，因为大小不匹配）
	err := copier.verifyCopy(fileInfo, targetFile, int64(len(sourceData)/2))
	if err == nil {
		t.Error("验证应该失败，因为文件大小不匹配")
	}

	// 修正目标文件大小
	if err := os.WriteFile(targetFile, sourceData, 0644); err != nil {
		t.Fatalf("修正目标文件失败: %v", err)
	}

	// 再次验证（应该成功）
	err = copier.verifyCopy(fileInfo, targetFile, int64(len(sourceData)))
	if err != nil {
		t.Errorf("验证应该成功，但失败: %v", err)
	}
}

// TestFileCopier_GetCopyStatistics 测试获取复制统计信息
func TestFileCopier_GetCopyStatistics(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Backup: config.BackupConfig{
			FileExtensions: []string{".opus"},
		},
	}

	log := logger.NewLogger(true)
	tracker := NewMockTracker()
	deviceInfo := &device.DeviceInfo{DeviceID: "test"}
	copier := NewFileCopier(cfg, log, tracker, deviceInfo)

	// 创建测试结果
	results := []*CopyResult{
		{
			File:        &utils.FileInfo{Name: "file1.opus", Size: 1024},
			Success:     true,
			BytesCopied: 1024,
			Duration:    100 * time.Millisecond,
		},
		{
			File:        &utils.FileInfo{Name: "file2.opus", Size: 2048},
			Success:     true,
			BytesCopied: 2048,
			Duration:    200 * time.Millisecond,
		},
		{
			File:        &utils.FileInfo{Name: "file3.opus", Size: 512},
			Skipped:     true,
			SkipReason:  "已备份",
		},
		{
			File:  &utils.FileInfo{Name: "file4.opus", Size: 4096},
			Success: false,
			Error:   fmt.Errorf("复制失败"),
		},
	}

	// 获取统计信息
	stats := copier.GetCopyStatistics(results)

	// 验证统计信息
	if stats["total_files"] != 4 {
		t.Errorf("总文件数错误，期望 4，实际 %v", stats["total_files"])
	}

	if stats["success_files"] != 2 {
		t.Errorf("成功文件数错误，期望 2，实际 %v", stats["success_files"])
	}

	if stats["skipped_files"] != 1 {
		t.Errorf("跳过文件数错误，期望 1，实际 %v", stats["skipped_files"])
	}

	if stats["error_files"] != 1 {
		t.Errorf("错误文件数错误，期望 1，实际 %v", stats["error_files"])
	}

	if stats["total_bytes"] != int64(3072) { // 1024 + 2048
		t.Errorf("总字节数错误，期望 3072，实际 %v", stats["total_bytes"])
	}

	// 验证平均速度
	if avgSpeed, ok := stats["average_speed"].(float64); ok && avgSpeed <= 0 {
		t.Error("平均速度应该大于0")
	}
}