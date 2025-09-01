package community

import (
	"context"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/graphql/models"
	"stormlink/tests/fixtures"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestClient(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return client
}

func TestCommunityUsecase_GetCommunityByID(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("existing community", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, fixtures.TestCommunity1.ID, community.ID)
		assert.Equal(t, fixtures.TestCommunity1.Name, community.Name)
		assert.Equal(t, fixtures.TestCommunity1.Slug, community.Slug)
		assert.Equal(t, fixtures.TestCommunity1.Description, community.Description)
		assert.Equal(t, fixtures.TestCommunity1.IsPrivate, community.IsPrivate)
		assert.Equal(t, fixtures.TestCommunity1.OwnerID, community.OwnerID)
	})

	t.Run("non-existing community", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, community)
		assert.True(t, ent.IsNotFound(err))
	})

	t.Run("private community", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, fixtures.PrivateCommunity.ID, community.ID)
		assert.Equal(t, fixtures.PrivateCommunity.Name, community.Name)
		assert.True(t, community.IsPrivate)
		assert.Equal(t, fixtures.PrivateCommunity.OwnerID, community.OwnerID)
	})

	t.Run("community with logo", func(t *testing.T) {
		// Create media for logo
		logo, err := fixtures.CreateTestMedia(ctx, client, "logo.png", "https://example.com/logo.png")
		require.NoError(t, err)

		// Create community with logo
		communityWithLogo, err := client.Community.Create().
			SetName("Community With Logo").
			SetSlug("community-with-logo").
			SetDescription("Community that has a logo").
			SetIsPrivate(false).
			SetOwnerID(fixtures.TestUser1.ID).
			SetLogoID(logo.ID).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		retrievedCommunity, err := uc.GetCommunityByID(ctx, communityWithLogo.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCommunity)

		// Load logo edge and verify
		logoEdge, err := retrievedCommunity.QueryLogo().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, logo.ID, logoEdge.ID)
		assert.Equal(t, "logo.png", logoEdge.Filename)
	})

	t.Run("community with community info", func(t *testing.T) {
		// Create community info
		communityInfo, err := client.CommunityInfo.Create().
			SetRules("Community rules").
			SetDescription("Extended description").
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Create community with info
		communityWithInfo, err := client.Community.Create().
			SetName("Community With Info").
			SetSlug("community-with-info").
			SetDescription("Community with extended info").
			SetIsPrivate(false).
			SetOwnerID(fixtures.TestUser1.ID).
			SetCommunityInfoID(communityInfo.ID).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		retrievedCommunity, err := uc.GetCommunityByID(ctx, communityWithInfo.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCommunity)

		// Load community info edge and verify
		infoEdge, err := retrievedCommunity.QueryCommunityInfo().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, communityInfo.ID, infoEdge.ID)
		assert.Equal(t, "Community rules", infoEdge.Rules)
	})

	t.Run("community with roles", func(t *testing.T) {
		// Create roles for the community
		moderatorRole, err := client.Role.Create().
			SetName("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"delete_posts", "ban_users"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		memberRole, err := client.Role.Create().
			SetName("Member").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"create_posts", "comment"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		retrievedCommunity, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCommunity)

		// Load roles edge and verify
		roles, err := retrievedCommunity.QueryRoles().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, roles, 2)

		// Verify both roles are present
		roleNames := make(map[string]bool)
		for _, role := range roles {
			roleNames[role.Name] = true
			assert.Equal(t, fixtures.TestCommunity1.ID, role.CommunityID)
		}
		assert.True(t, roleNames["Moderator"])
		assert.True(t, roleNames["Member"])

		// Verify permissions
		for _, role := range roles {
			if role.ID == moderatorRole.ID {
				assert.Contains(t, role.Permissions, "delete_posts")
				assert.Contains(t, role.Permissions, "ban_users")
			}
			if role.ID == memberRole.ID {
				assert.Contains(t, role.Permissions, "create_posts")
				assert.Contains(t, role.Permissions, "comment")
			}
		}
	})
}

