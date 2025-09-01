package service

import (
	"context"
	"stormlink/server/ent"
	mediapb "stormlink/server/grpc/media/protobuf"
	errorsx "stormlink/shared/errors"
	shareds3 "stormlink/shared/s3"

	"google.golang.org/grpc/codes"
)

// S3ClientInterface defines the interface for S3 operations
type S3ClientInterface interface {
	UploadFile(ctx context.Context, dir, filename string, fileContent []byte) (url, sanitized string, err error)
}

type MediaService struct {
	mediapb.UnimplementedMediaServiceServer
	s3     S3ClientInterface
	client *ent.Client
}

func NewMediaServiceWithClient(s3client S3ClientInterface, client *ent.Client) *MediaService {
	return &MediaService{s3: s3client, client: client}
}

func NewMediaService(s3client *shareds3.S3Client, client *ent.Client) *MediaService {
	return &MediaService{s3: s3client, client: client}
}

func (s *MediaService) UploadMedia(ctx context.Context, req *mediapb.UploadMediaRequest) (*mediapb.UploadMediaResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errorsx.FromGRPCCode(codes.InvalidArgument, "validation error", err)
	}
	dir := req.GetDir()
	if dir == "" {
		dir = "media"
	}
	filename := req.GetFilename()
	fileContent := req.GetFileContent()

	url, sanitized, err := s.s3.UploadFile(ctx, dir, filename, fileContent)
	if err != nil {
		return nil, errorsx.FromGRPCCode(codes.Internal, "failed to upload file to S3", err)
	}

	m, err := s.client.Media.Create().SetFilename(sanitized).SetURL(url).Save(ctx)
	if err != nil {
		return nil, errorsx.FromGRPCCode(codes.Internal, "failed to save media in DB", err)
	}

	return &mediapb.UploadMediaResponse{Url: url, Filename: sanitized, Id: int64(m.ID)}, nil
}
