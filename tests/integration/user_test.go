package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	userusecase "stormlink/server/usecase/user"
	"stormlink/tests/fixtures"
	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SimpleUserIntegrationTestSuite struct {
	suite.Suite
	ctx    context.Context
	helper *testhelper.PostgresTestHelper
}

func (suite *SimpleUserIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Set JWT_SECRET for testing
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-for-testing")

	// Setup PostgreSQL test helper
	suite.helper = testhelper.NewPostgresTestHelper(suite.T())
	suite.helper.WaitForDatabase(suite.T())
}

func (suite *SimpleUserIntegrationTestSuite) TearDownSuite() {
	if suite.helper != nil {
		suite.helper.Cleanup()
	}
}

func (suite *SimpleUserIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	suite.helper.CleanDatabase(suite.T())
}

func (suite *SimpleUserIntegrationTestSuite) TestUserCreationAndRetrieval() {
	client := suite.helper.GetClient()

	// Create a unique test user
	testUser := fixtures.TestUser1
	testUser.Email = fmt.Sprintf("test-creation-%d@example.com", time.Now().UnixNano())
	testUser.Slug = fmt.Sprintf("test-creation-%d", time.Now().UnixNano())

	user, err := fixtures.CreateTestUser(suite.ctx, client, testUser)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user)

	// Verify user properties
	assert.Equal(suite.T(), testUser.Name, user.Name)
	assert.Equal(suite.T(), testUser.Email, user.Email)
	assert.Equal(suite.T(), testUser.Slug, user.Slug)
	assert.Equal(suite.T(), testUser.IsVerified, user.IsVerified)
	assert.NotZero(suite.T(), user.ID)

	// Create usecase and test retrieval
	userUC := userusecase.NewUserUsecase(client)
	retrievedUser, err := userUC.GetUserByID(suite.ctx, user.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedUser)

	// Verify retrieved user matches created user
	assert.Equal(suite.T(), user.ID, retrievedUser.ID)
	assert.Equal(suite.T(), user.Name, retrievedUser.Name)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)
	assert.Equal(suite.T(), user.Slug, retrievedUser.Slug)
}

func (suite *SimpleUserIntegrationTestSuite) TestUserStatus() {
	client := suite.helper.GetClient()

	// Create a unique test user
	testUser := fixtures.TestUser1
	testUser.Email = fmt.Sprintf("test-status-%d@example.com", time.Now().UnixNano())
	testUser.Slug = fmt.Sprintf("test-status-%d", time.Now().UnixNano())

	user1, err := fixtures.CreateTestUser(suite.ctx, client, testUser)
	require.NoError(suite.T(), err)

	// Create usecase and test user status
	userUC := userusecase.NewUserUsecase(client)
	status, err := userUC.GetUserStatus(suite.ctx, user1.ID, user1.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), status)

	// Verify status fields are initialized
	assert.NotNil(suite.T(), status.FollowersCount)
	assert.NotNil(suite.T(), status.FollowingCount)
	assert.NotNil(suite.T(), status.PostsCount)
	assert.False(suite.T(), status.IsHostBanned)
	assert.False(suite.T(), status.IsHostMuted)
}

func (suite *SimpleUserIntegrationTestSuite) TestMultipleUsers() {
	client := suite.helper.GetClient()

	// Create multiple unique test users
	testUser1 := fixtures.TestUser1
	testUser1.Email = fmt.Sprintf("test-multiple-1-%d@example.com", time.Now().UnixNano())
	testUser1.Slug = fmt.Sprintf("test-multiple-1-%d", time.Now().UnixNano())

	testUser2 := fixtures.TestUser2
	testUser2.Email = fmt.Sprintf("test-multiple-2-%d@example.com", time.Now().UnixNano())
	testUser2.Slug = fmt.Sprintf("test-multiple-2-%d", time.Now().UnixNano())

	testUserUnverified := fixtures.UnverifiedUser
	testUserUnverified.Email = fmt.Sprintf("test-multiple-unverified-%d@example.com", time.Now().UnixNano())
	testUserUnverified.Slug = fmt.Sprintf("test-multiple-unverified-%d", time.Now().UnixNano())

	user1, err := fixtures.CreateTestUser(suite.ctx, client, testUser1)
	require.NoError(suite.T(), err)

	user2, err := fixtures.CreateTestUser(suite.ctx, client, testUser2)
	require.NoError(suite.T(), err)

	unverifiedUser, err := fixtures.CreateTestUser(suite.ctx, client, testUserUnverified)
	require.NoError(suite.T(), err)

	// Verify all users were created with correct properties
	assert.True(suite.T(), user1.IsVerified)
	assert.True(suite.T(), user2.IsVerified)
	assert.False(suite.T(), unverifiedUser.IsVerified)

	// Verify all users have unique IDs
	assert.NotEqual(suite.T(), user1.ID, user2.ID)
	assert.NotEqual(suite.T(), user1.ID, unverifiedUser.ID)
	assert.NotEqual(suite.T(), user2.ID, unverifiedUser.ID)

	// Test usecase with different users
	userUC := userusecase.NewUserUsecase(client)

	retrievedUser1, err := userUC.GetUserByID(suite.ctx, user1.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user1.Email, retrievedUser1.Email)

	retrievedUser2, err := userUC.GetUserByID(suite.ctx, user2.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user2.Email, retrievedUser2.Email)
}

