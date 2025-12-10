package device

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/allanpk716/record_center/internal/logger"
)

// SR302设备信息常量
const (
	SR302_NAME = "SR302"
	SR302_VID  = "2207"
	SR302_PID  = "0011"
)

// DeviceInfo 设备信息结构
type DeviceInfo struct {
	DeviceID   string    `json:"device_id"`
	Name       string    `json:"name"`
	VID        string    `json:"vid"`
	PID        string    `json:"pid"`
	IsMTP      bool      `json:"is_mtp"`
	IsADB      bool      `json:"is_adb"`
	ConnectedAt time.Time `json:"connected_at"`
}

// USBDevice USB设备信息
type USBDevice struct {
	DeviceID   string
	Name       string
	VID        string
	PID        string
	DeviceType string
}

// DetectSR302 检测SR302设备
func DetectSR302() (*DeviceInfo, error) {
	// 1. 通过WMI查询USB设备
	devices, err := enumerateUSBDevices()
	if err != nil {
		return nil, fmt.Errorf("枚举USB设备失败: %w", err)
	}

	// 2. 查找SR302设备
	for _, device := range devices {
		if strings.Contains(strings.ToUpper(device.Name), strings.ToUpper(SR302_NAME)) &&
			device.VID == SR302_VID &&
			device.PID == SR302_PID {

			// 创建设备信息
			deviceInfo := &DeviceInfo{
				DeviceID:    device.DeviceID,
				Name:        device.Name,
				VID:         device.VID,
				PID:         device.PID,
				IsMTP:       strings.Contains(strings.ToUpper(device.DeviceType), "MTP"),
				IsADB:       strings.Contains(strings.ToUpper(device.DeviceType), "ADB"),
				ConnectedAt: time.Now(),
			}

			return deviceInfo, nil
		}
	}

	return nil, fmt.Errorf("未找到SR302设备 (VID:%s, PID:%s)", SR302_VID, SR302_PID)
}

// enumerateUSBDevices 通过WMI枚举USB设备
func enumerateUSBDevices() ([]*USBDevice, error) {
	// 使用wmic命令查询USB设备
	cmd := exec.Command("wmic", "path", "win32_pnpentity", "where",
		`(deviceid like '%USB%')`, "get", "deviceid,name", "/format:csv")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行WMI查询失败: %w", err)
	}

	return parseWMICOutput(string(output))
}

