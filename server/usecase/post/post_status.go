package post

import (
	"context"
	"fmt"
	"stormlink/server/ent/communityfollow"
	"stormlink/server/ent/communityuserban"
	"stormlink/server/ent/communityusermute"
	"stormlink/server/ent/post"
	"stormlink/server/graphql/models"
)

func (uc *postUsecase) GetPostStatus(
    ctx context.Context,
    userID int,
    postID int,
) (*models.CommunityStatus, error) {
    // 1) Считаем общее число подписчиков
    followersCount, err := uc.client.CommunityFollow.
        Query().
        Where(communityfollow.CommunityIDEQ(communityID)).
        Count(ctx)
    if err != nil {
        return nil, fmt.Errorf("count followers: %w", err)
    }

    // 2) Считаем общее число постов
    postsCount, err := uc.client.Post.
        Query().
        Where(post.CommunityIDEQ(communityID)).
        Count(ctx)
    if err != nil {
        return nil, fmt.Errorf("count posts: %w", err)
    }

    // 3) Проверяем, подписан ли текущий юзер
    isFollowing, err := uc.client.CommunityFollow.
        Query().
        Where(
            communityfollow.CommunityIDEQ(communityID),
            communityfollow.UserIDEQ(userID),
        ).
        Exist(ctx)
    if err != nil {
        return nil, fmt.Errorf("check following: %w", err)
    }

    // 4) Проверяем, забанен ли текущий юзер
    isBanned, err := uc.client.CommunityUserBan.
        Query().
        Where(
            communityuserban.CommunityIDEQ(communityID),
            communityuserban.UserIDEQ(userID),
        ).
        Exist(ctx)
    if err != nil {
        return nil, fmt.Errorf("check ban: %w", err)
    }

    // 5) Проверяем, замучен ли текущий юзер
    isMuted, err := uc.client.CommunityUserMute.
        Query().
        Where(
            communityusermute.CommunityIDEQ(communityID),
            communityusermute.UserIDEQ(userID),
        ).
        Exist(ctx)
    if err != nil {
        return nil, fmt.Errorf("check mute: %w", err)
    }

    return &models.CommunityStatus{
        FollowersCount: fmt.Sprintf("%d", followersCount),
        PostsCount:     fmt.Sprintf("%d", postsCount),
        IsFollowing:    isFollowing,
        IsBanned:       isBanned,
        IsMuted:        isMuted,
    }, nil
}
