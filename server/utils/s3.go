package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// S3Client обёртка для AWS SDK v1.
type S3Client struct {
	svc          *s3.S3
	bucket       string
	endpoint     string
	region       string
	usePathStyle bool
	aliasHost    string
}

// NewS3Client создаёт новый S3Client, читая настройки из окружения.
func NewS3Client() (*S3Client, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	region := os.Getenv("S3_REGION")
	bucket := os.Getenv("S3_BUCKET")
	access := os.Getenv("S3_ACCESS_KEY_ID")
	secret := os.Getenv("S3_SECRET_ACCESS_KEY")
	usePathStyle := os.Getenv("S3_USE_PATH_STYLE") == "true"
	aliasHost := os.Getenv("S3_ALIAS_HOST")

	if endpoint == "" {
		return nil, fmt.Errorf("S3_ENDPOINT is not set")
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	if region == "" {
		return nil, fmt.Errorf("S3_REGION is not set")
	}
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is not set")
	}
	if access == "" || secret == "" {
		return nil, fmt.Errorf("S3_ACCESS_KEY_ID or S3_SECRET_ACCESS_KEY is not set")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(usePathStyle),
		Credentials:      credentials.NewStaticCredentials(access, secret, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	svc := s3.New(sess)
	return &S3Client{
		svc:          svc,
		bucket:       bucket,
		endpoint:     endpoint,
		region:       region,
		usePathStyle: usePathStyle,
		aliasHost:    aliasHost,
	}, nil
}

func sanitizeFilename(name string) string {
	ext := filepath.Ext(name)
	return uuid.New().String() + ext
}

// UploadFile загружает файл и возвращает публичный URL.
func (c *S3Client) UploadFile(ctx context.Context, dir, filename string, fileContent []byte) (url, sanitized string, err error) {
	sanitized = sanitizeFilename(filename)
	key := filepath.ToSlash(filepath.Join(dir, sanitized))

	contentType := "application/octet-stream"
	switch ext := strings.ToLower(filepath.Ext(sanitized)); ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileContent),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	}

	if _, err = c.svc.PutObjectWithContext(ctx, input); err != nil {
		return "", "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return "/storage/" + key, sanitized, nil
}

// GetFile скачивает объект по ключу key из S3 и возвращает его Content-Type и содержимое.
func (c *S3Client) GetFile(ctx context.Context, key string) (string, []byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	result, err := c.svc.GetObjectWithContext(ctx, input)
	if err != nil {
		return "", nil, err
	}
	defer result.Body.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, result.Body); err != nil {
		return "", nil, err
	}

	contentType := aws.StringValue(result.ContentType)
	return contentType, buf.Bytes(), nil
}
