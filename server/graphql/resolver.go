package graphql

import (
	"context"

	"stormlink/server/ent"
	"stormlink/server/ent/community"
	"stormlink/server/ent/user"
)

type Resolver struct {
	Client *ent.Client
}

var _ QueryResolver = (*Resolver)(nil)

// Получить сообщества, с опцией onlyNotBanned (по умолчанию true)
func (r *Resolver) Communities(ctx context.Context, onlyNotBanned *bool) ([]*ent.Community, error) {
	q := r.Client.Community.Query()
	if onlyNotBanned == nil || *onlyNotBanned {
		q = q.Where(community.CommunityHasBanned(false))
	}
	return q.Order(ent.Asc("id")).All(ctx)
}

// Получить всех юзеров
func (r *Resolver) Users(ctx context.Context) ([]*ent.User, error) {
	return r.Client.User.
		Query().
		Order(ent.Asc(user.FieldID)).
		All(ctx)
}

// Заглушка для Node — если нуждаетесь, можно делегировать entgql или реализовать сами
func (r *Resolver) Node(ctx context.Context, id string) (ent.Noder, error) {
	// Простая заглушка:
	return nil, nil
}

// Заглушка для Nodes
func (r *Resolver) Nodes(ctx context.Context, ids []string) ([]ent.Noder, error) {
	return nil, nil
}
