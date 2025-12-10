package utils

// CalculateTotalSize 计算文件列表的总大小
func CalculateTotalSize(files []*FileInfo) int64 {
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}
	return totalSize
}