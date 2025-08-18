package comment

import (
	"context"
	"encoding/base64"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/comment"
	"stormlink/server/ent/post"
	"stormlink/server/graphql/models"
	"strconv"
	"strings"
	"time"
)

type CommentUsecase interface {
	GetCommentsByPostID(ctx context.Context, postID int, hasDeleted *bool) ([]*ent.Comment, error)
	GetCommentsByPostIDLight(ctx context.Context, postID int, hasDeleted *bool) ([]*ent.Comment, error)
	GetCommentsByPostIDLightPaginated(ctx context.Context, postID int, hasDeleted *bool, limit int32, offset int32) ([]*ent.Comment, error)
	GetCommentsFeed(ctx context.Context, limit int32) ([]*ent.Comment, error)
	GetAllComments(ctx context.Context, hasDeleted *bool) ([]*ent.Comment, error)

	// Keyset pagination: курсоры формируются как base64("createdAt|id"), порядок ASC
	CommentsByPostConnection(ctx context.Context, postID int, hasDeleted *bool, first *int, after *string, last *int, before *string) (*models.CommentsConnection, error)
	// Общая лента: двунаправленная keyset пагинация по всем комментариям
	CommentsFeedConnection(ctx context.Context, hasDeleted *bool, first *int, after *string, last *int, before *string) (*models.CommentsConnection, error)
	// Окно вокруг якоря (включая сам якорь)
	CommentsWindow(ctx context.Context, postID int, anchorID int, beforeN int, afterN int, hasDeleted *bool) (*models.CommentsConnection, error)
	// Утилита для получения одного комментария
	CommentByID(ctx context.Context, id int) (*ent.Comment, error)
	// Получение статуса комментария для текущего пользователя
	GetCommentStatus(ctx context.Context, userID int, commentID int) (*models.CommentStatus, error)
}

type commentUsecase struct {
	client *ent.Client
}

func NewCommentUsecase(client *ent.Client) CommentUsecase {
	return &commentUsecase{client: client}
}

// GetCommentsByPostID возвращает плоский список комментариев к посту (полная версия для админки)
func (uc *commentUsecase) GetCommentsByPostID(ctx context.Context, postID int, hasDeleted *bool) ([]*ent.Comment, error) {
	q := uc.client.Comment.Query().
		Where(comment.PostIDEQ(postID))
	
	if hasDeleted != nil {
		q = q.Where(comment.HasDeletedEQ(*hasDeleted))
	}
	
	return q.
		WithAuthor().
		WithCommunity().
		WithMedia().
		Order(ent.Asc(comment.FieldCreatedAt)).
		All(ctx)
}

// GetCommentsByPostIDLight возвращает облегченную версию комментариев к посту (только основные поля + media для URL)
func (uc *commentUsecase) GetCommentsByPostIDLight(ctx context.Context, postID int, hasDeleted *bool) ([]*ent.Comment, error) {
	q := uc.client.Comment.Query().
		Where(comment.PostIDEQ(postID))
	
	if hasDeleted != nil {
		q = q.Where(comment.HasDeletedEQ(*hasDeleted))
	}
	
	// Убираем WithAuthor(), WithCommunity(), WithPost() чтобы снизить нагрузку на БД
	// Оставляем только WithMedia() если клиенту нужны URL медиафайлов
	return q.
		WithMedia().
		Order(ent.Asc(comment.FieldCreatedAt)).
		All(ctx)
}

// GetCommentsByPostIDLightPaginated возвращает облегченную версию комментариев к посту с пагинацией
func (uc *commentUsecase) GetCommentsByPostIDLightPaginated(ctx context.Context, postID int, hasDeleted *bool, limit int32, offset int32) ([]*ent.Comment, error) {
	q := uc.client.Comment.Query().
		Where(comment.PostIDEQ(postID))

	if hasDeleted != nil {
		q = q.Where(comment.HasDeletedEQ(*hasDeleted))
	}

	if limit > 0 {
		q = q.Limit(int(limit))
	}
	if offset > 0 {
		q = q.Offset(int(offset))
	}

	return q.
		WithMedia().
		Order(ent.Asc(comment.FieldCreatedAt)).
		All(ctx)
}

