package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"time"
)

// schema/userfollow.go
type UserFollow struct{ ent.Schema }

func (UserFollow) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.UUID("follower_id", uuid.New()).Default(uuid.New),
		field.UUID("followee_id", uuid.New()).Default(uuid.New),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (UserFollow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("follower", User.Type).
			Ref("following").
			Field("follower_id").
			Required().
			Unique(),

		edge.From("followee", User.Type).
			Ref("followers").
			Field("followee_id").
			Required().
			Unique(),
	}
}
