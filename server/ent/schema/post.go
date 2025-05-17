package schema

import (
	"github.com/google/uuid"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Post struct{ ent.Schema }

func (Post) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("title"),
		field.JSON("content", map[string]interface{}{}),

		field.Int("hero_image_id").Optional().Nillable(),
		field.Int("community_id"),
		field.UUID("author_id", uuid.New()).Default(uuid.New),

		field.JSON("meta", []struct {
			Title       *string `json:"title,omitempty"`
			Description *string `json:"description,omitempty"`
		}{}).
			Optional().
			StructTag(`json:"meta,omitempty"`),

		field.Int("views").Default(0),
		field.Bool("has_deleted").Default(false),
		field.Time("published_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Post) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("heroImage", Media.Type).
			Field("hero_image_id").
			Unique(),

		edge.To("comments", Comment.Type),
		edge.To("related_post", Post.Type),

		edge.From("community", Community.Type).
			Ref("posts").
			Field("community_id").
			Required().
			Unique(),

		edge.From("author", User.Type).
			Ref("posts").
			Field("author_id").
			Required().
			Unique(),

		edge.To("likes", PostLike.Type),
		edge.To("bookmarks", Bookmark.Type),
	}
}