// GetCommentsFeed возвращает плоский список последних комментариев для общей ленты
func (uc *commentUsecase) GetCommentsFeed(ctx context.Context, limit int32) ([]*ent.Comment, error) {
	return uc.client.Comment.Query().
		Where(comment.HasDeletedEQ(false)).
		Where(comment.HasPostWith(post.VisibilityEQ(post.VisibilityPublished))).
		WithAuthor().
		WithPost().
		WithCommunity().
		WithMedia().
		Order(ent.Desc(comment.FieldCreatedAt)).
		Limit(int(limit)).
		All(ctx)
}

// GetAllComments возвращает все комментарии платформы (плоский список)
func (uc *commentUsecase) GetAllComments(ctx context.Context, hasDeleted *bool) ([]*ent.Comment, error) {
	q := uc.client.Comment.Query()
	
	if hasDeleted != nil {
		q = q.Where(comment.HasDeletedEQ(*hasDeleted))
	} else {
		q = q.Where(comment.HasDeletedEQ(false))
	}

	// Только комментарии постов с visibility = published
	q = q.Where(comment.HasPostWith(post.VisibilityEQ(post.VisibilityPublished)))
	
	return q.
		WithAuthor().
		WithPost().
		WithCommunity().
		WithMedia().
		Order(ent.Desc(comment.FieldCreatedAt)).
		All(ctx)
}

// CommentByID возвращает комментарий по ID
func (uc *commentUsecase) CommentByID(ctx context.Context, id int) (*ent.Comment, error) {
	return uc.client.Comment.Get(ctx, id)
}

// cursorKey создает строку ключа курсора "createdAt|id"
func cursorKey(c *ent.Comment) string {
	return c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.Itoa(c.ID)
}

// encodeCursor кодирует ключ в base64
func encodeCursor(key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key))
}

// decodeCursor декодирует base64 в (createdAt, id)
func decodeCursor(cur string) (time.Time, int, error) {
	b, err := base64.StdEncoding.DecodeString(cur)
	if err != nil { return time.Time{}, 0, err }
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 { return time.Time{}, 0, fmt.Errorf("invalid cursor") }
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil { return time.Time{}, 0, err }
	id, err := strconv.Atoi(parts[1])
	if err != nil { return time.Time{}, 0, err }
	return t, id, nil
}

// CommentsByPostConnection реализация двунаправленной keyset пагинации
func (uc *commentUsecase) CommentsByPostConnection(ctx context.Context, postID int, hasDeleted *bool, first *int, after *string, last *int, before *string) (*models.CommentsConnection, error) {
	// базовый запрос
	base := uc.client.Comment.Query().Where(comment.PostIDEQ(postID))
	if hasDeleted != nil { base = base.Where(comment.HasDeletedEQ(*hasDeleted)) }

	var edges []*models.CommentEdge
	var startCur, endCur *string
	var hasPrev, hasNext bool

	// Ветка загрузки вниз (first/after)
	if first != nil {
		lim := *first
		q := base.Clone()
		if after != nil && *after != "" {
			ta, ia, err := decodeCursor(*after)
			if err != nil { return nil, err }
			q = q.Where(
				comment.Or(
					comment.CreatedAtGT(ta),
					comment.And(comment.CreatedAtEQ(ta), comment.IDGT(ia)),
				),
			)
		}
		items, err := q.
			Order(ent.Asc(comment.FieldCreatedAt), ent.Asc(comment.FieldID)).
			Limit(lim + 1).
			All(ctx)
		if err != nil { return nil, err }
		if len(items) > lim { hasNext = true; items = items[:lim] }
		for _, c := range items {
			cur := encodeCursor(cursorKey(c))
			edges = append(edges, &models.CommentEdge{ Cursor: cur, Node: c })
		}
	} else if last != nil { // Ветка загрузки вверх (last/before)
		lim := *last
		q := base.Clone()
		if before != nil && *before != "" {
			tb, ib, err := decodeCursor(*before)
			if err != nil { return nil, err }
			q = q.Where(
				comment.Or(
					comment.CreatedAtLT(tb),
					comment.And(comment.CreatedAtEQ(tb), comment.IDLT(ib)),
				),
			)
		}
		items, err := q.
			Order(ent.Desc(comment.FieldCreatedAt), ent.Desc(comment.FieldID)).
			Limit(lim + 1).
			All(ctx)
		if err != nil { return nil, err }
		if len(items) > lim { hasPrev = true; items = items[:lim] }
		// разворачиваем в ASC для клиента
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
		for _, c := range items {
			cur := encodeCursor(cursorKey(c))
			edges = append(edges, &models.CommentEdge{ Cursor: cur, Node: c })
		}
	} else {
		// если не указаны ни first, ни last — вернём пустое окно
		return &models.CommentsConnection{Edges: []*models.CommentEdge{}, PageInfo: &models.PageInfo{HasNextPage: false, HasPreviousPage: false}}, nil
	}

	if len(edges) > 0 {
		s := edges[0].Cursor
		e := edges[len(edges)-1].Cursor
		startCur = &s
		endCur = &e
	}

	return &models.CommentsConnection{
		Edges: edges,
		PageInfo: &models.PageInfo{
			HasNextPage: hasNext,
			HasPreviousPage: hasPrev,
			StartCursor:    startCur,
			EndCursor:      endCur,
		},
	}, nil
}

