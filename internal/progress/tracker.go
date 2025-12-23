package progress

import (
	"sync"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
	"github.com/allanpk716/record_center/pkg/utils"
)

const (
	// MaxSpeedSamples 最大速度采样点数量
	MaxSpeedSamples = 60
	// SpeedSampleAge 速度采样保留时间
	SpeedSampleAge = 10 * time.Second
)

// SpeedSample 速度采样数据
type SpeedSample struct {
	Timestamp   time.Time `json:"timestamp"`
	BytesCopied int64     `json:"bytes_copied"`
}

// SpeedCalculator 速度计算器
type SpeedCalculator struct {
	samples    []SpeedSample `json:"samples"`
	maxSamples int          `json:"max_samples"`
	maxAge     time.Duration `json:"max_age"`
	mu         sync.Mutex   `json:"-"`
}

// NewSpeedCalculator 创建新的速度计算器
func NewSpeedCalculator() *SpeedCalculator {
	return &SpeedCalculator{
		samples:    make([]SpeedSample, 0),
		maxSamples: MaxSpeedSamples,
		maxAge:     SpeedSampleAge,
	}
}

// AddSample 添加速度采样
func (sc *SpeedCalculator) AddSample(bytesCopied int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 兜底检查：如果切片过大，强制清理
	if len(sc.samples) >= sc.maxSamples*2 {
		sc.samples = sc.samples[len(sc.samples)-sc.maxSamples:]
	}

	now := time.Now()
	sample := SpeedSample{
		Timestamp:   now,
		BytesCopied: bytesCopied,
	}

	// 添加新采样
	sc.samples = append(sc.samples, sample)

	// 清理过期数据
	cutoff := now.Add(-sc.maxAge)
	validSamples := make([]SpeedSample, 0)
	for _, s := range sc.samples {
		if s.Timestamp.After(cutoff) {
			validSamples = append(validSamples, s)
		}
	}
	sc.samples = validSamples

	// 限制采样数量
	if len(sc.samples) > sc.maxSamples {
		sc.samples = sc.samples[len(sc.samples)-sc.maxSamples:]
	}
}

// GetCurrentSpeed 获取当前速度（MB/s）
func (sc *SpeedCalculator) GetCurrentSpeed() float64 {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if len(sc.samples) < 2 {
		return 0
	}

	first := sc.samples[0]
	last := sc.samples[len(sc.samples)-1]

	duration := last.Timestamp.Sub(first.Timestamp).Seconds()
	bytesDiff := last.BytesCopied - first.BytesCopied

	if duration <= 0 {
		return 0
	}

	return float64(bytesDiff) / duration / 1024 / 1024 // MB/s
}

// GetAverageSpeed 获取平均速度（MB/s）
func (sc *SpeedCalculator) GetAverageSpeed() float64 {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if len(sc.samples) < 2 {
		return 0
	}

	first := sc.samples[0]
	last := sc.samples[len(sc.samples)-1]

	totalDuration := last.Timestamp.Sub(first.Timestamp).Seconds()
	totalBytes := last.BytesCopied - first.BytesCopied

	if totalDuration <= 0 {
		return 0
	}

	return float64(totalBytes) / totalDuration / 1024 / 1024 // MB/s
}

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	totalFiles      int                `json:"total_files"`
	completedFiles  int                `json:"completed_files"`
	totalSize       int64              `json:"total_size"`
	copiedSize      int64              `json:"copied_size"`
	startTime       time.Time          `json:"start_time"`
	currentFile     *utils.FileInfo    `json:"current_file,omitempty"`
	speedCalculator *SpeedCalculator   `json:"speed_calculator"`
	elapsedTime     time.Duration      `json:"elapsed_time"`
	lastUpdateTime  time.Time          `json:"last_update_time"`
	mu              sync.RWMutex       `json:"-"`
	log             *logger.Logger     `json:"-"`
}

// NewProgressTracker 创建新的进度跟踪器
func NewProgressTracker(log *logger.Logger) *ProgressTracker {
	return &ProgressTracker{
		speedCalculator: NewSpeedCalculator(),
		log:             log,
	}
}

// Start 开始进度跟踪
func (pt *ProgressTracker) Start(files []*utils.FileInfo) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.totalFiles = len(files)
	pt.completedFiles = 0
	pt.totalSize = 0
	pt.copiedSize = 0
	pt.startTime = time.Now()
	pt.lastUpdateTime = time.Now()

	// 计算总大小
	for _, file := range files {
		pt.totalSize += file.Size
	}

	pt.log.Info("开始备份 %d 个文件，总大小: %s", pt.totalFiles, utils.FormatBytes(pt.totalSize))
	return nil
}

