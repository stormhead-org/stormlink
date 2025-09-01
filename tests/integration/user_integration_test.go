package integration

import (
	"context"
	"testing"
	"time"

	"stormlink/server/usecase/user"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UserIntegrationTestSuite struct {
	suite.Suite
	containers *testcontainers.TestContainers
	userUC     user.UserUsecase
	ctx        context.Context
}

func (suite *UserIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers (PostgreSQL + Redis)
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create usecase
	suite.userUC = user.NewUserUsecase(containers.EntClient)
}

func (suite *UserIntegrationTestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *UserIntegrationTestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *UserIntegrationTestSuite) TestUserWorkflow_CompleteLifecycle() {
	// Test complete user lifecycle from creation to complex interactions

	// Step 1: Create test users and communities
	err := fixtures.SeedExtendedData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	// Step 2: Test user retrieval with all relationships
	user, err := suite.userUC.GetUserByID(suite.ctx, fixtures.AdminUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(user)
	suite.Assert().Equal(fixtures.AdminUser.Name, user.Name)

	// Step 3: Test permissions across multiple communities
	communityIDs := []int{fixtures.LargeCommunity.ID, fixtures.RestrictedCommunity.ID}
	permissions, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, fixtures.AdminUser.ID, communityIDs)
	suite.Assert().NoError(err)
	suite.Assert().Len(permissions, 2)

	// Step 4: Test user status relationships
	status, err := suite.userUC.GetUserStatus(suite.ctx, fixtures.TestUser1.ID, fixtures.AdminUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(status)
	suite.Assert().False(status.IsOwn)
}

func (suite *UserIntegrationTestSuite) TestUserPermissions_WithRealDatabase() {
	// Test complex permission scenarios with PostgreSQL

	// Seed complex scenario
	err := fixtures.SeedComplexScenario(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	// Create community moderator relationship
	moderatorRole, err := suite.containers.EntClient.Role.Create().
		SetName("Test Moderator").
		SetCommunityID(fixtures.LargeCommunity.ID).
		SetPermissions([]string{"delete_posts", "ban_users", "mute_users"}).
		SetCreatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	_, err = suite.containers.EntClient.CommunityModerator.Create().
		SetUserID(fixtures.ModeratorUser.ID).
		SetCommunityID(fixtures.LargeCommunity.ID).
		SetRoleID(moderatorRole.ID).
		SetCreatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Test permissions for moderator
	permissions, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, fixtures.ModeratorUser.ID, []int{fixtures.LargeCommunity.ID})
	suite.Assert().NoError(err)
	suite.Assert().NotEmpty(permissions)

	communityPermissions := permissions[fixtures.LargeCommunity.ID]
	suite.Assert().NotNil(communityPermissions)
	suite.Assert().True(communityPermissions.CanDeletePosts)
	suite.Assert().True(communityPermissions.CanBanUsers)
}

func (suite *UserIntegrationTestSuite) TestUserCaching_WithRedis() {
	// Test user data caching with Redis

	// Seed basic data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	// First request - should hit database and cache
	start := time.Now()
	user1, err := suite.userUC.GetUserByID(suite.ctx, fixtures.TestUser1.ID)
	firstRequestDuration := time.Since(start)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(user1)

	// Second request - should hit cache (if implemented)
	start = time.Now()
	user2, err := suite.userUC.GetUserByID(suite.ctx, fixtures.TestUser1.ID)
	secondRequestDuration := time.Since(start)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(user2)

	// Verify same data
	suite.Assert().Equal(user1.ID, user2.ID)
	suite.Assert().Equal(user1.Name, user2.Name)
	suite.Assert().Equal(user1.Email, user2.Email)

	// Note: Cache performance comparison would depend on actual cache implementation
	suite.T().Logf("First request: %v, Second request: %v", firstRequestDuration, secondRequestDuration)
}

func (suite *UserIntegrationTestSuite) TestUserConcurrency_WithPostgreSQL() {
	// Test concurrent user operations with PostgreSQL

	// Seed basic data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	concurrency := 20
	results := make(chan error, concurrency)

	// Concurrent user retrievals
	for i := 0; i < concurrency; i++ {
		go func(iteration int) {
			user, err := suite.userUC.GetUserByID(suite.ctx, fixtures.TestUser1.ID)
			if err != nil {
				results <- err
				return
			}

			if user == nil {
				results <- assert.AnError
				return
			}

			// Verify data integrity
			if user.Name != fixtures.TestUser1.Name {
				results <- assert.AnError
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		suite.Assert().NoError(err, "Concurrent user retrieval should succeed")
	}
}

func (suite *UserIntegrationTestSuite) TestUserPermissions_ComplexHierarchy() {
	// Test complex permission hierarchy with multiple roles

	// Create additional users
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 5)
	suite.Require().NoError(err)
	suite.Require().Len(users, 5)

	// Create additional communities
	communities, err := fixtures.CreateBulkCommunities(suite.ctx, suite.containers.EntClient, 3, users[0].ID)
	suite.Require().NoError(err)
	suite.Require().Len(communities, 3)

	// Create roles for each community
	var roleIDs []int
	for _, community := range communities {
		roles := []fixtures.RoleFixture{
			{
				Name:        "Admin",
				CommunityID: community.ID,
				Permissions: []string{"all", "manage_community"},
				CreatedAt:   time.Now(),
			},
			{
				Name:        "Moderator",
				CommunityID: community.ID,
				Permissions: []string{"delete_posts", "ban_users"},
				CreatedAt:   time.Now(),
			},
			{
				Name:        "Member",
				CommunityID: community.ID,
				Permissions: []string{"create_posts", "comment"},
				CreatedAt:   time.Now(),
			},
		}

		for _, roleFixture := range roles {
			role, err := fixtures.CreateTestRole(suite.ctx, suite.containers.EntClient, roleFixture)
			suite.Require().NoError(err)
			roleIDs = append(roleIDs, role.ID)
		}
	}

	// Assign roles to users
	for i, user := range users[1:] { // Skip owner
		communityIdx := i % len(communities)
		roleIdx := i % 3 // Cycle through role types

		_, err := fixtures.CreateTestCommunityModerator(suite.ctx, suite.containers.EntClient, fixtures.CommunityModeratorFixture{
			UserID:      user.ID,
			CommunityID: communities[communityIdx].ID,
			RoleID:      roleIDs[communityIdx*3+roleIdx],
			CreatedAt:   time.Now(),
		})
		suite.Require().NoError(err)
	}

	// Test permissions for each user across all communities
	for _, user := range users {
		var communityIDs []int
		for _, community := range communities {
			communityIDs = append(communityIDs, community.ID)
		}

		permissions, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, user.ID, communityIDs)
		suite.Assert().NoError(err)
		suite.Assert().Len(permissions, len(communityIDs))

		// Verify each community has permission data
		for _, communityID := range communityIDs {
			suite.Assert().Contains(permissions, communityID)
			suite.Assert().NotNil(permissions[communityID])
		}
	}
}

func (suite *UserIntegrationTestSuite) TestUserStatus_ComplexRelationships() {
	// Test user status with complex follow/block relationships

	// Create users
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 10)
	suite.Require().NoError(err)

	// Create follow network (each user follows the next one)
	for i := 0; i < len(users)-1; i++ {
		_, err := fixtures.CreateTestFollow(suite.ctx, suite.containers.EntClient, fixtures.FollowFixture{
			FollowerID:  users[i].ID,
			FollowingID: users[i+1].ID,
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Hour),
		})
		suite.Require().NoError(err)
	}

	// Test status between followed users
	for i := 0; i < len(users)-1; i++ {
		status, err := suite.userUC.GetUserStatus(suite.ctx, users[i].ID, users[i+1].ID)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)
		suite.Assert().True(status.IsFollowing)
		suite.Assert().False(status.IsOwn)
	}

	// Test mutual follows
	_, err = fixtures.CreateTestFollow(suite.ctx, suite.containers.EntClient, fixtures.FollowFixture{
		FollowerID:  users[len(users)-1].ID,
		FollowingID: users[0].ID,
		CreatedAt:   time.Now(),
	})
	suite.Require().NoError(err)

	// Both directions should show following
	status1, err := suite.userUC.GetUserStatus(suite.ctx, users[0].ID, users[len(users)-1].ID)
	suite.Assert().NoError(err)
	suite.Assert().True(status1.IsFollowing)

	status2, err := suite.userUC.GetUserStatus(suite.ctx, users[len(users)-1].ID, users[0].ID)
	suite.Assert().NoError(err)
	suite.Assert().True(status2.IsFollowing)
}

