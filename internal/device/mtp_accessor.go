package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/allanpk716/record_center/internal/logger"
	"golang.org/x/sys/windows"
)

// Windows API 常量
const (
	CSIDL_DESKTOP            = 0x0000
	CSIDL_PERSONAL           = 0x0005
	SHGFI_PIDL              = 0x000000008
	SHGFI_DISPLAYNAME       = 0x000000200
	SFGAO_FOLDER            = 0x20000000
	SFGAO_FILESYSTEM        = 0x40000000

	// Shell 特殊文件夹
	DESKTOP = "::{20D04FE0-3AEA-1069-A2D8-08002B30309D}" // 此电脑
)

// MTPAccessor MTP设备访问器
type MTPAccessor struct {
	log           *logger.Logger
	retryManager *MTPRetryManager
	bridge        *DeviceBridgeImpl
	config        *ConnectionConfig
}

// NewMTPAccessor 创建MTP访问器
func NewMTPAccessor(log *logger.Logger) *MTPAccessor {
	retryManager := NewMTPRetryManager(log, 3) // 最多重试3次
	config := DefaultConnectionConfig()
	bridge := NewDeviceBridge(log, config)

	return &MTPAccessor{
		log:           log,
		retryManager: retryManager,
		bridge:        bridge,
		config:        config,
	}
}

// GetMTPDevicePath 获取MTP设备的Shell路径
func (ma *MTPAccessor) GetMTPDevicePath(deviceName string) (string, error) {
	ma.log.Debug("查找MTP设备: %s", deviceName)

	// 使用设备桥接器获取设备路径
	// 首先尝试检测设备以获取VID/PID
	devices, err := ma.bridge.ListAvailableDevices()
	if err != nil {
		ma.log.Warn("设备列表获取失败，使用传统方法: %v", err)
		return ma.getTraditionalDevicePath(deviceName)
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
		ma.log.Warn("未找到设备，使用传统方法: %s", deviceName)
		return ma.getTraditionalDevicePath(deviceName)
	}

	// 使用桥接器获取设备路径
	devicePath, err := ma.bridge.GetDevicePath(targetDevice.Name, targetDevice.VID, targetDevice.PID)
	if err != nil {
		ma.log.Warn("桥接器路径获取失败，使用传统方法: %v", err)
		return ma.getTraditionalDevicePath(deviceName)
	}

	ma.log.Info("通过设备桥接器找到路径: %s", devicePath)
	return devicePath, nil
}

// getTraditionalDevicePath 传统设备路径获取方法（降级方案）
func (ma *MTPAccessor) getTraditionalDevicePath(deviceName string) (string, error) {
	// 首先尝试检查设备是否作为驱动器挂载
	drives := ma.getAvailableDrives()
	for _, drive := range drives {
		volumeName, err := ma.getVolumeName(drive)
		if err == nil && strings.Contains(volumeName, deviceName) {
			ma.log.Info("找到挂载的驱动器: %s (%s)", drive, volumeName)
			return drive, nil
		}
	}

	// 如果没有找到挂载的驱动器，尝试通过WPD外壳命名空间
	// Windows 10/11 中，MTP设备通常可以通过特殊路径访问
	mtpPath := filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Microsoft", "Windows", "Explorer", "shell", deviceName)
	if _, err := os.Stat(mtpPath); err == nil {
		ma.log.Info("找到MTP设备路径: %s", mtpPath)
		return mtpPath, nil
	}

	return "", fmt.Errorf("MTP设备 %s 未找到或无法访问", deviceName)
}

// getAvailableDrives 获取所有可用的驱动器
func (ma *MTPAccessor) getAvailableDrives() []string {
	var drives []string

	// 获取所有逻辑驱动器
	buffer := make([]uint16, 256)
	length, err := windows.GetLogicalDriveStrings(uint32(len(buffer)), &buffer[0])
	if err != nil {
		ma.log.Warn("获取驱动器列表失败: %v", err)
		return drives
	}

	driveStr := windows.UTF16ToString(buffer[:length])
	for _, drive := range strings.Split(driveStr, "\x00") {
		if drive != "" {
			drives = append(drives, drive)
		}
	}

	return drives
}

