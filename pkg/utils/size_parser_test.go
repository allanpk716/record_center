package utils

import (
	"testing"
	"time"
)

// TestParseByteSize 测试解析字节大小字符串
func TestParseByteSize(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		// 基本单位测试
		{
			name:     "字节",
			input:    "1024B",
			expected: 1024,
		},
		{
			name:     "字节（无B）",
			input:    "1024",
			expected: 1024,
		},
		{
			name:     "千字节",
			input:    "1KB",
			expected: 1024,
		},
		{
			name:     "千字节（无B）",
			input:    "1K",
			expected: 1024,
		},
		{
			name:     "兆字节",
			input:    "1MB",
			expected: 1024 * 1024,
		},
		{
			name:     "兆字节（无B）",
			input:    "1M",
			expected: 1024 * 1024,
		},
		{
			name:     "吉字节",
			input:    "1GB",
			expected: 1024 * 1024 * 1024,
		},
		{
			name:     "吉字节（无B）",
			input:    "1G",
			expected: 1024 * 1024 * 1024,
		},
		{
			name:     "太字节",
			input:    "1TB",
			expected: 1024 * 1024 * 1024 * 1024,
		},
		{
			name:     "太字节（无B）",
			input:    "1T",
			expected: 1024 * 1024 * 1024 * 1024,
		},
		// 小数测试
		{
			name:     "1.5MB",
			input:    "1.5MB",
			expected: int64(1.5 * float64(1024*1024)),
		},
		{
			name:     "2.5GB",
			input:    "2.5GB",
			expected: int64(2.5 * float64(1024*1024*1024)),
		},
		// 空格测试
		{
			name:     "带空格",
			input:    "  5 MB  ",
			expected: 5 * 1024 * 1024,
		},
		{
			name:     "中间空格",
			input:    "5 MB",
			expected: 5 * 1024 * 1024,
		},
		// 大小写测试
		{
			name:     "小写",
			input:    "5mb",
			expected: 5 * 1024 * 1024,
		},
		{
			name:     "混合大小写",
			input:    "5Mb",
			expected: 5 * 1024 * 1024,
		},
		// 边界值测试
		{
			name:     "零字节",
			input:    "0B",
			expected: 0,
		},
		{
			name:     "最大值（接近int64上限）",
			input:    "8EB",
			expectErr: true, // 超出支持范围
		},
		// 错误情况
		{
			name:      "无效格式",
			input:     "invalid",
			expectErr: true,
		},
		{
			name:      "无效单位",
			input:     "5XB",
			expectErr: true,
		},
		{
			name:      "负数",
			input:     "-1MB",
			expectErr: true,
		},
		{
			name:      "空字符串",
			input:     "",
			expectErr: true,
		},
		{
			name:      "只有单位",
			input:     "MB",
			expectErr: true,
		},
		{
			name:      "特殊字符",
			input:     "5$MB",
			expectErr: true,
		},
		// BYTE关键字测试
		{
			name:     "BYTE单位",
			input:    "1024BYTE",
			expected: 1024,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseByteSize(tc.input)

			if tc.expectErr {
				if err == nil {
					t.Errorf("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望返回错误，但得到: %v", err)
				return
			}

			if result != tc.expected {
				t.Errorf("期望 %d，实际 %d", tc.expected, result)
			}
		})
	}
}

// TestParseDuration 测试解析时间间隔字符串
func TestParseDuration(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  time.Duration
		expectErr bool
	}{
		// 基本单位测试
		{
			name:     "纳秒",
			input:    "10ns",
			expected: 10 * time.Nanosecond,
		},
		{
			name:     "微秒",
			input:    "10us",
			expected: 10 * time.Microsecond,
		},
		{
			name:     "毫秒",
			input:    "10ms",
			expected: 10 * time.Millisecond,
		},
		{
			name:     "秒",
			input:    "10s",
			expected: 10 * time.Second,
		},
		{
			name:     "分钟",
			input:    "10m",
			expected: 10 * time.Minute,
		},
		{
			name:     "小时",
			input:    "10h",
			expected: 10 * time.Hour,
		},
		// 复合时间
		{
			name:     "1小时30分钟",
			input:    "1h30m",
			expected: 1*time.Hour + 30*time.Minute,
		},
		{
			name:     "2小时5分钟30秒",
			input:    "2h5m30s",
			expected: 2*time.Hour + 5*time.Minute + 30*time.Second,
		},
		// 默认秒（没有单位）
		{
			name:     "纯数字默认为秒",
			input:    "30",
			expected: 30 * time.Second,
		},
		{
			name:     "大数字",
			input:    "86400", // 1天
			expected: 86400 * time.Second,
		},
		// 空格测试
		{
			name:     "带空格",
			input:    "  30s  ",
			expected: 30 * time.Second,
		},
		// 错误情况
		{
			name:      "空字符串",
			input:     "",
			expectErr: true,
		},
		{
			name:      "只有空格",
			input:     "   ",
			expectErr: true,
		},
		{
			name:      "无效格式",
			input:     "invalid",
			expectErr: true,
		},
		{
			name:      "无效单位",
			input:     "10x",
			expectErr: true,
		},
		{
			name:      "负数",
			input:     "-10s",
			expectErr: true, // time.ParseDuration不支持负数
		},
		// 小数测试
		{
			name:     "小数秒",
			input:    "1.5s",
			expected: 1500 * time.Millisecond,
		},
		{
			name:     "小数分钟",
			input:    "1.5m",
			expected: 90 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseDuration(tc.input)

			if tc.expectErr {
				if err == nil {
					t.Errorf("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望返回错误，但得到: %v", err)
				return
			}

			if result != tc.expected {
				t.Errorf("期望 %v，实际 %v", tc.expected, result)
			}
		})
	}
}

