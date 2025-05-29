package auth

import (
	"context"
	"stormlink/server/ent/user"
	"stormlink/server/pkg/http"
	"stormlink/server/pkg/jwt"
	"strconv"

	"stormlink/server/pkg/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"stormlink/server/grpc/auth/protobuf"

	"google.golang.org/protobuf/types/known/emptypb"

	"stormlink/server/ent"
	"stormlink/server/pkg/mapper"
)

func (s *AuthService) Login(ctx context.Context, req *protobuf.LoginRequest) (*protobuf.LoginResponse, error) {
	// Проверяем входные данные
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	email := req.GetEmail()
	password := req.GetPassword()

	// Ищем пользователя по email
	user, err := s.client.User.
		Query().
		Where(user.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// Проверяем пароль
	err = jwt.ComparePassword(user.PasswordHash, password, user.Salt)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// Проверяем, верифицирован ли email
	if !user.IsVerified {
		return nil, status.Errorf(codes.FailedPrecondition, "user email not verified")
	}

	// Генерируем токены
	accessToken, err := jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error generating access token: %v", err)
	}
	refreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error generating refresh token: %v", err)
	}

	return &protobuf.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserId:       int32(user.ID),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, _ *emptypb.Empty) (*protobuf.LogoutResponse, error) {
	return &protobuf.LogoutResponse{
		Message: "Successfully logged out",
	}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *protobuf.ValidateTokenRequest) (*protobuf.ValidateTokenResponse, error) {
	claims, err := jwt.ParseAccessToken(req.Token)
	if err != nil {
			return &protobuf.ValidateTokenResponse{Valid: false}, nil
	}
	return &protobuf.ValidateTokenResponse{
			UserId: int32(claims.UserID),
			Valid:  true,
	}, nil
}

func (s *AuthService) GetMe(ctx context.Context, _ *emptypb.Empty) (*protobuf.GetMeResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated: %v", err)
	}

	user, err := s.uc.GetUserByID(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &protobuf.GetMeResponse{
		User: mapper.UserToProto(user),
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *protobuf.RefreshTokenRequest) (*protobuf.LoginResponse, error) {
	// Проверка входных данных
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	refreshToken := req.GetRefreshToken()

	// Попробуем получить refreshToken из куки, если он не передан в запросе
	if refreshToken == "" {
		httpReq := http.GetHTTPRequest(ctx)
		if httpReq != nil {
			cookie, err := httpReq.Cookie("refresh_token")
			if err == nil && cookie != nil {
				refreshToken = cookie.Value
			}
		}
	}

	if refreshToken == "" {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token is required")
	}

	// Извлекаем claims
	claims, err := jwt.ParseToken(refreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token: %v", err)
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user_id claim missing")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user_id format")
	}

	// Убедимся, что пользователь существует
	_, err = s.client.User.
		Query().
		Where(user.IDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	// Генерируем новые токены
	newAccessToken, err := jwt.GenerateAccessToken(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token")
	}
	newRefreshToken, err := jwt.GenerateRefreshToken(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token")
	}

	return &protobuf.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
