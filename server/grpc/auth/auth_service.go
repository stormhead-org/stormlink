package auth

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/auth/protobuf"
	"stormlink/server/usecase/user"
)

type AuthService struct {
	protobuf.UnimplementedAuthServiceServer
	client *ent.Client
	uc     user.UserUsecase
}

func NewAuthService(client *ent.Client, uc user.UserUsecase) *AuthService {
	return &AuthService{
		client: client,
		uc:     uc,
	}
}
