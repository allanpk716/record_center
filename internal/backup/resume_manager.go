package backup

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// ResumeInfo 断点续传信息
type ResumeInfo struct {
	FilePath      string            `json:"file_path"`
	TempPath      string            `json:"temp_path"`
	CopiedBytes   int64             `json:"copied_bytes"`
	TotalBytes    int64             `json:"total_bytes"`
	LastUpdated   time.Time         `json:"last_updated"`
	Checksums     []string          `json:"checksums"`     // 分块校验和
	ChunkSize     int64             `json:"chunk_size"`   // 块大小
	Metadata      map[string]string `json:"metadata"`     // 额外的元数据
}

// ResumeManager 断点续传管理器
type ResumeManager struct {
	storagePath string
	tempDir     string
	log         *logger.Logger
	mu          sync.RWMutex
	cache       map[string]*ResumeInfo // 内存缓存
}

// NewResumeManager 创建断点续传管理器
func NewResumeManager(storagePath, tempDir string, log *logger.Logger) *ResumeManager {
	rm := &ResumeManager{
		storagePath: storagePath,
		tempDir:     tempDir,
		log:         log,
		cache:       make(map[string]*ResumeInfo),
	}

	// 确保目录存在
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		rm.log.Error("创建断点续传目录失败: %v", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		rm.log.Error("创建临时目录失败: %v", err)
	}

	return rm
}

// SaveResumeInfo 保存断点信息
func (rm *ResumeManager) SaveResumeInfo(info *ResumeInfo) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 更新时间戳
	info.LastUpdated = time.Now()

	// 保存到内存缓存
	rm.cache[info.FilePath] = info

	// 保存到文件
	return rm.saveToFile(info)
}

// GetResumeInfo 获取断点信息
func (rm *ResumeManager) GetResumeInfo(filePath string) (*ResumeInfo, error) {
	rm.mu.RLock()

	// 先从内存缓存查找
	if info, exists := rm.cache[filePath]; exists {
		rm.mu.RUnlock()
		return info, nil
	}
	rm.mu.RUnlock()

	// 从文件加载
	info, err := rm.loadFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// 更新内存缓存
	rm.mu.Lock()
	rm.cache[filePath] = info
	rm.mu.Unlock()

	return info, nil
}

// UpdateProgress 更新复制进度
func (rm *ResumeManager) UpdateProgress(filePath string, copiedBytes int64) error {
	info, err := rm.GetResumeInfo(filePath)
	if err != nil {
		// 如果不存在，创建新的
		info = &ResumeInfo{
			FilePath:    filePath,
			TempPath:    rm.getTempPath(filePath),
			CopiedBytes: copiedBytes,
			ChunkSize:   5 * 1024 * 1024, // 默认5MB块
			Metadata:    make(map[string]string),
		}
	}

	info.CopiedBytes = copiedBytes
	info.LastUpdated = time.Now()

	return rm.SaveResumeInfo(info)
}

// ClearResumeInfo 清除断点信息
func (rm *ResumeManager) ClearResumeInfo(filePath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 从内存缓存删除
	delete(rm.cache, filePath)

	// 删除临时文件
	tempPath := rm.getTempPath(filePath)
	if _, err := os.Stat(tempPath); err == nil {
		if removeErr := os.Remove(tempPath); removeErr != nil {
			rm.log.Warn("删除临时文件失败: %s, %v", tempPath, removeErr)
		}
	}

	// 删除断点信息文件
	resumeFilePath := rm.getResumeFilePath(filePath)
	if _, err := os.Stat(resumeFilePath); err == nil {
		if removeErr := os.Remove(resumeFilePath); removeErr != nil {
			rm.log.Warn("删除断点信息文件失败: %s, %v", resumeFilePath, removeErr)
			return removeErr
		}
	}

	return nil
}

// GetTempPath 获取临时文件路径
func (rm *ResumeManager) GetTempPath(filePath string) string {
	return rm.getTempPath(filePath)
}

// CleanupExpired 清理过期的断点信息
func (rm *ResumeManager) CleanupExpired(maxAge time.Duration) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(rm.storagePath, "*.resume"))
	if err != nil {
		return fmt.Errorf("扫描断点信息文件失败: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	cleanedCount := 0

	for _, file := range files {
		info, err := rm.loadResumeFile(file)
		if err != nil {
			rm.log.Warn("加载断点信息失败: %s, %v", file, err)
			continue
		}

		if info.LastUpdated.Before(cutoff) {
			// 删除过期的断点信息
			if err := os.Remove(file); err != nil {
				rm.log.Warn("删除过期断点信息失败: %s, %v", file, err)
			} else {
				// 同时删除临时文件
				if _, err := os.Stat(info.TempPath); err == nil {
					os.Remove(info.TempPath)
				}
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		rm.log.Info("清理了 %d 个过期的断点信息", cleanedCount)
	}

	return nil
}

// 私有方法

// getTempPath 获取临时文件路径
func (rm *ResumeManager) getTempPath(filePath string) string {
	// 使用文件路径的哈希作为临时文件名，避免路径过长
	hash := fmt.Sprintf("%x", time.Now().UnixNano())
	return filepath.Join(rm.tempDir, fmt.Sprintf("tmp_%s_%s", filepath.Base(filePath), hash))
}

// getResumeFilePath 获取断点信息文件路径
func (rm *ResumeManager) getResumeFilePath(filePath string) string {
	// 使用文件路径的简单哈希作为文件名，避免中文路径问题
	hash := rm.simpleHash(filePath)
	return filepath.Join(rm.storagePath, fmt.Sprintf("%s.resume", hash))
}

// simpleHash 使用标准库 FNV 哈希函数生成文件名
func (rm *ResumeManager) simpleHash(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%016x", h.Sum64())
}

// saveToFile 保存断点信息到文件
func (rm *ResumeManager) saveToFile(info *ResumeInfo) error {
	filePath := rm.getResumeFilePath(info.FilePath)

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化断点信息失败: %w", err)
	}

	// 原子性写入
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 重命名为目标文件
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// loadFromFile 从文件加载断点信息
func (rm *ResumeManager) loadFromFile(filePath string) (*ResumeInfo, error) {
	resumeFilePath := rm.getResumeFilePath(filePath)
	return rm.loadResumeFile(resumeFilePath)
}

// loadResumeFile 加载指定的断点信息文件
func (rm *ResumeManager) loadResumeFile(resumeFilePath string) (*ResumeInfo, error) {
	data, err := os.ReadFile(resumeFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取断点信息文件失败: %w", err)
	}

	var info ResumeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("解析断点信息失败: %w", err)
	}

	return &info, nil
}