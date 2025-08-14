package post

import (
	"context"
	"fmt"
	"stormlink/server/ent/bookmark"
	"stormlink/server/ent/comment"
	"stormlink/server/ent/post"
	"stormlink/server/ent/postlike"
	"stormlink/server/graphql/models"
)

func (uc *postUsecase) GetPostStatus(
    ctx context.Context,
    userID int,
    postID int,
) (*models.PostStatus, error) {
    // 1) Считаем лайки/комментарии/закладки
    likesCount, err := uc.client.PostLike.Query().
        Where(postlike.HasPostWith(post.IDEQ(postID))).
        Count(ctx)
    if err != nil { return nil, fmt.Errorf("count likes: %w", err) }

    commentsCount, err := uc.client.Comment.Query().
        Where(comment.PostIDEQ(postID)).
        Count(ctx)
    if err != nil { return nil, fmt.Errorf("count comments: %w", err) }

    bookmarksCount, err := uc.client.Bookmark.Query().
        Where(bookmark.PostIDEQ(postID)).
        Count(ctx)
    if err != nil { return nil, fmt.Errorf("count bookmarks: %w", err) }

    // Флаги по текущему пользователю
    isLiked := false
    hasBookmark := false
    if userID > 0 {
        liked, err := uc.client.PostLike.Query().
            Where(
                postlike.PostIDEQ(postID),
                postlike.UserIDEQ(userID),
            ).
            Exist(ctx)
        if err != nil { return nil, fmt.Errorf("check liked: %w", err) }
        isLiked = liked

        bm, err := uc.client.Bookmark.Query().
            Where(
                bookmark.PostIDEQ(postID),
                bookmark.UserIDEQ(userID),
            ).
            Exist(ctx)
        if err != nil { return nil, fmt.Errorf("check bookmark: %w", err) }
        hasBookmark = bm
    }

    return &models.PostStatus{
        LikesCount:     fmt.Sprintf("%d", likesCount),
        CommentsCount:  fmt.Sprintf("%d", commentsCount),
        BookmarksCount: fmt.Sprintf("%d", bookmarksCount),
        IsLiked:        isLiked,
        HasBookmark:    hasBookmark,
    }, nil
}
