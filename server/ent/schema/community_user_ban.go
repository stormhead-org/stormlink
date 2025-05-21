package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CommunityUserBan struct{ ent.Schema }

func (CommunityUserBan) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("user_id"),
		field.Int("community_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CommunityUserBan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("communities_bans").
			Field("user_id").
			Required().
			Unique(),

		edge.From("community", Community.Type).
			Ref("bans").
			Field("community_id").
			Required().
			Unique(),
	}
}