func (suite *UserIntegrationTestSuite) TestDatabaseTransactions_Consistency() {
	// Test data consistency under concurrent operations

	// Seed basic data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	concurrency := 10
	results := make(chan error, concurrency)

	// Concurrent operations that modify user relationships
	for i := 0; i < concurrency; i++ {
		go func(iteration int) {
			// Create unique users for each goroutine to avoid conflicts
			randomUser, err := fixtures.CreateRandomUser(suite.ctx, suite.containers.EntClient)
			if err != nil {
				results <- err
				return
			}

			// Create follow relationship
			_, err = fixtures.CreateTestFollow(suite.ctx, suite.containers.EntClient, fixtures.FollowFixture{
				FollowerID:  randomUser.ID,
				FollowingID: fixtures.TestUser1.ID,
				CreatedAt:   time.Now(),
			})
			if err != nil {
				results <- err
				return
			}

			// Verify user status
			status, err := suite.userUC.GetUserStatus(suite.ctx, randomUser.ID, fixtures.TestUser1.ID)
			if err != nil {
				results <- err
				return
			}

			if status == nil || !status.IsFollowing {
				results <- assert.AnError
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		suite.Assert().NoError(err, "Concurrent user operations should maintain consistency")
	}

	// Verify final state
	user, err := suite.userUC.GetUserByID(suite.ctx, fixtures.TestUser1.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(user)

	// Count followers
	followers, err := suite.containers.EntClient.UserFollow.Query().
		Where(user.FollowingUserFollows.IDEQ(fixtures.TestUser1.ID)).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(followers, concurrency, "All follow relationships should be created")
}

func (suite *UserIntegrationTestSuite) TestUserPermissions_WithRedisCache() {
	// Test permission caching behavior with Redis

	// Seed complex scenario
	err := fixtures.SeedComplexScenario(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	// Create roles in large community
	moderatorRole, err := suite.containers.EntClient.Role.Create().
		SetName("Test Moderator").
		SetCommunityID(fixtures.LargeCommunity.ID).
		SetPermissions([]string{"delete_posts", "ban_users"}).
		SetCreatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Assign moderator role
	_, err = suite.containers.EntClient.CommunityModerator.Create().
		SetUserID(fixtures.ModeratorUser.ID).
		SetCommunityID(fixtures.LargeCommunity.ID).
		SetRoleID(moderatorRole.ID).
		SetCreatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// First permission check - should hit database
	start := time.Now()
	permissions1, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, fixtures.ModeratorUser.ID, []int{fixtures.LargeCommunity.ID})
	firstDuration := time.Since(start)
	suite.Assert().NoError(err)
	suite.Assert().NotEmpty(permissions1)

	// Second permission check - could hit cache if implemented
	start = time.Now()
	permissions2, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, fixtures.ModeratorUser.ID, []int{fixtures.LargeCommunity.ID})
	secondDuration := time.Since(start)
	suite.Assert().NoError(err)
	suite.Assert().NotEmpty(permissions2)

	// Verify data consistency
	suite.Assert().Equal(permissions1[fixtures.LargeCommunity.ID].CanDeletePosts,
		permissions2[fixtures.LargeCommunity.ID].CanDeletePosts)
	suite.Assert().Equal(permissions1[fixtures.LargeCommunity.ID].CanBanUsers,
		permissions2[fixtures.LargeCommunity.ID].CanBanUsers)

	suite.T().Logf("First permission check: %v, Second: %v", firstDuration, secondDuration)
}

func (suite *UserIntegrationTestSuite) TestUserSearch_Performance() {
	// Test user search performance with large dataset

	// Create many users
	userCount := 100
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, userCount)
	suite.Require().NoError(err)
	suite.Assert().Len(users, userCount)

	// Test individual user retrieval performance
	start := time.Now()
	for _, user := range users[:10] { // Test first 10 users
		retrievedUser, err := suite.userUC.GetUserByID(suite.ctx, user.ID)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(retrievedUser)
		suite.Assert().Equal(user.ID, retrievedUser.ID)
	}
	duration := time.Since(start)

	suite.T().Logf("Retrieved 10 users in %v (avg: %v per user)", duration, duration/10)
	suite.Assert().Less(duration, 1*time.Second, "User retrieval should be fast even with large dataset")
}

func (suite *UserIntegrationTestSuite) TestUserRelationships_NetworkEffects() {
	// Test complex user relationship networks

	// Create a network of users
	networkSize := 20
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, networkSize)
	suite.Require().NoError(err)

	// Create communities
	communities, err := fixtures.CreateBulkCommunities(suite.ctx, suite.containers.EntClient, 5, users[0].ID)
	suite.Require().NoError(err)

	// Create complex follow network (star pattern - everyone follows user[0])
	for i := 1; i < networkSize; i++ {
		_, err := fixtures.CreateTestFollow(suite.ctx, suite.containers.EntClient, fixtures.FollowFixture{
			FollowerID:  users[i].ID,
			FollowingID: users[0].ID,
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Minute),
		})
		suite.Require().NoError(err)
	}

	// Create community follows (distribute users across communities)
	for i, user := range users {
		communityIdx := i % len(communities)
		_, err := fixtures.CreateTestCommunityFollow(suite.ctx, suite.containers.EntClient, fixtures.CommunityFollowFixture{
			UserID:      user.ID,
			CommunityID: communities[communityIdx].ID,
			CreatedAt:   time.Now(),
		})
		suite.Require().NoError(err)
	}

	// Test user status for central user (should have many followers)
	for i := 1; i < min(networkSize, 5); i++ { // Test first 5 followers
		status, err := suite.userUC.GetUserStatus(suite.ctx, users[i].ID, users[0].ID)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)
		suite.Assert().True(status.IsFollowing)
		suite.Assert().False(status.IsOwn)
	}

	// Test permissions across multiple communities for each user
	var communityIDs []int
	for _, community := range communities {
		communityIDs = append(communityIDs, community.ID)
	}

	for i := 0; i < min(networkSize, 5); i++ { // Test first 5 users
		permissions, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, users[i].ID, communityIDs)
		suite.Assert().NoError(err)
		suite.Assert().Len(permissions, len(communityIDs))

		// Each user should have some level of permissions in each community
		for _, communityID := range communityIDs {
			suite.Assert().Contains(permissions, communityID)
			suite.Assert().NotNil(permissions[communityID])
		}
	}
}

