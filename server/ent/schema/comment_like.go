package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type LikeComment struct{ ent.Schema }

func (LikeComment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (LikeComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("comment_likes").
			Unique(),
		edge.From("comment", Comment.Type).
			Ref("likes").
			Unique(),
	}
}
