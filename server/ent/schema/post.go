package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Post holds the schema definition for the Post entity.
type Post struct {
	ent.Schema
}

// Fields of the Post.
func (Post) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").NotEmpty(),
		field.String("content").NotEmpty(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the Post.
func (Post) Edges() []ent.Edge {
	return nil
}
