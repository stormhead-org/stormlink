package schema

import (
	"github.com/google/uuid"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Community struct{ ent.Schema }

func (Community) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.Int("logo_id").Optional().Nillable(),
		field.Int("banner_id").Optional().Nillable(),
		field.UUID("owner_id", uuid.New()).Default(uuid.New),
		field.String("title"),
		field.String("contacts").Optional().Nillable(),
		field.String("description").Optional().Nillable(),
		field.JSON("table_info", []struct {
			Label string  `json:"label"`
			Value string  `json:"value"`
			ID    *string `json:"id,omitempty"`
		}{}).
			Optional().
			StructTag(`json:"table_info,omitempty"`),

		field.Bool("community_has_banned").Default(false),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Community) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("logo", Media.Type).
			Field("logo_id").
			Unique(),

		edge.To("banner", Media.Type).
			Field("banner_id").
			Unique(),

		edge.From("owner", User.Type).
			Ref("communities_owner").
			Field("owner_id").
			Required().
			Unique(),

		edge.To("moderators", CommunityModerators.Type),
		edge.To("roles", Role.Type),
		edge.To("rules", CommunityRule.Type),
		edge.To("followers", CommunityFollow.Type),
		edge.To("bans", CommunityUsersBan.Type),
		edge.To("mutes", CommunityUsersMute.Type),
		edge.To("posts", Post.Type),
		edge.To("comments", Comment.Type),
	}
}