// parseWMICOutput 解析WMI命令输出
func parseWMICOutput(output string) ([]*USBDevice, error) {
	var devices []*USBDevice
	lines := strings.Split(output, "\n")

	// 跳过标题行和空行
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析CSV格式（简化处理）
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

			deviceID := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])

		if deviceID == "" || name == "" {
			continue
		}

		// 解析VID和PID
		vid, pid := extractVIDPID(deviceID)
		if vid == "" || pid == "" {
			continue
		}

		device := &USBDevice{
			DeviceID:   deviceID,
			Name:       name,
			VID:        vid,
			PID:        pid,
			DeviceType: determineDeviceType(name, deviceID),
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// extractVIDPID 从设备ID中提取VID和PID
func extractVIDPID(deviceID string) (string, string) {
	// 查找VID和PID模式
	vidPattern := "VID_"
	pidPattern := "PID_"

	vidIndex := strings.Index(strings.ToUpper(deviceID), vidPattern)
	if vidIndex == -1 {
		return "", ""
	}

	vidStart := vidIndex + len(vidPattern)
	if vidStart+4 > len(deviceID) {
		return "", ""
	}
	vid := strings.ToUpper(deviceID[vidStart : vidStart+4])

	pidIndex := strings.Index(strings.ToUpper(deviceID), pidPattern)
	if pidIndex == -1 {
		return "", ""
	}

	pidStart := pidIndex + len(pidPattern)
	if pidStart+4 > len(deviceID) {
		return "", ""
	}
	pid := strings.ToUpper(deviceID[pidStart : pidStart+4])

	return vid, pid
}

// determineDeviceType 确定设备类型
func determineDeviceType(name, deviceID string) string {
	nameUpper := strings.ToUpper(name)
	deviceIDUpper := strings.ToUpper(deviceID)

	if strings.Contains(nameUpper, "MTP") || strings.Contains(deviceIDUpper, "MTP") {
		return "MTP"
	}
	if strings.Contains(nameUpper, "ADB") || strings.Contains(deviceIDUpper, "ADB") {
		return "ADB"
	}
	if strings.Contains(nameUpper, "STORAGE") || strings.Contains(nameUpper, "USB") {
		return "STORAGE"
	}

	return "UNKNOWN"
}

// IsDeviceConnected 检查设备是否仍然连接
func IsDeviceConnected(deviceInfo *DeviceInfo) (bool, error) {
	devices, err := enumerateUSBDevices()
	if err != nil {
		return false, err
	}

	for _, device := range devices {
		if device.DeviceID == deviceInfo.DeviceID &&
			device.VID == deviceInfo.VID &&
			device.PID == deviceInfo.PID {
			return true, nil
		}
	}

	return false, nil
}

// WaitForDevice 等待设备连接（超时时间单位为秒）
func WaitForDevice(timeout int) (*DeviceInfo, error) {
	log := logger.NewLogger(true)
	log.Info("等待SR302设备连接...")

	start := time.Now()
	for time.Since(start).Seconds() < float64(timeout) {
		device, err := DetectSR302()
		if err == nil {
			log.Info("设备已连接: %s", device.Name)
			return device, nil
		}

		// 等待1秒后重试
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("等待设备超时 (%d秒)", timeout)
}

// ListAllConnectedDevices 列出所有连接的USB设备（调试用）
func ListAllConnectedDevices() ([]*USBDevice, error) {
	return enumerateUSBDevices()
}

// ScanAllUSBDevices 扫描所有USB设备并转换为DeviceInfo格式
func ScanAllUSBDevices() ([]*DeviceInfo, error) {
	usbDevices, err := enumerateUSBDevices()
	if err != nil {
		return nil, err
	}

	var deviceInfos []*DeviceInfo
	for _, usbDevice := range usbDevices {
		deviceInfo := &DeviceInfo{
			DeviceID:    usbDevice.DeviceID,
			Name:        usbDevice.Name,
			VID:         usbDevice.VID,
			PID:         usbDevice.PID,
			IsMTP:       strings.Contains(strings.ToUpper(usbDevice.DeviceType), "MTP"),
			IsADB:       strings.Contains(strings.ToUpper(usbDevice.DeviceType), "ADB"),
			ConnectedAt: time.Now(),
		}
		deviceInfos = append(deviceInfos, deviceInfo)
	}

	return deviceInfos, nil
}

// GetDeviceInfoFromPath 从路径获取设备信息（如果适用）
func GetDeviceInfoFromPath(path string) (*DeviceInfo, error) {
	// 这个函数用于处理设备挂载为驱动器的情况
	// 对于SR302 MTP设备，可能不适用，但保留接口以便扩展

	// 如果是驱动器路径，检查是否为可移动存储
	if len(path) >= 2 && path[1] == ':' {
		cmd := exec.Command("wmic", "logicaldisk", "where",
			fmt.Sprintf("deviceid='%s'", path[:2]), "get", "drivetype", "/format:csv")

		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("查询驱动器类型失败: %w", err)
		}

		// 解析输出，检查是否为可移动存储
		if strings.Contains(string(output), "2") { // DriveType=2 表示可移动存储
			return &DeviceInfo{
				DeviceID:    path,
				Name:        "Removable Storage",
				VID:         "",
				PID:         "",
				IsMTP:       false,
				IsADB:       false,
				ConnectedAt: time.Now(),
			}, nil
		}
	}

	return nil, fmt.Errorf("路径不支持: %s", path)
}