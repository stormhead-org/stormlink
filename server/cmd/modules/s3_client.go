package modules

import (
	"log"
	shareds3 "stormlink/shared/s3"
)

var S3Client *shareds3.S3Client

func InitS3Client() {
	var err error
    S3Client, err = shareds3.NewS3Client()
	if err != nil {
		log.Fatalf("failed to init S3 client: %v", err)
	}
}
