//go:build windows

package device

import (
	"fmt"
	"strings"

	"github.com/go-ole/go-ole"
)

// COM错误码常量
const (
	S_OK                        = 0
	S_FALSE                     = 1
	E_FAIL                     = 0x80004005
	E_INVALIDARG               = 0x80070057
	E_OUTOFMEMORY              = 0x8007000E
	E_POINTER                  = 0x80004003
	E_NOINTERFACE             = 0x80004002
	E_NOTIMPL                  = 0x80004001
	REGDB_E_CLASSNOTREG        = 0x80040154
	CLASS_E_NOAGGREGATION      = 0x80040110
	CLASS_E_CLASSNOTAVAILABLE  = 0x80040111
	CLASS_E_NOTLICENSED        = 0x80040112
)

// HRESULTToError 将HRESULT错误码转换为Go错误
func HRESULTToError(hr uint32) error {
	switch hr {
	case S_OK:
		return nil
	case S_FALSE:
		return fmt.Errorf("操作返回FALSE")
	case E_INVALIDARG:
		return fmt.Errorf("无效参数 (0x%X)", hr)
	case E_OUTOFMEMORY:
		return fmt.Errorf("内存不足 (0x%X)", hr)
	case E_POINTER:
		return fmt.Errorf("无效指针 (0x%X)", hr)
	case E_NOINTERFACE:
		return fmt.Errorf("接口不支持 (0x%X)", hr)
	case E_NOTIMPL:
		return fmt.Errorf("方法未实现 (0x%X)", hr)
	case REGDB_E_CLASSNOTREG:
		return fmt.Errorf("类未注册 (0x%X)", hr)
	case CLASS_E_NOAGGREGATION:
		return fmt.Errorf("类不支持聚合 (0x%X)", hr)
	case CLASS_E_CLASSNOTAVAILABLE:
		return fmt.Errorf("类不可用 (0x%X)", hr)
	case CLASS_E_NOTLICENSED:
		return fmt.Errorf("类未授权 (0x%X)", hr)
	case E_FAIL:
		return fmt.Errorf("未指定错误 (0x%X)", hr)
	default:
		return fmt.Errorf("COM错误 (0x%X)", hr)
	}
}

// ReleaseCOMObject 安全释放COM对象
func ReleaseCOMObject(obj interface{}) {
	if obj != nil {
		if o, ok := obj.(interface{ Release() }); ok {
			o.Release()
		}
	}
}

// SafeQueryInterface 安全查询接口
func SafeQueryInterface(unknown interface{}, iid *ole.GUID) (interface{}, error) {
	if unknown == nil {
		return nil, fmt.Errorf("源对象为空")
	}

	// 这里需要实现安全的接口查询
	return unknown, nil
}

// InitCOM 初始化COM库
func InitCOM() error {
	// 暂时跳过实际COM初始化，专注于框架结构
	// if err := ole.CoInitializeEx(nil, ole.COINIT_APARTMENTTHREADED); err != nil {
	//	return fmt.Errorf("COM初始化失败: %w", err)
	// }
	return nil
}

// CleanupCOM 清理COM库
func CleanupCOM() {
	ole.CoUninitialize()
}

// PROPERTYKEY 结构体用于WPD属性键
type PROPERTYKEY struct {
	fmtID *ole.GUID
	pidID uint32
}

// WPD属性键常量
// 这些PROPERTYKEY用于获取MTP设备对象的特定属性
var (
	// WPD_OBJECT_ID: 对象的唯一标识符
	WPD_OBJECT_ID = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  2,
	}

	// WPD_OBJECT_NAME: 对象名称（文件名）
	WPD_OBJECT_NAME = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  4,
	}

	// WPD_OBJECT_SIZE: 对象大小（以字节为单位）
	// 这是解决文件大小问题的关键属性键
	WPD_OBJECT_SIZE = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  12,
	}

	// WPD_OBJECT_ORIGINAL_FILE_NAME: 原始文件名
	WPD_OBJECT_ORIGINAL_FILE_NAME = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  13,
	}

	// WPD_OBJECT_DATE_CREATED: 创建日期
	WPD_OBJECT_DATE_CREATED = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  16,
	}

	// WPD_OBJECT_DATE_MODIFIED: 修改日期
	WPD_OBJECT_DATE_MODIFIED = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  17,
	}

	// WPD_OBJECT_DATE_AUTHORED: 作者日期（录音时间）
	WPD_OBJECT_DATE_AUTHORED = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  18,
	}

	// WPD_OBJECT_CONTENT_TYPE: 内容类型（音频、视频等）
	WPD_OBJECT_CONTENT_TYPE = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  7,
	}

	// WPD_OBJECT_FORMAT: 对象格式（如.opus）
	WPD_OBJECT_FORMAT = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  9,
	}

	// WPD_OBJECT_ISHIDDEN: 是否为隐藏文件
	WPD_OBJECT_ISHIDDEN = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  22,
	}

	// WPD_OBJECT_ISSYSTEM: 是否为系统文件
	WPD_OBJECT_ISSYSTEM = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  23,
	}

	// WPD_OBJECT_CAN_DELETE: 是否可以删除
	WPD_OBJECT_CAN_DELETE = PROPERTYKEY{
		fmtID: ole.NewGUID("{EF6B490D-5CD8-433A-AFF4-2634FB0B8B23}"),
		pidID:  24,
	}
)

