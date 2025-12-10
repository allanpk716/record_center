package device

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// String 返回方法的字符串表示
func (m AccessMethod) String() string {
	switch m {
	case MethodPowerShell:
		return "PowerShell"
	case MethodWMI:
		return "WMI"
	case MethodDirectFile:
		return "直接文件访问"
	case MethodWindowsShellCOM:
		return "Windows Shell COM"
	case "PowerShellEnhanced":
		return "增强PowerShell"
	default:
		return "未知方法"
	}
}

// MethodStatistics 方法统计信息
type MethodStatistics struct {
	Method         AccessMethod
	SuccessCount   int
	FailureCount   int
	LastSuccessTime time.Time
	LastFailureTime time.Time
	SuccessRate    float64
}

// MTPRetryManager MTP重试管理器
type MTPRetryManager struct {
	log           *logger.Logger
	maxAttempts   int
	retryDelay    time.Duration
	statistics    map[AccessMethod]*MethodStatistics
	methodOrder   []AccessMethod // 访问方法的优先级顺序
}

// NewMTPRetryManager 创建MTP重试管理器
func NewMTPRetryManager(log *logger.Logger, maxAttempts int) *MTPRetryManager {
	manager := &MTPRetryManager{
		log:         log,
		maxAttempts: maxAttempts,
		retryDelay:  time.Second,
		statistics:  make(map[AccessMethod]*MethodStatistics),
		methodOrder: []AccessMethod{
			"PowerShellEnhanced",  // 首选增强PowerShell方法
			MethodPowerShell,      // 标准PowerShell方法
			MethodWMI,             // 备选WMI方法
			MethodDirectFile,      // 最后尝试直接文件访问
		},
	}

	// 初始化统计信息
	for _, method := range manager.methodOrder {
		manager.statistics[method] = &MethodStatistics{
			Method: method,
		}
	}

	return manager
}

// ScanWithRetry 使用重试机制扫描MTP设备
func (manager *MTPRetryManager) ScanWithRetry(accessor *MTPAccessor, deviceName, basePath string) ([]*FileInfo, error) {
	manager.log.Debug("开始MTP重试扫描: %s", deviceName)

	var lastError error
	var lastFiles []*FileInfo

	// 尝试每种访问方法
	for methodIndex, method := range manager.methodOrder {
		manager.log.Debug("尝试访问方法 %d/%d: %s", methodIndex+1, len(manager.methodOrder), method)

		// 检查方法成功率，跳过长期失败的方法
		stats := manager.statistics[method]
		if stats.FailureCount > 10 && stats.SuccessRate < 0.1 {
			manager.log.Debug("跳过低成功率方法: %s (成功率: %.1f%%)", method, stats.SuccessRate*100)
			continue
		}

		// 尝试指定方法
		files, err := manager.tryMethod(accessor, method, deviceName, basePath)
		if err != nil {
			manager.recordFailure(method, err)
			lastError = err
			manager.log.Warn("方法 %s 失败: %v", method, err)
			continue
		}

		// 成功获取文件
		manager.recordSuccess(method, len(files))
		lastFiles = files
		manager.log.Info("成功使用 %s 方法获取到 %d 个文件", method, len(files))
		break
	}

	if lastFiles == nil && lastError != nil {
		manager.log.Error("所有访问方法都失败了")
		manager.log.Error("最后错误: %v", lastError)
		manager.printStatistics()
		return nil, lastError
	}

	return lastFiles, nil
}

// tryMethod 尝试指定的访问方法
func (manager *MTPRetryManager) tryMethod(accessor *MTPAccessor, method AccessMethod, deviceName, basePath string) ([]*FileInfo, error) {
	switch method {
	case "PowerShellEnhanced":
		return manager.tryPowerShellEnhancedMethod(accessor, deviceName, basePath)
	case MethodPowerShell:
		return manager.tryPowerShellMethod(accessor, deviceName, basePath)
	case MethodWMI:
		return manager.tryWMIMethod(accessor, deviceName)
	case MethodDirectFile:
		return manager.tryDirectFileMethod(accessor, deviceName, basePath)
	default:
		return nil, fmt.Errorf("未知的访问方法: %v", method)
	}
}

// convertMTPFilesToFileInfo 转换MTPFileEntry到FileInfo
func (manager *MTPRetryManager) convertMTPFilesToFileInfo(mtpFiles []*MTPFileEntry) []*FileInfo {
	var files []*FileInfo
	for _, mtpFile := range mtpFiles {
		// 跳过目录
		if mtpFile.IsDir {
			continue
		}

		// 根据扩展名判断是否为Opus文件
		isOpus := false
		if ext := strings.ToLower(filepath.Ext(mtpFile.Name)); ext == ".opus" {
			isOpus = true
		}

		fileInfo := &FileInfo{
			Path:         mtpFile.Path,
			RelativePath: mtpFile.RelativePath,
			Name:         mtpFile.Name,
			Size:         mtpFile.Size,
			IsOpus:       isOpus,
			ModTime:      mtpFile.ModTime,
		}
		files = append(files, fileInfo)
	}
	return files
}

