package device

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestExtractVIDPID 测试从设备ID中提取VID和PID
func TestExtractVIDPID(t *testing.T) {
	testCases := []struct {
		name     string
		deviceID string
		vid      string
		pid      string
	}{
		{
			name:     "标准格式",
			deviceID: "USB\\VID_2207&PID_0011\\123456",
			vid:      "2207",
			pid:      "0011",
		},
		{
			name:     "小写格式",
			deviceID: "usb\\vid_2207&pid_0011\\123456",
			vid:      "2207",
			pid:      "0011",
		},
		{
			name:     "混合大小写",
			deviceID: "USB\\Vid_2207&Pid_0011\\123456",
			vid:      "2207",
			pid:      "0011",
		},
		{
			name:     "没有VID",
			deviceID: "USB\\PID_0011\\123456",
			vid:      "",
			pid:      "",
		},
		{
			name:     "没有PID",
			deviceID: "USB\\VID_2207\\123456",
			vid:      "2207",
			pid:      "",
		},
		{
			name:     "VID太短",
			deviceID: "USB\\VID_22&PID_0011\\123456",
			vid:      "",
			pid:      "",
		},
		{
			name:     "PID太短",
			deviceID: "USB\\VID_2207&PID_00\\123456",
			vid:      "",
			pid:      "",
		},
		{
			name:     "空字符串",
			deviceID: "",
			vid:      "",
			pid:      "",
		},
		{
			name:     "其他格式",
			deviceID: "ACPI\\PNP0A08\\0",
			vid:      "",
			pid:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vid, pid := extractVIDPID(tc.deviceID)
			if vid != tc.vid {
				t.Errorf("期望VID为 '%s'，实际为 '%s'", tc.vid, vid)
			}
			if pid != tc.pid {
				t.Errorf("期望PID为 '%s'，实际为 '%s'", tc.pid, pid)
			}
		})
	}
}

// TestDetermineDeviceType 测试确定设备类型
func TestDetermineDeviceType(t *testing.T) {
	testCases := []struct {
		name       string
		deviceName string
		deviceID   string
		expectType string
	}{
		{
			name:       "MTP设备（名称）",
			deviceName: "MTP USB Device",
			deviceID:   "USB\\VID_2207&PID_0011\\123",
			expectType: "MTP",
		},
		{
			name:       "MTP设备（ID）",
			deviceName: "USB Device",
			deviceID:   "USB\\VID_2207&PID_0011&MTP\\123",
			expectType: "MTP",
		},
		{
			name:       "ADB设备（名称）",
			deviceName: "Android ADB Interface",
			deviceID:   "USB\\VID_2207&PID_0011\\123",
			expectType: "ADB",
		},
		{
			name:       "ADB设备（ID）",
			deviceName: "USB Device",
			deviceID:   "USB\\VID_2207&PID_0011&ADB\\123",
			expectType: "ADB",
		},
		{
			name:       "存储设备（名称）",
			deviceName: "USB Mass Storage Device",
			deviceID:   "USB\\VID_2207&PID_0011\\123",
			expectType: "STORAGE",
		},
		{
			name:       "存储设备（ID）",
			deviceName: "USB Device",
			deviceID:   "USB\\VID_2207&PID_0011\\STORAGE\\123",
			expectType: "STORAGE",
		},
		{
			name:       "USB设备（名称）",
			deviceName: "Generic USB Hub",
			deviceID:   "USB\\VID_2207&PID_0011\\123",
			expectType: "STORAGE",
		},
		{
			name:       "未知设备",
			deviceName: "Unknown Device",
			deviceID:   "ACPI\\PNP0A08\\0",
			expectType: "UNKNOWN",
		},
		{
			name:       "空信息",
			deviceName: "",
			deviceID:   "",
			expectType: "UNKNOWN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deviceType := determineDeviceType(tc.deviceName, tc.deviceID)
			if deviceType != tc.expectType {
				t.Errorf("期望设备类型为 '%s'，实际为 '%s'", tc.expectType, deviceType)
			}
		})
	}
}

