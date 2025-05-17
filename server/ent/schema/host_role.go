package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type HostRole struct{ ent.Schema }

func (HostRole) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("name"),
		field.String("color").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("badge_media", Media.Type).
			Ref("hostrole_badge").
			Field("id").
			Unique(),
		edge.To("users", User.Type), // many Users can have this host role
	}
}
