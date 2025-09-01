package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"stormlink/server/ent/comment"
	"stormlink/server/ent/post"
	postusecase "stormlink/server/usecase/post"
	"stormlink/tests/fixtures"

	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SimplePostUsecaseTestSuite struct {
	suite.Suite
	ctx    context.Context
	helper *testhelper.PostgresTestHelper
}

func (suite *SimplePostUsecaseTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.helper = testhelper.NewPostgresTestHelper(suite.T())
	suite.helper.WaitForDatabase(suite.T())
}

func (suite *SimplePostUsecaseTestSuite) TearDownSuite() {
	if suite.helper != nil {
		suite.helper.Cleanup()
	}
}

func (suite *SimplePostUsecaseTestSuite) SetupTest() {
	suite.helper.CleanDatabase(suite.T())
}

func (suite *SimplePostUsecaseTestSuite) TestPostCreation() {
	client := suite.helper.GetClient()

	// Create test data
	user, err := fixtures.CreateTestUser(suite.ctx, client, fixtures.UserFixture{
		Name:       "Test Author",
		Slug:       fmt.Sprintf("test-author-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("author-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})
	require.NoError(suite.T(), err)

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, fixtures.CommunityFixture{
		Name:        "Test Community",
		Slug:        fmt.Sprintf("test-community-%d", time.Now().UnixNano()),
		Description: "A test community",
		IsPrivate:   false,
		OwnerID:     user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	// Test post creation through fixtures
	post, err := fixtures.CreateTestPost(suite.ctx, client, fixtures.PostFixture{
		Title:       "Test Post",
		Content:     "This is a test post content",
		CommunityID: community.ID,
		AuthorID:    user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)
	assert.NotZero(suite.T(), post.ID)

	// Verify post was created correctly
	assert.Equal(suite.T(), "Test Post", post.Title)
	assert.Equal(suite.T(), community.ID, post.CommunityID)
	assert.Equal(suite.T(), user.ID, post.AuthorID)
	assert.NotNil(suite.T(), post.Content)

	// Create post usecase and test retrieval
	postUC := postusecase.NewPostUsecase(client)
	retrievedPost, err := postUC.GetPostByID(suite.ctx, post.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), post.ID, retrievedPost.ID)
}

func (suite *SimplePostUsecaseTestSuite) TestPostRetrieval() {
	client := suite.helper.GetClient()

	// Create test data using fixtures
	user, err := fixtures.CreateTestUser(suite.ctx, client, fixtures.UserFixture{
		Name:       "Test Author",
		Slug:       fmt.Sprintf("test-author-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("author-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})
	require.NoError(suite.T(), err)

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, fixtures.CommunityFixture{
		Name:        "Test Community",
		Slug:        fmt.Sprintf("test-community-%d", time.Now().UnixNano()),
		Description: "A test community",
		IsPrivate:   false,
		OwnerID:     user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	post, err := fixtures.CreateTestPost(suite.ctx, client, fixtures.PostFixture{
		Title:       "Test Retrieval Post",
		Content:     "Content for retrieval test",
		CommunityID: community.ID,
		AuthorID:    user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	// Test post retrieval through usecase
	postUC := postusecase.NewPostUsecase(client)
	retrievedPost, err := postUC.GetPostByID(suite.ctx, post.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedPost)

	assert.Equal(suite.T(), post.ID, retrievedPost.ID)
	assert.Equal(suite.T(), post.Title, retrievedPost.Title)
	assert.Equal(suite.T(), post.CommunityID, retrievedPost.CommunityID)
	assert.Equal(suite.T(), post.AuthorID, retrievedPost.AuthorID)
}

func (suite *SimplePostUsecaseTestSuite) TestPostStatus() {
	client := suite.helper.GetClient()

	// Create test data
	user, err := fixtures.CreateTestUser(suite.ctx, client, fixtures.UserFixture{
		Name:       "Test User",
		Slug:       fmt.Sprintf("test-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("user-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})
	require.NoError(suite.T(), err)

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, fixtures.CommunityFixture{
		Name:        "Test Community",
		Slug:        fmt.Sprintf("test-community-%d", time.Now().UnixNano()),
		Description: "A test community",
		IsPrivate:   false,
		OwnerID:     user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	post, err := fixtures.CreateTestPost(suite.ctx, client, fixtures.PostFixture{
		Title:       "Test Status Post",
		Content:     "Content for status test",
		CommunityID: community.ID,
		AuthorID:    user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	// Test post status through usecase
	postUC := postusecase.NewPostUsecase(client)
	status, err := postUC.GetPostStatus(suite.ctx, user.ID, post.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), status)

	// Basic validation - PostStatus should exist and have basic fields
	assert.NotNil(suite.T(), status.LikesCount)
	assert.NotNil(suite.T(), status.CommentsCount)
}

func (suite *SimplePostUsecaseTestSuite) TestPostWithComments() {
	client := suite.helper.GetClient()

	// Create test data
	user1, err := fixtures.CreateTestUser(suite.ctx, client, fixtures.UserFixture{
		Name:       "Post Author",
		Slug:       fmt.Sprintf("post-author-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("author-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt-1",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})
	require.NoError(suite.T(), err)

	user2, err := fixtures.CreateTestUser(suite.ctx, client, fixtures.UserFixture{
		Name:       "Commenter",
		Slug:       fmt.Sprintf("commenter-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("commenter-%d@example.com", time.Now().UnixNano()),
		Password:   "password456",
		Salt:       "test-salt-2",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})
	require.NoError(suite.T(), err)

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, fixtures.CommunityFixture{
		Name:        "Test Community",
		Slug:        fmt.Sprintf("test-community-%d", time.Now().UnixNano()),
		Description: "A test community",
		IsPrivate:   false,
		OwnerID:     user1.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	testPost, err := fixtures.CreateTestPost(suite.ctx, client, fixtures.PostFixture{
		Title:       "Post with Comments",
		Content:     "This post will have comments",
		CommunityID: community.ID,
		AuthorID:    user1.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	// Create comments
	testComment, err := fixtures.CreateTestComment(suite.ctx, client, fixtures.CommentFixture{
		Content:   "This is a test comment",
		PostID:    testPost.ID,
		AuthorID:  user2.ID,
		CreatedAt: time.Now(),
	})
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), testComment)

	// Verify comments exist
	comments, err := client.Comment.Query().Where(
		comment.PostIDEQ(testPost.ID),
	).All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), comments, 1)
	assert.Equal(suite.T(), "This is a test comment", comments[0].Content)
	assert.Equal(suite.T(), user2.ID, comments[0].AuthorID)
}

func (suite *SimplePostUsecaseTestSuite) TestMultiplePosts() {
	client := suite.helper.GetClient()

	// Create test user and community
	user, err := fixtures.CreateTestUser(suite.ctx, client, fixtures.UserFixture{
		Name:       "Prolific Author",
		Slug:       fmt.Sprintf("prolific-author-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("prolific-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})
	require.NoError(suite.T(), err)

	community, err := fixtures.CreateTestCommunity(suite.ctx, client, fixtures.CommunityFixture{
		Name:        "Active Community",
		Slug:        fmt.Sprintf("active-community-%d", time.Now().UnixNano()),
		Description: "A very active community",
		IsPrivate:   false,
		OwnerID:     user.ID,
		CreatedAt:   time.Now(),
	})
	require.NoError(suite.T(), err)

	// Create multiple posts
	const numPosts = 5
	postIDs := make([]int, 0, numPosts)

	for i := 0; i < numPosts; i++ {
		post, err := fixtures.CreateTestPost(suite.ctx, client, fixtures.PostFixture{
			Title:       fmt.Sprintf("Post %d", i+1),
			Content:     fmt.Sprintf("Content for post %d", i+1),
			CommunityID: community.ID,
			AuthorID:    user.ID,
			CreatedAt:   time.Now(),
		})
		require.NoError(suite.T(), err)
		postIDs = append(postIDs, post.ID)
	}

	postUC := postusecase.NewPostUsecase(client)

	// Verify all posts were created
	assert.Len(suite.T(), postIDs, numPosts)

	// Test retrieving posts by community
	posts, err := client.Post.Query().Where(
		post.CommunityIDEQ(community.ID),
	).All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), posts, numPosts)

	// Verify each post can be retrieved individually
	for _, postID := range postIDs {
		retrievedPost, err := postUC.GetPostByID(suite.ctx, postID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), retrievedPost)
		assert.Equal(suite.T(), community.ID, retrievedPost.CommunityID)
		assert.Equal(suite.T(), user.ID, retrievedPost.AuthorID)
	}
}

func (suite *SimplePostUsecaseTestSuite) TestPostNotFound() {
	client := suite.helper.GetClient()

	postUC := postusecase.NewPostUsecase(client)

	// Try to get a post that doesn't exist
	_, err := postUC.GetPostByID(suite.ctx, 99999)
	assert.Error(suite.T(), err)
}

func TestSimplePostUsecase(t *testing.T) {
	suite.Run(t, new(SimplePostUsecaseTestSuite))
}