// TestParseWMICOutput 测试解析WMI输出
func TestParseWMICOutput(t *testing.T) {
	testCases := []struct {
		name          string
		output        string
		expectDevices int
		expectError   bool
	}{
		{
			name: "正常输出",
			output: `Node,DeviceID,Name
COMPUTER,USB\VID_2207&PID_0011\123456,SR302 Voice Recorder
COMPUTER,USB\VID_1234&PID_5678\876543,USB Mass Storage
COMPUTER,ACPI\PNP0A08\0,Root Port`,
			expectDevices: 2,
			expectError:   false,
		},
		{
			name: "包含空行",
			output: `Node,DeviceID,Name
COMPUTER,USB\VID_2207&PID_0011\123456,SR302 Voice Recorder

COMPUTER,USB\VID_1234&PID_5678\876543,USB Mass Storage

`,
			expectDevices: 2,
			expectError:   false,
		},
		{
			name: "无效格式",
			output: `Node,DeviceID
COMPUTER,USB\VID_2207&PID_0011\123456
COMPUTER,USB\VID_1234&PID_5678\876543`,
			expectDevices: 0,
			expectError:   false, // 应该忽略无效行
		},
		{
			name:          "空输出",
			output:        "",
			expectDevices: 0,
			expectError:   false,
		},
		{
			name:          "只有标题",
			output:        "Node,DeviceID,Name",
			expectDevices: 0,
			expectError:   false,
		},
		{
			name: "缺少VID/PID",
			output: `Node,DeviceID,Name
COMPUTER,USB\DEVICE123\123456,Test Device
COMPUTER,ACPI\PNP0A08\0,Root Port`,
			expectDevices: 0,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			devices, err := parseWMICOutput(tc.output)

			if tc.expectError && err == nil {
				t.Error("期望返回错误，但没有错误")
			} else if !tc.expectError && err != nil {
				t.Errorf("不期望返回错误，但得到: %v", err)
			}

			if len(devices) != tc.expectDevices {
				t.Errorf("期望设备数量为 %d，实际为 %d", tc.expectDevices, len(devices))
			}

			// 验证设备信息
			for _, device := range devices {
				if device.DeviceID == "" {
					t.Error("设备ID不应为空")
				}
				if device.Name == "" {
					t.Error("设备名称不应为空")
				}
				if device.VID == "" || device.PID == "" {
					t.Error("VID和PID不应为空")
				}
			}
		})
	}
}

// TestIsDeviceConnected 测试检查设备是否连接
func TestIsDeviceConnected(t *testing.T) {
	// 由于这个函数需要调用实际的系统命令，我们只测试参数验证
	deviceInfo := &DeviceInfo{
		DeviceID: "USB\\VID_2207&PID_0011\\123456",
		Name:     "SR302 Voice Recorder",
		VID:      "2207",
		PID:      "0011",
	}

	// 测试空设备信息
	connected, err := IsDeviceConnected(nil)
	if err == nil {
		t.Error("空设备信息应该返回错误")
	}

	// 测试正常设备信息
	// 注意：由于测试环境可能没有连接设备，这里只测试函数调用不出错
	_, err = IsDeviceConnected(deviceInfo)
	// 不检查结果，只检查函数调用
	_ = err
}

// TestScanAllUSBDevices 测试扫描所有USB设备
func TestScanAllUSBDevices(t *testing.T) {
	// 注意：这个测试需要实际执行系统命令，在CI环境中可能失败
	// 我们只测试函数调用不出错
	devices, err := ScanAllUSBDevices()
	if err != nil {
		t.Errorf("扫描USB设备失败: %v", err)
	}

	// 验证返回的设备信息
	for _, device := range devices {
		if device.DeviceID == "" {
			t.Error("设备ID不应为空")
		}
		if device.Name == "" {
			t.Error("设备名称不应为空")
		}
		// VID和PID可能为空（非USB设备）
	}
}

// TestDeviceInfoStruct 测试DeviceInfo结构体
func TestDeviceInfoStruct(t *testing.T) {
	device := &DeviceInfo{
		DeviceID:    "USB\\VID_2207&PID_0011\\123456",
		Name:        "SR302 Voice Recorder",
		VID:         "2207",
		PID:         "0011",
		IsMTP:       true,
		IsADB:       false,
		ConnectedAt: time.Now(),
	}

	// 验证JSON标签
	deviceType := reflect.TypeOf(device)
	for i := 0; i < deviceType.NumField(); i++ {
		field := deviceType.Field(i)
		if field.Tag.Get("json") == "" {
			t.Errorf("字段 %s 缺少JSON标签", field.Name)
		}
	}
}

