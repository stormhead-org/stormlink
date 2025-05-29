package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

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
