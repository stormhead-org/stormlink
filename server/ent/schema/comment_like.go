package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CommentLike struct{ ent.Schema }

func (CommentLike) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("user_id"),
		field.Int("comment_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CommentLike) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("comments_likes").
			Field("user_id").
			Required().
			Unique(),

		edge.From("comment", Comment.Type).
			Ref("likes").
			Field("comment_id").
			Required().
			Unique(),
	}
}