// tryPowerShellMethod 尝试PowerShell方法
func (manager *MTPRetryManager) tryPowerShellMethod(accessor *MTPAccessor, deviceName, basePath string) ([]*FileInfo, error) {
	manager.log.Debug("使用PowerShell方法访问MTP设备")

	// 获取PowerShell访问器
	psAccessor := NewPowerShellMTPAccessor(manager.log)
	if psAccessor == nil {
		return nil, fmt.Errorf("PowerShell访问器创建失败")
	}

	// 获取设备路径
	devicePath, err := psAccessor.GetMTPDevicePath(deviceName)
	if err != nil {
		return nil, fmt.Errorf("获取设备路径失败: %w", err)
	}

	// 扫描文件
	mtpFiles, err := psAccessor.ListMTPFiles(devicePath, basePath)
	if err != nil {
		return nil, fmt.Errorf("PowerShell文件扫描失败: %w", err)
	}

	// 转换格式
	return manager.convertMTPFilesToFileInfo(mtpFiles), nil
}

// tryPowerShellEnhancedMethod 尝试增强PowerShell方法
func (manager *MTPRetryManager) tryPowerShellEnhancedMethod(accessor *MTPAccessor, deviceName, basePath string) ([]*FileInfo, error) {
	manager.log.Debug("使用增强PowerShell方法")

	// 创建增强PowerShell访问器
	enhanced := NewPowerShellEnhanced(manager.log)
	if enhanced == nil {
		return nil, fmt.Errorf("增强PowerShell访问器创建失败")
	}

	// 暂时使用默认VID/PID，后续应该从设备信息获取
	err := enhanced.ConnectToDevice(deviceName, "2207", "0011")
	if err != nil {
		return nil, fmt.Errorf("增强PowerShell连接失败: %w", err)
	}
	defer enhanced.Close()

	// 列出文件
	return enhanced.ListFiles(basePath)
}

// tryWMIMethod 尝试WMI方法
func (manager *MTPRetryManager) tryWMIMethod(accessor *MTPAccessor, deviceName string) ([]*FileInfo, error) {
	manager.log.Debug("使用WMI方法")

	// WMI主要用于设备管理，文件访问需要其他方法
	return nil, fmt.Errorf("WMI方法主要用于设备管理，不支持文件访问")
}

// tryDirectFileMethod 尝试直接文件访问方法
func (manager *MTPRetryManager) tryDirectFileMethod(accessor *MTPAccessor, deviceName, basePath string) ([]*FileInfo, error) {
	manager.log.Debug("使用直接文件访问方法")

	// 尝试直接访问设备作为文件系统
	_, err := accessor.GetMTPDevicePath(deviceName)
	if err != nil {
		return nil, fmt.Errorf("无法获取设备路径: %w", err)
	}

	// 这里可以实现直接文件系统访问
	return nil, fmt.Errorf("直接文件访问方法尚未完全实现")
}

// recordSuccess 记录成功
func (manager *MTPRetryManager) recordSuccess(method AccessMethod, fileCount int) {
	stats := manager.statistics[method]
	stats.SuccessCount++
	stats.LastSuccessTime = time.Now()
	stats.calculateSuccessRate()
}

// recordFailure 记录失败
func (manager *MTPRetryManager) recordFailure(method AccessMethod, err error) {
	stats := manager.statistics[method]
	stats.FailureCount++
	stats.LastFailureTime = time.Now()
	stats.calculateSuccessRate()
}

// calculateSuccessRate 计算成功率
func (s *MethodStatistics) calculateSuccessRate() {
	total := s.SuccessCount + s.FailureCount
	if total > 0 {
		s.SuccessRate = float64(s.SuccessCount) / float64(total)
	}
}

// printStatistics 打印统计信息
func (manager *MTPRetryManager) printStatistics() {
	manager.log.Info("MTP访问方法统计:")
	for _, method := range manager.methodOrder {
		stats := manager.statistics[method]
		manager.log.Info("  %s: 成功 %d 次, 失败 %d 次, 成功率 %.1f%%",
			method, stats.SuccessCount, stats.FailureCount, stats.SuccessRate*100)
	}
}

// GetStatistics 获取统计信息
func (manager *MTPRetryManager) GetStatistics() map[AccessMethod]*MethodStatistics {
	// 返回统计信息的副本
	result := make(map[AccessMethod]*MethodStatistics)
	for k, v := range manager.statistics {
		result[k] = &MethodStatistics{
			Method:         v.Method,
			SuccessCount:   v.SuccessCount,
			FailureCount:   v.FailureCount,
			LastSuccessTime: v.LastSuccessTime,
			LastFailureTime: v.LastFailureTime,
			SuccessRate:    v.SuccessRate,
		}
	}
	return result
}