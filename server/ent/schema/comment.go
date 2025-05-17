package schema

import (
	"github.com/google/uuid"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Comment struct{ ent.Schema }

func (Comment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.UUID("author_id", uuid.New()).Default(uuid.New),
		field.Int("post_id"),
		field.Int("community_id"),
		field.Int("parent_comment_id").Optional().Nillable(),
		field.Int("media_id").Optional().Nillable(),

		field.Bool("has_deleted").Default(false),
		field.Bool("has_updated").Default(false),

		field.String("content"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Comment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("author", User.Type).
			Ref("comments").
			Field("author_id").
			Required().
			Unique(),

		edge.From("post", Post.Type).
			Ref("comments").
			Field("post_id").
			Required().
			Unique(),

		edge.From("community", Community.Type).
			Ref("comments").
			Field("community_id").
			Required().
			Unique(),

		edge.To("media", Media.Type).
			Field("media_id").
			Unique(),

		edge.From("parent_comment", Comment.Type).
			Ref("children_comment").
			Field("parent_comment_id").
			Unique(),

		edge.To("children_comment", Comment.Type),

		edge.To("likes", CommentLike.Type),
	}
}
