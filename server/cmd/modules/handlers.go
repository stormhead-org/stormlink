package modules

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	authpb "stormlink/server/grpc/auth/protobuf"
	mediapb "stormlink/server/grpc/media/protobuf"
	"stormlink/server/utils"
)

// loginHandler обрабатывает запрос на вход пользователя
func LoginHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("📥 [HTTP] Request: POST /v1/users/login")
	var req authpb.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ [Login] Invalid request body: %v", err)
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}
	authClient := authpb.NewAuthServiceClient(grpcConn)
	resp, err := authClient.Login(r.Context(), &req)
	if err != nil {
		log.Printf("❌ [Login] gRPC error: %v", err)
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}
	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	log.Printf("✅ [Login] Set auth cookies")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("❌ [Login] Failed to encode response: %v", err)
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
		return
	}
	log.Printf("📤 [HTTP] Response: Login successful")
}

// logoutHandler обрабатывает запрос на выход пользователя
func LogoutHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("📥 [HTTP] Request: POST /v1/users/logout")
	authHeader := r.Header.Get("Authorization")
	log.Printf("🔍 [Logout] HTTP Authorization header: %s", authHeader)
	md := metadata.New(map[string]string{})
	if authHeader != "" {
		md.Set("authorization", authHeader)
	}
	ctx := metadata.NewOutgoingContext(r.Context(), md)
	authClient := authpb.NewAuthServiceClient(grpcConn)
	resp, err := authClient.Logout(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("❌ [Logout] gRPC error: %v", err)
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Domain:   "localhost",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Domain:   "localhost",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	log.Printf("✅ [Logout] Cleared auth cookies")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("❌ [Logout] Failed to encode response: %v", err)
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
		return
	}
	log.Printf("📤 [HTTP] Response: Successfully logged out")
}

// refreshTokenHandler обрабатывает запрос на обновление токена
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("📥 [HTTP] Request: POST /v1/users/refresh-token")
	var req authpb.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		cookie, err := r.Cookie("refresh_token")
		if err == nil && cookie != nil {
			req.RefreshToken = cookie.Value
		} else {
			log.Printf("❌ [RefreshToken] Refresh token required: %v", err)
			http.Error(w, `{"error": "refresh token required"}`, http.StatusBadRequest)
			return
		}
	}
	authClient := authpb.NewAuthServiceClient(grpcConn)
	resp, err := authClient.RefreshToken(r.Context(), &req)
	if err != nil {
		log.Printf("❌ [RefreshToken] gRPC error: %v", err)
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}
	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	log.Printf("✅ [RefreshToken] Set auth cookies")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("❌ [RefreshToken] Failed to encode response: %v", err)
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
		return
	}
	log.Printf("📤 [HTTP] Response: Token refreshed")
}

// mediaUploadHandler обрабатывает запрос на загрузку медиа
func MediaUploadHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("📥 [HTTP] Request: POST /v1/media/upload")
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
	log.Printf("🔍 [MediaUpload] HTTP Authorization header: %s", authHeader)
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
	resp, err := mediaClient.UploadMedia(ctx, &mediapb.UploadMediaRequest{
		Dir:         dir,
		Filename:    handler.Filename,
		FileContent: fileContent,
	})
	if err != nil {
		log.Printf("❌ [MediaUpload] gRPC error: %v", err)
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	log.Printf("✅ [MediaUpload] File uploaded: %s", resp.Url)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("❌ [MediaUpload] Failed to encode response: %v", err)
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
		return
	}
	log.Printf("📤 [HTTP] Response: File uploaded")
}

// storageHandler обрабатывает запросы к хранилищу
func StorageHandler(w http.ResponseWriter, r *http.Request, s3Client *utils.S3Client) {
	key := strings.TrimPrefix(r.URL.Path, "/storage/")
	if key == "" {
		http.Error(w, "Bad storage path", http.StatusBadRequest)
		return
	}
	ctype, data, err := s3Client.GetFile(r.Context(), key)
	if err != nil {
		log.Printf("❌ StorageProxy GetFile(%q): %v", key, err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
