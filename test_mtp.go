package main

import (
	"fmt"
	"github.com/allanpk716/record_center/internal/device"
	"github.com/allanpk716/record_center/internal/logger"
)

func main() {
	log := logger.NewLogger(true)
	// 测试NewMTPAccessor是否可访问
	accessor := device.NewMTPAccessor(log)
	fmt.Printf("MTPAccessor created: %v\n", accessor != nil)
}
