package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type ProfileTableInfoItem struct {
	ent.Schema
}

func (ProfileTableInfoItem) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("key").NotEmpty(),
		field.String("value").NotEmpty(),

		field.Int("community_id").Optional(),
		field.Int("user_id").Optional(),

		field.Enum("type").
			Values("user", "community"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ProfileTableInfoItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("community", Community.Type).
      Field("community_id").
      Unique(),

    edge.To("user", User.Type).
      Field("user_id").
      Unique(),
	}
}