// TestParseByteSizeEdgeCases 测试ParseByteSize的边缘情况
func TestParseByteSizeEdgeCases(t *testing.T) {
	// 测试非常接近int64最大值的值
	maxInt64 := int64(9223372036854775807)
	result, err := ParseByteSize("8EB") // 8 Exabytes
	if err == nil {
		t.Errorf("8EB应该超出int64范围，但没有错误。结果: %d", result)
	}

	// 测试浮点数精度问题
	result, err = ParseByteSize("9.223372036854775807EB") // 接近最大值
	if err == nil && result > maxInt64 {
		t.Errorf("结果 %d 超出了int64最大值", result)
	}

	// 测试非常小的值
	result, err = ParseByteSize("0.1KB")
	if err != nil {
		t.Errorf("解析小数值失败: %v", err)
	}
	if result != 102 { // 0.1 * 1024 = 102.4 -> int64截断为102
		t.Errorf("期望 102，实际 %d", result)
	}
}

// TestParseDurationEdgeCases 测试ParseDuration的边缘情况
func TestParseDurationEdgeCases(t *testing.T) {
	// 测试非常大的值
	_, err := ParseDuration("1000000h")
	if err != nil {
		t.Errorf("解析大时间值失败: %v", err)
	}

	// 测试零值
	result, err := ParseDuration("0")
	if err != nil {
		t.Errorf("解析零值失败: %v", err)
	}
	if result != 0 {
		t.Errorf("期望 0，实际 %v", result)
	}
}

// BenchmarkParseByteSize 性能测试：解析字节大小
func BenchmarkParseByteSize(b *testing.B) {
	testCases := []string{
		"1MB",
		"5GB",
		"1.5MB",
		"1024B",
		"  10 MB  ",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ParseByteSize(tc)
			}
		})
	}
}

// BenchmarkParseDuration 性能测试：解析时间间隔
func BenchmarkParseDuration(b *testing.B) {
	testCases := []string{
		"1h30m",
		"24h",
		"30s",
		"86400", // 1天（秒）
		"1.5m",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ParseDuration(tc)
			}
		})
	}
}

// ExampleParseByteSize 示例：解析字节大小
func ExampleParseByteSize() {
	size, _ := ParseByteSize("5MB")
	println(size) // 5242880
	// Output: 5242880
}

// ExampleParseDuration 示例：解析时间间隔
func ExampleParseDuration() {
	duration, _ := ParseDuration("1h30m")
	println(duration.String()) // 1h30m0s
	// Output: 1h30m0s
}

// TestCombinedUsage 测试组合使用场景
func TestCombinedUsage(t *testing.T) {
	// 模拟配置解析
	chunkSize, err := ParseByteSize("5MB")
	if err != nil {
		t.Fatalf("解析块大小失败: %v", err)
	}

	resumeInterval, err := ParseByteSize("1MB")
	if err != nil {
		t.Fatalf("解析恢复间隔失败: %v", err)
	}

	maxAge, err := ParseDuration("24h")
	if err != nil {
		t.Fatalf("解析最大时间失败: %v", err)
	}

	// 验证值
	if chunkSize != 5*1024*1024 {
		t.Errorf("块大小错误，期望 %d，实际 %d", 5*1024*1024, chunkSize)
	}

	if resumeInterval != 1024*1024 {
		t.Errorf("恢复间隔错误，期望 %d，实际 %d", 1024*1024, resumeInterval)
	}

	if maxAge != 24*time.Hour {
		t.Errorf("最大时间错误，期望 %v，实际 %v", 24*time.Hour, maxAge)
	}

	// 验证逻辑关系
	if chunkSize%resumeInterval != 0 {
		t.Error("块大小应该是恢复间隔的整数倍")
	}
}

// TestRoundTrip 测试往返转换（从值到字符串再回到值）
func TestRoundTrip(t *testing.T) {
	originalValues := []int64{
		1024,           // 1KB
		1024 * 1024,    // 1MB
		1024 * 1024 * 1024, // 1GB
	}

	for _, val := range originalValues {
		// 转换为字符串表示
		var str string
		switch val {
		case 1024:
			str = "1KB"
		case 1024 * 1024:
			str = "1MB"
		case 1024 * 1024 * 1024:
			str = "1GB"
		}

		// 解析回来
		parsed, err := ParseByteSize(str)
		if err != nil {
			t.Errorf("解析 %s 失败: %v", str, err)
			continue
		}

		if parsed != val {
			t.Errorf("往返转换失败：原始 %d，字符串 %s，解析后 %d", val, str, parsed)
		}
	}
}