func (suite *SimpleUserIntegrationTestSuite) TestUserWithCommunityData() {
	client := suite.helper.GetClient()

	// Create unique users first, then communities and posts
	testUser1 := fixtures.TestUser1
	testUser1.Email = fmt.Sprintf("test-community-1-%d@example.com", time.Now().UnixNano())
	testUser1.Slug = fmt.Sprintf("test-community-1-%d", time.Now().UnixNano())

	testUser2 := fixtures.TestUser2
	testUser2.Email = fmt.Sprintf("test-community-2-%d@example.com", time.Now().UnixNano())
	testUser2.Slug = fmt.Sprintf("test-community-2-%d", time.Now().UnixNano())

	testUserUnverified := fixtures.UnverifiedUser
	testUserUnverified.Email = fmt.Sprintf("test-community-unverified-%d@example.com", time.Now().UnixNano())
	testUserUnverified.Slug = fmt.Sprintf("test-community-unverified-%d", time.Now().UnixNano())

	// Create users manually
	// Create users and capture their real IDs
	user1, err := fixtures.CreateTestUser(suite.ctx, client, testUser1)
	require.NoError(suite.T(), err)

	user2, err := fixtures.CreateTestUser(suite.ctx, client, testUser2)
	require.NoError(suite.T(), err)

	_, err = fixtures.CreateTestUser(suite.ctx, client, testUserUnverified)
	require.NoError(suite.T(), err)

	// Create communities with correct owner IDs
	communityFixture1 := fixtures.TestCommunity1
	communityFixture1.OwnerID = user1.ID
	community1, err := fixtures.CreateTestCommunity(suite.ctx, client, communityFixture1)
	require.NoError(suite.T(), err)

	communityFixture2 := fixtures.PrivateCommunity
	communityFixture2.OwnerID = user2.ID
	_, err = fixtures.CreateTestCommunity(suite.ctx, client, communityFixture2)
	require.NoError(suite.T(), err)

	// Create posts with correct IDs
	postFixture1 := fixtures.TestPost1
	postFixture1.AuthorID = user1.ID
	postFixture1.CommunityID = community1.ID
	post1, err := fixtures.CreateTestPost(suite.ctx, client, postFixture1)
	require.NoError(suite.T(), err)

	postFixture2 := fixtures.TestPost2
	postFixture2.AuthorID = user2.ID
	postFixture2.CommunityID = community1.ID
	_, err = fixtures.CreateTestPost(suite.ctx, client, postFixture2)
	require.NoError(suite.T(), err)

	// Create comments with correct IDs
	commentFixture1 := fixtures.TestComment1
	commentFixture1.PostID = post1.ID
	commentFixture1.AuthorID = user2.ID
	_, err = fixtures.CreateTestComment(suite.ctx, client, commentFixture1)
	require.NoError(suite.T(), err)

	replyFixture := fixtures.TestReply1
	replyFixture.PostID = post1.ID
	replyFixture.AuthorID = user1.ID
	_, err = fixtures.CreateTestComment(suite.ctx, client, replyFixture)
	require.NoError(suite.T(), err)

	// Verify users were created
	users, err := client.User.Query().All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), users, 3) // TestUser1, TestUser2, UnverifiedUser

	// Verify communities were created
	communities, err := client.Community.Query().All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), communities, 2) // TestCommunity1, PrivateCommunity

	// Verify posts were created
	posts, err := client.Post.Query().All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), posts, 2) // TestPost1, TestPost2

	// Verify comments were created
	comments, err := client.Comment.Query().All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), comments, 2) // TestComment1, TestReply1

	// Test user status with real data
	userUC := userusecase.NewUserUsecase(client)

	// Find a user to test with
	testUser := users[0]
	status, err := userUC.GetUserStatus(suite.ctx, testUser.ID, testUser.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), status)

	// Status should have valid string values
	assert.NotEmpty(suite.T(), status.FollowersCount)
	assert.NotEmpty(suite.T(), status.FollowingCount)
	assert.NotEmpty(suite.T(), status.PostsCount)
}

func (suite *SimpleUserIntegrationTestSuite) TestJWTTokenGeneration() {
	client := suite.helper.GetClient()

	// Create a unique test user
	testUser := fixtures.TestUser1
	testUser.Email = fmt.Sprintf("test-jwt-%d@example.com", time.Now().UnixNano())
	testUser.Slug = fmt.Sprintf("test-jwt-%d", time.Now().UnixNano())

	user, err := fixtures.CreateTestUser(suite.ctx, client, testUser)
	require.NoError(suite.T(), err)

	// Test JWT generation
	token, err := fixtures.GenerateTestJWT(user.ID)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	// Test refresh token generation
	refreshToken, err := fixtures.GenerateTestRefreshToken(user.ID)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), refreshToken)

	// Tokens should be different
	assert.NotEqual(suite.T(), token, refreshToken)
}

func (suite *SimpleUserIntegrationTestSuite) TestUserNotFound() {
	client := suite.helper.GetClient()

	userUC := userusecase.NewUserUsecase(client)

	// Try to get a user that doesn't exist
	_, err := userUC.GetUserByID(suite.ctx, 99999)
	assert.Error(suite.T(), err)
}

func TestSimpleUserIntegration(t *testing.T) {
	suite.Run(t, new(SimpleUserIntegrationTestSuite))
}