// CommentsWindow возвращает окно вокруг якоря
func (uc *commentUsecase) CommentsWindow(ctx context.Context, postID int, anchorID int, beforeN int, afterN int, hasDeleted *bool) (*models.CommentsConnection, error) {
	// грузим якорь и валидируем пост
	anchor, err := uc.client.Comment.Get(ctx, anchorID)
	if err != nil { return nil, err }
	if anchor.PostID != postID { return nil, fmt.Errorf("anchor does not belong to post") }
	if hasDeleted != nil && *hasDeleted != anchor.HasDeleted { /* допускаем, фильтрация ниже */ }

	base := uc.client.Comment.Query().Where(comment.PostIDEQ(postID))
	if hasDeleted != nil { base = base.Where(comment.HasDeletedEQ(*hasDeleted)) }

	// before (меньше якоря)
	beforeQ := base.Clone().
		Where(comment.Or(
			comment.CreatedAtLT(anchor.CreatedAt),
			comment.And(comment.CreatedAtEQ(anchor.CreatedAt), comment.IDLT(anchor.ID)),
		)).
		Order(ent.Desc(comment.FieldCreatedAt), ent.Desc(comment.FieldID)).
		Limit(beforeN)
	beforeItems, err := beforeQ.All(ctx)
	if err != nil { return nil, err }
	// реверс в ASC
	for i, j := 0, len(beforeItems)-1; i < j; i, j = i+1, j-1 { beforeItems[i], beforeItems[j] = beforeItems[j], beforeItems[i] }

	// after (больше якоря)
	afterQ := base.Clone().
		Where(comment.Or(
			comment.CreatedAtGT(anchor.CreatedAt),
			comment.And(comment.CreatedAtEQ(anchor.CreatedAt), comment.IDGT(anchor.ID)),
		)).
		Order(ent.Asc(comment.FieldCreatedAt), ent.Asc(comment.FieldID)).
		Limit(afterN)
	afterItems, err := afterQ.All(ctx)
	if err != nil { return nil, err }

	// собираем edges: before + anchor + after
	var edges []*models.CommentEdge
	for _, c := range beforeItems { edges = append(edges, &models.CommentEdge{ Cursor: encodeCursor(cursorKey(c)), Node: c }) }
	// включаем якорь, если не отфильтрован
	if hasDeleted == nil || anchor.HasDeleted == *hasDeleted {
		edges = append(edges, &models.CommentEdge{ Cursor: encodeCursor(cursorKey(anchor)), Node: anchor })
	}
	for _, c := range afterItems { edges = append(edges, &models.CommentEdge{ Cursor: encodeCursor(cursorKey(c)), Node: c }) }

	var startCur, endCur *string
	if len(edges) > 0 {
		s := edges[0].Cursor
		e := edges[len(edges)-1].Cursor
		startCur = &s
		endCur = &e
	}

	// флаги наличия страниц выше/ниже
	// предыдущие есть, если было ровно beforeN записей слева
	// следующие есть, если было ровно afterN записей справа
	hasPrev := len(beforeItems) == beforeN
	hasNext := len(afterItems) == afterN

	return &models.CommentsConnection{
		Edges: edges,
		PageInfo: &models.PageInfo{
			HasNextPage: hasNext,
			HasPreviousPage: hasPrev,
			StartCursor:    startCur,
			EndCursor:      endCur,
		},
	}, nil
}

