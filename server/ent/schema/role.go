package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Role struct{ ent.Schema }

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("name"),
		field.Int("badge_id").Optional().Nillable(),
		field.String("color").Optional().Nillable(),
		field.Int("community_id"),

		field.Bool("community_roles_management").Default(false),
		field.Bool("community_user_ban").Default(false),
		field.Bool("community_user_mute").Default(false),
		field.Bool("community_delete_post").Default(false),
		field.Bool("community_remove_post_from_publication").Default(false),
		field.Bool("community_delete_comments").Default(false),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("badge", Media.Type).
			Field("badge_id").
			Unique(),

		edge.From("community", Community.Type).
			Field("community_id").
			Ref("roles").
			Required().
			Unique(),

		edge.From("users", User.Type).
			Ref("communities_roles"),
	}
}
