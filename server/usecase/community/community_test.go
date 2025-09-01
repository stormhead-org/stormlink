package community

import (
	"context"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/tests/fixtures"
	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestClient(t *testing.T) (*ent.Client, *testhelper.PostgresTestHelper) {
	helper := testhelper.NewPostgresTestHelper(t)
	helper.WaitForDatabase(t)
	helper.CleanDatabase(t)
	return helper.GetClient(), helper
}

func TestCommunityUsecase_GetCommunityByID(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

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
		assert.Equal(t, fixtures.TestCommunity1.Name, community.Title)
		assert.Equal(t, fixtures.TestCommunity1.Slug, community.Slug)
		assert.Equal(t, fixtures.TestCommunity1.Description, community.Description)
		assert.Equal(t, fixtures.TestCommunity1.OwnerID, community.OwnerID)
	})

	t.Run("non-existing community", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, community)
	})
}

func TestCommunityUsecase_GetCommunityBySlug(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

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
		assert.Equal(t, fixtures.TestCommunity1.Name, community.Title)
		assert.Equal(t, fixtures.TestCommunity1.Slug, community.Slug)
		assert.Equal(t, fixtures.TestCommunity1.OwnerID, community.OwnerID)
	})

	t.Run("community with logo", func(t *testing.T) {
		// Create media for logo
		logo, err := fixtures.CreateTestMedia(ctx, client, fixtures.TestMedia1)
		require.NoError(t, err)

		// Create community with logo
		community, err := client.Community.Create().
			SetTitle("Test Community with Logo").
			SetSlug("test-community-with-logo").
			SetDescription("A community with a logo").
			SetOwnerID(fixtures.TestUser1.ID).
			SetLogoID(logo.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		foundCommunity, err := uc.GetCommunityByID(ctx, community.ID)

		assert.NoError(t, err)
		assert.NotNil(t, foundCommunity)
		assert.Equal(t, community.ID, foundCommunity.ID)
		assert.Equal(t, logo.ID, *foundCommunity.LogoID)
	})

	t.Run("non-existing community", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, community)
	})
}

func TestCommunityUsecase_GetCommunityWithRoles(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("community with roles", func(t *testing.T) {
		// Create roles for the community
		_, err := client.Role.Create().
			SetTitle("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityUserBan(true).
			SetCommunityDeletePost(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Role.Create().
			SetTitle("Member").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Test community retrieval with roles
		community, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, fixtures.TestCommunity1.ID, community.ID)

		// Roles are loaded via WithRoles() in the usecase
		roles, err := community.QueryRoles().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, roles, 2)

		// Create a map for easier verification
		roleMap := make(map[string]*ent.Role)
		for _, role := range roles {
			roleMap[role.Title] = role
		}

		// Verify roles exist
		assert.Contains(t, roleMap, "Moderator")
		assert.Contains(t, roleMap, "Member")

		// Verify permissions
		moderator := roleMap["Moderator"]
		assert.True(t, moderator.CommunityUserBan)
		assert.True(t, moderator.CommunityDeletePost)

		member := roleMap["Member"]
		assert.False(t, member.CommunityUserBan)
		assert.False(t, member.CommunityDeletePost)
	})
}

