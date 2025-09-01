package user

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/tests/fixtures"
	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SimpleUserUsecaseTestSuite struct {
	suite.Suite
	ctx    context.Context
	helper *testhelper.PostgresTestHelper
}

func (suite *SimpleUserUsecaseTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Set JWT_SECRET for testing
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-for-testing")

	// Setup PostgreSQL test helper
	suite.helper = testhelper.NewPostgresTestHelper(suite.T())
	suite.helper.WaitForDatabase(suite.T())
}

func (suite *SimpleUserUsecaseTestSuite) TearDownSuite() {
	if suite.helper != nil {
		suite.helper.Cleanup()
	}
}

func (suite *SimpleUserUsecaseTestSuite) SetupTest() {
	// Clean database before each test
	suite.helper.CleanDatabase(suite.T())
}

func (suite *SimpleUserUsecaseTestSuite) TestGetUserByID() {
	client := suite.helper.GetClient()

	// Create test user
	testUser := fixtures.UserFixture{
		Name:       "Test User",
		Slug:       fmt.Sprintf("test-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("testuser-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, client, testUser)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user)

	// Test usecase
	userUC := NewUserUsecase(client)
	retrievedUser, err := userUC.GetUserByID(suite.ctx, user.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedUser)

	// Verify user data
	assert.Equal(suite.T(), user.ID, retrievedUser.ID)
	assert.Equal(suite.T(), user.Name, retrievedUser.Name)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)
	assert.Equal(suite.T(), user.Slug, retrievedUser.Slug)
	assert.Equal(suite.T(), user.IsVerified, retrievedUser.IsVerified)
}

func (suite *SimpleUserUsecaseTestSuite) TestGetUserByIDNotFound() {
	client := suite.helper.GetClient()

	userUC := NewUserUsecase(client)

	// Try to get non-existent user
	_, err := userUC.GetUserByID(suite.ctx, 99999)
	assert.Error(suite.T(), err)
}

func (suite *SimpleUserUsecaseTestSuite) TestGetUserStatus() {
	client := suite.helper.GetClient()

	// Create test users
	user1Fixture := fixtures.UserFixture{
		Name:       "User One",
		Slug:       fmt.Sprintf("user-one-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("user1-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt-1",
		IsVerified: true,
		CreatedAt:  time.Now(),
	}

	user2Fixture := fixtures.UserFixture{
		Name:       "User Two",
		Slug:       fmt.Sprintf("user-two-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("user2-%d@example.com", time.Now().UnixNano()),
		Password:   "password456",
		Salt:       "test-salt-2",
		IsVerified: true,
		CreatedAt:  time.Now(),
	}

	user1, err := fixtures.CreateTestUser(suite.ctx, client, user1Fixture)
	require.NoError(suite.T(), err)

	user2, err := fixtures.CreateTestUser(suite.ctx, client, user2Fixture)
	require.NoError(suite.T(), err)

	// Test usecase
	userUC := NewUserUsecase(client)

	// Test user status for self
	status1, err := userUC.GetUserStatus(suite.ctx, user1.ID, user1.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), status1)

	// Basic validation - UserStatus should have string counters
	assert.NotNil(suite.T(), status1.FollowersCount)
	assert.NotNil(suite.T(), status1.FollowingCount)
	assert.NotNil(suite.T(), status1.PostsCount)
	assert.False(suite.T(), status1.IsHostBanned)
	assert.False(suite.T(), status1.IsHostMuted)

	// Test user status for other user
	status2, err := userUC.GetUserStatus(suite.ctx, user1.ID, user2.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), status2)

	assert.NotNil(suite.T(), status2.FollowersCount)
	assert.NotNil(suite.T(), status2.FollowingCount)
	assert.NotNil(suite.T(), status2.PostsCount)
	assert.False(suite.T(), status2.IsFollowing)
}

