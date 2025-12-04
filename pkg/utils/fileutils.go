package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// FileInfo 文件信息结构
type FileInfo struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsOpus       bool      `json:"is_opus"`
	Hash         string    `json:"hash,omitempty"`
}

// IsOpusFile 检查文件是否为.opus格式
func IsOpusFile(filename string) bool {
	return strings.ToLower(filepath.Ext(filename)) == ".opus"
}

// CalculateFileHash 计算文件的SHA256哈希值
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("计算哈希失败: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// GetFileInfo 获取文件详细信息
func GetFileInfo(filePath, basePath string) (*FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("路径是目录不是文件: %s", filePath)
	}

	relativePath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		relativePath = filePath
	}

	return &FileInfo{
		Path:         filePath,
		RelativePath: filepath.ToSlash(relativePath), // 统一使用正斜杠
		Name:         stat.Name(),
		Size:         stat.Size(),
		ModTime:      stat.ModTime(),
		IsOpus:       IsOpusFile(stat.Name()),
	}, nil
}

// ScanDirectory 递归扫描目录，查找所有.opus文件
func ScanDirectory(dirPath string, log *logger.Logger) ([]*FileInfo, error) {
	var files []*FileInfo

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn("访问路径失败: %s, 错误: %v", path, err)
			return nil // 继续扫描其他文件
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理.opus文件
		if IsOpusFile(info.Name()) {
			fileInfo, err := GetFileInfo(path, dirPath)
			if err != nil {
				log.Warn("获取文件信息失败: %s, 错误: %v", path, err)
				return nil
			}

			files = append(files, fileInfo)
			log.Debug("发现文件: %s (%.2f MB)", fileInfo.RelativePath, float64(fileInfo.Size)/1024/1024)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("扫描目录失败: %w", err)
	}

	log.Info("扫描完成，共找到 %d 个.opus文件", len(files))
	return files, nil
}

// EnsureDir 确保目录存在
func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// FileExists 检查文件是否存在
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// IsNewerFile 比较两个文件的修改时间，判断file1是否比file2新
func IsNewerFile(file1, file2 string) (bool, error) {
	info1, err := os.Stat(file1)
	if err != nil {
		return false, fmt.Errorf("获取文件信息失败: %s, %w", file1, err)
	}

	info2, err := os.Stat(file2)
	if err != nil {
		return false, fmt.Errorf("获取文件信息失败: %s, %w", file2, err)
	}

	return info1.ModTime().After(info2.ModTime()), nil
}

// FormatBytes 格式化字节数为人类可读的格式
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 格式化时间间隔为人类可读的格式
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), float64(int(d.Seconds())%60))
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
}

// SafeFileName 清理文件名，移除不安全的字符
func SafeFileName(name string) string {
	// 替换不安全的字符
	unsafe := []string{"<", ">", ":", "\"", "|", "?", "*"}
	safeName := name
	for _, char := range unsafe {
		safeName = strings.ReplaceAll(safeName, char, "_")
	}

	// 移除前后空格和点
	safeName = strings.Trim(safeName, " .")

	// 如果为空，使用默认名称
	if safeName == "" {
		safeName = "unnamed_file"
	}

	return safeName
}

// CopyFile 复制文件
func CopyFile(src, dst string, log *logger.Logger) error {
	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制文件内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 获取源文件大小以确保复制完整
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %w", err)
	}

	if written != srcInfo.Size() {
		return fmt.Errorf("文件复制不完整: 期望 %d 字节，实际复制 %d 字节", srcInfo.Size(), written)
	}

	log.Debug("文件复制完成: %s -> %s (%s)", src, dst, FormatBytes(written))
	return nil
}

// GetDirectorySize 获取目录中所有文件的总大小
func GetDirectorySize(dirPath string, log *logger.Logger) (int64, error) {
	var totalSize int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn("访问路径失败: %s, 错误: %v", path, err)
			return nil
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("计算目录大小失败: %w", err)
	}

	return totalSize, nil
}

// CleanOldFiles 清理指定天数之前的旧文件
func CleanOldFiles(dirPath string, days int, log *logger.Logger) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	cleaned := 0

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn("访问路径失败: %s, 错误: %v", path, err)
			return nil
		}

		if !info.IsDir() && info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				log.Warn("删除旧文件失败: %s, 错误: %v", path, err)
			} else {
				log.Debug("已删除旧文件: %s", path)
				cleaned++
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("清理旧文件失败: %w", err)
	}

	log.Info("清理完成，删除了 %d 个超过 %d 天的旧文件", cleaned, days)
	return nil
}