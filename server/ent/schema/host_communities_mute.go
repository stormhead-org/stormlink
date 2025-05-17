package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type HostCommunitiesMute struct{ ent.Schema }

func (HostCommunitiesMute) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("community_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostCommunitiesMute) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("community", Community.Type).
			Field("community_id").
			Required().
			Unique(),
	}
}
