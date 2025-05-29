package user

import (
	"context"
	"stormlink/server/ent"
	"stormlink/server/ent/user"
	"stormlink/server/model"
)

type UserUsecase interface {
	GetUserByID(ctx context.Context, id int) (*ent.User, error)
	GetPermissionsByCommunities(ctx context.Context, userID int, communityIDs []int) (map[int]*model.CommunityPermissions, error)
}

type userUsecase struct {
	client *ent.Client
}

func NewUserUsecase(client *ent.Client) UserUsecase {
	return &userUsecase{client: client}
}

func (uc *userUsecase) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
	return uc.client.User.
			Query().
			Where(user.IDEQ(id)).
			WithAvatar().
			WithUserInfo().
			WithHostRoles().
			WithCommunitiesRoles().
			Only(ctx)
}
