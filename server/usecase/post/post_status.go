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
    // 1) Получаем пост с автором и сообществом
    p, err := uc.client.Post.Query().
        Where(post.IDEQ(postID)).
        WithAuthor().
        WithCommunity().
        Only(ctx)
    if err != nil { return nil, fmt.Errorf("get post: %w", err) }

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

    // 3) Флаги по текущему пользователю
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

    // 4) Проверяем, является ли автор поста владельцем сообщества
    authorCommunityOwner := false
    if p.Edges.Community != nil && p.Edges.Author != nil {
        authorCommunityOwner = p.Edges.Community.OwnerID == p.Edges.Author.ID
    }

    // 5) Проверяем, является ли автор поста владельцем платформы (host)
    authorHostOwner := false
    if p.Edges.Author != nil {
        host, err := uc.client.Host.Query().
            Where().
            Only(ctx)
        if err == nil && host.OwnerID != nil && *host.OwnerID == p.Edges.Author.ID {
            authorHostOwner = true
        }
    }

    return &models.PostStatus{
        LikesCount:           fmt.Sprintf("%d", likesCount),
        CommentsCount:        fmt.Sprintf("%d", commentsCount),
        BookmarksCount:       fmt.Sprintf("%d", bookmarksCount),
        IsLiked:              isLiked,
        HasBookmark:          hasBookmark,
        AuthorCommunityOwner: authorCommunityOwner,
        AuthorHostOwner:      authorHostOwner,
    }, nil
}
