package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type HostSidebarNavigation struct {
	ent.Schema
}

func (HostSidebarNavigation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostSidebarNavigation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("items", HostSidebarNavigationItem.Type),
	}
}
