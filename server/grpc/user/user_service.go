package user

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/user/protobuf"
	"stormlink/server/usecase/user"
)

type UserService struct {
	protobuf.UnimplementedUserServiceServer
	client *ent.Client
	uc     user.UserUsecase
}

func NewUserService(client *ent.Client, uc user.UserUsecase) *UserService {
	return &UserService{
		client: client,
		uc:     uc,
	}
}
