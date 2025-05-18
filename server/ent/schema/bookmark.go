package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Bookmark struct{ ent.Schema }

func (Bookmark) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("user_id"),
		field.Int("post_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Bookmark) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("bookmarks").
			Field("user_id").
			Required().
			Unique(),

		edge.From("post", Post.Type).
			Ref("bookmarks").
			Field("post_id").
			Required().
			Unique(),
	}
}