func TestCommunityUsecase_GetCommunityStatus(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("community status for owner", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, fixtures.TestCommunity1.OwnerID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipOwner, status.Relationship)
	})

	t.Run("community status for non-member", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipNone, status.Relationship)
	})

	t.Run("user following community", func(t *testing.T) {
		// Create follow relationship
		_, err := client.CommunityFollow.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.True(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipFollowing, status.Relationship)
	})

	t.Run("user with moderator role", func(t *testing.T) {
		// Create moderator role
		moderatorRole, err := client.Role.Create().
			SetName("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"delete_posts", "ban_users"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Create community moderator relationship
		_, err = client.CommunityModerator.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetRoleID(moderatorRole.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing) // Following should be separate from moderation
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipModerator, status.Relationship)
	})

	t.Run("banned user", func(t *testing.T) {
		// Create ban
		_, err := client.CommunityUserBan.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetReason("Spam").
			SetCreatedAt(time.Now()).
			SetExpiresAt(time.Now().Add(24 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.True(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipBanned, status.Relationship)
	})

	t.Run("muted user", func(t *testing.T) {
		// Create mute
		_, err := client.CommunityUserMute.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetReason("Too chatty").
			SetCreatedAt(time.Now()).
			SetExpiresAt(time.Now().Add(2 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.True(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipMuted, status.Relationship)
	})

	t.Run("anonymous user viewing community", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, 0, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipNone, status.Relationship)
	})

	t.Run("non-existing community", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, 99999)

		assert.Error(t, err)
		assert.Nil(t, status)
	})

	t.Run("non-existing user", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, 99999, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipNone, status.Relationship)
	})
}

func TestCommunityUsecase_CommunityWithComplexRelationships(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("community with all relationships", func(t *testing.T) {
		// Create logo
		logo, err := fixtures.CreateTestMedia(ctx, client, "community-logo.png", "https://example.com/community-logo.png")
		require.NoError(t, err)

		// Create community info
		communityInfo, err := client.CommunityInfo.Create().
			SetRules("1. Be respectful\n2. No spam\n3. Stay on topic").
			SetDescription("Extended community description with more details").
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Create roles
		adminRole, err := client.Role.Create().
			SetName("Admin").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"all"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		moderatorRole, err := client.Role.Create().
			SetName("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"delete_posts", "ban_users", "mute_users"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		memberRole, err := client.Role.Create().
			SetName("Member").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"create_posts", "comment", "like"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Update community with all relationships
		_, err = client.Community.UpdateOneID(fixtures.TestCommunity1.ID).
			SetLogoID(logo.ID).
			SetCommunityInfoID(communityInfo.ID).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		retrievedCommunity, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCommunity)

		// Verify logo
		logoEdge, err := retrievedCommunity.QueryLogo().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, logo.ID, logoEdge.ID)

		// Verify community info
		infoEdge, err := retrievedCommunity.QueryCommunityInfo().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, communityInfo.ID, infoEdge.ID)
		assert.Contains(t, infoEdge.Rules, "Be respectful")

		// Verify roles
		roles, err := retrievedCommunity.QueryRoles().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, roles, 3)

		roleNames := make(map[string]*ent.Role)
		for _, role := range roles {
			roleNames[role.Name] = role
		}

		// Verify admin role
		assert.Contains(t, roleNames, "Admin")
		assert.Equal(t, adminRole.ID, roleNames["Admin"].ID)
		assert.Contains(t, roleNames["Admin"].Permissions, "all")

		// Verify moderator role
		assert.Contains(t, roleNames, "Moderator")
		assert.Equal(t, moderatorRole.ID, roleNames["Moderator"].ID)
		assert.Contains(t, roleNames["Moderator"].Permissions, "delete_posts")

		// Verify member role
		assert.Contains(t, roleNames, "Member")
		assert.Equal(t, memberRole.ID, roleNames["Member"].ID)
		assert.Contains(t, roleNames["Member"].Permissions, "create_posts")
	})
}

