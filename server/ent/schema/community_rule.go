package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CommunityRule struct {
	ent.Schema
}

func (CommunityRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("community_id").Optional().Nillable(),
		field.String("title").NotEmpty(),
		field.String("description").Optional(),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CommunityRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("community", Community.Type).
			Ref("rules").
			Field("community_id").
			Unique(),
	}
}
