package auth

import (
	"context"
	"fmt"
	"stormlink/server/ent/user" //
	"stormlink/server/utils"

	"stormlink/server/grpc/auth/protobuf"
)

func (s *AuthService) Login(ctx context.Context, req *protobuf.LoginRequest) (*protobuf.LoginResponse, error) {
	email := req.GetEmail()
	password := req.GetPassword()

	// Ищем пользователя по email
	user, err := s.client.User.
		Query().
		Where(user.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Проверяем пароль
	err = utils.ComparePassword(user.PasswordHash, password, user.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Генерируем токены
	accessToken, err := utils.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("error generating access token: %v", err)
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %v", err)
	}

	return &protobuf.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
