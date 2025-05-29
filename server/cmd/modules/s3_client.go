package modules

import (
	"log"
	"stormlink/server/pkg/s3"
)

var S3Client *s3.S3Client

func InitS3Client() {
	var err error
	S3Client, err = s3.NewS3Client()
	if err != nil {
		log.Fatalf("failed to init S3 client: %v", err)
	}
}
