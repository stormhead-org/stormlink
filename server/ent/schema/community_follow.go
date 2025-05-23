package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CommunityFollow struct{ ent.Schema }

func (CommunityFollow) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("user_id"),
		field.Int("community_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CommunityFollow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("communities_follow").
			Field("user_id").
			Required().
			Unique(),

		edge.From("community", Community.Type).
			Ref("followers").
			Field("community_id").
			Required().
			Unique(),
	}
}
