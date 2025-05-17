package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CommunityUsersMute struct{ ent.Schema }

func (CommunityUsersMute) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("user_id"),
		field.Int("community_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CommunityUsersMute) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("communities_mutes").
			Field("user_id").
			Required().
			Unique(),
			
		edge.From("community", Community.Type).
			Ref("mutes").
			Field("community_id").
			Required().
			Unique(),
	}
}
