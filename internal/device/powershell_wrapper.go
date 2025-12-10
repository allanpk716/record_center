//go:build windows

package device

import (
	"fmt"
	"io"
	"os"

	"github.com/allanpk716/record_center/internal/logger"
)

// PowerShellMTPWrapper PowerShell MTP访问器包装器，用于实现MTPInterface
type PowerShellMTPWrapper struct {
	log           *logger.Logger
	accessor     *PowerShellMTPAccessor
	connected     bool
	device        *DeviceInfo
}

// NewPowerShellMTPWrapper 创建PowerShell MTP包装器
func NewPowerShellMTPWrapper(log *logger.Logger) *PowerShellMTPWrapper {
	accessor := NewPowerShellMTPAccessor(log)
	return &PowerShellMTPWrapper{
		log:       log,
		accessor: accessor,
		connected: false,
	}
}

// ConnectToDevice 连接到设备
func (wrapper *PowerShellMTPWrapper) ConnectToDevice(deviceName, vid, pid string) error {
	wrapper.log.Debug("PowerShell包装器连接设备: %s", deviceName)
	wrapper.device = &DeviceInfo{
		Name:      deviceName,
		VID:       vid,
		PID:       pid,
		DeviceID:  fmt.Sprintf("USB\\VID_%s&PID_%s", vid, pid),
	}
	wrapper.connected = true
	return nil
}

// ListFiles 列出文件
func (wrapper *PowerShellMTPWrapper) ListFiles(basePath string) ([]*FileInfo, error) {
	if !wrapper.connected {
		return nil, fmt.Errorf("设备未连接")
	}

	wrapper.log.Debug("PowerShell包装器列出文件: %s", basePath)

	// 获取设备路径
	devicePath, err := wrapper.accessor.GetMTPDevicePath(wrapper.device.Name)
	if err != nil {
		return nil, fmt.Errorf("获取设备路径失败: %w", err)
	}

	// 使用PowerShell访问器列出文件
	mtpFiles, err := wrapper.accessor.ListMTPFiles(devicePath, basePath)
	if err != nil {
		return nil, fmt.Errorf("PowerShell文件列表获取失败: %w", err)
	}

	// 转换为FileInfo格式
	var files []*FileInfo
	for _, mtpFile := range mtpFiles {
		// 跳过目录
		if mtpFile.IsDir {
			continue
		}

		fileInfo := &FileInfo{
			Path:         mtpFile.Path,
			RelativePath: mtpFile.RelativePath,
			Name:         mtpFile.Name,
			Size:         mtpFile.Size,
			IsOpus:       true, // 假设都是Opus文件
			ModTime:      mtpFile.ModTime,
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// GetFileStream 获取文件流
func (wrapper *PowerShellMTPWrapper) GetFileStream(filePath string) (io.ReadCloser, error) {
	wrapper.log.Debug("PowerShell包装器获取文件流: %s", filePath)

	// 对于PowerShell MTP，直接打开文件可能不工作
	// 这里提供一个基本的实现，实际上可能需要其他方法
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}
	return file, nil
}

// Close 关闭连接
func (wrapper *PowerShellMTPWrapper) Close() error {
	wrapper.connected = false
	wrapper.device = nil
	return nil
}

// IsConnected 检查连接状态
func (wrapper *PowerShellMTPWrapper) IsConnected() bool {
	return wrapper.connected
}

// GetDeviceInfo 获取设备信息
func (wrapper *PowerShellMTPWrapper) GetDeviceInfo() *DeviceInfo {
	return wrapper.device
}