// CommentsFeedConnection — двунаправленная keyset пагинация по всем комментариям
func (uc *commentUsecase) CommentsFeedConnection(ctx context.Context, hasDeleted *bool, first *int, after *string, last *int, before *string) (*models.CommentsConnection, error) {
    base := uc.client.Comment.Query().
        Where(comment.HasPostWith(post.VisibilityEQ(post.VisibilityPublished)))
    if hasDeleted != nil { base = base.Where(comment.HasDeletedEQ(*hasDeleted)) } else { base = base.Where(comment.HasDeletedEQ(false)) }

    var edges []*models.CommentEdge
    var startCur, endCur *string
    var hasPrev, hasNext bool

    if first != nil {
        // Вниз по ленте (к более старым), выдаём в DESC (новые сверху)
        lim := *first
        q := base.Clone()
        if after != nil && *after != "" {
            ta, ia, err := decodeCursor(*after)
            if err != nil { return nil, err }
            // После курсора в DESC-порядке = строго старее курсора
            q = q.Where(comment.Or(
                comment.CreatedAtLT(ta),
                comment.And(comment.CreatedAtEQ(ta), comment.IDLT(ia)),
            ))
        }
        items, err := q.Order(ent.Desc(comment.FieldCreatedAt), ent.Desc(comment.FieldID)).Limit(lim + 1).All(ctx)
        if err != nil { return nil, err }
        if len(items) > lim { hasNext = true; items = items[:lim] }
        for _, c := range items { cur := encodeCursor(cursorKey(c)); edges = append(edges, &models.CommentEdge{Cursor: cur, Node: c}) }
    } else if last != nil {
        // Вверх по ленте (к более новым), отдаём в DESC
        lim := *last
        q := base.Clone()
        if before != nil && *before != "" {
            tb, ib, err := decodeCursor(*before)
            if err != nil { return nil, err }
            // Перед курсором в DESC-порядке = строго новее курсора
            q = q.Where(comment.Or(
                comment.CreatedAtGT(tb),
                comment.And(comment.CreatedAtEQ(tb), comment.IDGT(ib)),
            ))
        }
        // Грузим ASC, чтобы затем взять ближние к курсору и развернуть в DESC
        items, err := q.Order(ent.Asc(comment.FieldCreatedAt), ent.Asc(comment.FieldID)).Limit(lim + 1).All(ctx)
        if err != nil { return nil, err }
        if len(items) > lim { hasPrev = true; items = items[:lim] }
        // Разворот в DESC для клиента
        for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 { items[i], items[j] = items[j], items[i] }
        for _, c := range items { cur := encodeCursor(cursorKey(c)); edges = append(edges, &models.CommentEdge{Cursor: cur, Node: c}) }
    } else {
        return &models.CommentsConnection{Edges: []*models.CommentEdge{}, PageInfo: &models.PageInfo{HasNextPage: false, HasPreviousPage: false}}, nil
    }

    if len(edges) > 0 { s := edges[0].Cursor; e := edges[len(edges)-1].Cursor; startCur = &s; endCur = &e }

    return &models.CommentsConnection{
        Edges: edges,
        PageInfo: &models.PageInfo{ HasNextPage: hasNext, HasPreviousPage: hasPrev, StartCursor: startCur, EndCursor: endCur },
    }, nil
}
