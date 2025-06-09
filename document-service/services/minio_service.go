package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	"forgecrud-backend/shared/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOService struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOService() (*MinIOService, error) {
	cfg := config.GetConfig()

	// Parse endpoint URL to get host
	parsedURL, err := url.Parse(cfg.MinIOServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid MinIO endpoint: %v", err)
	}

	endpoint := parsedURL.Host
	useSSL := cfg.MinIOUseSSL

	log.Printf("🔗 Connecting to MinIO: %s (SSL: %v)", endpoint, useSSL)

	// Initialize MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIORootUser, cfg.MinIORootPassword, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %v", err)
	}

	service := &MinIOService{
		client:     minioClient,
		bucketName: cfg.MinIOBucketName,
	}

	// Test connection and create bucket if needed
	if err := service.initializeBucket(); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *MinIOService) initializeBucket() error {
	ctx := context.Background()

	log.Printf("🪣 Checking bucket: %s", s.bucketName)

	// Check if bucket exists
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %v", err)
	}

	if !exists {
		// Create bucket
		err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %v", err)
		}
		log.Printf("✅ MinIO bucket '%s' created successfully", s.bucketName)
	} else {
		log.Printf("✅ MinIO bucket '%s' already exists", s.bucketName)
	}

	return nil
}

// Test connection
func (s *MinIOService) TestConnection() error {
	ctx := context.Background()

	// List buckets to test connection
	buckets, err := s.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to MinIO: %v", err)
	}

	log.Printf("✅ MinIO connection successful. Found %d buckets", len(buckets))
	for _, bucket := range buckets {
		log.Printf("   📁 Bucket: %s (created: %s)", bucket.Name, bucket.CreationDate)
	}

	return nil
}

// GetClient returns the MinIO client
func (s *MinIOService) GetClient() *minio.Client {
	return s.client
}

// GetBucketName returns the bucket name
func (s *MinIOService) GetBucketName() string {
	return s.bucketName
}

// CreateFolder creates a folder in MinIO bucket
func (s *MinIOService) CreateFolder(folderPath string) error {
	ctx := context.Background()

	// Clean path and ensure it ends with /
	cleanPath := strings.Trim(folderPath, "/")
	if cleanPath != "" {
		cleanPath = cleanPath + "/"
	}

	// Create empty folder marker object
	objectKey := cleanPath + ".foldermarker"
	reader := strings.NewReader("")

	_, err := s.client.PutObject(ctx, s.bucketName, objectKey, reader, 0, minio.PutObjectOptions{
		ContentType: "text/plain",
	})

	if err != nil {
		return fmt.Errorf("failed to create folder marker: %v", err)
	}

	log.Printf("✅ MinIO folder created: %s", objectKey)
	return nil
}

// DeleteFolder removes a folder and ALL its contents from MinIO bucket
func (s *MinIOService) DeleteFolder(folderPath string) error {
	ctx := context.Background()

	// Clean path
	cleanPath := strings.Trim(folderPath, "/")
	if cleanPath != "" {
		cleanPath = cleanPath + "/"
	}

	// 🔥 1. List all objects in the folder (recursive)
	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    cleanPath,
		Recursive: true,
	})

	var errors []string
	objectCount := 0

	// 🔥 2. Delete all objects in the folder
	for object := range objectCh {
		if object.Err != nil {
			errors = append(errors, fmt.Sprintf("list error: %v", object.Err))
			continue
		}

		// Delete each object
		err := s.client.RemoveObject(ctx, s.bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete %s: %v", object.Key, err))
		} else {
			objectCount++
			log.Printf("🗑️ Deleted object: %s", object.Key)
		}
	}

	// 🔥 3. Delete folder marker if exists
	folderMarker := cleanPath + ".foldermarker"
	err := s.client.RemoveObject(ctx, s.bucketName, folderMarker, minio.RemoveObjectOptions{})
	if err != nil {
		// Folder marker may not exist, don't fail
		log.Printf("Warning: Could not delete folder marker %s: %v", folderMarker, err)
	}

	// Report results
	if len(errors) > 0 {
		return fmt.Errorf("failed to delete some objects: %v", errors)
	}

	log.Printf("✅ MinIO folder deleted: %s (removed %d objects)", cleanPath, objectCount)
	return nil
}

// ListFolderContents lists all objects in a folder
func (s *MinIOService) ListFolderContents(folderPath string) ([]string, error) {
	ctx := context.Background()

	prefix := strings.Trim(folderPath, "/")
	if prefix != "" {
		prefix = prefix + "/"
	}

	var objects []string
	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix: prefix,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

// FolderExists checks if folder exists in MinIO
func (s *MinIOService) FolderExists(folderPath string) (bool, error) {
	ctx := context.Background()

	cleanPath := strings.Trim(folderPath, "/")
	if cleanPath != "" {
		cleanPath = cleanPath + "/"
	}

	objectKey := cleanPath + ".foldermarker"

	_, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		// If object not found, folder doesn't exist
		return false, nil
	}

	return true, nil
}

