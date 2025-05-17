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
		field.Int("badge_id").Optional().Nillable(),
		field.String("color").Optional().Nillable(),

		field.Bool("community_roles_management").Default(false),
		field.Bool("host_user_ban").Default(false),
		field.Bool("host_user_mute").Default(false),
		field.Bool("host_community_delete_post").Default(false),
		field.Bool("host_community_remove_post_from_publication").Default(false),
		field.Bool("host_community_delete_comments").Default(false),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("badge", Media.Type).
			Field("badge_id").
			Unique(),

		edge.From("users", User.Type).
			Ref("host_roles"),
	}
}
