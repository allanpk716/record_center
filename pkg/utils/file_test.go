package utils

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// TestIsOpusFile 测试检查文件是否为.opus格式
func TestIsOpusFile(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.opus", true},
		{"test.OPUS", true},
		{"test.Opus", true},
		{"test.mp3", false},
		{"test.wav", false},
		{"test.opus.mp3", false},
		{"", false},
		{"opus", false},
		{".opus", false},
		{"test.opus.bak", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := IsOpusFile(tc.filename)
			if result != tc.expected {
				t.Errorf("文件名 '%s': 期望 %v，实际 %v", tc.filename, tc.expected, result)
			}
		})
	}
}

// TestCalculateFileHash 测试计算文件哈希值
func TestCalculateFileHash(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.opus")

	// 创建测试文件
	testData := []byte("test audio data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 计算哈希值
	hash, err := CalculateFileHash(testFile)
	if err != nil {
		t.Fatalf("计算哈希失败: %v", err)
	}

	// 验证哈希值
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(testData))
	if hash != expectedHash {
		t.Errorf("哈希值不匹配，期望 %s，实际 %s", expectedHash[:16]+"...", hash[:16]+"...")
	}

	// 测试不存在的文件
	_, err = CalculateFileHash(filepath.Join(tempDir, "not_exist.opus"))
	if err == nil {
		t.Error("不存在的文件应该返回错误")
	}
}

// TestGetFileInfo 测试获取文件信息
func TestGetFileInfo(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}

	// 创建测试文件
	testData := []byte("test audio data")
	testFile := filepath.Join(subDir, "test.opus")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 获取文件信息
	fileInfo, err := GetFileInfo(testFile, tempDir)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	// 验证文件信息
	if fileInfo.Path != testFile {
		t.Errorf("路径不匹配，期望 %s，实际 %s", testFile, fileInfo.Path)
	}

	expectedRelativePath := filepath.Join("subdir", "test.opus")
	if fileInfo.RelativePath != expectedRelativePath {
		t.Errorf("相对路径不匹配，期望 %s，实际 %s", expectedRelativePath, fileInfo.RelativePath)
	}

	if fileInfo.Name != "test.opus" {
		t.Errorf("文件名不匹配，期望 'test.opus'，实际 '%s'", fileInfo.Name)
	}

	if fileInfo.Size != int64(len(testData)) {
		t.Errorf("文件大小不匹配，期望 %d，实际 %d", len(testData), fileInfo.Size)
	}

	if !fileInfo.IsOpus {
		t.Error("文件应该被识别为opus格式")
	}

	// 测试目录（应该返回错误）
	_, err = GetFileInfo(tempDir, tempDir)
	if err == nil {
		t.Error("目录应该返回错误")
	}
}

// TestScanDirectory 测试扫描目录
func TestScanDirectory(t *testing.T) {
	// 创建临时目录结构
	tempDir := t.TempDir()
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2")
	if err := os.MkdirAll(subDir1, 0755); err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}

	// 创建测试文件
	files := map[string]string{
		"test1.opus":  "audio data 1",
		"test2.opus":  "audio data 2",
		"test3.mp3":  "audio data 3",
		"subdir1/test4.opus": "audio data 4",
		"subdir2/test5.opus": "audio data 5",
		"subdir2/test6.txt": "text data 6",
	}

	for filePath, data := range files {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(data), 0644); err != nil {
			t.Fatalf("创建文件 %s 失败: %v", filePath, err)
		}
	}

	// 扫描目录
	log := logger.NewLogger(true)
	foundFiles, err := ScanDirectory(tempDir, log)
	if err != nil {
		t.Fatalf("扫描目录失败: %v", err)
	}

	// 验证找到的文件
	expectedCount := 4 // test1.opus, test2.opus, test4.opus, test5.opus
	if len(foundFiles) != expectedCount {
		t.Errorf("期望找到 %d 个opus文件，实际找到 %d 个", expectedCount, len(foundFiles))
	}

	// 验证所有找到的文件都是opus格式
	for _, file := range foundFiles {
		if !file.IsOpus {
			t.Errorf("文件 %s 不是opus格式", file.Name)
		}
	}

	// 测试不存在的目录
	_, err = ScanDirectory(filepath.Join(tempDir, "not_exist"), log)
	if err == nil {
		t.Error("不存在的目录应该返回错误")
	}
}

