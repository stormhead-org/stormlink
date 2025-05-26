package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("name").NotEmpty(),
		field.String("slug").Unique().NotEmpty(),
		field.Int("avatar_id").Optional().Nillable(),
		field.Int("banner_id").Optional().Nillable(),
		field.String("description").Optional().Nillable(),

		field.String("email").Unique().NotEmpty(),
		field.String("password_hash").NotEmpty(),
		field.String("salt").NotEmpty(),
		field.Bool("is_verified").Default(false),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{

		edge.To("avatar", Media.Type).
			Field("avatar_id").
			Unique(),

		edge.To("banner", Media.Type).
			Field("banner_id").
			Unique(),

		edge.From("user_info", ProfileTableInfoItem.Type).
      Ref("user"),

		// Роли хоста (HostRole)
		edge.To("host_roles", HostRole.Type),

		// Роли в сообществах (Role)
		edge.To("communities_roles", Role.Type),

		// Баны и муты в сообществах
		edge.To("communities_bans", CommunityUserBan.Type),
		edge.To("communities_mutes", CommunityUserMute.Type),

		// Посты пользователя
		edge.To("posts", Post.Type),

		// Комментарии пользователя
		edge.To("comments", Comment.Type),

		// «Я на кого подписан»:
		edge.To("following", UserFollow.Type),

		// «На меня подписаны»:
		edge.To("followers", UserFollow.Type),

		// Подписка/фоллоу сообществ
		edge.To("communities_follow", CommunityFollow.Type),

		// Сообщества, которыми владеет пользователь
		edge.To("communities_owner", Community.Type),

		// Сообщества, в которых пользователь модератор
		edge.To("communities_moderator", CommunityModerator.Type),

		// Лайки к постам и комментариям
		edge.To("posts_likes", PostLike.Type),
		edge.To("comments_likes", CommentLike.Type),

		// Закладки (Post)
		edge.To("bookmarks", Bookmark.Type),

		// Связь с EmailVerification
		edge.To("email_verifications", EmailVerification.Type),
	}
}
