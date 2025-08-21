package graphql

import (
	"stormlink/server/ent"
	"stormlink/server/usecase/ban"
	"stormlink/server/usecase/comment"
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/communityrole"
	"stormlink/server/usecase/communityrule"
	"stormlink/server/usecase/hostmute"
	"stormlink/server/usecase/hostrole"
	"stormlink/server/usecase/hostrule"
	"stormlink/server/usecase/post"
	"stormlink/server/usecase/profiletableinfoitem"
	"stormlink/server/usecase/user"

	authpb "stormlink/server/grpc/auth/protobuf"
	mailpb "stormlink/server/grpc/mail/protobuf"
	mediapb "stormlink/server/grpc/media/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
)

type Resolver struct {
	Client *ent.Client
	UserUC user.UserUsecase
	CommunityUC community.CommunityUsecase
	PostUC post.PostUsecase
	CommentUC comment.CommentUsecase
	HostRoleUC hostrole.HostRoleUsecase
	CommunityRoleUC communityrole.CommunityRoleUsecase
	CommunityRuleUsecase communityrule.CommunityRuleUsecase
	HostRuleUC hostrule.HostRuleUsecase
	HostMuteUC hostmute.HostMuteUsecase
	BanUC ban.BanUsecase
	ProfileTableInfoItemUC profiletableinfoitem.ProfileTableInfoItemUsecase
	AuthClient authpb.AuthServiceClient
	UserClient userpb.UserServiceClient
	MailClient mailpb.MailServiceClient
	MediaClient mediapb.MediaServiceClient
}