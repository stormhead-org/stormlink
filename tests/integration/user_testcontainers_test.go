package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	entuser "stormlink/server/ent/user"
	"stormlink/server/usecase/user"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserIntegrationTestSuite struct {
	suite.Suite
	containers *testcontainers.TestContainers
	entClient  *ent.Client
	userUC     user.UserUsecase
	ctx        context.Context
}

func (suite *UserIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Set JWT secret for tests
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-for-integration-tests")

	// Setup test containers (PostgreSQL + Redis)
	containers, err := testcontainers.Setup(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create Ent client
	suite.entClient = enttest.Open(suite.T(), "postgres", suite.containers.GetPostgresDSN())

	// Create usecase
	suite.userUC = user.NewUserUsecase(suite.entClient)
}

func (suite *UserIntegrationTestSuite) TearDownSuite() {
	if suite.entClient != nil {
		suite.entClient.Close()
	}
	if suite.containers != nil {
		suite.containers.Cleanup()
	}
}

func (suite *UserIntegrationTestSuite) SetupTest() {
	// Clean up database by removing all data
	suite.cleanDatabase()
}

func (suite *UserIntegrationTestSuite) TearDownTest() {
	// Clean up after each test
	suite.cleanDatabase()
}

func (suite *UserIntegrationTestSuite) cleanDatabase() {
	// Delete all data in reverse dependency order
	suite.entClient.Comment.Delete().ExecX(suite.ctx)
	suite.entClient.Post.Delete().ExecX(suite.ctx)
	suite.entClient.Community.Delete().ExecX(suite.ctx)
	suite.entClient.User.Delete().ExecX(suite.ctx)
}

func (suite *UserIntegrationTestSuite) TestUserWorkflow_BasicLifecycle() {
	// Test basic user creation and retrieval
	user, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser1)
	suite.Require().NoError(err)
	suite.Require().NotNil(user)

	// Verify user was created correctly
	assert.Equal(suite.T(), fixtures.TestUser1.Name, user.Name)
	assert.Equal(suite.T(), fixtures.TestUser1.Email, user.Email)
	assert.Equal(suite.T(), fixtures.TestUser1.Slug, user.Slug)
	assert.Equal(suite.T(), fixtures.TestUser1.IsVerified, user.IsVerified)

	// Test user retrieval
	retrievedUser, err := suite.userUC.GetUserByID(suite.ctx, user.ID)
	suite.Require().NoError(err)
	suite.Require().NotNil(retrievedUser)

	assert.Equal(suite.T(), user.ID, retrievedUser.ID)
	assert.Equal(suite.T(), user.Name, retrievedUser.Name)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)
}

func (suite *UserIntegrationTestSuite) TestUserCreationWithCommunities() {
	// Create a user
	user, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Create a community owned by this user - use the actual user ID
	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user.ID
	community, err := fixtures.CreateTestCommunity(suite.ctx, suite.entClient, communityFixture)
	suite.Require().NoError(err)
	suite.Require().NotNil(community)

	// Verify the community was created correctly
	assert.Equal(suite.T(), fixtures.TestCommunity1.Name, community.Title)
	assert.Equal(suite.T(), fixtures.TestCommunity1.Slug, community.Slug)
	assert.Equal(suite.T(), user.ID, community.OwnerID)

	// Test user status for their own community
	status, err := suite.userUC.GetUserStatus(suite.ctx, user.ID, user.ID)
	suite.Require().NoError(err)
	suite.Require().NotNil(status)

	// Basic status validation - UserStatus should have string counters
	assert.NotNil(suite.T(), status.FollowersCount)
	assert.NotNil(suite.T(), status.FollowingCount)
	assert.NotNil(suite.T(), status.PostsCount)
}

func (suite *UserIntegrationTestSuite) TestUserWithPosts() {
	// Create test data
	user1, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	user2, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser2)
	suite.Require().NoError(err)

	// Create community with correct owner ID
	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user1.ID
	community, err := fixtures.CreateTestCommunity(suite.ctx, suite.entClient, communityFixture)
	suite.Require().NoError(err)

	// Create posts with correct IDs
	postFixture1 := fixtures.TestPost1
	postFixture1.AuthorID = user1.ID
	postFixture1.CommunityID = community.ID
	post1, err := fixtures.CreateTestPost(suite.ctx, suite.entClient, postFixture1)
	suite.Require().NoError(err)

	postFixture2 := fixtures.TestPost2
	postFixture2.AuthorID = user2.ID
	postFixture2.CommunityID = community.ID
	post2, err := fixtures.CreateTestPost(suite.ctx, suite.entClient, postFixture2)
	suite.Require().NoError(err)

	// Verify posts were created
	assert.Equal(suite.T(), fixtures.TestPost1.Title, post1.Title)
	assert.Equal(suite.T(), user1.ID, post1.AuthorID)
	assert.Equal(suite.T(), community.ID, post1.CommunityID)

	assert.Equal(suite.T(), fixtures.TestPost2.Title, post2.Title)
	assert.Equal(suite.T(), user2.ID, post2.AuthorID)
	assert.Equal(suite.T(), community.ID, post2.CommunityID)

	// Test getting user status shows basic information
	status1, err := suite.userUC.GetUserStatus(suite.ctx, user1.ID, user1.ID)
	suite.Require().NoError(err)
	suite.Require().NotNil(status1)

	// Basic validation of UserStatus fields
	assert.NotEmpty(suite.T(), status1.FollowersCount)
	assert.NotEmpty(suite.T(), status1.FollowingCount)
	assert.NotEmpty(suite.T(), status1.PostsCount)

	status2, err := suite.userUC.GetUserStatus(suite.ctx, user2.ID, user2.ID)
	suite.Require().NoError(err)
	suite.Require().NotNil(status2)
}