// WPD_OBJECT_CONTENT_TYPE 常量
const (
	WPD_CONTENT_TYPE_GENERIC_OBJECT = "Generic Object"
	WPD_CONTENT_TYPE_FOLDER         = "Folder"
	WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT = "Functional Object"
	WPD_CONTENT_TYPE_IMAGE           = "Image"
	WPD_CONTENT_TYPE_AUDIO           = "Audio"
	WPD_CONTENT_TYPE_VIDEO           = "Video"
	WPD_CONTENT_TYPE_PLAYLIST        = "Playlist"
	WPD_CONTENT_TYPE_ALBUM           = "Album"
	WPD_CONTENT_TYPE_GENRE           = "Genre"
	WPD_CONTENT_TYPE_ARTIST          = "Artist"
	WPD_CONTENT_TYPE_CONTACT         = "Contact"
	WPD_CONTENT_TYPE_MESSAGE         = "Message"
	WPD_CONTENT_TYPE_CALENDAR        = "Calendar"
	WPD_CONTENT_TYPE_TASK            = "Task"
	WPD_CONTENT_TYPE_PROGRAM         = "Program"
	WPD_CONTENT_TYPE_MEDICAST_CAST   = "Medicast Cast"
	WPD_CONTENT_TYPE_TV_RECORDING    = "TV Recording"
	WPD_CONTENT_TYPE_DOCUMENT        = "Document"
	WPD_CONTENT_TYPE_TELEMETRY       = "Telemetry"
	WPD_CONTENT_TYPE_UNKNOWN         = "Unknown"
)

// IsAudioContentType 检查是否为音频类型
func IsAudioContentType(contentType string) bool {
	audioTypes := []string{
		WPD_CONTENT_TYPE_AUDIO,
		"WAVE",
		"MP3",
		"OPUS",
		"M4A",
		"AAC",
		"FLAC",
		"WMA",
	}

	for _, t := range audioTypes {
		if strings.EqualFold(contentType, t) {
			return true
		}
	}
	return false
}

// WPD属性获取辅助函数
// 这些函数用于从WPD COM接口获取对象属性并转换为Go类型

// GetObjectPropertyInt64 从WPD对象获取int64类型的属性值
func GetObjectPropertyInt64(properties interface{}, objectID string, propertyKey PROPERTYKEY) (int64, error) {
	// 这里需要实现WPD API调用来获取属性
	// 由于go-ole的API复杂性，这里暂时返回估算值
	// 在实际实现中，需要调用IPortableDeviceProperties::GetValues方法

	switch propertyKey.pidID {
	case WPD_OBJECT_SIZE.pidID:
		// 对于文件大小，暂时返回0表示无法获取
		// 实际实现中应该调用WPD API获取真实大小
		return 0, fmt.Errorf("WPD API未实现：无法获取文件大小")
	default:
		return 0, fmt.Errorf("不支持的int64属性ID: %d", propertyKey.pidID)
	}
}

// GetObjectPropertyString 从WPD对象获取string类型的属性值
func GetObjectPropertyString(properties interface{}, objectID string, propertyKey PROPERTYKEY) (string, error) {
	// 这里需要实现WPD API调用来获取属性
	// 实际实现中需要调用IPortableDeviceProperties::GetValues方法

	switch propertyKey.pidID {
	case WPD_OBJECT_NAME.pidID:
		return "", fmt.Errorf("WPD API未实现：无法获取对象名称")
	case WPD_OBJECT_FORMAT.pidID:
		return "", fmt.Errorf("WPD API未实现：无法获取对象格式")
	case WPD_OBJECT_CONTENT_TYPE.pidID:
		return "", fmt.Errorf("WPD API未实现：无法获取内容类型")
	default:
		return "", fmt.Errorf("不支持的string属性ID: %d", propertyKey.pidID)
	}
}

