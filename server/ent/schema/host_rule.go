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

		field.Int("host_id").Optional().Nillable(),
		field.String("title").Optional().Nillable(),
		field.String("description").Optional().Nillable(),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).
			Ref("rules").
			Field("host_id").
			Unique(),
	}
}
