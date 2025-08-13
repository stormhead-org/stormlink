package modules

import (
	"log"
	"net/http"
	"strings"

	shareds3 "stormlink/shared/s3"
)

// NewStorageHandler возвращает HTTP‑обработчик, который проксирует файлы из S3 совместимого хранилища
func NewStorageHandler(s3client *shareds3.S3Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        key := strings.TrimPrefix(r.URL.Path, "/storage/")
        if key == "" {
            http.Error(w, "Bad storage path", http.StatusBadRequest)
            return
        }
        contentType, data, err := s3client.GetFile(r.Context(), key)
        if err != nil {
            log.Printf("❌ StorageHandler GetFile(%q): %v", key, err)
            http.Error(w, "Not found", http.StatusNotFound)
            return
        }
        w.Header().Set("Content-Type", contentType)
        w.Header().Set("Cache-Control", "public, max-age=86400")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(data)
    }
}