func (suite *SimpleUserUsecaseTestSuite) TestGetUserStatusWithCommunityData() {
	client := suite.helper.GetClient()

	// Create test user
	userFixture := fixtures.UserFixture{
		Name:       "Community Owner",
		Slug:       fmt.Sprintf("community-owner-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("owner-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, client, userFixture)
	require.NoError(suite.T(), err)

	// Create test community
	communityFixture := fixtures.CommunityFixture{
		Name:        "Test Community",
		Slug:        fmt.Sprintf("test-community-%d", time.Now().UnixNano()),
		Description: "A test community",
		IsPrivate:   false,
		OwnerID:     user.ID,
		CreatedAt:   time.Now(),
	}

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, communityFixture)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), community)

	// Test usecase with community data
	userUC := NewUserUsecase(client)
	status, err := userUC.GetUserStatus(suite.ctx, user.ID, user.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), status)

	// Basic validation
	assert.NotNil(suite.T(), status.FollowersCount)
	assert.NotNil(suite.T(), status.FollowingCount)
	assert.NotNil(suite.T(), status.PostsCount)
}

func (suite *SimpleUserUsecaseTestSuite) TestGetPermissionsByCommunities() {
	client := suite.helper.GetClient()

	// Create test user
	userFixture := fixtures.UserFixture{
		Name:       "Test User",
		Slug:       fmt.Sprintf("test-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("user-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, client, userFixture)
	require.NoError(suite.T(), err)

	// Create test community
	communityFixture := fixtures.CommunityFixture{
		Name:        "Test Community",
		Slug:        fmt.Sprintf("test-community-%d", time.Now().UnixNano()),
		Description: "A test community",
		IsPrivate:   false,
		OwnerID:     user.ID,
		CreatedAt:   time.Now(),
	}

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, communityFixture)
	require.NoError(suite.T(), err)

	// Test permissions
	userUC := NewUserUsecase(client)
	permissions, err := userUC.GetPermissionsByCommunities(suite.ctx, user.ID, []int{community.ID})
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), permissions)

	// Check permissions exist for the community
	assert.Contains(suite.T(), permissions, community.ID)
	communityPermissions := permissions[community.ID]
	assert.NotNil(suite.T(), communityPermissions)

	// Basic permission validation - just check that we get a permissions object
	assert.NotNil(suite.T(), communityPermissions)
}

func (suite *SimpleUserUsecaseTestSuite) TestMultipleUsersInteraction() {
	client := suite.helper.GetClient()

	// Create multiple users
	const numUsers = 3
	users := make([]*ent.User, 0, numUsers)

	for i := 0; i < numUsers; i++ {
		userFixture := fixtures.UserFixture{
			Name:       fmt.Sprintf("User %d", i+1),
			Slug:       fmt.Sprintf("user-%d-%d", i+1, time.Now().UnixNano()),
			Email:      fmt.Sprintf("user%d-%d@example.com", i+1, time.Now().UnixNano()),
			Password:   "password123",
			Salt:       fmt.Sprintf("test-salt-%d", i+1),
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		user, err := fixtures.CreateTestUser(suite.ctx, client, userFixture)
		require.NoError(suite.T(), err)
		users = append(users, user)
	}

	// Test usecase with all users
	userUC := NewUserUsecase(client)

	for _, user := range users {
		retrievedUser, err := userUC.GetUserByID(suite.ctx, user.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.ID, retrievedUser.ID)
		assert.Equal(suite.T(), user.Name, retrievedUser.Name)

		// Test user status
		status, err := userUC.GetUserStatus(suite.ctx, user.ID, user.ID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), status)
		assert.NotNil(suite.T(), status.FollowersCount)
		assert.NotNil(suite.T(), status.FollowingCount)
		assert.NotNil(suite.T(), status.PostsCount)
	}

	// Test cross-user status checks
	for i, user1 := range users {
		for j, user2 := range users {
			if i != j {
				status, err := userUC.GetUserStatus(suite.ctx, user1.ID, user2.ID)
				require.NoError(suite.T(), err)
				assert.NotNil(suite.T(), status)
				assert.False(suite.T(), status.IsFollowing) // Should not be following by default
			}
		}
	}
}

func TestSimpleUserUsecase(t *testing.T) {
	suite.Run(t, new(SimpleUserUsecaseTestSuite))
}
