//go:build windows

package device

import (
	"fmt"
	"sync"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// DeviceBridgeImpl 设备桥接实现
type DeviceBridgeImpl struct {
	log           *logger.Logger
	config        *ConnectionConfig
	resolvers     []PathResolver
	accessResults map[AccessMethod]*AccessResult
	mutex         sync.RWMutex
	stats         *PerformanceStats
}

// NewDeviceBridge 创建新的设备桥接器
func NewDeviceBridge(log *logger.Logger, config *ConnectionConfig) *DeviceBridgeImpl {
	if config == nil {
		config = DefaultConnectionConfig()
	}

	bridge := &DeviceBridgeImpl{
		log:           log,
		config:        config,
		accessResults: make(map[AccessMethod]*AccessResult),
		stats: &PerformanceStats{
			MethodStats: make(map[AccessMethod]*MethodStats),
		},
	}

	// 初始化路径解析器
	bridge.initResolvers()

	return bridge
}

// initResolvers 初始化路径解析器
func (db *DeviceBridgeImpl) initResolvers() {
	// 按优先级添加解析器
	db.resolvers = []PathResolver{
		NewPowerShellEnhancedResolver(db.log), // 最高优先级，使用增强的PowerShell
		NewPowerShellResolver(db.log),         // 标准PowerShell方案
		NewWMIResolver(db.log),                // 备选方案
		NewDirectFileResolver(db.log),         // 最低优先级
	}
}

// DetectAndBridge 检测设备并创建MTP访问接口
func (db *DeviceBridgeImpl) DetectAndBridge(deviceName string) (MTPInterface, error) {
	db.log.Debug("开始检测和桥接设备: %s", deviceName)

	// 首先检测设备
	devices, err := db.ListAvailableDevices()
	if err != nil {
		return nil, NewMTPError(ERROR_DEVICE_NOT_FOUND, "无法列出可用设备", err)
	}

	// 查找目标设备
	var targetDevice *DeviceInfo
	for _, device := range devices {
		if device.Name == deviceName {
			targetDevice = device
			break
		}
	}

	if targetDevice == nil {
		return nil, NewMTPError(ERROR_DEVICE_NOT_FOUND,
			fmt.Sprintf("未找到设备: %s", deviceName), nil)
	}

	db.log.Debug("找到目标设备: %s (VID:%s, PID:%s)",
		targetDevice.Name, targetDevice.VID, targetDevice.PID)

	// 尝试不同的访问方法
	for _, resolver := range db.resolvers {
		if !resolver.IsAvailable() {
			db.log.Debug("跳过不可用的解析器: %T", resolver)
			continue
		}

		methodName := db.getMethodName(resolver)
		db.log.Debug("尝试访问方法: %s (优先级: %d)", methodName, resolver.GetPriority())

		startTime := time.Now()

		// 尝试解析设备路径
		devicePath, err := resolver.Resolve(targetDevice.Name, targetDevice.VID, targetDevice.PID)
		duration := time.Since(startTime)

		// 记录访问结果
		result := &AccessResult{
			Method:     methodName,
			Success:    err == nil,
			DevicePath: devicePath,
			Duration:   duration,
			Error:      err,
		}

		db.recordAccessResult(methodName, result)

		if err != nil {
			db.log.Warn("访问方法 %s 失败: %v (耗时: %v)", methodName, err, duration)
			continue
		}

		db.log.Info("成功使用 %s 方法访问设备 (耗时: %v)", methodName, duration)
		db.log.Debug("设备路径: %s", devicePath)

		// 根据解析器类型创建对应的MTP接口
		mtpInterface, err := db.createMTPInterface(resolver, targetDevice, devicePath)
		if err != nil {
			db.log.Warn("创建MTP接口失败: %v", err)
			continue
		}

		return mtpInterface, nil
	}

	// 所有方法都失败了
	db.log.Error("所有访问方法都失败了")
	db.printAccessSummary()

	return nil, NewMTPError(ERROR_DEVICE_NOT_FOUND,
		fmt.Sprintf("无法通过任何方法访问设备: %s", deviceName), nil)
}

// GetDevicePath 获取设备访问路径
func (db *DeviceBridgeImpl) GetDevicePath(deviceName, vid, pid string) (string, error) {
	db.log.Debug("获取设备路径: %s (VID:%s, PID:%s)", deviceName, vid, pid)

	for _, resolver := range db.resolvers {
		if !resolver.IsAvailable() {
			continue
		}

		devicePath, err := resolver.Resolve(deviceName, vid, pid)
		if err != nil {
			db.log.Debug("解析器 %v 失败: %v", resolver, err)
			continue
		}

		if devicePath != "" {
			db.log.Debug("找到设备路径: %s", devicePath)
			return devicePath, nil
		}
	}

	return "", NewMTPError(ERROR_DEVICE_NOT_FOUND,
		fmt.Sprintf("无法找到设备路径: %s", deviceName), nil)
}

// ListAvailableDevices 列出所有可用的MTP设备
func (db *DeviceBridgeImpl) ListAvailableDevices() ([]*DeviceInfo, error) {
	db.log.Debug("列出所有可用的MTP设备")

	// 使用现有的设备检测功能
	// 这里可以集成现有的 DetectSR302 和其他设备检测逻辑

	// 暂时使用现有的检测方法
	sr302Device, err := DetectSR302()
	if err != nil {
		db.log.Debug("SR302设备检测失败: %v", err)
		return []*DeviceInfo{}, nil
	}

	return []*DeviceInfo{sr302Device}, nil
}

// createMTPInterface 根据解析器类型创建对应的MTP接口
func (db *DeviceBridgeImpl) createMTPInterface(resolver PathResolver, device *DeviceInfo, devicePath string) (MTPInterface, error) {
	// 最高优先级：尝试WPD COM访问器
	db.log.Debug("尝试WPD COM访问器")
	wpdAccessor := NewWPDComAccessor(db.log)
	wpdErr := wpdAccessor.ConnectToDevice(device.Name, device.VID, device.PID)
	if wpdErr == nil {
		db.log.Info("成功使用WPD COM访问器")
		return wpdAccessor, nil
	}
	db.log.Debug("WPD COM访问器失败: %v", wpdErr)

	// 第二优先级：Windows原生MTP访问器
	windowsNative := NewWindowsNativeMTP(db.log)
	if windowsNativeErr := windowsNative.ConnectToDevice(device.Name, device.VID, device.PID); windowsNativeErr == nil {
		db.log.Info("使用Windows原生MTP访问器")
		return windowsNative, nil
	}
	db.log.Debug("Windows原生MTP访问器失败，尝试其他方法")

	// 备选方案
	switch resolver.(type) {
	case *PowerShellEnhancedResolver:
		enhanced := NewPowerShellEnhanced(db.log)
		err := enhanced.ConnectToDevice(device.Name, device.VID, device.PID)
		if err != nil {
			return nil, fmt.Errorf("增强PowerShell连接失败: %w", err)
		}
		return enhanced, nil
	case *PowerShellResolver:
		// 为PowerShellMTPAccessor添加包装器以实现MTPInterface
		return NewPowerShellMTPWrapper(db.log), nil
	case *WMIResolver:
		return NewWMIMTPAccessor(db.log), nil
	case *DirectFileResolver:
		return NewDirectFileAccessor(db.log, devicePath), nil
	default:
		return nil, NewMTPError(ERROR_NOT_SUPPORTED,
			fmt.Sprintf("不支持的解析器类型: %T", resolver), nil)
	}
}

// recordAccessResult 记录访问结果
func (db *DeviceBridgeImpl) recordAccessResult(method AccessMethod, result *AccessResult) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.accessResults[method] = result

	// 更新性能统计
	stats, exists := db.stats.MethodStats[method]
	if !exists {
		stats = &MethodStats{
			Method: method,
		}
		db.stats.MethodStats[method] = stats
	}

	if result.Success {
		stats.SuccessCount++
		stats.LastSuccess = time.Now()
	} else {
		stats.FailureCount++
		stats.LastFailure = time.Now()
	}

	stats.TotalTime += result.Duration
	stats.UpdateAverageTime()
	stats.CalculateSuccessRate()

	// 更新总体统计
	db.stats.TotalAttempts++
	db.stats.TotalTime += result.Duration
	if result.Success {
		db.stats.SuccessCount++
		db.stats.LastSuccessTime = time.Now()
	} else {
		db.stats.LastError = result.Error
	}

	if db.stats.TotalAttempts > 0 {
		db.stats.AverageTime = db.stats.TotalTime / time.Duration(db.stats.TotalAttempts)
	}
}

