package media

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"stormlink/server/grpc/media/protobuf"
	"stormlink/server/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MediaService struct {
	protobuf.UnimplementedMediaServiceServer
	s3 *utils.S3Client
}

// используем фабрику с внешним S3Client
func NewMediaServiceWithClient(client *utils.S3Client) *MediaService {
	return &MediaService{s3: client}
}

func (s *MediaService) UploadMedia(ctx context.Context, req *protobuf.UploadMediaRequest) (*protobuf.UploadMediaResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	dir := req.GetDir()
	if dir == "" {
		dir = "media"
	}
	filename := req.GetFilename()
	fileContent := req.GetFileContent()

	// мы знаем ключ внутри бакета
	key := filepath.ToSlash(filepath.Join(dir, filename))

	// отправляем в S3
	if err := s.s3.Put(ctx, key, fileContent); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload file: %v", err)
	}

	// возвращаем прокси-путь, по которому клиент будет брать картинку
	return &protobuf.UploadMediaResponse{
		Url: fmt.Sprintf("/storage/%s", strings.TrimPrefix(key, "/")),
	}, nil
}
