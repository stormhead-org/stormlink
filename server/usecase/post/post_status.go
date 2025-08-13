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
    // 1) Загружаем пост (для определения published/draft по published_at)
    p, err := uc.client.Post.Query().
        Where(post.IDEQ(postID)).
        Only(ctx)
    if err != nil {
        return nil, fmt.Errorf("load post: %w", err)
    }

    // 2) Считаем лайки/комментарии/закладки
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

    // 3) Вычисляем статус (по умолчанию DRAFT)
    vis := models.PostVisibilityDraft
    if p.PublishedAt != nil && !p.PublishedAt.IsZero() {
        vis = models.PostVisibilityPublished
    }

    // Views не храним — виртуально 0 (или подключите счётчик позже)
    return &models.PostStatus{
        LikesCount:     fmt.Sprintf("%d", likesCount),
        CommentsCount:  fmt.Sprintf("%d", commentsCount),
        BookmarksCount: fmt.Sprintf("%d", bookmarksCount),
        ViewsCount:     "0",
        Status:         vis,
    }, nil
}
