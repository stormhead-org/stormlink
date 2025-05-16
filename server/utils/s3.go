package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Client struct {
    client   *s3.Client
    bucket   string
    endpoint string
}

func NewS3Client() (*S3Client, error) {
    // Читаем ENV
    raw := os.Getenv("S3_ENDPOINT")     // https://media.onestorage.ru или media.onestorage.ru
    var endpoint string
    secure := strings.HasPrefix(raw, "https://")
    if secure {
        endpoint = raw
    } else if strings.HasPrefix(raw, "http://") {
        endpoint = "http://" + strings.TrimPrefix(raw, "http://")
    } else {
        endpoint = "https://" + raw
    }

    region := os.Getenv("S3_REGION")    // например ru-1
    bucket := os.Getenv("S3_BUCKET")
    access := os.Getenv("S3_ACCESS_KEY_ID")
    secret := os.Getenv("S3_SECRET_ACCESS_KEY")

    // Загружаем базовый aws.Config, привязываем endpoint
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion(region),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(access, secret, ""),
        ),
        config.WithEndpointResolverWithOptions(
            aws.EndpointResolverWithOptionsFunc(func(service, rgn string, opts ...interface{}) (aws.Endpoint, error) {
                return aws.Endpoint{
                    URL:               endpoint,
                    SigningRegion:     region,
                    HostnameImmutable: true,
                }, nil
            }),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("load AWS config: %w", err)
    }

    // Создаём S3-клиент в path-style и без compute-checksums
    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle           = true
        o.DisableComputeChecksums = true
    })

    return &S3Client{client: client, bucket: bucket, endpoint: endpoint}, nil
}

func (c *S3Client) UploadFile(ctx context.Context, dir, filename string, fileContent []byte) (string, error) {
    // формируем ключ: avatar/myface.png
    key := filepath.Join(dir, filename)

    // определяем Content-Type по расширению
    contentType := "application/octet-stream"
    switch strings.ToLower(filepath.Ext(filename)) {
    case ".jpg", ".jpeg":
        contentType = "image/jpeg"
    case ".png":
        contentType = "image/png"
    case ".gif":
        contentType = "image/gif"
    }

    _, err := c.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(c.bucket),
        Key:         aws.String(key),
        Body:        bytes.NewReader(fileContent),
        ACL:         types.ObjectCannedACLPublicRead,
        ContentType: aws.String(contentType),
    })
    if err != nil {
        return "", fmt.Errorf("upload to S3: %w", err)
    }

    // Возвращаем публичную ссылку, например:
    // https://media.onestorage.ru/avatar/face.png
    return fmt.Sprintf("%s/%s", c.endpoint, key), nil
}