func (suite *UserIntegrationTestSuite) TestDataIntegrity_ConstraintValidation() {
	// Test database constraints and data integrity

	// Test unique constraints
	_, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Try to create user with duplicate email
	duplicateEmailUser := fixtures.TestUser1
	duplicateEmailUser.ID = 999
	duplicateEmailUser.Slug = "different-slug"
	_, err = fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, duplicateEmailUser)
	suite.Assert().Error(err, "Should not allow duplicate email")

	// Try to create user with duplicate slug
	duplicateSlugUser := fixtures.TestUser1
	duplicateSlugUser.ID = 998
	duplicateSlugUser.Email = "different@email.com"
	_, err = fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, duplicateSlugUser)
	suite.Assert().Error(err, "Should not allow duplicate slug")
}

func (suite *UserIntegrationTestSuite) TestDatabaseCleanup_BetweenTests() {
	// Test that database cleanup works properly between tests

	// Create test data
	user, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)
	suite.Assert().NotNil(user)

	// Verify data exists
	retrievedUser, err := suite.userUC.GetUserByID(suite.ctx, user.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(retrievedUser)

	// Reset database
	err = suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Verify data is gone
	retrievedUser, err = suite.userUC.GetUserByID(suite.ctx, user.ID)
	suite.Assert().Error(err)
	suite.Assert().Nil(retrievedUser)
	suite.Assert().True(suite.containers.EntClient.IsNotFound(err))
}

