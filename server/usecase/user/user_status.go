package user

import (
	"context"
	"fmt"
	"stormlink/server/ent/hostuserban"
	"stormlink/server/ent/hostusermute"
	"stormlink/server/ent/post"
	"stormlink/server/ent/user"
	"stormlink/server/ent/userfollow"
	"stormlink/server/graphql/models"
)

func (uc *userUsecase) GetUserStatus(
    ctx context.Context,
    currentUserID int,
    userID int,
) (*models.UserStatus, error) {
    // 1) Считаем общее число подписчиков
    followersCount, err := uc.client.UserFollow.
        Query().
        Where(userfollow.FolloweeIDEQ(userID)).
        Count(ctx)
    if err != nil {
        return nil, fmt.Errorf("count followers: %w", err)
    }

    // 2) Считаем на кого пользователь подписан
    followingCount, err := uc.client.UserFollow.
        Query().
        Where(userfollow.FollowerIDEQ(userID)).
        Count(ctx)
    if err != nil {
        return nil, fmt.Errorf("count following: %w", err)
    }

    // 3) Считаем общее число постов
    postsCount, err := uc.client.Post.
        Query().
        Where(post.AuthorIDEQ(userID)).
        Count(ctx)
    if err != nil {
        return nil, fmt.Errorf("count posts: %w", err)
    }

    // 4) Проверяем, подписан ли текущий юзер
    isFollowing, err := uc.client.UserFollow.
        Query().
        Where(
            userfollow.FollowerIDEQ(currentUserID),
            userfollow.FolloweeIDEQ(userID),
        ).
        Exist(ctx)
    if err != nil {
        return nil, fmt.Errorf("check following: %w", err)
    }

    // 5) Проверяем, забанен ли текущий юзер
    isHostBanned, err := uc.client.HostUserBan.
        Query().
        Where(
            hostuserban.HasUserWith(user.IDEQ(userID)),
        ).
        Exist(ctx)
    if err != nil {
        return nil, fmt.Errorf("check host ban: %w", err)
    }

    // 6) Проверяем, замучен ли текущий юзер
    isHostMuted, err := uc.client.HostUserMute.
        Query().
        Where(
            hostusermute.HasUserWith(user.IDEQ(userID)),
        ).
        Exist(ctx)
    if err != nil {
        return nil, fmt.Errorf("check host mute: %w", err)
    }

    return &models.UserStatus{
        FollowersCount: fmt.Sprintf("%d", followersCount),
        FollowingCount: fmt.Sprintf("%d", followingCount),
        PostsCount:     fmt.Sprintf("%d", postsCount),
        IsHostBanned:       isHostBanned,
        IsHostMuted:        isHostMuted,
        IsFollowing:    isFollowing,
    }, nil
}
