package document

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// ValidateUploadedFile validates uploaded file
func ValidateUploadedFile(header *multipart.FileHeader) error {
	if header.Size == 0 {
		return fmt.Errorf("file is empty")
	}

	if header.Size > 100*1024*1024 { // 100MB limit
		return fmt.Errorf("file size exceeds 100MB limit")
	}

	return nil
}

// CalculateFileChecksum calculates MD5 checksum
func CalculateFileChecksum(file multipart.File) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Reset file pointer
	file.Seek(0, 0)

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GenerateVersionedFileName generates versioned filename for MinIO
func GenerateVersionedFileName(baseName string, version int) string {
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	return fmt.Sprintf("%s-v%d%s", nameWithoutExt, version, ext)
}

// GenerateDisplayPath generates display path for UI
func GenerateDisplayPath(folderPath, fileName string, version int) string {
	versionedFileName := GenerateVersionedFileName(fileName, version)
	return folderPath + versionedFileName
}

// GenerateMinIOPath generates MinIO object key
func GenerateMinIOPath(folderPath, fileName string, version int) string {
	versionedFileName := GenerateVersionedFileName(fileName, version)
	return folderPath + versionedFileName
}