// StartWithParams 使用参数开始进度跟踪
func (pt *ProgressTracker) StartWithParams(totalFiles int, totalSize int64) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.totalFiles = totalFiles
	pt.completedFiles = 0
	pt.totalSize = totalSize
	pt.copiedSize = 0
	pt.startTime = time.Now()
	pt.lastUpdateTime = time.Now()

	pt.log.Info("开始备份 %d 个文件，总大小: %s", pt.totalFiles, utils.FormatBytes(pt.totalSize))
	return nil
}

// UpdateCurrentFile 更新当前处理的文件
func (pt *ProgressTracker) UpdateCurrentFile(file *utils.FileInfo) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.currentFile = file
	pt.lastUpdateTime = time.Now()
}

// UpdateProgress 更新文件复制进度
func (pt *ProgressTracker) UpdateProgress(bytesCopied int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.copiedSize = bytesCopied
	pt.speedCalculator.AddSample(bytesCopied)
	pt.lastUpdateTime = time.Now()
}

// CompleteFile 标记文件完成
func (pt *ProgressTracker) CompleteFile() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completedFiles++
	pt.lastUpdateTime = time.Now()
}

// GetProgressInfo 获取当前进度信息
func (pt *ProgressTracker) GetProgressInfo() *ProgressInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// 使用局部变量而不是修改字段，避免在读锁内进行写操作
	elapsedTime := time.Since(pt.startTime)

	progressPercent := float64(0)
	if pt.totalFiles > 0 {
		progressPercent = float64(pt.completedFiles) / float64(pt.totalFiles) * 100
	}

	var estimatedTime time.Duration
	currentSpeed := pt.speedCalculator.GetCurrentSpeed()
	if currentSpeed > 0 && pt.totalSize > pt.copiedSize {
		remainingBytes := pt.totalSize - pt.copiedSize
		estimatedSeconds := remainingBytes / (int64(currentSpeed*1024*1024))
		estimatedTime = time.Duration(estimatedSeconds) * time.Second
	}

	currentFileName := ""
	if pt.currentFile != nil {
		currentFileName = pt.currentFile.RelativePath
	}

	return &ProgressInfo{
		TotalFiles:     pt.totalFiles,
		CompletedFiles: pt.completedFiles,
		CurrentFile:    currentFileName,
		TotalSize:      pt.totalSize,
		CopiedSize:     pt.copiedSize,
		Speed:          currentSpeed,
		ElapsedTime:    elapsedTime,
		EstimatedTime:  estimatedTime,
		ProgressPercent: progressPercent,
	}
}

// ProgressInfo 进度信息结构
type ProgressInfo struct {
	TotalFiles      int           `json:"total_files"`
	CompletedFiles  int           `json:"completed_files"`
	CurrentFile     string        `json:"current_file"`
	TotalSize       int64         `json:"total_size"`
	CopiedSize      int64         `json:"copied_size"`
	Speed           float64       `json:"speed"`
	ElapsedTime     time.Duration `json:"elapsed_time"`
	EstimatedTime   time.Duration `json:"estimated_time"`
	ProgressPercent float64       `json:"progress_percent"`
}

// IsCompleted 检查是否完成
func (pt *ProgressTracker) IsCompleted() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.completedFiles >= pt.totalFiles
}

// GetElapsedTime 获取已用时间
func (pt *ProgressTracker) GetElapsedTime() time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return time.Since(pt.startTime)
}

// GetRemainingFiles 获取剩余文件数
func (pt *ProgressTracker) GetRemainingFiles() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.totalFiles - pt.completedFiles
}

// GetCopySpeed 获取当前拷贝速度
func (pt *ProgressTracker) GetCopySpeed() float64 {
	return pt.speedCalculator.GetCurrentSpeed()
}

// GetAverageSpeed 获取平均拷贝速度
func (pt *ProgressTracker) GetAverageSpeed() float64 {
	return pt.speedCalculator.GetAverageSpeed()
}

// Reset 重置进度跟踪器
func (pt *ProgressTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.totalFiles = 0
	pt.completedFiles = 0
	pt.totalSize = 0
	pt.copiedSize = 0
	pt.startTime = time.Now()
	pt.lastUpdateTime = time.Now()
	pt.currentFile = nil
	pt.speedCalculator = NewSpeedCalculator()
}

// LogProgress 记录当前进度到日志
func (pt *ProgressTracker) LogProgress() {
	info := pt.GetProgressInfo()
	pt.log.Info("进度: %d/%d (%.1f%%), 速度: %.2f MB/s, 已用: %s",
		info.CompletedFiles, info.TotalFiles, info.ProgressPercent,
		info.Speed, utils.FormatDuration(info.ElapsedTime))
}