package media

import (
	"context"
	"stormlink/server/grpc/media/protobuf"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	url, sanitized, err := s.s3.UploadFile(ctx, dir, filename, fileContent)

	// отправляем в S3
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload file: %v", err)
	}

	// возвращаем прокси-путь, по которому клиент будет брать картинку
	return &protobuf.UploadMediaResponse{
		Url:      url,
		Filename: sanitized,
	}, nil
}
