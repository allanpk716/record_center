package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/pkg/utils"
)

// TestBackupTracker_NewBackupTracker 测试创建备份跟踪器
func TestBackupTracker_NewBackupTracker(t *testing.T) {
	log := logger.NewLogger(true)
	tracker := NewBackupTracker("test_records.json", log)

	if tracker == nil {
		t.Fatal("创建备份跟踪器失败")
	}

	if tracker.storagePath != "test_records.json" {
		t.Errorf("期望存储路径为 'test_records.json'，实际为 '%s'", tracker.storagePath)
	}

	if tracker.storage == nil {
		t.Fatal("存储对象未初始化")
	}

	if tracker.storage.Version != "1.0" {
		t.Errorf("期望版本为 '1.0'，实际为 '%s'", tracker.storage.Version)
	}
}

// TestBackupTracker_LoadAndSave 测试加载和保存备份记录
func TestBackupTracker_LoadAndSave(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)

	// 测试保存新记录
	err := tracker.Save()
	if err != nil {
		t.Fatalf("保存备份记录失败: %v", err)
	}

	// 验证文件是否存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("备份记录文件未创建")
	}

	// 创建新的跟踪器并加载
	tracker2 := NewBackupTracker(testFile, log)
	err = tracker2.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	if tracker2.storage.Version != "1.0" {
		t.Errorf("期望版本为 '1.0'，实际为 '%s'", tracker2.storage.Version)
	}
}

// TestBackupTracker_AddRecord 测试添加备份记录
func TestBackupTracker_AddRecord(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 添加测试记录
	err = tracker.AddRecord(
		"/test/source/file1.opus",
		"/test/target/file1.opus",
		"device123",
		1024,
		"hash123",
	)
	if err != nil {
		t.Fatalf("添加备份记录失败: %v", err)
	}

	// 验证记录数量
	if len(tracker.storage.Records) != 1 {
		t.Errorf("期望记录数量为 1，实际为 %d", len(tracker.storage.Records))
	}

	// 验证记录内容
	record := tracker.storage.Records[0]
	if record.SourcePath != "/test/source/file1.opus" {
		t.Errorf("期望源路径为 '/test/source/file1.opus'，实际为 '%s'", record.SourcePath)
	}
	if record.TargetPath != "/test/target/file1.opus" {
		t.Errorf("期望目标路径为 '/test/target/file1.opus'，实际为 '%s'", record.TargetPath)
	}
	if record.FileSize != 1024 {
		t.Errorf("期望文件大小为 1024，实际为 %d", record.FileSize)
	}
	if record.DeviceID != "device123" {
		t.Errorf("期望设备ID为 'device123'，实际为 '%s'", record.DeviceID)
	}
}

// TestBackupTracker_AddRecordWithVerify 测试添加带完整性验证的备份记录
func TestBackupTracker_AddRecordWithVerify(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 添加带验证的测试记录
	err = tracker.AddRecordWithVerify(
		"/test/source/file2.opus",
		"/test/target/file2.opus",
		"device456",
		2048,
		"hash456",
		true,
		"sha256",
	)
	if err != nil {
		t.Fatalf("添加备份记录失败: %v", err)
	}

	// 验证记录
	record := tracker.storage.Records[0]
	if !record.IntegrityCheck {
		t.Error("期望完整性验证为 true")
	}
	if !record.Verified {
		t.Error("期望验证状态为 true")
	}
	if record.HashAlgorithm != "sha256" {
		t.Errorf("期望哈希算法为 'sha256'，实际为 '%s'", record.HashAlgorithm)
	}
}

// TestBackupTracker_IsFileBackedUp 测试检查文件是否已备份
func TestBackupTracker_IsFileBackedUp(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 测试未备份的文件
	backedUp, _, err := tracker.IsFileBackedUp("/test/not/backed.opus")
	if err != nil {
		t.Fatalf("检查备份状态失败: %v", err)
	}
	if backedUp {
		t.Error("未备份的文件被误判为已备份")
	}

	// 添加记录
	err = tracker.AddRecord(
		"/test/source/file1.opus",
		"/test/target/file1.opus",
		"device123",
		1024,
		"hash123",
	)
	if err != nil {
		t.Fatalf("添加备份记录失败: %v", err)
	}

	// 测试已备份的文件
	backedUp, record, err := tracker.IsFileBackedUp("/test/source/file1.opus")
	if err != nil {
		t.Fatalf("检查备份状态失败: %v", err)
	}
	if !backedUp {
		t.Error("已备份的文件被误判为未备份")
	}
	if record == nil {
		t.Fatal("未返回备份记录")
	}
}