// TestEnsureDir 测试确保目录存在
func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()

	// 测试创建单层目录
	dir1 := filepath.Join(tempDir, "test1")
	if err := EnsureDir(dir1); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	if _, err := os.Stat(dir1); os.IsNotExist(err) {
		t.Error("目录未创建")
	}

	// 测试创建多层目录
	dir2 := filepath.Join(tempDir, "level1", "level2", "level3")
	if err := EnsureDir(dir2); err != nil {
		t.Fatalf("创建多层目录失败: %v", err)
	}

	if _, err := os.Stat(dir2); os.IsNotExist(err) {
		t.Error("多层目录未创建")
	}

	// 测试已存在的目录
	if err := EnsureDir(dir1); err != nil {
		t.Errorf("已存在的目录不应该返回错误: %v", err)
	}
}

// TestFileExists 测试检查文件是否存在
func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	// 测试存在的文件
	testFile := filepath.Join(tempDir, "test.opus")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	if !FileExists(testFile) {
		t.Error("存在的文件应该返回true")
	}

	// 测试不存在的文件
	if FileExists(filepath.Join(tempDir, "not_exist.opus")) {
		t.Error("不存在的文件应该返回false")
	}

	// 测试存在的目录
	if !FileExists(tempDir) {
		t.Error("存在的目录应该返回true")
	}
}

// TestIsNewerFile 测试比较文件修改时间
func TestIsNewerFile(t *testing.T) {
	tempDir := t.TempDir()

	// 创建第一个文件
	file1 := filepath.Join(tempDir, "file1.opus")
	if err := os.WriteFile(file1, []byte("test1"), 0644); err != nil {
		t.Fatalf("创建文件1失败: %v", err)
	}

	// 等待一下确保时间不同
	time.Sleep(10 * time.Millisecond)

	// 创建第二个文件
	file2 := filepath.Join(tempDir, "file2.opus")
	if err := os.WriteFile(file2, []byte("test2"), 0644); err != nil {
		t.Fatalf("创建文件2失败: %v", err)
	}

	// file2应该比file1新
	newer, err := IsNewerFile(file2, file1)
	if err != nil {
		t.Fatalf("比较文件时间失败: %v", err)
	}
	if !newer {
		t.Error("file2应该比file1新")
	}

	// file1不应该比file2新
	newer, err = IsNewerFile(file1, file2)
	if err != nil {
		t.Fatalf("比较文件时间失败: %v", err)
	}
	if newer {
		t.Error("file1不应该比file2新")
	}

	// 测试不存在的文件
	_, err = IsNewerFile(filepath.Join(tempDir, "not_exist1"), file1)
	if err == nil {
		t.Error("不存在的文件应该返回错误")
	}
}

// TestFormatBytes 测试格式化字节数
func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TiB"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d", tc.bytes), func(t *testing.T) {
			result := FormatBytes(tc.bytes)
			if result != tc.expected {
				t.Errorf("期望 '%s'，实际 '%s'", tc.expected, result)
			}
		})
	}
}

// TestFormatDuration 测试格式化时间间隔
func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "0s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{2*time.Minute + 30*time.Second, "2m 30s"},
		{1*time.Hour + 2*time.Minute + 3*time.Second, "1h 2m 3s"},
		{2*time.Hour + 5*time.Minute + 7*time.Second, "2h 5m 7s"},
	}

	for _, tc := range testCases {
		t.Run(tc.duration.String(), func(t *testing.T) {
			result := FormatDuration(tc.duration)
			if result != tc.expected {
				t.Errorf("期望 '%s'，实际 '%s'", tc.expected, result)
			}
		})
	}
}

