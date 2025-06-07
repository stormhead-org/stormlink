package media

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/media/protobuf"
	"stormlink/server/pkg/s3"
)

type MediaService struct {
	protobuf.UnimplementedMediaServiceServer
	s3 *s3.S3Client
	client *ent.Client
}

func NewMediaServiceWithClient(s3Client *s3.S3Client, client *ent.Client) *MediaService {
	return &MediaService{
		s3:        s3Client,
		client: client,
}
}
