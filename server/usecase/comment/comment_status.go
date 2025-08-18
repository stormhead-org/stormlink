package comment

import (
	"context"
	"fmt"
	"stormlink/server/ent/comment"
	"stormlink/server/ent/commentlike"
	"stormlink/server/graphql/models"
)

func (uc *commentUsecase) GetCommentStatus(
    ctx context.Context,
    userID int,
    commentID int,
) (*models.CommentStatus, error) {
    // 1) Получаем коммент с автором и сообществом
    p, err := uc.client.Comment.Query().
        Where(comment.IDEQ(commentID)).
        WithAuthor().
        WithCommunity().
        Only(ctx)
    if err != nil { return nil, fmt.Errorf("get comment: %w", err) }

    // 2) Считаем лайки/комментарии/закладки
    likesCount, err := uc.client.CommentLike.Query().
        Where(commentlike.HasCommentWith(comment.IDEQ(commentID))).
        Count(ctx)
    if err != nil { return nil, fmt.Errorf("count likes: %w", err) }

    // 3) Флаги по текущему пользователю
    isLiked := false
    if userID > 0 {
        liked, err := uc.client.CommentLike.Query().
        Where(
                commentlike.CommentIDEQ(commentID),
                commentlike.UserIDEQ(userID),
        ).
        Exist(ctx)
        if err != nil { return nil, fmt.Errorf("check liked: %w", err) }
        isLiked = liked
    }

    // 4) Проверяем, является ли автор коммента владельцем сообщества
    authorCommunityOwner := false
    if p.Edges.Community != nil && p.Edges.Author != nil {
        authorCommunityOwner = p.Edges.Community.OwnerID == p.Edges.Author.ID
    }

    // 5) Проверяем, является ли автор коммента владельцем платформы (host)
    authorHostOwner := false
    if p.Edges.Author != nil {
        host, err := uc.client.Host.Query().
            Where().
            Only(ctx)
        if err == nil && host.OwnerID != nil && *host.OwnerID == p.Edges.Author.ID {
            authorHostOwner = true
        }
    }

    return &models.CommentStatus{
        LikesCount:           fmt.Sprintf("%d", likesCount),
        IsLiked:              isLiked,
        AuthorCommunityOwner: authorCommunityOwner,
        AuthorHostOwner:      authorHostOwner,
    }, nil
}
