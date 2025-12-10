//go:build windows

package device

import (
	"io"
	"time"
)

// MTPInterface 定义统一的MTP设备访问接口
type MTPInterface interface {
	// ConnectToDevice 连接到指定的MTP设备
	ConnectToDevice(deviceName, vid, pid string) error

	// ListFiles 列出指定路径下的文件
	ListFiles(basePath string) ([]*FileInfo, error)

	// GetFileStream 获取文件读取流
	GetFileStream(filePath string) (io.ReadCloser, error)

	// Close 关闭连接并释放资源
	Close() error

	// IsConnected 检查是否已连接到设备
	IsConnected() bool

	// GetDeviceInfo 获取设备信息
	GetDeviceInfo() *DeviceInfo
}

// DeviceBridge 定义设备检测与MTP访问桥接接口
type DeviceBridge interface {
	// DetectAndBridge 检测设备并创建MTP访问接口
	DetectAndBridge(deviceName string) (MTPInterface, error)

	// GetDevicePath 获取设备访问路径
	GetDevicePath(deviceName, vid, pid string) (string, error)

	// ListAvailableDevices 列出所有可用的MTP设备
	ListAvailableDevices() ([]*DeviceInfo, error)
}

// PathResolver 定义设备路径解析接口
type PathResolver interface {
	// Resolve 解析设备路径
	Resolve(deviceName, vid, pid string) (string, error)

	// GetPriority 获取解析策略的优先级
	GetPriority() int

	// IsAvailable 检查策略是否可用
	IsAvailable() bool
}

// MTPErrorCode 定义MTP错误代码类型
type MTPErrorCode int

const (
	// ERROR_NONE 无错误
	ERROR_NONE MTPErrorCode = iota
	// ERROR_DEVICE_NOT_FOUND 设备未找到
	ERROR_DEVICE_NOT_FOUND
	// ERROR_ACCESS_DENIED 访问被拒绝
	ERROR_ACCESS_DENIED
	// ERROR_DEVICE_BUSY 设备忙碌
	ERROR_DEVICE_BUSY
	// ERROR_NOT_SUPPORTED 不支持的操作
	ERROR_NOT_SUPPORTED
	// ERROR_TIMEOUT 操作超时
	ERROR_TIMEOUT
	// ERROR_INVALID_PARAMETER 参数无效
	ERROR_INVALID_PARAMETER
	// ERROR_COM_ERROR COM接口错误
	ERROR_COM_ERROR
	// ERROR_POWER_SHELL_FAILED PowerShell执行失败
	ERROR_POWER_SHELL_FAILED
)

// MTPError 定义MTP访问错误
type MTPError struct {
	Code      MTPErrorCode
	Message   string
	Cause     error
	Context   map[string]interface{}
	Retryable bool
}

// Error 实现error接口
func (e *MTPError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// IsRetryable 检查错误是否可重试
func (e *MTPError) IsRetryable() bool {
	return e.Retryable ||
		e.Code == ERROR_DEVICE_BUSY ||
		e.Code == ERROR_TIMEOUT ||
		e.Code == ERROR_ACCESS_DENIED
}

// NewMTPError 创建新的MTP错误
func NewMTPError(code MTPErrorCode, message string, cause error) *MTPError {
	return &MTPError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
		Retryable: false,
	}
}

// NewRetryableMTPError 创建可重试的MTP错误
func NewRetryableMTPError(code MTPErrorCode, message string, cause error) *MTPError {
	return &MTPError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
		Retryable: true,
	}
}

// AddContext 添加错误上下文信息
func (e *MTPError) AddContext(key string, value interface{}) {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
}

// ConnectionConfig 定义连接配置
type ConnectionConfig struct {
	Timeout       time.Duration // 连接超时时间
	MaxRetries    int           // 最大重试次数
	RetryDelay    time.Duration // 重试延迟
	UseFallback   bool          // 是否使用降级策略
	Verbose       bool          // 是否启用详细日志
}

// DefaultConnectionConfig 返回默认连接配置
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  1 * time.Second,
		UseFallback: true,
		Verbose:     false,
	}
}

// AccessMethod 定义访问方法类型
type AccessMethod string

const (
	// MethodWindowsShellCOM Windows Shell COM接口
	MethodWindowsShellCOM AccessMethod = "WindowsShellCOM"
	// MethodPowerShell PowerShell访问
	MethodPowerShell AccessMethod = "PowerShell"
	// MethodWMI WMI访问
	MethodWMI AccessMethod = "WMI"
	// MethodDirectFile 直接文件系统访问
	MethodDirectFile AccessMethod = "DirectFile"
)

// AccessResult 定义访问结果
type AccessResult struct {
	Method      AccessMethod
	Success     bool
	FilesCount  int
	Error       error
	Duration    time.Duration
	DevicePath  string
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	TotalAttempts   int
	SuccessCount    int
	TotalFiles      int64
	TotalTime       time.Duration
	AverageTime     time.Duration
	LastSuccessTime time.Time
	LastError       error
	MethodStats     map[AccessMethod]*MethodStats
}

// MethodStats 方法统计
type MethodStats struct {
	Method         AccessMethod
	SuccessCount   int
	FailureCount   int
	TotalFiles     int64
	TotalTime      time.Duration
	AverageTime    time.Duration
	LastSuccess    time.Time
	LastFailure    time.Time
	SuccessRate    float64
}

// CalculateSuccessRate 计算成功率
func (ms *MethodStats) CalculateSuccessRate() {
	total := ms.SuccessCount + ms.FailureCount
	if total > 0 {
		ms.SuccessRate = float64(ms.SuccessCount) / float64(total)
	}
}

// UpdateAverageTime 更新平均时间
func (ms *MethodStats) UpdateAverageTime() {
	totalAttempts := ms.SuccessCount + ms.FailureCount
	if totalAttempts > 0 {
		ms.AverageTime = ms.TotalTime / time.Duration(totalAttempts)
	}
}