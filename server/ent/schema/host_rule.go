package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type HostRule struct {
	ent.Schema
}

func (HostRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("rule_id").Optional().Nillable(),
		field.String("name_rule").Optional().Nillable(),
		field.String("description_rule").Optional().Nillable(),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).
			Ref("rules").
			Field("rule_id").
			Unique(),
	}
}
