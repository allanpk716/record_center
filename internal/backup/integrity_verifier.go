package backup

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/allanpk716/record_center/internal/logger"
)

// IntegrityVerifier 文件完整性验证器
type IntegrityVerifier struct {
	log            *logger.Logger
	hashAlgorithm  string
}

// NewIntegrityVerifier 创建完整性验证器
func NewIntegrityVerifier(log *logger.Logger, hashAlgorithm string) *IntegrityVerifier {
	return &IntegrityVerifier{
		log:           log,
		hashAlgorithm: hashAlgorithm,
	}
}

// VerifyFileIntegrity 验证文件完整性
func (iv *IntegrityVerifier) VerifyFileIntegrity(sourcePath, targetPath, expectedHash string) (bool, string, error) {
	// 计算目标文件哈希
	actualHash, err := iv.CalculateFileHash(targetPath)
	if err != nil {
		return false, "", fmt.Errorf("计算目标文件哈希失败: %w", err)
	}

	// 如果提供了期望的哈希值，进行对比
	if expectedHash != "" {
		if actualHash != expectedHash {
			iv.log.Error("文件完整性验证失败: %s", targetPath)
			iv.log.Error("期望哈希: %s", expectedHash)
			iv.log.Error("实际哈希: %s", actualHash)
			return false, actualHash, fmt.Errorf("哈希值不匹配")
		}
		iv.log.Debug("文件完整性验证通过: %s", targetPath)
	}

	return true, actualHash, nil
}

// CalculateFileHash 计算文件哈希
func (iv *IntegrityVerifier) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	var hasher hash.Hash
	switch iv.hashAlgorithm {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	default:
		// 默认使用SHA256
		hasher = sha256.New()
		iv.log.Warn("未知的哈希算法: %s，使用默认的SHA256", iv.hashAlgorithm)
	}

	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// CopyWithVerification 带完整性验证的文件复制
func (iv *IntegrityVerifier) CopyWithVerification(src io.Reader, dst io.Writer, expectedSize int64) (int64, string, error) {
	// 创建多写入器，同时写入目标和计算哈希
	var hasher hash.Hash
	switch iv.hashAlgorithm {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	default:
		hasher = sha256.New()
	}

	// 创建多写入器
	multiWriter := io.MultiWriter(dst, hasher)

	// 复制数据，同时计算哈希
	copiedBytes, err := io.CopyN(multiWriter, src, expectedSize)
	if err != nil && err != io.EOF {
		return 0, "", fmt.Errorf("复制数据失败: %w", err)
	}

	// 获取哈希值
	hashValue := fmt.Sprintf("%x", hasher.Sum(nil))

	return copiedBytes, hashValue, nil
}