// TestBackupTracker_GetNewFiles 测试获取需要备份的新文件
func TestBackupTracker_GetNewFiles(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 创建测试文件列表
	files := []*utils.FileInfo{
		{
			Path:         "/test/source/file1.opus",
			RelativePath: "file1.opus",
			Name:         "file1.opus",
			Size:         1024,
		},
		{
			Path:         "/test/source/file2.opus",
			RelativePath: "file2.opus",
			Name:         "file2.opus",
			Size:         2048,
		},
		{
			Path:         "/test/source/file3.opus",
			RelativePath: "file3.opus",
			Name:         "file3.opus",
			Size:         3072,
		},
	}

	// 添加一个已备份的记录
	err = tracker.AddRecord(
		"/test/source/file2.opus",
		"/test/target/file2.opus",
		"device123",
		2048,
		"hash456",
	)
	if err != nil {
		t.Fatalf("添加备份记录失败: %v", err)
	}

	// 获取新文件
	newFiles, err := tracker.GetNewFiles(files, "device123")
	if err != nil {
		t.Fatalf("获取新文件失败: %v", err)
	}

	// 验证新文件数量
	if len(newFiles) != 2 {
		t.Errorf("期望新文件数量为 2，实际为 %d", len(newFiles))
	}

	// 验证新文件内容
	newFilePaths := make(map[string]bool)
	for _, file := range newFiles {
		newFilePaths[file.Path] = true
	}

	if !newFilePaths["/test/source/file1.opus"] {
		t.Error("file1.opus 应该在新文件列表中")
	}
	if !newFilePaths["/test/source/file3.opus"] {
		t.Error("file3.opus 应该在新文件列表中")
	}
	if newFilePaths["/test/source/file2.opus"] {
		t.Error("file2.opus 不应该在新文件列表中（已备份）")
	}
}

// TestBackupTracker_RemoveRecord 测试移除备份记录
func TestBackupTracker_RemoveRecord(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 添加测试记录
	err = tracker.AddRecord(
		"/test/source/file1.opus",
		"/test/target/file1.opus",
		"device123",
		1024,
		"hash123",
	)
	if err != nil {
		t.Fatalf("添加备份记录失败: %v", err)
	}

	// 验证记录存在
	if len(tracker.storage.Records) != 1 {
		t.Fatalf("期望记录数量为 1，实际为 %d", len(tracker.storage.Records))
	}

	// 移除记录
	err = tracker.RemoveRecord("/test/source/file1.opus")
	if err != nil {
		t.Fatalf("移除备份记录失败: %v", err)
	}

	// 验证记录已移除
	if len(tracker.storage.Records) != 0 {
		t.Errorf("期望记录数量为 0，实际为 %d", len(tracker.storage.Records))
	}

	// 测试移除不存在的记录
	err = tracker.RemoveRecord("/test/not/exist.opus")
	if err == nil {
		t.Error("移除不存在的记录应该返回错误")
	}
}

// TestBackupTracker_ClearRecords 测试清空所有备份记录
func TestBackupTracker_ClearRecords(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 添加多个测试记录
	for i := 0; i < 5; i++ {
		err = tracker.AddRecord(
			"/test/source/file"+string(rune('1'+i))+".opus",
			"/test/target/file"+string(rune('1'+i))+".opus",
			"device123",
			int64(1024*(i+1)),
			"hash123",
		)
		if err != nil {
			t.Fatalf("添加备份记录失败: %v", err)
		}
	}

	// 验证记录数量
	if len(tracker.storage.Records) != 5 {
		t.Fatalf("期望记录数量为 5，实际为 %d", len(tracker.storage.Records))
	}

	// 清空记录
	err = tracker.ClearRecords()
	if err != nil {
		t.Fatalf("清空备份记录失败: %v", err)
	}

	// 验证记录已清空
	if len(tracker.storage.Records) != 0 {
		t.Errorf("期望记录数量为 0，实际为 %d", len(tracker.storage.Records))
	}

	if tracker.storage.TotalFilesBackedUp != 0 {
		t.Errorf("期望总备份文件数为 0，实际为 %d", tracker.storage.TotalFilesBackedUp)
	}

	if tracker.storage.TotalSize != 0 {
		t.Errorf("期望总大小为 0，实际为 %d", tracker.storage.TotalSize)
	}
}

// TestBackupTracker_GetRecordsByDevice 测试获取指定设备的备份记录
func TestBackupTracker_GetRecordsByDevice(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 添加不同设备的记录
	devices := []string{"device1", "device2", "device1"}
	for i, deviceID := range devices {
		err = tracker.AddRecord(
			"/test/source/file"+string(rune('1'+i))+".opus",
			"/test/target/file"+string(rune('1'+i))+".opus",
			deviceID,
			int64(1024*(i+1)),
			"hash123",
		)
		if err != nil {
			t.Fatalf("添加备份记录失败: %v", err)
		}
	}

	// 获取 device1 的记录
	records := tracker.GetRecordsByDevice("device1")
	if len(records) != 2 {
		t.Errorf("期望 device1 的记录数量为 2，实际为 %d", len(records))
	}

	// 验证记录都属于 device1
	for _, record := range records {
		if record.DeviceID != "device1" {
			t.Errorf("记录设备ID为 '%s'，期望为 'device1'", record.DeviceID)
		}
	}

	// 获取 device2 的记录
	records = tracker.GetRecordsByDevice("device2")
	if len(records) != 1 {
		t.Errorf("期望 device2 的记录数量为 1，实际为 %d", len(records))
	}

	// 获取不存在的设备的记录
	records = tracker.GetRecordsByDevice("device3")
	if len(records) != 0 {
		t.Errorf("期望 device3 的记录数量为 0，实际为 %d", len(records))
	}
}

