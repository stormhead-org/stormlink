package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Host struct{ ent.Schema }

func (Host) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("title").Optional().Nillable(),
		field.String("slogan").Optional().Nillable(),
		field.String("contacts").Optional().Nillable(),
		field.String("description").Optional().Nillable(),

		field.Int("logo_id").Optional().Nillable(),
		field.Int("banner_id").Optional().Nillable(),
		field.Int("auth_banner_id").Optional().Nillable(),
		field.Int("owner_id").Optional().Nillable(),

		field.Bool("first_settings").Default(true),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Host) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("logo", Media.Type).
			Field("logo_id").
			Unique(),

		edge.To("banner", Media.Type).
			Field("banner_id").
			Unique(),

		edge.To("auth_banner", Media.Type).
			Field("auth_banner_id").
			Unique(),

		edge.To("owner", User.Type).
			Field("owner_id").
			Unique(),

		edge.To("rules", HostRule.Type),
	}
}
