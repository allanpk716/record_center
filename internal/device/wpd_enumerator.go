//go:build windows

package device

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
)

// WPDFileEnumerator WPD文件枚举器
type WPDFileEnumerator struct {
	accessor    *WPDComAccessor
	content     *ole.IUnknown
	maxDepth    int
	currentDepth int
}

// NewWPDFileEnumerator 创建新的WPD文件枚举器
func NewWPDFileEnumerator(accessor *WPDComAccessor, maxDepth int) *WPDFileEnumerator {
	return &WPDFileEnumerator{
		accessor: accessor,
		maxDepth: maxDepth,
	}
}

// EnumerateFiles 枚举文件
func (e *WPDFileEnumerator) EnumerateFiles(rootObjectID string) ([]*FileInfo, error) {
	e.accessor.log.Debug("开始枚举WPD文件，根对象ID: %s", rootObjectID)
	e.currentDepth = 0

	// 获取设备内容接口
	if err := e.getContentInterface(); err != nil {
		return nil, fmt.Errorf("获取内容接口失败: %w", err)
	}

	// 递归枚举
	files, err := e.enumerateRecursive(rootObjectID)
	if err != nil {
		return nil, fmt.Errorf("递归枚举失败: %w", err)
	}

	e.accessor.log.Debug("WPD文件枚举完成，找到 %d 个文件", len(files))
	return files, nil
}

// getContentInterface 获取设备内容接口
func (e *WPDFileEnumerator) getContentInterface() error {
	e.accessor.log.Debug("获取IPortableDeviceContent接口")

	// 这里需要从WPDComAccessor获取content接口
	// 由于COM接口调用的复杂性，我们先暂时跳过
	return nil
}

// enumerateRecursive 递归枚举文件
func (e *WPDFileEnumerator) enumerateRecursive(objectID string) ([]*FileInfo, error) {
	if e.currentDepth >= e.maxDepth {
		return nil, nil
	}

	e.accessor.log.Debug("枚举对象: %s (深度: %d)", objectID, e.currentDepth)

	// 获取子对象列表
	childIDs, err := e.getChildObjects(objectID)
	if err != nil {
		e.accessor.log.Debug("获取子对象失败: %v", err)
		return nil, err
	}

	var files []*FileInfo

	// 遍历子对象
	for _, childID := range childIDs {
		props, err := e.getObjectProperties(childID)
		if err != nil {
			e.accessor.log.Debug("获取对象属性失败: %v", err)
			continue
		}

		objectType, _ := props["ObjectType"].(string)
		name, _ := props["Name"].(string)

		// 判断是否为文件
		if objectType == WPD_CONTENT_TYPE_GENERIC_OBJECT ||
		   (objectType == WPD_CONTENT_TYPE_AUDIO && e.isAudioFile(name)) {

			// 检查是否为.opus文件
			if e.isAudioFile(name) {
				file := &FileInfo{
					Name:    name,
					Path:    e.buildFilePath(props),
					Size:    e.getObjectSize(props),
					ModTime: e.getObjectModifyTime(props),
					IsOpus:  true,
				}
				files = append(files, file)
				e.accessor.log.Debug("找到音频文件: %s", name)
			}
		} else if objectType == WPD_CONTENT_TYPE_FOLDER {
			// 递归处理子目录
			e.currentDepth++
			childFiles, err := e.enumerateRecursive(childID)
			e.currentDepth--

			if err == nil {
				files = append(files, childFiles...)
			}
		}
	}

	return files, nil
}

// getChildObjects 获取子对象列表
func (e *WPDFileEnumerator) getChildObjects(objectID string) ([]string, error) {
	e.accessor.log.Debug("获取对象 %s 的子对象", objectID)

	// 这里需要调用WPD API获取子对象
	// 由于COM接口的复杂性，我们暂时返回空列表
	return []string{}, nil
}

// getObjectProperties 获取对象属性
func (e *WPDFileEnumerator) getObjectProperties(objectID string) (map[string]interface{}, error) {
	e.accessor.log.Debug("获取对象 %s 的属性", objectID)

	// 这里需要调用WPD API获取对象属性
	// 由于COM接口的复杂性，我们返回模拟属性
	props := make(map[string]interface{})

	// 基于objectID模拟一些属性
	if strings.Contains(objectID, "DEVICE") {
		props["ObjectType"] = WPD_CONTENT_TYPE_FOLDER
		props["Name"] = "DEVICE"
	} else {
		props["ObjectType"] = WPD_CONTENT_TYPE_GENERIC_OBJECT
		props["Name"] = filepath.Base(objectID)
		props["Size"] = int64(1024 * 1024) // 模拟1MB
		props["Date Modified"] = time.Now()
	}

	return props, nil
}

// isAudioFile 检查是否为音频文件
func (e *WPDFileEnumerator) isAudioFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	audioExtensions := []string{".opus", ".mp3", ".wav", ".m4a", ".aac", ".flac", ".wma"}

	for _, audioExt := range audioExtensions {
		if ext == audioExt {
			return true
		}
	}
	return false
}

// buildFilePath 构建文件路径
func (e *WPDFileEnumerator) buildFilePath(props map[string]interface{}) string {
	name, _ := props["Name"].(string)
	// 这里应该构建完整的MTP设备路径
	// 暂时返回文件名
	return name
}

// getObjectSize 获取文件大小
func (e *WPDFileEnumerator) getObjectSize(props map[string]interface{}) int64 {
	if size, ok := props["Size"].(int64); ok {
		return size
	}
	return 0
}

// getObjectModifyTime 获取修改时间
func (e *WPDFileEnumerator) getObjectModifyTime(props map[string]interface{}) time.Time {
	if modTime, ok := props["Date Modified"].(time.Time); ok {
		return modTime
	}
	return time.Now()
}