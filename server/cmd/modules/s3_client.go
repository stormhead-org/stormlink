package modules

import (
	"log"
	shareds3 "stormlink/shared/s3"
)

// InitS3 возвращает инициализированный клиент S3, без глобальных синглтонов
func InitS3() *shareds3.S3Client {
    c, err := shareds3.NewS3Client()
    if err != nil {
        log.Fatalf("failed to init S3 client: %v", err)
    }
    return c
}
