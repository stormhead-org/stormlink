package auth

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/auth/protobuf"
)

type AuthService struct {
	protobuf.UnimplementedAuthServiceServer
	client *ent.Client
}

func NewAuthService(client *ent.Client) *AuthService {
	return &AuthService{client: client}
}