// getMethodName 获取方法名称
func (db *DeviceBridgeImpl) getMethodName(resolver PathResolver) AccessMethod {
	switch resolver.(type) {
	case *PowerShellEnhancedResolver:
		return "PowerShellEnhanced"
	case *WindowsShellResolver:
		return MethodWindowsShellCOM
	case *PowerShellResolver:
		return MethodPowerShell
	case *WMIResolver:
		return MethodWMI
	case *DirectFileResolver:
		return MethodDirectFile
	default:
		return "Unknown"
	}
}

// printAccessSummary 打印访问摘要
func (db *DeviceBridgeImpl) printAccessSummary() {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	db.log.Info("MTP访问方法统计:")
	for method, result := range db.accessResults {
		status := "失败"
		if result.Success {
			status = "成功"
		}
		db.log.Info("  %s: %s (耗时: %v)", method, status, result.Duration)
		if result.Error != nil {
			db.log.Info("    错误: %v", result.Error)
		}
	}
}

// GetStats 获取性能统计
func (db *DeviceBridgeImpl) GetStats() *PerformanceStats {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	// 返回统计信息的副本
	stats := &PerformanceStats{
		TotalAttempts:   db.stats.TotalAttempts,
		SuccessCount:    db.stats.SuccessCount,
		TotalTime:       db.stats.TotalTime,
		AverageTime:     db.stats.AverageTime,
		LastSuccessTime: db.stats.LastSuccessTime,
		LastError:       db.stats.LastError,
		MethodStats:     make(map[AccessMethod]*MethodStats),
	}

	for method, methodStats := range db.stats.MethodStats {
		stats.MethodStats[method] = &MethodStats{
			Method:         methodStats.Method,
			SuccessCount:   methodStats.SuccessCount,
			FailureCount:   methodStats.FailureCount,
			TotalTime:      methodStats.TotalTime,
			AverageTime:    methodStats.AverageTime,
			LastSuccess:    methodStats.LastSuccess,
			LastFailure:    methodStats.LastFailure,
			SuccessRate:    methodStats.SuccessRate,
		}
	}

	return stats
}

// Close 关闭桥接器并释放资源
func (db *DeviceBridgeImpl) Close() error {
	db.log.Debug("关闭设备桥接器")
	// 这里可以添加清理逻辑
	return nil
}