// UploadFile uploads a file to the specified folder in the bucket
func (s *MinIOService) UploadFile(ctx context.Context, file io.Reader, fileName, folderName string, fileSize int64) error {
	log.Printf("⬆️ Uploading file to: %s/%s (size: %d bytes)", s.bucketName, fileName, fileSize)

	// Ensure the folder name ends with a slash
	if !strings.HasSuffix(folderName, "/") {
		folderName += "/"
	}

	// Upload the file
	_, err := s.client.PutObject(ctx, s.bucketName, folderName+fileName, file, fileSize, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	log.Printf("✅ File '%s' uploaded successfully", fileName)
	return nil
}

// DownloadFile downloads a file from the bucket
func (s *MinIOService) DownloadFile(ctx context.Context, fileName, folderName string) (io.ReadCloser, error) {
	log.Printf("⬇️ Downloading file: %s/%s", s.bucketName, fileName)

	// Ensure the folder name ends with a slash
	if !strings.HasSuffix(folderName, "/") {
		folderName += "/"
	}

	// Download the file
	object, err := s.client.GetObject(ctx, s.bucketName, folderName+fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}

	log.Printf("✅ File '%s' downloaded successfully", fileName)
	return object, nil
}

// RemoveFile removes a file from the bucket
func (s *MinIOService) RemoveFile(ctx context.Context, fileName, folderName string) error {
	log.Printf("🗑️ Removing file: %s/%s", s.bucketName, fileName)

	// Ensure the folder name ends with a slash
	if !strings.HasSuffix(folderName, "/") {
		folderName += "/"
	}

	// Remove the file
	err := s.client.RemoveObject(ctx, s.bucketName, folderName+fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove file: %v", err)
	}

	log.Printf("✅ File '%s' removed successfully", fileName)
	return nil
}

// MoveObject moves an object from one location to another
func (m *MinIOService) MoveObject(sourceKey, destKey string) error {
	// Copy object to new location
	src := minio.CopySrcOptions{
		Bucket: m.bucketName,
		Object: sourceKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: m.bucketName,
		Object: destKey,
	}

	_, err := m.client.CopyObject(context.Background(), dst, src)
	if err != nil {
		return err
	}

	// Remove original object
	return m.client.RemoveObject(context.Background(), m.bucketName, sourceKey, minio.RemoveObjectOptions{})
}

// MoveFolder moves all objects from old folder path to new folder path in MinIO
func (m *MinIOService) MoveFolder(oldPath, newPath string) error {
	// Clean paths
	oldPath = strings.Trim(oldPath, "/")
	newPath = strings.Trim(newPath, "/")

	if oldPath == "" || newPath == "" {
		return fmt.Errorf("invalid folder paths")
	}

	// Add trailing slash to ensure we're working with folders
	oldPrefix := oldPath + "/"
	newPrefix := newPath + "/"

	ctx := context.Background()

	// List all objects with the old prefix
	objectCh := m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    oldPrefix,
		Recursive: true,
	})

	// Move each object
	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("failed to list objects: %v", object.Err)
		}

		// Skip if object key doesn't have the expected prefix
		if !strings.HasPrefix(object.Key, oldPrefix) {
			continue
		}

		// Calculate new object key
		relativePath := strings.TrimPrefix(object.Key, oldPrefix)
		newObjectKey := newPrefix + relativePath

		// Copy object to new location
		src := minio.CopySrcOptions{
			Bucket: m.bucketName,
			Object: object.Key,
		}

		dst := minio.CopyDestOptions{
			Bucket: m.bucketName,
			Object: newObjectKey,
		}

		_, err := m.client.CopyObject(ctx, dst, src)
		if err != nil {
			return fmt.Errorf("failed to copy object %s to %s: %v", object.Key, newObjectKey, err)
		}

		// Remove original object
		err = m.client.RemoveObject(ctx, m.bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to remove original object %s: %v", object.Key, err)
		}
	}

	return nil
}

// CopyObject copies an object from source to destination
func (s *MinIOService) CopyObject(sourceKey, destKey string) error {
	ctx := context.Background()

	// Copy object
	_, err := s.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: s.bucketName,
		Object: destKey,
	}, minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: sourceKey,
	})

	if err != nil {
		return fmt.Errorf("failed to copy object: %v", err)
	}

	log.Printf("✅ Object copied: %s -> %s", sourceKey, destKey)
	return nil
}
