package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Media holds the schema definition for the Media entity.
type HostSocialNavigation struct {
	ent.Schema
}

// Fields of the Media.
func (HostSocialNavigation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.String("github").Optional().Nillable(),
		field.String("site").Optional().Nillable(),
		field.String("telegram").Optional().Nillable(),
		field.String("instagram").Optional().Nillable(),
		field.String("twitter").Optional().Nillable(),
		field.String("mastodon").Optional().Nillable(),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Media.
func (HostSocialNavigation) Edges() []ent.Edge {
	return nil
}