func (suite *UserIntegrationTestSuite) TestRedisCleanup_BetweenTests() {
	// Test that Redis cleanup works properly

	// Set some test data in Redis
	testKey := "test:user:123"
	testValue := "test-value"

	err := suite.containers.RedisClient.Set(suite.ctx, testKey, testValue, time.Hour).Err()
	suite.Require().NoError(err)

	// Verify data exists
	retrievedValue, err := suite.containers.RedisClient.Get(suite.ctx, testKey).Result()
	suite.Assert().NoError(err)
	suite.Assert().Equal(testValue, retrievedValue)

	// Flush Redis
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)

	// Verify data is gone
	_, err = suite.containers.RedisClient.Get(suite.ctx, testKey).Result()
	suite.Assert().Error(err) // Should be redis.Nil error
}

func (suite *UserIntegrationTestSuite) TestErrorHandling_DatabaseFailures() {
	// Test error handling when database operations fail

	// Close the database connection to simulate failure
	originalClient := suite.containers.EntClient
	err := originalClient.Close()
	suite.Require().NoError(err)

	// Try to use closed client
	_, err = suite.userUC.GetUserByID(suite.ctx, 1)
	suite.Assert().Error(err, "Should handle database connection errors gracefully")

	// Note: We can't easily restore the connection in this test
	// In a real scenario, you might want to test connection retry logic
}

// Helper function for min operation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Run integration test suite
func TestUserIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(UserIntegrationTestSuite))
}

// Performance benchmark with real PostgreSQL
func BenchmarkUserIntegration_PostgreSQL(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	ctx := context.Background()

	// Setup containers
	containers, err := testcontainers.SetupTestContainers(ctx)
	require.NoError(b, err)
	defer func() {
		err := containers.Cleanup(ctx)
		require.NoError(b, err)
	}()

	// Setup usecase
	userUC := user.NewUserUsecase(containers.EntClient)

	// Seed test data
	err = fixtures.SeedBasicData(ctx, containers.EntClient)
	require.NoError(b, err)

	b.Run("GetUserByID_PostgreSQL", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := userUC.GetUserByID(ctx, fixtures.TestUser1.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetPermissionsByCommunities_PostgreSQL", func(b *testing.B) {
		communityIDs := []int{fixtures.TestCommunity1.ID, fixtures.PrivateCommunity.ID}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := userUC.GetPermissionsByCommunities(ctx, fixtures.TestUser1.ID, communityIDs)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetUserStatus_PostgreSQL", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := userUC.GetUserStatus(ctx, fixtures.TestUser1.ID, fixtures.TestUser2.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
