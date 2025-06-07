package community

import (
	"context"
	"stormlink/server/ent"
	"stormlink/server/ent/community"
	"stormlink/server/model"
)

type CommunityUsecase interface {
	GetCommunityByID(ctx context.Context, id int) (*ent.Community, error)
	GetCommunityStatus(ctx context.Context, userID int, communityID int) (map[int]*model.CommunityStatus, error)
}

type communityUsecase struct {
	client *ent.Client
}

func NewCommunityUsecase(client *ent.Client) communityUsecase {
	return &communityUsecase{client: client}
}

func (uc *communityUsecase) GetCommunityByID(ctx context.Context, id int) (*ent.Community, error) {
	return uc.client.Community.
			Query().
			Where(community.IDEQ(id)).
			WithLogo().
			WithCommunityInfo().
			WithRoles().
			Only(ctx)
}
