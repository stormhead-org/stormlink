package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Media holds the schema definition for the Media entity.
type Media struct {
	ent.Schema
}

// Fields of the Media.
func (Media) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("alt").Optional().Nillable(),
		field.String("url").Optional().Nillable(),
		field.String("thumbnail_url").Optional().Nillable(),
		field.String("filename").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Media.
func (Media) Edges() []ent.Edge {
	return nil
}