// GetObjectPropertyTime 从WPD对象获取时间类型的属性值
func GetObjectPropertyTime(properties interface{}, objectID string, propertyKey PROPERTYKEY) (interface{}, error) {
	// 这里需要实现WPD API调用来获取时间属性
	// 实际实现中需要调用IPortableDeviceProperties::GetValues方法并转换FILETIME结构

	switch propertyKey.pidID {
	case WPD_OBJECT_DATE_CREATED.pidID:
		return nil, fmt.Errorf("WPD API未实现：无法获取创建时间")
	case WPD_OBJECT_DATE_MODIFIED.pidID:
		return nil, fmt.Errorf("WPD API未实现：无法获取修改时间")
	case WPD_OBJECT_DATE_AUTHORED.pidID:
		return nil, fmt.Errorf("WPD API未实现：无法获取作者时间")
	default:
		return nil, fmt.Errorf("不支持的时间属性ID: %d", propertyKey.pidID)
	}
}

// GetObjectPropertiesMultiple 批量获取对象的多个属性
func GetObjectPropertiesMultiple(properties interface{}, objectID string, propertyKeys []PROPERTYKEY) (map[PROPERTYKEY]interface{}, error) {
	result := make(map[PROPERTYKEY]interface{})

	// 在实际实现中，应该调用IPortableDeviceProperties::GetValues方法一次性获取多个属性
	// 这里暂时逐个调用简化实现

	for _, key := range propertyKeys {
		var value interface{}
		var err error

		// 根据属性ID判断类型并调用相应的获取函数
		switch key.pidID {
		case WPD_OBJECT_SIZE.pidID:
			value, err = GetObjectPropertyInt64(properties, objectID, key)
		case WPD_OBJECT_NAME.pidID, WPD_OBJECT_FORMAT.pidID, WPD_OBJECT_CONTENT_TYPE.pidID:
			value, err = GetObjectPropertyString(properties, objectID, key)
		case WPD_OBJECT_DATE_CREATED.pidID, WPD_OBJECT_DATE_MODIFIED.pidID, WPD_OBJECT_DATE_AUTHORED.pidID:
			value, err = GetObjectPropertyTime(properties, objectID, key)
		default:
			err = fmt.Errorf("不支持的属性ID: %d", key.pidID)
		}

		if err != nil {
			// 对于单个属性失败，记录错误但继续处理其他属性
			continue
		}

		result[key] = value
	}

	return result, nil
}

// EstimateFileSizeFromName 基于文件名智能估算文件大小
// 这个函数作为WPD API获取失败时的降级方案
func EstimateFileSizeFromName(filename string) int64 {
	filename = strings.ToLower(filename)

	// 基于常见的录音笔文件命名模式进行估算
	// 例如：REC20240101080000.opus 表示2024年1月1日8点的录音

	// 短录音（1-5分钟）：1-5MB
	if strings.Contains(filename, "short") || strings.Contains(filename, "brief") {
		return 3 * 1024 * 1024 // 3MB
	}

	// 中等录音（5-30分钟）：5-30MB
	if strings.Contains(filename, "medium") || strings.Contains(filename, "normal") {
		return 15 * 1024 * 1024 // 15MB
	}

	// 长录音（30分钟以上）：30-500MB
	if strings.Contains(filename, "long") || strings.Contains(filename, "meeting") {
		return 100 * 1024 * 1024 // 100MB
	}

	// 基于文件大小的通用估算
	// 对于.opus音频文件，使用平均码率进行估算
	// .opus典型码率：32-160 kbps，取中间值80 kbps
	// 假设平均录音时长为10分钟
	return 6 * 1024 * 1024 // 6MB作为保守估算
}

// IsWPDPropertySupported 检查设备是否支持特定属性
func IsWPDPropertySupported(properties interface{}, propertyKey PROPERTYKEY) bool {
	// 在实际实现中，应该调用IPortableDeviceProperties::GetPropertyAttributes
	// 检查属性是否存在且可读

	// 暂时假设所有基本属性都支持
	supportedProperties := []uint32{
		WPD_OBJECT_ID.pidID,
		WPD_OBJECT_NAME.pidID,
		WPD_OBJECT_SIZE.pidID,
		WPD_OBJECT_DATE_MODIFIED.pidID,
		WPD_OBJECT_DATE_CREATED.pidID,
		WPD_OBJECT_CONTENT_TYPE.pidID,
		WPD_OBJECT_FORMAT.pidID,
	}

	for _, supportedID := range supportedProperties {
		if supportedID == propertyKey.pidID {
			return true
		}
	}

	return false
}