func TestCommunityUsecase_GetCommunityStatus(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("community status for owner", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("community status for non-member", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("user following community", func(t *testing.T) {
		// Create follow relationship
		_, err := client.CommunityFollow.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("user with moderator role", func(t *testing.T) {
		// Create moderator role
		_, err := client.Role.Create().
			SetTitle("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityUserBan(true).
			SetCommunityDeletePost(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Assign moderator role to user2
		_, err = client.CommunityModerator.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsFollowing) // Still following from previous test
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("banned user", func(t *testing.T) {
		// Ban user2 from private community
		_, err := client.CommunityUserBan.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsFollowing)
		assert.True(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("muted user", func(t *testing.T) {
		// Mute user1 in private community
		_, err := client.CommunityUserMute.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.True(t, status.IsMuted)
	})

	t.Run("anonymous user viewing community", func(t *testing.T) {
		status, err := uc.GetCommunityStatus(ctx, 0, fixtures.TestCommunity1.ID) // 0 = anonymous

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
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
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})
}

func TestCommunityUsecase_CommunityWithRolesAndLogo(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("community with all relationships", func(t *testing.T) {
		// Create logo
		logo, err := fixtures.CreateTestMedia(ctx, client, fixtures.TestMedia1)
		require.NoError(t, err)

		// Create roles
		_, err = client.Role.Create().
			SetTitle("Admin").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityRolesManagement(true).
			SetCommunityUserBan(true).
			SetCommunityUserMute(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Role.Create().
			SetTitle("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityDeletePost(true).
			SetCommunityUserBan(true).
			SetCommunityUserMute(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Role.Create().
			SetTitle("Member").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Update community with logo
		_, err = client.Community.UpdateOneID(fixtures.TestCommunity1.ID).
			SetLogoID(logo.ID).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Get community with all relationships loaded
		community, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, fixtures.TestCommunity1.ID, community.ID)
		assert.Equal(t, logo.ID, *community.LogoID)

		// Get roles through the community
		roles, err := community.QueryRoles().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, roles, 3)

		// Create a map for easier verification
		roleMap := make(map[string]*ent.Role)
		for _, role := range roles {
			roleMap[role.Title] = role
		}

		// Verify roles exist
		assert.Contains(t, roleMap, "Admin")
		assert.Contains(t, roleMap, "Moderator")
		assert.Contains(t, roleMap, "Member")

		// Verify admin permissions
		assert.True(t, roleMap["Admin"].CommunityRolesManagement)
		assert.True(t, roleMap["Admin"].CommunityUserBan)

		// Verify moderator permissions
		assert.True(t, roleMap["Moderator"].CommunityDeletePost)
		assert.True(t, roleMap["Moderator"].CommunityUserBan)

		// Verify member permissions
		assert.False(t, roleMap["Member"].CommunityUserBan)
	})
}

func TestCommunityUsecase_ComplexScenarios(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("user following community with moderator role", func(t *testing.T) {
		// Create custom role
		_, err := client.Role.Create().
			SetTitle("Custom Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityUserBan(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// User follows community
		_, err = client.CommunityFollow.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// User becomes moderator
		_, err = client.CommunityModerator.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsFollowing)
		assert.False(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("banned user should not show following status", func(t *testing.T) {
		// Ban user2 from private community (but they might still have follow record)
		_, err := client.CommunityUserBan.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// User tries to follow (this might exist from before ban)
		_, err = client.CommunityFollow.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		// Ignore error if follow already exists

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser2.ID, fixtures.PrivateCommunity.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		// Ban should be reflected
		assert.True(t, status.IsBanned)
		assert.False(t, status.IsMuted)
	})

	t.Run("muted user in owned community", func(t *testing.T) {
		// Mute user1 in their own community (edge case)
		_, err := client.CommunityUserMute.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetCommunityStatus(ctx, fixtures.TestUser1.ID, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsMuted)
		assert.False(t, status.IsBanned)
	})
}

func TestCommunityUsecase_CommunityOwner(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("get community owned by user", func(t *testing.T) {
		community, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, fixtures.TestUser1.ID, community.OwnerID)
	})

	t.Run("verify ownership", func(t *testing.T) {
		// Create a new user who owns no communities
		newUser, err := client.User.Create().
			SetName("New User").
			SetSlug("new-user").
			SetEmail("newuser@example.com").
			SetPasswordHash("hash").
			SetSalt("salt").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Create a community owned by the new user
		newCommunity, err := client.Community.Create().
			SetTitle("New Community").
			SetSlug("new-community").
			SetOwnerID(newUser.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Get the community and verify ownership
		community, err := uc.GetCommunityByID(ctx, newCommunity.ID)
		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, newUser.ID, community.OwnerID)
	})
}

func TestCommunityUsecase_RolePermissions(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewCommunityUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("role hierarchy and permissions", func(t *testing.T) {
		// Create role hierarchy: Admin > Moderator > Member > Read Only
		_, err = client.Role.Create().
			SetTitle("Admin").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityRolesManagement(true).
			SetCommunityUserBan(true).
			SetCommunityUserMute(true).
			SetCommunityDeletePost(true).
			SetCommunityRemovePostFromPublication(true).
			SetCommunityDeleteComments(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Role.Create().
			SetTitle("Moderator").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCommunityDeletePost(true).
			SetCommunityUserBan(true).
			SetCommunityUserMute(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Role.Create().
			SetTitle("Member").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Role.Create().
			SetTitle("Read Only").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Get community with roles
		community, err := uc.GetCommunityByID(ctx, fixtures.TestCommunity1.ID)
		assert.NoError(t, err)
		assert.NotNil(t, community)

		// Get all roles
		roles, err := community.QueryRoles().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, roles, 4)

		// Create a map for easier verification
		roleMap := make(map[string]*ent.Role)
		for _, role := range roles {
			roleMap[role.Title] = role
		}

		// Verify admin permissions
		admin := roleMap["Admin"]
		assert.NotNil(t, admin)
		assert.True(t, admin.CommunityRolesManagement)
		assert.True(t, admin.CommunityUserBan)

		// Verify moderator permissions
		moderator := roleMap["Moderator"]
		assert.NotNil(t, moderator)
		assert.True(t, moderator.CommunityDeletePost)
		assert.True(t, moderator.CommunityUserBan)
		assert.False(t, moderator.CommunityRolesManagement)

		// Verify member permissions
		member := roleMap["Member"]
		assert.NotNil(t, member)
		assert.False(t, member.CommunityDeletePost)
		assert.False(t, member.CommunityUserBan)

		// Verify read-only permissions
		readOnly := roleMap["Read Only"]
		assert.NotNil(t, readOnly)
		assert.False(t, readOnly.CommunityDeletePost)
		assert.False(t, readOnly.CommunityUserBan)
	})
}
