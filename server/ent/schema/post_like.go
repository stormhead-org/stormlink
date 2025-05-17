package schema

import (
	"github.com/google/uuid"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PostLike struct{ ent.Schema }

func (PostLike) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.UUID("user_id", uuid.New()).Default(uuid.New),
		field.Int("post_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PostLike) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("posts_likes").
			Field("user_id").
			Required().
			Unique(),

		edge.From("post", Post.Type).
			Ref("likes").
			Field("post_id").
			Required().
			Unique(),
	}
}