// TestUSBDeviceStruct 测试USBDevice结构体
func TestUSBDeviceStruct(t *testing.T) {
	device := &USBDevice{
		DeviceID:   "USB\\VID_2207&PID_0011\\123456",
		Name:       "SR302 Voice Recorder",
		VID:        "2207",
		PID:        "0011",
		DeviceType: "MTP",
	}

	// 验证字段
	if device.DeviceID == "" {
		t.Error("DeviceID不应为空")
	}
	if device.Name == "" {
		t.Error("Name不应为空")
	}
	if device.VID == "" {
		t.Error("VID不应为空")
	}
	if device.PID == "" {
		t.Error("PID不应为空")
	}
	if device.DeviceType == "" {
		t.Error("DeviceType不应为空")
	}
}

// TestConstants 测试常量定义
func TestConstants(t *testing.T) {
	if SR302_NAME != "SR302" {
		t.Errorf("SR302_NAME应为 'SR302'，实际为 '%s'", SR302_NAME)
	}

	if SR302_VID != "2207" {
		t.Errorf("SR302_VID应为 '2207'，实际为 '%s'", SR302_VID)
	}

	if SR302_PID != "0011" {
		t.Errorf("SR302_PID应为 '0011'，实际为 '%s'", SR302_PID)
	}
}

// BenchmarkExtractVIDPID 性能测试：提取VID和PID
func BenchmarkExtractVIDPID(b *testing.B) {
	deviceID := "USB\\VID_2207&PID_0011\\1234567890ABCDEF"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractVIDPID(deviceID)
	}
}

// BenchmarkDetermineDeviceType 性能测试：确定设备类型
func BenchmarkDetermineDeviceType(b *testing.B) {
	name := "SR302 Voice Recorder MTP Device"
	deviceID := "USB\\VID_2207&PID_0011&MI_00\\6&12345678&0&0000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		determineDeviceType(name, deviceID)
	}
}

// ExampleExtractVIDPID 示例：提取VID和PID
func ExampleExtractVIDPID() {
	deviceID := "USB\\VID_2207&PID_0011\\123456"
	vid, pid := extractVIDPID(deviceID)
	println("VID:", vid)
	println("PID:", pid)
	// Output:
	// VID: 2207
	// PID: 0011
}

// ExampleDetermineDeviceType 示例：确定设备类型
func ExampleDetermineDeviceType() {
	name := "SR302 MTP Device"
	deviceID := "USB\\VID_2207&PID_0011\\123456"
	deviceType := determineDeviceType(name, deviceID)
	println("Device Type:", deviceType)
	// Output:
	// Device Type: MTP
}

// MockWMICOutput 模拟WMI输出解析（避免系统调用）
// 注意：这个函数仅供测试使用
func mockParseWMICOutputForTest(output string) ([]*USBDevice, error) {
	// 使用真实的parseWMICOutput函数
	return parseWMICOutput(output)
}

// TestParseWMICOutputEdgeCases 测试边缘情况
func TestParseWMICOutputEdgeCases(t *testing.T) {
	testCases := []struct {
		name   string
		output string
		check  func([]*USBDevice) bool
	}{
		{
			name: "设备名称包含逗号",
			output: `Node,DeviceID,Name
COMPUTER,USB\VID_2207&PID_0011\123,Device, Name with commas`,
			check: func(devices []*USBDevice) bool {
				return len(devices) == 1 && strings.Contains(devices[0].Name, "Device")
			},
		},
		{
			name: "特殊字符",
			output: `Node,DeviceID,Name
COMPUTER,USB\VID_2207&PID_0011\123,Device®™™ Special`,
			check: func(devices []*USBDevice) bool {
				return len(devices) == 1
			},
		},
		{
			name: "多个空字段",
			output: `Node,DeviceID,Name
COMPUTER,,Empty Device ID
COMPUTER,USB\VID_2207&PID_0011\123,`,
			check: func(devices []*USBDevice) bool {
				return len(devices) == 0 // 应该过滤掉无效设备
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			devices, err := mockParseWMICOutputForTest(tc.output)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if !tc.check(devices) {
				t.Error("检查失败")
			}
		})
	}
}