// TestSafeFileName 测试清理文件名
func TestSafeFileName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"normal_file.opus", "normal_file.opus"},
		{"file<with>brackets.opus", "file_with_brackets.opus"},
		{"file:with:colons.opus", "file_with_colons.opus"},
		{"file\"with\"quotes.opus", "file_with_quotes.opus"},
		{"file|with|pipes.opus", "file_with_pipes.opus"},
		{"file?with?questions.opus", "file_with_questions.opus"},
		{"file*with*asterisks.opus", "file_with_asterisks.opus"},
		{"  spaced_file.opus  ", "spaced_file.opus"},
		{".hidden_file.opus.", "hidden_file.opus"},
		{"<>:\"|?*", "unnamed_file"},
		{"", "unnamed_file"},
		{"   .  ", "unnamed_file"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := SafeFileName(tc.input)
			if result != tc.expected {
				t.Errorf("期望 '%s'，实际 '%s'", tc.expected, result)
			}
		})
	}
}

// TestCopyFile 测试复制文件
func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	// 创建源目录
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建源文件
	sourceFile := filepath.Join(sourceDir, "test.opus")
	testData := []byte("test audio data")
	if err := os.WriteFile(sourceFile, testData, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	// 复制文件
	targetFile := filepath.Join(targetDir, "test.opus")
	log := logger.NewLogger(true)
	if err := CopyFile(sourceFile, targetFile, log); err != nil {
		t.Fatalf("复制文件失败: %v", err)
	}

	// 验证目标文件存在
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

	// 测试不存在的源文件
	err = CopyFile(filepath.Join(sourceDir, "not_exist"), targetFile, log)
	if err == nil {
		t.Error("不存在的源文件应该返回错误")
	}
}

// TestGetDirectorySize 测试获取目录大小
func TestGetDirectorySize(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")

	// 创建子目录
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}

	// 创建测试文件
	files := []struct {
		path string
		data []byte
	}{
		{"file1.opus", []byte("data1")},
		{"file2.opus", []byte("data2")},
		{"subdir/file3.opus", []byte("data3")},
	}

	totalSize := int64(0)
	for _, file := range files {
		fullPath := filepath.Join(tempDir, file.path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(fullPath, file.data, 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
		totalSize += int64(len(file.data))
	}

	// 获取目录大小
	log := logger.NewLogger(true)
	calculatedSize, err := GetDirectorySize(tempDir, log)
	if err != nil {
		t.Fatalf("获取目录大小失败: %v", err)
	}

	if calculatedSize != totalSize {
		t.Errorf("期望目录大小为 %d，实际为 %d", totalSize, calculatedSize)
	}

	// 测试空目录
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("创建空目录失败: %v", err)
	}

	size, err := GetDirectorySize(emptyDir, log)
	if err != nil {
		t.Fatalf("获取空目录大小失败: %v", err)
	}

	if size != 0 {
		t.Errorf("空目录大小应为0，实际为 %d", size)
	}
}

// TestCleanOldFiles 测试清理旧文件
func TestCleanOldFiles(t *testing.T) {
	tempDir := t.TempDir()

	// 创建测试文件（不同时间）
	now := time.Now()
	oldTime := now.AddDate(0, 0, -10) // 10天前

	files := []struct {
		name    string
		data    []byte
		modTime time.Time
		shouldKeep bool
	}{
		{"new.opus", []byte("new data"), now, true},
		{"old.opus", []byte("old data"), oldTime, false},
		{"recent.opus", []byte("recent data"), now.AddDate(0, 0, -2), true}, // 2天前
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file.name)
		if err := os.WriteFile(filePath, file.data, 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
		// 设置修改时间
		if err := os.Chtimes(filePath, file.modTime, file.modTime); err != nil {
			t.Fatalf("设置文件时间失败: %v", err)
		}
	}

	// 清理7天前的文件
	log := logger.NewLogger(true)
	if err := CleanOldFiles(tempDir, 7, log); err != nil {
		t.Fatalf("清理旧文件失败: %v", err)
	}

	// 验证文件状态
	for _, file := range files {
		filePath := filepath.Join(tempDir, file.name)
		exists := FileExists(filePath)
		if exists && !file.shouldKeep {
			t.Errorf("文件 %s 应该被删除但仍存在", file.name)
		}
		if !exists && file.shouldKeep {
			t.Errorf("文件 %s 应该保留但已被删除", file.name)
		}
	}
}

