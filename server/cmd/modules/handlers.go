package modules

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"stormlink/server/ent"
	mediapb "stormlink/server/grpc/media/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mediaUploadHandler обрабатывает запрос на загрузку медиа
func MediaUploadHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn, client *ent.Client) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("❌ [MediaUpload] Failed to parse form: %v", err)
		http.Error(w, `{"error": "failed to parse form"}`, http.StatusBadRequest)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Printf("❌ [MediaUpload] No file provided: %v", err)
		http.Error(w, `{"error": "file is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()
	fileContent, err := io.ReadAll(file)
	if err != nil {
		log.Printf("❌ [MediaUpload] Failed to read file: %v", err)
		http.Error(w, `{"error": "failed to read file"}`, http.StatusInternalServerError)
		return
	}
	authHeader := r.Header.Get("Authorization")
	md := metadata.New(map[string]string{})
	if authHeader != "" {
		md.Set("authorization", authHeader)
	}
	ctx := metadata.NewOutgoingContext(r.Context(), md)
	mediaClient := mediapb.NewMediaServiceClient(grpcConn)
	dir := r.FormValue("dir")
	if dir == "" {
		dir = "media"
	}
	grpcResp, err := mediaClient.UploadMedia(ctx, &mediapb.UploadMediaRequest{
		Dir:         dir,
		Filename:    handler.Filename,
		FileContent: fileContent,
	})
	if err != nil {
		log.Printf("❌ [MediaUpload] gRPC error: %v", err)
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	m, err := client.Media.
		Create().
		SetFilename(grpcResp.GetFilename()).
		SetURL(grpcResp.GetUrl()).
		Save(r.Context())
	if err != nil {
		log.Printf("❌ [MediaUpload] Save DB error: %v", err)
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}

	grpcResp.Id = int64(m.ID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(grpcResp); err != nil {
    log.Printf("❌ [MediaUpload] Failed to encode response: %v", err)
    http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
    return
	}
}

// storageHandler обрабатывает запросы к хранилищу
func StorageHandler(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/storage/")
	if key == "" {
		http.Error(w, "Bad storage path", http.StatusBadRequest)
		return
	}
	contentType, data, err := S3Client.GetFile(r.Context(), key)
	if err != nil {
		log.Printf("❌ StorageHandler GetFile(%q): %v", key, err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
