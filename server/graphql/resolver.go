package graphql

import (
	"stormlink/server/ent"
	"stormlink/server/usecase/user"

	authpb "stormlink/server/grpc/auth/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
)

type Resolver struct {
	Client *ent.Client
	UC     user.UserUsecase
	AuthClient authpb.AuthServiceClient
  UserClient userpb.UserServiceClient
}