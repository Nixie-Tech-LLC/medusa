package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rs/zerolog/log"
)

type Storage interface {
	SaveFile(fileHeader *multipart.FileHeader, filename string) (string, error)
}

type LocalStorage struct {
	uploadDir string
}

type SpacesStorage struct {
	client   *s3.S3
	bucket   string
	cdnURL   string
	endpoint string
}

func NewLocalStorage(uploadDir string) *LocalStorage {
	return &LocalStorage{uploadDir: uploadDir}
}

func NewSpacesStorage(endpoint, region, bucket, cdnURL, accessKey, secretKey string) (*SpacesStorage, error) {
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(false),
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SpacesStorage{
		client:   s3.New(sess),
		bucket:   bucket,
		cdnURL:   cdnURL,
		endpoint: endpoint,
	}, nil
}

// normalizeFilename creates a unique, normalized filename without spaces
func normalizeFilename(originalFilename string) string {
	// Get file extension
	ext := filepath.Ext(originalFilename)
	baseName := strings.TrimSuffix(originalFilename, ext)

	// Remove or replace problematic characters
	// Replace spaces with underscores
	baseName = strings.ReplaceAll(baseName, " ", "_")

	// Remove or replace other problematic characters, keeping only alphanumeric, dash, underscore
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	baseName = reg.ReplaceAllString(baseName, "")

	// Ensure the filename isn't empty after cleaning
	if baseName == "" {
		baseName = "file"
	}

	// Add timestamp to make it unique and traceable
	timestamp := time.Now().Format("20060102_150405")

	// Construct final filename: basename_timestamp.ext
	return fmt.Sprintf("%s_%s%s", baseName, timestamp, ext)
}

func (ls *LocalStorage) SaveFile(fileHeader *multipart.FileHeader, filename string) (string, error) {
	normalizedFilename := normalizeFilename(filename)
	log.Debug().Str("original", filename).Str("normalized", normalizedFilename).Msg("File upload normalized")
	uploadPath := filepath.Join(ls.uploadDir, normalizedFilename)

	// Ensure upload directory exists
	if err := os.MkdirAll(ls.uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			return
		}
	}(src)

	dst, err := os.Create(uploadPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			return
		}
	}(dst)

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return uploadPath, nil
}

func (ss *SpacesStorage) SaveFile(fileHeader *multipart.FileHeader, filename string) (string, error) {
	normalizedFilename := normalizeFilename(filename)
	log.Debug().Str("original", filename).Str("normalized", normalizedFilename).Msg("File upload normalized")

	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			return
		}
	}(src)

	key := fmt.Sprintf("uploads/%s", normalizedFilename)

	// Determine content type based on file extension
	contentType := getContentType(normalizedFilename)

	_, err = ss.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(ss.bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file to Spaces")
		return "", fmt.Errorf("failed to upload to Spaces: %w", err)
	}

	// Return the CDN URL
	cdnURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(ss.cdnURL, "/"), key)
	return cdnURL, nil
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
