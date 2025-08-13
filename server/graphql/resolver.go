package graphql

import (
	"stormlink/server/ent"
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/post"
	"stormlink/server/usecase/user"

	authpb "stormlink/server/grpc/auth/protobuf"
	mailpb "stormlink/server/grpc/mail/protobuf"
	mediapb "stormlink/server/grpc/media/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
)

type Resolver struct {
	Client *ent.Client
    UserUC      user.UserUsecase
    CommunityUC community.CommunityUsecase
		PostUC post.PostUsecase
    AuthClient  authpb.AuthServiceClient
    UserClient  userpb.UserServiceClient
    MailClient  mailpb.MailServiceClient
    MediaClient mediapb.MediaServiceClient
}