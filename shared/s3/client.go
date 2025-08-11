package s3

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
)

type S3Client struct {
    svc          *awss3.S3
    bucket       string
    endpoint     string
    region       string
    usePathStyle bool
    aliasHost    string
}

func NewS3Client() (*S3Client, error) {
    endpoint := os.Getenv("S3_ENDPOINT")
    region := os.Getenv("S3_REGION")
    bucket := os.Getenv("S3_BUCKET")
    access := os.Getenv("S3_ACCESS_KEY_ID")
    secret := os.Getenv("S3_SECRET_ACCESS_KEY")
    usePathStyle := os.Getenv("S3_USE_PATH_STYLE") == "true"
    aliasHost := os.Getenv("S3_ALIAS_HOST")

    if endpoint == "" { return nil, fmt.Errorf("S3_ENDPOINT is not set") }
    if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
        endpoint = "https://" + endpoint
    }
    if region == "" { return nil, fmt.Errorf("S3_REGION is not set") }
    if bucket == "" { return nil, fmt.Errorf("S3_BUCKET is not set") }
    if access == "" || secret == "" { return nil, fmt.Errorf("S3_ACCESS_KEY_ID or S3_SECRET_ACCESS_KEY is not set") }

    sess, err := session.NewSession(&aws.Config{
        Region:           aws.String(region),
        Endpoint:         aws.String(endpoint),
        S3ForcePathStyle: aws.Bool(usePathStyle),
        Credentials:      credentials.NewStaticCredentials(access, secret, ""),
    })
    if err != nil { return nil, fmt.Errorf("failed to create AWS session: %w", err) }

    svc := awss3.New(sess)
    return &S3Client{svc: svc, bucket: bucket, endpoint: endpoint, region: region, usePathStyle: usePathStyle, aliasHost: aliasHost}, nil
}


