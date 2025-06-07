package media

import (
	"context"
	"stormlink/server/grpc/media/protobuf"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *MediaService) UploadMedia(ctx context.Context, req *protobuf.UploadMediaRequest) (*protobuf.UploadMediaResponse, error) {
    // Валидация входных данных
    if err := req.Validate(); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
    }

    // Директория из запроса, по-умолчанию "media"
    dir := req.GetDir()
    if dir == "" {
        dir = "media"
    }

    filename := req.GetFilename()
    fileContent := req.GetFileContent()

    // 1) Загружаем в S3, получаем публичный URL и "санитизированное" имя
    url, sanitized, err := s.s3.UploadFile(ctx, dir, filename, fileContent)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to upload file to S3: %v", err)
    }

    // 2) Сохраняем запись в таблице media через ent
    mEntity, err := s.client.Media.
        Create().
        SetFilename(sanitized).
        SetURL(url).
        Save(ctx)
    if err != nil {
        // Если S3-загрузка прошла, но БД упала — можно даже попытаться удалить объект из S3,
        // но здесь для краткости просто возвращаем ошибку
        return nil, status.Errorf(codes.Internal, "failed to save media in DB: %v", err)
    }

    // 3) Формируем ответ, где id — это ID из БД, url и filename — из S3
    return &protobuf.UploadMediaResponse{
        Url:      url,
        Filename: sanitized,
        Id:       int64(mEntity.ID),
    }, nil
}