// TestBackupTracker_CleanOldRecords 测试清理旧的备份记录
func TestBackupTracker_CleanOldRecords(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 手动添加不同时间的记录
	now := time.Now()
	oldTime := now.AddDate(0, 0, -10) // 10天前

	tracker.storage.Records = []BackupRecord{
		{
			SourcePath: "/test/source/new1.opus",
			TargetPath: "/test/target/new1.opus",
			BackupTime: now,
			FileSize:   1024,
		},
		{
			SourcePath: "/test/source/old1.opus",
			TargetPath: "/test/target/old1.opus",
			BackupTime: oldTime,
			FileSize:   2048,
		},
		{
			SourcePath: "/test/source/new2.opus",
			TargetPath: "/test/target/new2.opus",
			BackupTime: now,
			FileSize:   3072,
		},
		{
			SourcePath: "/test/source/old2.opus",
			TargetPath: "/test/target/old2.opus",
			BackupTime: oldTime,
			FileSize:   4096,
		},
	}

	// 清理7天前的记录
	err = tracker.CleanOldRecords(7)
	if err != nil {
		t.Fatalf("清理旧记录失败: %v", err)
	}

	// 验证只有新记录保留
	if len(tracker.storage.Records) != 2 {
		t.Errorf("期望清理后记录数量为 2，实际为 %d", len(tracker.storage.Records))
	}

	for _, record := range tracker.storage.Records {
		if record.BackupTime.Before(now.AddDate(0, 0, -7)) {
			t.Errorf("记录 '%s' 的备份时间早于7天前，应该被清理", record.SourcePath)
		}
	}
}

// TestBackupTracker_ExportRecords 测试导出备份记录
func TestBackupTracker_ExportRecords(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")
	exportFile := filepath.Join(tempDir, "export.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 添加测试记录
	err = tracker.AddRecord(
		"/test/source/file1.opus",
		"/test/target/file1.opus",
		"device123",
		1024,
		"hash123",
	)
	if err != nil {
		t.Fatalf("添加备份记录失败: %v", err)
	}

	// 导出记录
	err = tracker.ExportRecords(exportFile)
	if err != nil {
		t.Fatalf("导出备份记录失败: %v", err)
	}

	// 验证导出文件
	data, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}

	var exported BackupStorage
	err = json.Unmarshal(data, &exported)
	if err != nil {
		t.Fatalf("解析导出文件失败: %v", err)
	}

	if len(exported.Records) != 1 {
		t.Errorf("期望导出记录数量为 1，实际为 %d", len(exported.Records))
	}
}

// TestBackupTracker_ConcurrentAccess 测试并发访问安全性
func TestBackupTracker_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_backup.json")

	log := logger.NewLogger(true)
	tracker := NewBackupTracker(testFile, log)
	err := tracker.Load()
	if err != nil {
		t.Fatalf("加载备份记录失败: %v", err)
	}

	// 并发添加记录
	const numGoroutines = 10
	const recordsPerGoroutine = 5

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()
			for j := 0; j < recordsPerGoroutine; j++ {
				err := tracker.AddRecord(
					"/test/source/file"+string(rune('A'+goroutineID))+string(rune('0'+j))+".opus",
					"/test/target/file"+string(rune('A'+goroutineID))+string(rune('0'+j))+".opus",
					"device"+string(rune('A'+goroutineID)),
					int64(1024*(j+1)),
					"hash",
				)
				if err != nil {
					t.Errorf("并发添加记录失败 (goroutine %d, record %d): %v", goroutineID, j, err)
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证记录数量
	expectedRecords := numGoroutines * recordsPerGoroutine
	if len(tracker.storage.Records) != expectedRecords {
		t.Errorf("期望记录数量为 %d，实际为 %d", expectedRecords, len(tracker.storage.Records))
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()
			_, err := tracker.GetStatistics()
			if err != nil {
				t.Errorf("并发获取统计信息失败 (goroutine %d): %v", goroutineID, err)
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestBackupTracker_LoadInvalidJSON 测试加载无效JSON文件
func TestBackupTracker_LoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.json")

	log := logger.NewLogger(true)

	// 创建无效的JSON文件
	err := os.WriteFile(testFile, []byte("{ invalid json"), 0644)
	if err != nil {
		t.Fatalf("创建无效JSON文件失败: %v", err)
	}

	tracker := NewBackupTracker(testFile, log)
	err = tracker.Load()
	if err != nil {
		t.Fatalf("加载无效JSON应该创建新记录而不是返回错误: %v", err)
	}

	// 验证创建了新的存储
	if len(tracker.storage.Records) != 0 {
		t.Errorf("期望记录数量为 0，实际为 %d", len(tracker.storage.Records))
	}
}