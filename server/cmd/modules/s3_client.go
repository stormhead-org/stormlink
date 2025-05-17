package modules

import (
	"log"
	"stormlink/server/utils"
)

var S3Client *utils.S3Client

func InitS3Client() {
	var err error
	S3Client, err = utils.NewS3Client()
	if err != nil {
		log.Fatalf("failed to init S3 client: %v", err)
	}
}
