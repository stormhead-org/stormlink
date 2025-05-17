package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

type HostSidebarNavigationItem struct {
	ent.Schema
}

func (HostSidebarNavigationItem) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),

		field.Int("sidebar_navigation_id"),
		field.Int("post_id"),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (HostSidebarNavigationItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sidebar_navigation", HostSidebarNavigation.Type).
			Ref("items").
			Field("sidebar_navigation_id").
			Required().
			Unique(),

		edge.To("post", Post.Type).
			Field("post_id").
			Required().
			Unique(),
	}
}