// TestIsEmptyDirectory 测试检查目录是否为空
func TestIsEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// 测试空目录
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("创建空目录失败: %v", err)
	}

	isEmpty, err := IsEmptyDirectory(emptyDir)
	if err != nil {
		t.Fatalf("检查空目录失败: %v", err)
	}
	if !isEmpty {
		t.Error("目录应该是空的")
	}

	// 测试非空目录
	notEmptyDir := filepath.Join(tempDir, "not_empty")
	if err := os.Mkdir(notEmptyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	testFile := filepath.Join(notEmptyDir, "test.opus")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	isEmpty, err = IsEmptyDirectory(notEmptyDir)
	if err != nil {
		t.Fatalf("检查非空目录失败: %v", err)
	}
	if isEmpty {
		t.Error("目录不应该为空")
	}

	// 测试不存在的目录
	_, err = IsEmptyDirectory(filepath.Join(tempDir, "not_exist"))
	if err == nil {
		t.Error("不存在的目录应该返回错误")
	}
}

// TestRemoveEmptyDirectories 测试删除空目录
func TestRemoveEmptyDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// 创建目录结构
	// tempDir/
	//   dir1/          (空)
	//   dir2/          (包含文件)
	//     file1.opus
	//   dir3/          (包含空子目录)
	//     subdir/      (空)
	//   dir4/          (嵌套空目录)
	//     level1/
	//       level2/    (空)

	dirs := []string{"dir1", "dir2", "dir3/subdir", "dir4/level1/level2"}
	for _, dir := range dirs {
		fullPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}

	// 创建文件
	testFile := filepath.Join(tempDir, "dir2", "file1.opus")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 删除空目录
	log := logger.NewLogger(true)
	removed, err := RemoveEmptyDirectories(tempDir, log, false)
	if err != nil {
		t.Fatalf("删除空目录失败: %v", err)
	}

	// 应该删除3个空目录（dir1, subdir, level2）
	if removed != 3 {
		t.Errorf("期望删除3个空目录，实际删除 %d 个", removed)
	}

	// 验证目录状态
	// dir1应该被删除
	if FileExists(filepath.Join(tempDir, "dir1")) {
		t.Error("dir1应该被删除")
	}

	// dir2应该保留（包含文件）
	if !FileExists(filepath.Join(tempDir, "dir2")) {
		t.Error("dir2应该保留")
	}

	// subdir应该被删除
	if FileExists(filepath.Join(tempDir, "dir3", "subdir")) {
		t.Error("subdir应该被删除")
	}

	// level2应该被删除，但level1应该保留
	if FileExists(filepath.Join(tempDir, "dir4", "level1", "level2")) {
		t.Error("level2应该被删除")
	}
	if !FileExists(filepath.Join(tempDir, "dir4", "level1")) {
		t.Error("level1应该保留")
	}

	// 测试干运行模式（dry run）
	tempDir2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir2, "empty"), 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}

	removed, err = RemoveEmptyDirectories(tempDir2, log, true)
	if err != nil {
		t.Fatalf("干运行失败: %v", err)
	}

	if removed != 1 {
		t.Errorf("干运行期望报告删除1个目录，实际报告 %d 个", removed)
	}

	// 验证目录仍然存在（干运行不实际删除）
	if !FileExists(filepath.Join(tempDir2, "empty")) {
		t.Error("干运行不应该实际删除目录")
	}
}

// BenchmarkCalculateFileHash 性能测试：计算文件哈希
func BenchmarkCalculateFileHash(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark.opus")

	// 创建测试文件
	testData := make([]byte, 1024*1024) // 1MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		b.Fatalf("创建测试文件失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateFileHash(testFile)
	}
}

// BenchmarkFormatBytes 性能测试：格式化字节数
func BenchmarkFormatBytes(b *testing.B) {
	size := int64(1024 * 1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatBytes(size)
	}
}

// ExampleIsOpusFile 示例：检查opus文件
func ExampleIsOpusFile() {
	filename := "test.opus"
	if IsOpusFile(filename) {
		println("这是一个opus文件")
	}
	// Output: 这是一个opus文件
}

// ExampleFormatBytes 示例：格式化字节数
func ExampleFormatBytes() {
	size := int64(1024 * 1024 * 1.5)
	formatted := FormatBytes(size)
	println(formatted)
	// Output: 1.5 MiB
}