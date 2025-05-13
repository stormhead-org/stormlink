package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailVerification holds the schema definition for the EmailVerification entity.
type EmailVerification struct {
    ent.Schema
}

// Fields of the EmailVerification.
func (EmailVerification) Fields() []ent.Field {
    return []ent.Field{
        field.String("token").Unique(),
        field.Time("expires_at"),
        field.Time("created_at").Default(time.Now),
    }
}

// Edges of the EmailVerification.
func (EmailVerification) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("user", User.Type).
            Ref("email_verifications").
            Unique(),
    }
}

// Indexes of the EmailVerification.
func (EmailVerification) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("token").
            Unique(),
    }
}