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

		field.Int("rule_id").Optional().Nillable(),
		field.String("community_name_rule").Optional().Nillable(),
		field.String("community_description_rule").Optional().Nillable(),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CommunityRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("community", Community.Type).
			Ref("rules").
			Field("rule_id").
			Unique(),
	}
}
