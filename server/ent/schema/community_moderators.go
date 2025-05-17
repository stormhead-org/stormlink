package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type FollowCommunity struct{ ent.Schema }

func (FollowCommunity) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FollowCommunity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("follow_communities").
			Unique(),
		edge.From("community", Community.Type).
			Ref("followers").
			Unique(),
	}
}