func TestCommunityUsecase_CommunityStatusComplexScenarios(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("user both following and moderating", func(t *testing.T) {
		// Create follow relationship
		_, err := client.CommunityFollow.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Create moderator role
		moderatorRole, err := client.Role.Create().
			SetName("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"delete_posts"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Add user as moderator
		_, err = client.CommunityModerator.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetRoleID(moderatorRole.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.True(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
		// Moderator relationship should take precedence
		assert.Equal(t, models.CommunityStatusRelationshipModerator, status.Relationship)
	})

	t.Run("expired ban should not affect status", func(t *testing.T) {
		// Create expired ban
		_, err := client.CommunityUserBan.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetReason("Old violation").
			SetCreatedAt(time.Now().Add(-48 * time.Hour)).
			SetExpiresAt(time.Now().Add(-24 * time.Hour)). // Expired
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsOwn) // User1 is owner of TestCommunity1
		assert.False(t, status.IsBanned) // Ban is expired
		assert.Equal(t, models.CommunityStatusRelationshipOwner, status.Relationship)
	})

	t.Run("active ban overrides other relationships", func(t *testing.T) {
		// Create follow relationship first
		_, err := client.CommunityFollow.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Create active ban
		_, err = client.CommunityUserBan.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetReason("Spam").
			SetCreatedAt(time.Now()).
			SetExpiresAt(time.Now().Add(24 * time.Hour)). // Active
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.True(t, status.IsFollowing) // Should still show following
		assert.True(t, status.IsBanned)    // But also banned
		assert.False(t, status.IsMuted)
		assert.Equal(t, models.CommunityStatusRelationshipBanned, status.Relationship) // Ban takes precedence
	})

	t.Run("expired mute should not affect status", func(t *testing.T) {
		// Create expired mute
		_, err := client.CommunityUserMute.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetReason("Old chatty behavior").
			SetCreatedAt(time.Now().Add(-48 * time.Hour)).
			SetExpiresAt(time.Now().Add(-1 * time.Hour)). // Expired
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsMuted) // Mute is expired
		assert.Equal(t, models.CommunityStatusRelationshipNone, status.Relationship)
	})

	t.Run("active mute affects status", func(t *testing.T) {
		// Create active mute
		_, err := client.CommunityUserMute.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetReason("Too chatty").
			SetCreatedAt(time.Now()).
			SetExpiresAt(time.Now().Add(2 * time.Hour)). // Active
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsOwn)   // Still owner
		assert.True(t, status.IsMuted) // But muted
		// Owner relationship should take precedence over mute
		assert.Equal(t, models.CommunityStatusRelationshipOwner, status.Relationship)
	})
}

func TestCommunityUsecase_PrivateCommunityAccess(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("owner can access private community", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.True(t, community.IsPrivate)
		assert.Equal(t, fixtures.PrivateCommunity.OwnerID, community.OwnerID)
	})

	t.Run("non-member can still get private community data", func(t *testing.T) {
		// Note: Access control should be handled at a higher level (resolver/middleware)
		// The usecase just retrieves data, so this should succeed
		community, err := uc.GetCommunityByID(ctx, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.True(t, community.IsPrivate)
	})
}

func TestCommunityUsecase_CommunityRoleHierarchy(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("role hierarchy and permissions", func(t *testing.T) {
		// Create role hierarchy: Admin > Moderator > Member
		adminRole, err := client.Role.Create().
			SetName("Admin").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"all", "manage_community", "delete_posts", "ban_users", "mute_users", "manage_roles"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		moderatorRole, err := client.Role.Create().
			SetName("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"delete_posts", "ban_users", "mute_users"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		memberRole, err := client.Role.Create().
			SetName("Member").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"create_posts", "comment", "like", "bookmark"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		readOnlyRole, err := client.Role.Create().
			SetName("Read Only").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetPermissions([]string{"view"}).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		community, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)
		assert.NoError(t, err)
		assert.NotNil(t, community)

		// Load roles and verify hierarchy
		roles, err := community.QueryRoles().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, roles, 4)

		// Create a map for easier verification
		roleMap := make(map[string]*ent.Role)
		for _, role := range roles {
			roleMap[role.Name] = role
		}

		// Verify admin permissions
		admin := roleMap["Admin"]
		assert.NotNil(t, admin)
		assert.Contains(t, admin.Permissions, "all")
		assert.Contains(t, admin.Permissions, "manage_community")

		// Verify moderator permissions
		moderator := roleMap["Moderator"]
		assert.NotNil(t, moderator)
		assert.Contains(t, moderator.Permissions, "delete_posts")
		assert.Contains(t, moderator.Permissions, "ban_users")
		assert.NotContains(t, moderator.Permissions, "manage_community")

		// Verify member permissions
		member := roleMap["Member"]
		assert.NotNil(t, member)
		assert.Contains(t, member.Permissions, "create_posts")
		assert.Contains(t, member.Permissions, "comment")
		assert.NotContains(t, member.Permissions, "delete_posts")

		// Verify read-only permissions
		readOnly := roleMap["Read Only"]
		assert.NotNil(t, readOnly)
		assert.Contains(t, readOnly.Permissions, "view")
		assert.NotContains(t, readOnly.Permissions, "create_posts")
	})
}

func TestCommunityUsecase_EdgeCases(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	t.Run("context cancellation during GetCommunityByID", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		community, err := uc.GetCommunityByID(cancelledCtx,