func (suite *UserIntegrationTestSuite) TestUserWithComments() {
	// Create users first
	user1, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	user2, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser2)
	suite.Require().NoError(err)

	// Create community with correct owner ID
	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user1.ID
	community, err := fixtures.CreateTestCommunity(suite.ctx, suite.entClient, communityFixture)
	suite.Require().NoError(err)

	// Create posts with correct IDs
	postFixture1 := fixtures.TestPost1
	postFixture1.AuthorID = user1.ID
	postFixture1.CommunityID = community.ID
	post1, err := fixtures.CreateTestPost(suite.ctx, suite.entClient, postFixture1)
	suite.Require().NoError(err)

	// Create comments with correct IDs
	commentFixture := fixtures.TestComment1
	commentFixture.PostID = post1.ID
	commentFixture.AuthorID = user2.ID
	_, err = fixtures.CreateTestComment(suite.ctx, suite.entClient, commentFixture)
	suite.Require().NoError(err)

	replyFixture := fixtures.TestReply1
	replyFixture.PostID = post1.ID
	replyFixture.AuthorID = user1.ID
	_, err = fixtures.CreateTestComment(suite.ctx, suite.entClient, replyFixture)
	suite.Require().NoError(err)

	// Verify that comments were created as part of seed data
	comments, err := suite.entClient.Comment.Query().All(suite.ctx)
	suite.Require().NoError(err)
	assert.Len(suite.T(), comments, 2) // TestComment1 and TestReply1

	// Verify comment structure
	for _, comment := range comments {
		assert.NotZero(suite.T(), comment.ID)
		assert.NotEmpty(suite.T(), comment.Content)
		assert.NotZero(suite.T(), comment.PostID)
		assert.NotZero(suite.T(), comment.AuthorID)
		assert.NotZero(suite.T(), comment.CommunityID)
	}
}

func (suite *UserIntegrationTestSuite) TestUserAuthentication() {
	// Create a verified user
	user, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Test JWT generation
	token, err := fixtures.GenerateTestJWT(user.ID)
	suite.Require().NoError(err)
	assert.NotEmpty(suite.T(), token)

	// Test refresh token generation
	refreshToken, err := fixtures.GenerateTestRefreshToken(user.ID)
	suite.Require().NoError(err)
	assert.NotEmpty(suite.T(), refreshToken)
}

func (suite *UserIntegrationTestSuite) TestUserValidation() {
	// Test with verified user
	verifiedUser, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.TestUser1)
	suite.Require().NoError(err)
	assert.True(suite.T(), verifiedUser.IsVerified)

	// Test with unverified user
	unverifiedUser, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)
	assert.False(suite.T(), unverifiedUser.IsVerified)
}

func (suite *UserIntegrationTestSuite) TestConcurrentUserOperations() {
	const numGoroutines = 5 // Reduce number to avoid overwhelming test DB

	// Create multiple users concurrently
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()

			userFixture := fixtures.UserFixture{
				Name:       fmt.Sprintf("Concurrent User %d", index),
				Slug:       fmt.Sprintf("concurrent-user-%d", index),
				Email:      fmt.Sprintf("concurrent%d@example.com", index),
				Password:   "password123",
				Salt:       fmt.Sprintf("salt-%d", index),
				IsVerified: true,
				CreatedAt:  time.Now(),
			}

			_, err := fixtures.CreateTestUser(suite.ctx, suite.entClient, userFixture)
			if err != nil {
				errors <- err
				return
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		suite.Fail("Concurrent operation failed", err.Error())
	}

	// Verify all users were created
	users, err := suite.entClient.User.Query().All(suite.ctx)
	suite.Require().NoError(err)
	assert.GreaterOrEqual(suite.T(), len(users), numGoroutines)
}

func (suite *UserIntegrationTestSuite) TestDatabaseTransactions() {
	// Test transaction rollback on error
	tx, err := suite.entClient.Tx(suite.ctx)
	suite.Require().NoError(err)

	// Create a user within transaction
	user, err := tx.User.Create().
		SetName("Transaction Test User").
		SetSlug("transaction-test-user").
		SetEmail("transaction@example.com").
		SetPasswordHash("hash").
		SetSalt("salt").
		SetIsVerified(true).
		Save(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(user)

	// Rollback the transaction
	err = tx.Rollback()
	suite.Require().NoError(err)

	// Verify user was not saved due to rollback
	exists, err := suite.entClient.User.Query().Where(entuser.IDEQ(user.ID)).Exist(suite.ctx)
	suite.Require().NoError(err)
	assert.False(suite.T(), exists) // Should not exist
}

func TestUserIntegration(t *testing.T) {
	suite.Run(t, new(UserIntegrationTestSuite))
}
