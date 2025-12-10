package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseByteSize 解析字节大小字符串（如 "5MB", "1.5GB"）
func ParseByteSize(sizeStr string) (int64, error) {
	// 匹配数字和单位
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]?B?)$`)
	matches := re.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(sizeStr)))
	if len(matches) != 3 {
		return 0, errors.New("invalid size format")
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	unit := matches[2]
	var multiplier int64 = 1

	switch unit {
	case "B", "BYTE":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return int64(value * float64(multiplier)), nil
}

// ParseDuration 解析时间间隔字符串（如 "24h", "30m"）
func ParseDuration(durationStr string) (time.Duration, error) {
	// time.ParseDuration 已经支持大部分格式
	// 这里只是做个包装，处理一些特殊情况
	durationStr = strings.TrimSpace(durationStr)
	if durationStr == "" {
		return 0, errors.New("empty duration")
	}

	// 如果没有单位，默认为秒
	if regexp.MustCompile(`^\d+$`).MatchString(durationStr) {
		durationStr += "s"
	}

	return time.ParseDuration(durationStr)
}