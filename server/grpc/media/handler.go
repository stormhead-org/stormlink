package media

import (
	"context"
	"log"
	"stormlink/server/grpc/media/protobuf"
	"stormlink/server/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MediaService struct {
    protobuf.UnimplementedMediaServiceServer
    s3 *utils.S3Client
}

func NewMediaService() *MediaService {
    s3Client, err := utils.NewS3Client()
    if err != nil {
        log.Fatalf("failed to initialize S3 client: %v", err)
    }
    return &MediaService{s3: s3Client}
}

func (s *MediaService) UploadMedia(ctx context.Context, req *protobuf.UploadMediaRequest) (*protobuf.UploadMediaResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
    }

    filename := req.GetFilename()
    fileContent := req.GetFileContent()

    url, err := s.s3.UploadFile(ctx, filename, fileContent)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to upload file: %v", err)
    }

    return &protobuf.UploadMediaResponse{
        Url: url,
    }, nil
}