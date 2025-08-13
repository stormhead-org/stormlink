package post

import (
	"context"
	"stormlink/server/ent"
	"stormlink/server/ent/post"
	"stormlink/server/graphql/models"
)

type PostUsecase interface {
  GetPostByID(ctx context.Context, id int) (*ent.Post, error)
  GetPostStatus(ctx context.Context, userID int, postID int) (*models.PostStatus, error)
}


type postUsecase struct {
	client *ent.Client
}

func NewPostUsecase(client *ent.Client) PostUsecase {
	return &postUsecase{client: client}
}

func (uc *postUsecase) GetPostByID(ctx context.Context, id int) (*ent.Post, error) {
	return uc.client.Post.
			Query().
			Where(post.IDEQ(id)).
			WithHeroImage().
			WithAuthor().
			WithBookmarks().
			WithComments().
			WithCommunity().
			WithLikes().
			Only(ctx)
}
