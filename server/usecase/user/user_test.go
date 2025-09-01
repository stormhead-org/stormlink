package user

import (
	"context"
	"fmt"
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

func TestUserUsecase_GetUserByID(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	// Create test user
	testUser, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	require.NoError(t, err)

	t.Run("existing user", func(t *testing.T) {
		user, err := uc.GetUserByID(ctx, testUser.ID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, fixtures.TestUser1.Name, user.Name)
		assert.Equal(t, fixtures.TestUser1.Email, user.Email)
		assert.Equal(t, fixtures.TestUser1.Slug, user.Slug)
	})

	t.Run("non-existing user", func(t *testing.T) {
		user, err := uc.GetUserByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, ent.IsNotFound(err))
	})

	t.Run("user with avatar", func(t *testing.T) {
		// Create media for avatar
		avatar, err := fixtures.CreateTestMedia(ctx, client, "avatar.png", "https://example.com/avatar.png")
		require.NoError(t, err)

		// Create user with avatar
		userWithAvatar, err := client.User.Create().
			SetName("User With Avatar").
			SetSlug("user-with-avatar").
			SetEmail("avatar@test.com").
			SetPasswordHash("hash").
			SetSalt("salt").
			SetAvatarID(avatar.ID).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		retrievedUser, err := uc.GetUserByID(ctx, userWithAvatar.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)

		// Load avatar edge and verify
		avatar_edge, err := retrievedUser.QueryAvatar().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, avatar.ID, avatar_edge.ID)
	})
}

func TestUserUsecase_GetPermissionsByCommunities(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("user with no special permissions", func(t *testing.T) {
		permissions, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, []int{fixtures.TestCommunity1.ID})

		assert.NoError(t, err)
		assert.NotNil(t, permissions)
		assert.Contains(t, permissions, fixtures.TestCommunity1.ID)

		communityPermissions := permissions[fixtures.TestCommunity1.ID]
		assert.NotNil(t, communityPermissions)
		// Default permissions for regular user
		assert.False(t, communityPermissions.CanDeletePosts)
		assert.False(t, communityPermissions.CanBanUsers)
		assert.False(t, communityPermissions.CanManageRoles)
	})

	t.Run("community owner permissions", func(t *testing.T) {
		permissions, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestCommunity1.OwnerID, []int{fixtures.TestCommunity1.ID})

		assert.NoError(t, err)
		assert.NotNil(t, permissions)
		assert.Contains(t, permissions, fixtures.TestCommunity1.ID)

		ownerPermissions := permissions[fixtures.TestCommunity1.ID]
		assert.NotNil(t, ownerPermissions)
		// Owner should have all permissions
		assert.True(t, ownerPermissions.CanDeletePosts)
		assert.True(t, ownerPermissions.CanBanUsers)
		assert.True(t, ownerPermissions.CanManageRoles)
	})

	t.Run("multiple communities", func(t *testing.T) {
		communityIDs := []int{fixtures.TestCommunity1.ID, fixtures.PrivateCommunity.ID}
		permissions, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, communityIDs)

		assert.NoError(t, err)
		assert.NotNil(t, permissions)
		assert.Len(t, permissions, 2)
		assert.Contains(t, permissions, fixtures.TestCommunity1.ID)
		assert.Contains(t, permissions, fixtures.PrivateCommunity.ID)
	})

	t.Run("non-existing user", func(t *testing.T) {
		permissions, err := uc.GetPermissionsByCommunities(ctx, 99999, []int{fixtures.TestCommunity1.ID})

		assert.NoError(t, err)
		assert.Empty(t, permissions)
	})

	t.Run("non-existing communities", func(t *testing.T) {
		permissions, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, []int{99999, 88888})

		assert.NoError(t, err)
		assert.Empty(t, permissions)
	})
}

func TestUserUsecase_GetUserStatus(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("user viewing own profile", func(t *testing.T) {
		status, err := uc.GetUserStatus(ctx, fixtures.TestUser1.ID, fixtures.TestUser1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBlocked)
		assert.Equal(t, models.UserStatusRelationshipSelf, status.Relationship)
	})

	t.Run("user viewing another user", func(t *testing.T) {
		status, err := uc.GetUserStatus(ctx, fixtures.TestUser1.ID, fixtures.TestUser2.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBlocked)
		assert.Equal(t, models.UserStatusRelationshipNone, status.Relationship)
	})

	t.Run("user following another user", func(t *testing.T) {
		// Create follow relationship
		_, err := client.UserFollow.Create().
			SetFollowerID(fixtures.TestUser1.ID).
			SetFollowingID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetUserStatus(ctx, fixtures.TestUser1.ID, fixtures.TestUser2.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.True(t, status.IsFollowing)
		assert.False(t, status.IsBlocked)
		assert.Equal(t, models.UserStatusRelationshipFollowing, status.Relationship)
	})

	t.Run("anonymous user viewing profile", func(t *testing.T) {
		status, err := uc.GetUserStatus(ctx, 0, fixtures.TestUser1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsOwn)
		assert.False(t, status.IsFollowing)
		assert.False(t, status.IsBlocked)
		assert.Equal(t, models.UserStatusRelationshipNone, status.Relationship)
	})

	t.Run("non-existing target user", func(t *testing.T) {
		status, err := uc.GetUserStatus(ctx, fixtures.TestUser1.ID, 99999)

		assert.Error(t, err)
		assert.Nil(t, status)
	})
}

func TestUserUsecase_EdgeCases(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	t.Run("context cancellation", func(t *testing.T) {
		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		user, err := uc.GetUserByID(cancelledCtx, 1)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("empty community IDs list", func(t *testing.T) {
		permissions, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, []int{})

		assert.NoError(t, err)
		assert.Empty(t, permissions)
	})

	t.Run("nil community IDs", func(t *testing.T) {
		permissions, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, nil)

		assert.NoError(t, err)
		assert.Empty(t, permissions)
	})
}

func TestUserUsecase_Performance(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	// Create multiple test users
	userIDs := make([]int, 100)
	for i := 0; i < 100; i++ {
		testUser := fixtures.UserFixture{
			Name:       fmt.Sprintf("User %d", i),
			Slug:       fmt.Sprintf("user-%d", i),
			Email:      fmt.Sprintf("user%d@test.com", i),
			Password:   "password",
			Salt:       fmt.Sprintf("salt-%d", i),
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		user, err := fixtures.CreateTestUser(ctx, client, testUser)
		require.NoError(t, err)
		userIDs[i] = user.ID
	}

	t.Run("bulk permission retrieval performance", func(t *testing.T) {
		start := time.Now()

		// Test retrieving permissions for many communities
		communityIDs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		permissions, err := uc.GetPermissionsByCommunities(ctx, userIDs[0], communityIDs)

		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, permissions)
		// Should complete reasonably quickly (under 100ms for this test)
		assert.Less(t, duration, 100*time.Millisecond, "Permission retrieval should be fast")
	})
}

// Benchmark tests
func BenchmarkUserUsecase_GetUserByID(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	// Create test user
	testUser, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetUserByID(ctx, testUser.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUserUsecase_GetPermissionsByCommunities(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	uc := NewUserUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(b, err)

	communityIDs := []int{fixtures.TestCommunity1.ID, fixtures.PrivateCommunity.ID}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, communityIDs)
		if err != nil {
			b.Fatal(err)
		}
	}
}
