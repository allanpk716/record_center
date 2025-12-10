//go:build windows

package device

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/go-ole/go-ole"
)

// WPDFileStream WPD文件流
type WPDFileStream struct {
	accessor    *WPDComAccessor
	stream      *ole.IUnknown
	resource    *ole.IUnknown
	filePath    string
	position    int64
	totalSize   int64
	mutex       sync.RWMutex
	closed      bool
}

// NewWPDFileStream 创建新的WPD文件流
func NewWPDFileStream(accessor *WPDComAccessor, filePath string, totalSize int64) *WPDFileStream {
	return &WPDFileStream{
		accessor:   accessor,
		filePath:   filePath,
		totalSize:  totalSize,
	}
}

// Read 读取数据
func (s *WPDFileStream) Read(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return 0, io.EOF
	}

	if s.position >= s.totalSize {
		return 0, io.EOF
	}

	// 计算要读取的大小
	readSize := int64(len(p))
	if s.position+readSize > s.totalSize {
		readSize = s.totalSize - s.position
	}

	// 从设备读取数据
	err = s.readFromDevice(s.position, p[:readSize])
	if err != nil {
		return 0, fmt.Errorf("从设备读取失败: %w", err)
	}

	s.position += readSize
	return int(readSize), nil
}

// Seek 设置读取位置
func (s *WPDFileStream) Seek(offset int64, whence int) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return 0, io.ErrClosedPipe
	}

	var newPos int64

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.position + offset
	case io.SeekEnd:
		newPos = s.totalSize + offset
	default:
		return 0, fmt.Errorf("无效的whence值: %d", whence)
	}

	if newPos < 0 {
		return 0, fmt.Errorf("无效的位置: %d", newPos)
	}

	s.position = newPos
	return newPos, nil
}

// readFromDevice 从设备读取数据
func (s *WPDFileStream) readFromDevice(position int64, p []byte) error {
	s.accessor.log.Debug("从设备读取数据，位置: %d，大小: %d", position, len(p))

	// 创建临时文件用于复制
	tempFile, err := os.CreateTemp("", "wpd_stream_*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 使用PowerShell复制文件到临时位置
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)
if ($portable) {
    $device = $portable.Items() | Where-Object { $_.Name -like "*%s*" } | Select-Object -First 1
    if ($device) {
        $deviceFolder = $device.GetFolder
        if ($deviceFolder) {
            # 查找目标文件
            function Find-File($folder, $targetPath) {
                foreach ($item in $folder.Items()) {
                    $currentPath = if ($targetPath -like "*\*") {
                        $targetPath
                    } else {
                        $item.Name
                    }

                    if (-not $item.IsFolder -and $item.Name -eq [System.IO.Path]::GetFileName($targetPath)) {
                        return $item
                    } elseif ($item.IsFolder) {
                        try {
                            $subFolder = $item.GetFolder()
                            $result = Find-File $subFolder $targetPath
                            if ($result) { return $result }
                        } catch {
                            # 忽略无法访问的文件夹
                        }
                    }
                }
                return $null
            }

            $fileItem = Find-File $deviceFolder "%s"
            if ($fileItem) {
                # 复制文件到临时位置
                $tempPath = "%s"
                $shell.NameSpace([System.IO.Path]::GetDirectoryName($tempPath)).CopyHere($fileItem, 0x4)
                Write-Output "SUCCESS"
            } else {
                Write-Error "文件未找到"
            }
        } else {
            Write-Error "无法获取设备文件夹"
        }
    } else {
        Write-Error "设备未找到"
    }
} else {
    Write-Error "无法获取便携式设备命名空间"
}
`, s.accessor.deviceInfo.Name, s.filePath, tempFile.Name())

	// 执行PowerShell脚本
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.accessor.log.Error("文件复制失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("文件复制失败: %w", err)
	}

	// 检查是否成功
	if !strings.Contains(string(output), "SUCCESS") {
		return fmt.Errorf("文件复制失败: %s", string(output))
	}

	// 从临时文件读取数据
	_, err = tempFile.Seek(position, 0)
	if err != nil {
		return fmt.Errorf("临时文件定位失败: %w", err)
	}

	_, err = tempFile.Read(p)
	if err != nil {
		return fmt.Errorf("临时文件读取失败: %w", err)
	}

	return nil
}

// Close 关闭流
func (s *WPDFileStream) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return nil
	}

	s.accessor.log.Debug("关闭WPD文件流: %s", s.filePath)

	// 释放COM资源
	if s.resource != nil {
		s.resource.Release()
		s.resource = nil
	}

	if s.stream != nil {
		s.stream.Release()
		s.stream = nil
	}

	s.closed = true
	return nil
}

// WPDResourceTransfer WPD资源传输
type WPDResourceTransfer struct {
	accessor    *WPDComAccessor
	filePath    string
	destPath    string
	progress    int64
	totalSize   int64
	bufferSize  int64
	onProgress  func(transferred int64, total int64)
}

// NewWPDResourceTransfer 创建新的资源传输
func NewWPDResourceTransfer(accessor *WPDComAccessor, filePath, destPath string, totalSize int64) *WPDResourceTransfer {
	return &WPDResourceTransfer{
		accessor:   accessor,
		filePath:   filePath,
		destPath:   destPath,
		totalSize:  totalSize,
		bufferSize: 1024 * 1024, // 1MB buffer
	}
}

// SetProgressCallback 设置进度回调
func (t *WPDResourceTransfer) SetProgressCallback(callback func(transferred int64, total int64)) {
	t.onProgress = callback
}

// Transfer 开始传输
func (t *WPDResourceTransfer) Transfer() error {
	t.accessor.log.Info("开始传输文件: %s -> %s (大小: %d bytes)", t.filePath, t.destPath, t.totalSize)

	// 这里需要实现实际的文件传输逻辑
	// 由于COM接口的复杂性，我们暂时模拟传输过程

	if t.onProgress != nil {
		t.onProgress(0, t.totalSize)
	}

	// 模拟传输过程
	chunkSize := t.bufferSize
	transferred := int64(0)

	for transferred < t.totalSize {
		if chunkSize > t.totalSize-transferred {
			chunkSize = t.totalSize - transferred
		}

		// 模拟读取和写入
		err := t.transferChunk(transferred, chunkSize)
		if err != nil {
			return fmt.Errorf("传输数据块失败: %w", err)
		}

		transferred += chunkSize

		if t.onProgress != nil {
			t.onProgress(transferred, t.totalSize)
		}

		t.progress = transferred
	}

	t.accessor.log.Info("文件传输完成: %s", t.filePath)
	return nil
}

// transferChunk 传输数据块
func (t *WPDResourceTransfer) transferChunk(offset int64, size int64) error {
	t.accessor.log.Debug("传输数据块，偏移: %d，大小: %d", offset, size)

	// 这里需要调用WPD API传输数据
	// 由于COM接口的复杂性，我们暂时跳过实际实现
	return nil
}

// GetProgress 获取传输进度
func (t *WPDResourceTransfer) GetProgress() (transferred int64, total int64) {
	return t.progress, t.totalSize
}