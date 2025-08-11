package service

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"stormlink/server/ent"
	entuser "stormlink/server/ent/user"
	authpb "stormlink/server/grpc/auth/protobuf"
	useruc "stormlink/server/usecase/user"
	"stormlink/shared/auth"
	httpCookies "stormlink/shared/http"
	"stormlink/shared/jwt"
	sharedmapper "stormlink/shared/mapper"
	redisx "stormlink/shared/redis"

	redis "github.com/redis/go-redis/v9"
)

type AuthService struct {
    authpb.UnimplementedAuthServiceServer
    client *ent.Client
    uc     useruc.UserUsecase
    // опционально: хранение токенов/сессий в Redis (revoke/rotation)
    // если redis не сконфигурирован — будет nil и функционал пропускается
    redis *redis.Client
}

func NewAuthService(client *ent.Client, uc useruc.UserUsecase) *AuthService {
    var rds *redis.Client
    if c, err := redisx.NewClient(); err == nil {
        rds = c
    }
    return &AuthService{client: client, uc: uc, redis: rds}
}

func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
    }
    email := req.GetEmail()
    password := req.GetPassword()

    u, err := s.client.User.
        Query().
        Where(entuser.EmailEQ(email)).
        WithAvatar().
        WithUserInfo().
        WithHostRoles().
        WithCommunitiesRoles().
        Only(ctx)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
    }

    if err := jwt.ComparePassword(u.PasswordHash, password, u.Salt); err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
    }
    if !u.IsVerified {
        return nil, status.Errorf(codes.FailedPrecondition, "user email not verified")
    }

    accessToken, err := jwt.GenerateAccessToken(u.ID)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "error generating access token: %v", err)
    }
    refreshToken, err := jwt.GenerateRefreshToken(u.ID)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "error generating refresh token: %v", err)
    }

    // Опционально: записать сессию/refresh в Redis с TTL, чтобы можно было делать revoke/rotation
    if s.redis != nil {
        ttl := 7 * 24 * time.Hour
        _ = s.redis.Set(ctx, "refresh:"+refreshToken, u.ID, ttl).Err()
    }

    if w := httpCookies.GetHTTPResponseWriter(ctx); w != nil {
        httpCookies.SetAuthCookies(w, accessToken, refreshToken)
        log.Println("✅ [Login] Cookies set successfully")
    } else {
        log.Println("⚠️ [Login] HTTP response writer not found, cookies not set")
    }

    return &authpb.LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken, User: sharedmapper.UserToProto(u)}, nil
}

func (s *AuthService) Logout(ctx context.Context, _ *emptypb.Empty) (*authpb.LogoutResponse, error) {
    userID, err := auth.UserIDFromContext(ctx)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "unauthenticated: %v", err)
    }
    if _, err := s.client.User.Query().Where(entuser.IDEQ(userID)).Only(ctx); err != nil {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    // Если используем Redis — удалим текущий refresh из хранилища
    if s.redis != nil {
        if r := httpCookies.GetHTTPRequest(ctx); r != nil {
            if c, err := r.Cookie("refresh_token"); err == nil && c != nil && c.Value != "" {
                _ = s.redis.Del(ctx, "refresh:"+c.Value).Err()
            }
        }
    }

    if w := httpCookies.GetHTTPResponseWriter(ctx); w != nil {
        httpCookies.ClearAuthCookies(w)
    } else {
        log.Println("⚠️ [Logout] HTTP response writer not found, cookies not cleared")
    }
    return &authpb.LogoutResponse{Message: "Successfully logged out"}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
    claims, err := jwt.ParseAccessToken(req.Token)
    if err != nil {
        return &authpb.ValidateTokenResponse{Valid: false}, nil
    }
    // Дополнительно: можно проверять revoke списка access по jti если перейдем на jti
    return &authpb.ValidateTokenResponse{UserId: int32(claims.UserID), Valid: true}, nil
}

func (s *AuthService) GetMe(ctx context.Context, _ *emptypb.Empty) (*authpb.GetMeResponse, error) {
    userID, err := auth.UserIDFromContext(ctx)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "unauthenticated: %v", err)
    }
    u, err := s.uc.GetUserByID(ctx, userID)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, status.Errorf(codes.NotFound, "user not found")
        }
        return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
    }
    return &authpb.GetMeResponse{User: sharedmapper.UserToProto(u)}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
    }
    refreshToken := req.GetRefreshToken()
    if refreshToken == "" {
        if r := httpCookies.GetHTTPRequest(ctx); r != nil {
            if c, err := r.Cookie("refresh_token"); err == nil && c != nil {
                refreshToken = c.Value
            }
        }
    }
    if refreshToken == "" {
        return nil, status.Errorf(codes.InvalidArgument, "refresh token is required")
    }
    claims, err := jwt.ParseRefreshToken(refreshToken)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token: %v", err)
    }
    // Проверяем, что refresh присутствует и не отозван (если есть Redis)
    if s.redis != nil {
        if _, err := s.redis.Get(ctx, "refresh:"+refreshToken).Result(); err != nil {
            return nil, status.Errorf(codes.Unauthenticated, "refresh token is revoked or unknown")
        }
    }
    userID := claims.UserID
    if _, err := s.client.User.Query().Where(entuser.IDEQ(userID)).Only(ctx); err != nil {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    newAccess, err := jwt.GenerateAccessToken(userID)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to generate access token")
    }
    newRefresh, err := jwt.GenerateRefreshToken(userID)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to generate refresh token")
    }

    // Ротация refresh: помечаем старый как использованный и сохраняем новый, если есть Redis
    if s.redis != nil {
        ttl := 7 * 24 * time.Hour
        // invalidate old
        _ = s.redis.Del(ctx, "refresh:"+refreshToken).Err()
        // set new
        _ = s.redis.Set(ctx, "refresh:"+newRefresh, userID, ttl).Err()
    }
    if w := httpCookies.GetHTTPResponseWriter(ctx); w != nil {
        httpCookies.SetAuthCookies(w, newAccess, newRefresh)
        log.Println("✅ [RefreshToken] Cookies set successfully")
    } else {
        log.Println("⚠️ [RefreshToken] HTTP response writer not found, cookies not set")
    }
    return &authpb.RefreshTokenResponse{AccessToken: newAccess, RefreshToken: newRefresh}, nil
}