// getVolumeName 获取驱动器的卷标
func (ma *MTPAccessor) getVolumeName(drive string) (string, error) {
	drivePath, err := windows.UTF16PtrFromString(drive)
	if err != nil {
		return "", err
	}

	buffer := make([]uint16, 256)
	err = windows.GetVolumeInformation(drivePath, nil, 0, nil, nil, nil, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(buffer), nil
}

// IsMTPPathAccessible 检查MTP路径是否可访问
func (ma *MTPAccessor) IsMTPPathAccessible(path string) bool {
	// 尝试列出目录内容
	file, err := os.Open(path)
	if err != nil {
		ma.log.Debug("无法访问路径 %s: %v", path, err)
		return false
	}
	defer file.Close()

	_, err = file.Readdirnames(1)
	return err == nil
}

// ScanMTPDevice 扫描MTP设备中的文件
func (ma *MTPAccessor) ScanMTPDevice(devicePath, basePath string) ([]*FileInfo, error) {
	ma.log.Info("开始扫描MTP设备: %s\\%s", devicePath, basePath)

	// 首先尝试使用重试机制
	deviceName := filepath.Base(devicePath)
	retryFiles, retryErr := ma.retryManager.ScanWithRetry(ma, deviceName, basePath)
	if retryErr == nil && retryFiles != nil {
		ma.log.Info("通过重试机制成功扫描到 %d 个文件", len(retryFiles))
		return retryFiles, nil
	}

	ma.log.Warn("重试机制失败，尝试直接PowerShell访问: %v", retryErr)

	// 回退到原始PowerShell访问
	if psAccessor := NewPowerShellMTPAccessor(ma.log); psAccessor != nil {
		if powerShellFiles, powerShellErr := ma.scanWithPowerShell(devicePath, basePath, psAccessor); powerShellErr == nil {
			ma.log.Info("直接PowerShell访问成功，扫描到 %d 个文件", len(powerShellFiles))
			return powerShellFiles, nil
		} else {
			ma.log.Warn("直接PowerShell访问也失败: %v", powerShellErr)
		}
	}

	fullPath := filepath.Join(devicePath, basePath)

	// 检查路径是否可访问
	if !ma.IsMTPPathAccessible(fullPath) {
		// 如果直接路径不可访问，尝试其他可能的路径
		alternativePaths := []string{
			basePath, // 直接使用基础路径
			filepath.Join(devicePath, strings.ReplaceAll(basePath, `\`, `/`)), // 使用正斜杠
		}

		for _, altPath := range alternativePaths {
			if ma.IsMTPPathAccessible(altPath) {
				fullPath = altPath
				ma.log.Info("使用替代路径: %s", fullPath)
				break
			}
		}

		// 如果仍然不可访问，返回错误提示
		if !ma.IsMTPPathAccessible(fullPath) {
			ma.log.Error("MTP设备路径不可访问，无法获取文件信息")
			ma.log.Error("请检查：")
			ma.log.Error("  1. 录音笔是否正确连接到电脑")
			ma.log.Error("  2. 设备驱动是否正常安装")
			ma.log.Error("  3. 设备是否处于可访问状态")
			ma.log.Error("  4. PowerShell执行策略是否允许脚本运行")
			return nil, fmt.Errorf("MTP设备路径不可访问: %s", fullPath)
		}
	}

	var files []*FileInfo
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ma.log.Warn("访问文件失败: %s, %v", path, err)
			return nil // 跳过错误继续处理
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理.opus文件
		if strings.ToLower(filepath.Ext(info.Name())) == ".opus" {
			relativePath, err := filepath.Rel(fullPath, path)
			if err != nil {
				relativePath = path
			}

			fileInfo := &FileInfo{
				Path:         path,
				RelativePath: relativePath,
				Name:         info.Name(),
				Size:         info.Size(),
				ModTime:      info.ModTime(),
			}

			files = append(files, fileInfo)
			ma.log.Debug("发现文件: %s (%.2f MB)", relativePath, float64(info.Size())/1024/1024)
		}

		return nil
	})

	if err != nil {
		ma.log.Error("扫描MTP设备时出错: %v", err)
		ma.log.Error("扫描失败，无法获取文件列表")
		ma.log.Error("可能的原因：")
		ma.log.Error("  - MTP设备连接中断")
		ma.log.Error("  - 设备权限不足")
		ma.log.Error("  - 设备正在被其他程序访问")
		ma.log.Error("  - PowerShell访问权限问题")
		return nil, fmt.Errorf("扫描MTP设备失败: %w", err)
	}

	ma.log.Info("扫描完成，发现 %d 个.opus文件", len(files))
	return files, nil
}

// CopyFromMTPDevice 从MTP设备复制文件
func (ma *MTPAccessor) CopyFromMTPDevice(sourcePath, targetPath string) error {
	ma.log.Debug("复制文件: %s -> %s", sourcePath, targetPath)

	// 尝试直接复制
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("无法打开源文件: %w", err)
	}
	defer sourceFile.Close()

	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 创建目标文件
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("无法创建目标文件: %w", err)
	}
	defer targetFile.Close()

	// 复制文件内容
	_, err = targetFile.ReadFrom(sourceFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	ma.log.Debug("文件复制完成: %s", targetPath)
	return nil
}

// Close 关闭MTP访问器并释放资源
func (ma *MTPAccessor) Close() error {
	if ma.bridge != nil {
		return ma.bridge.Close()
	}
	return nil
}

// scanWithPowerShell 使用PowerShell扫描文件
func (ma *MTPAccessor) scanWithPowerShell(devicePath, basePath string, psAccessor *PowerShellMTPAccessor) ([]*FileInfo, error) {
	// 使用PowerShell列出文件
	mtpFiles, err := psAccessor.ListMTPFiles(devicePath, basePath)
	if err != nil {
		return nil, fmt.Errorf("PowerShell扫描失败: %w", err)
	}

	// 转换为FileInfo格式
	var files []*FileInfo
	for _, mtpFile := range mtpFiles {
		// 跳过目录
		if mtpFile.IsDir {
			continue
		}

		// 只处理.opus文件
		if strings.ToLower(filepath.Ext(mtpFile.Name)) != ".opus" {
			continue
		}

		fileInfo := &FileInfo{
			Path:         mtpFile.Path,
			RelativePath: strings.TrimPrefix(mtpFile.RelativePath, basePath+"\\"),
			Name:         mtpFile.Name,
			Size:         mtpFile.Size,
			IsOpus:       true,
			ModTime:      mtpFile.ModTime,
		}

		files = append(files, fileInfo)
		ma.log.Debug("发现文件: %s (%.2f MB)", fileInfo.RelativePath, float64(fileInfo.Size)/1024/1024)
	}

	ma.log.Info("PowerShell扫描完成，发现 %d 个.opus文件", len(files))
	return files, nil
}

// FileInfo MTP设备文件信息
type FileInfo struct {
	Path         string
	RelativePath string
	Name         string
	Size         int64
	IsOpus       bool
	ModTime      interface{} // 可以是time.Time或其他类型
}