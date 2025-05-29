package media

import (
	"stormlink/server/grpc/media/protobuf"
	"stormlink/server/pkg/s3"
)

type MediaService struct {
	protobuf.UnimplementedMediaServiceServer
	s3 *s3.S3Client
}

func NewMediaServiceWithClient(client *s3.S3Client) *MediaService {
	return &MediaService{s3: client}
}
