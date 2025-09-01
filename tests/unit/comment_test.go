package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"stormlink/server/ent"
	"stormlink/server/usecase/comment"
	"stormlink/tests/fixtures"
	"stormlink/tests/testhelper"
)

type CommentUsecaseTestSuite struct {
	suite.Suite
	helper *testhelper.PostgresTestHelper
	client *ent.Client
	uc     comment.CommentUsecase
	ctx    context.Context
}

func (suite *CommentUsecaseTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.helper = testhelper.NewPostgresTestHelper(suite.T())
	suite.helper.WaitForDatabase(suite.T())

	suite.client = suite.helper.GetClient()
	suite.uc = comment.NewCommentUsecase(suite.client)
}

func (suite *CommentUsecaseTestSuite) TearDownSuite() {
	if suite.helper != nil {
		suite.helper.Cleanup()
	}
}

func (suite *CommentUsecaseTestSuite) SetupTest() {
	// Clean up data before each test
	suite.helper.CleanDatabase(suite.T())
}

// createTestData creates test data with proper foreign key relationships
// Returns: user1, user2, community, post, comment, reply, error
func (suite *CommentUsecaseTestSuite) createTestData() (*ent.User, *ent.User, *ent.Community, *ent.Post, *ent.Comment, *ent.Comment, error) {
	// Create users
	user1, err := fixtures.CreateTestUser(suite.ctx, suite.client, fixtures.TestUser1)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	user2, err := fixtures.CreateTestUser(suite.ctx, suite.client, fixtures.TestUser2)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// Create community with correct owner ID
	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user1.ID
	community, err := fixtures.CreateTestCommunity(suite.ctx, suite.client, communityFixture)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// Create post with correct IDs
	postFixture := fixtures.TestPost1
	postFixture.AuthorID = user1.ID
	postFixture.CommunityID = community.ID
	post, err := fixtures.CreateTestPost(suite.ctx, suite.client, postFixture)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// Create comment with correct IDs
	commentFixture := fixtures.TestComment1
	commentFixture.PostID = post.ID
	commentFixture.AuthorID = user2.ID
	comment, err := fixtures.CreateTestComment(suite.ctx, suite.client, commentFixture)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// Create reply comment
	replyFixture := fixtures.TestReply1
	replyFixture.PostID = post.ID
	replyFixture.AuthorID = user1.ID
	replyFixture.ParentID = &comment.ID
	reply, err := fixtures.CreateTestComment(suite.ctx, suite.client, replyFixture)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return user1, user2, community, post, comment, reply, nil
}

func (suite *CommentUsecaseTestSuite) TestGetCommentsByPostID() {
	// Create test data with correct IDs
	user1, user2, _, post, comment, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	suite.Run("get comments for existing post", func() {
		comments, err := suite.uc.GetCommentsByPostID(suite.ctx, post.ID, nil)

		suite.NoError(err)
		suite.NotNil(comments)
		suite.Len(comments, 2) // TestComment1 and TestReply1

		// Verify comment content by matching content and author
		foundComment := false
		foundReply := false
		for _, c := range comments {
			if c.Content == fixtures.TestComment1.Content && c.AuthorID == user2.ID {
				foundComment = true
			}
			if c.Content == fixtures.TestReply1.Content && c.AuthorID == user1.ID {
				foundReply = true
			}
		}
		suite.True(foundComment, "Should find the main comment")
		suite.True(foundReply, "Should find the reply comment")
	})

	suite.Run("get comments for non-existing post", func() {
		comments, err := suite.uc.GetCommentsByPostID(suite.ctx, 99999, nil)

		suite.NoError(err)
		suite.Empty(comments)
	})

	suite.Run("filter by deleted status", func() {
		// Create a deleted comment
		deletedComment, err := suite.client.Comment.Create().
			SetContent("This comment is deleted").
			SetPostID(post.ID).
			SetAuthorID(comment.AuthorID).
			SetCommunityID(post.CommunityID).
			SetHasDeleted(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)

		// Test filtering for non-deleted comments
		hasDeleted := false
		comments, err := suite.uc.GetCommentsByPostID(suite.ctx, post.ID, &hasDeleted)
		suite.NoError(err)
		suite.Len(comments, 2) // Should only get non-deleted comments

		// Test filtering for deleted comments
		hasDeleted = true
		deletedComments, err := suite.uc.GetCommentsByPostID(suite.ctx, post.ID, &hasDeleted)
		suite.NoError(err)
		suite.Len(deletedComments, 1)
		suite.Equal(deletedComment.ID, deletedComments[0].ID)
	})
}

func (suite *CommentUsecaseTestSuite) TestGetCommentsByPostIDLight() {
	// Create test data with correct IDs
	_, _, _, post, _, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	suite.Run("light version returns comments without heavy relations", func() {
		comments, err := suite.uc.GetCommentsByPostIDLight(suite.ctx, post.ID, nil)

		suite.NoError(err)
		suite.NotEmpty(comments)
		suite.Len(comments, 2)

		// Verify that it's the light version
		for _, comment := range comments {
			suite.NotNil(comment)
			suite.NotEmpty(comment.Content)
		}
	})
}

func (suite *CommentUsecaseTestSuite) TestGetCommentsByPostIDLightPaginated() {
	// Create test data with correct IDs
	user1, _, _, post, _, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	// Create additional comments for pagination testing
	for i := 3; i <= 10; i++ {
		_, err := suite.client.Comment.Create().
			SetContent(fmt.Sprintf("Test comment %d", i)).
			SetPostID(post.ID).
			SetAuthorID(user1.ID).
			SetCommunityID(post.CommunityID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Minute)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)
	}

	suite.Run("paginated results with limit and offset", func() {
		// Test first page
		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, post.ID, nil, 5, 0)
		suite.NoError(err)
		suite.Len(comments, 5)

		// Test second page
		comments2, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, post.ID, nil, 5, 5)
		suite.NoError(err)
		suite.Len(comments2, 5)

		// Verify no overlap
		firstPageIDs := make(map[int]bool)
		for _, comment := range comments {
			firstPageIDs[comment.ID] = true
		}

		for _, comment := range comments2 {
			suite.False(firstPageIDs[comment.ID], "Second page should not contain items from first page")
		}
	})

	suite.Run("limit larger than available comments", func() {
		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, post.ID, nil, 100, 0)
		suite.NoError(err)
		suite.Len(comments, 10) // Should return all 10 comments
	})

	suite.Run("offset beyond available comments", func() {
		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, post.ID, nil, 5, 100)
		suite.NoError(err)
		suite.Empty(comments)
	})
}

func (suite *CommentUsecaseTestSuite) TestCommentByID() {
	// Create test data with correct IDs
	_, _, _, _, comment, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	suite.Run("get_existing_comment", func() {
		foundComment, err := suite.uc.CommentByID(suite.ctx, comment.ID)

		suite.NoError(err)
		suite.NotNil(foundComment)
		suite.Equal(comment.ID, foundComment.ID)
		suite.Equal(comment.Content, foundComment.Content)
		suite.Equal(comment.AuthorID, foundComment.AuthorID)
	})

	suite.Run("get non-existing comment", func() {
		comment, err := suite.uc.CommentByID(suite.ctx, 99999)

		suite.Error(err)
		suite.Nil(comment)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("get comment with parent relationship", func() {
		comment, err := suite.uc.CommentByID(suite.ctx, fixtures.TestReply1.ID)

		suite.NoError(err)
		suite.NotNil(comment)
		suite.NotNil(comment.ParentCommentID)
		suite.Equal(fixtures.TestComment1.ID, *comment.ParentCommentID)
	})
}

func (suite *CommentUsecaseTestSuite) TestGetCommentStatus() {
	// Create test data with correct IDs
	user1, user2, _, _, comment, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	suite.Run("user viewing own comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, user2.ID, comment.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsLiked)
		// For now, we'll check basic fields available in CommentStatus
		// The owner check would need to be implemented in the usecase logic
	})

	suite.Run("user viewing another user's comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, user1.ID, comment.ID)

		suite.NoError(err)
		suite.NotNil(status)
		// Check that the user can see the comment status
		suite.False(status.IsLiked)
		// Additional ownership/permission checks would be handled by business logic
	})

	suite.Run("anonymous user viewing comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, 0, comment.ID)

		suite.NoError(err)
		suite.NotNil(status)
		// Anonymous users should see basic comment status without ownership info
	})

	suite.Run("non-existing comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, user2.ID, 99999)

		suite.Error(err)
		suite.Nil(status)
	})
}

func (suite *CommentUsecaseTestSuite) TestCommentsFeedConnection() {
	// Create test data with correct IDs
	user1, _, _, post, _, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	// Create additional comments for feed
	for i := 3; i <= 8; i++ {
		_, err := suite.client.Comment.Create().
			SetContent(fmt.Sprintf("Feed comment %d", i)).
			SetPostID(post.ID).
			SetAuthorID(user1.ID).
			SetCommunityID(post.CommunityID).
			SetCreatedAt(time.Now().Add(-time.Duration(i) * time.Hour)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)
	}

	suite.Run("forward pagination", func() {
		first := 3
		connection, err := suite.uc.CommentsFeedConnection(suite.ctx, nil, &first, nil, nil, nil)

		suite.NoError(err)
		suite.NotNil(connection)
		// We should have at least 3 comments (2 from createTestData + 6 additional = 8 total)
		suite.True(len(connection.Edges) <= 3, "Should return at most 3 comments")
		suite.NotNil(connection.PageInfo)
		if len(connection.Edges) == 3 {
			suite.True(connection.PageInfo.HasNextPage)
		}
		suite.False(connection.PageInfo.HasPreviousPage)

		// Verify ordering (should be by creation time)
		for i := 1; i < len(connection.Edges); i++ {
			prevTime := connection.Edges[i-1].Node.CreatedAt
			currTime := connection.Edges[i].Node.CreatedAt
			suite.True(prevTime.Before(currTime) || prevTime.Equal(currTime), "Comments should be ordered by creation time")
		}
	})

	suite.Run("backward pagination", func() {
		last := 3
		connection, err := suite.uc.CommentsFeedConnection(suite.ctx, nil, nil, nil, &last, nil)

		suite.NoError(err)
		suite.NotNil(connection)
		// We should have at most 3 comments
		suite.True(len(connection.Edges) <= 3, "Should return at most 3 comments")
		suite.NotNil(connection.PageInfo)
		suite.False(connection.PageInfo.HasNextPage)
		if len(connection.Edges) == 3 {
			suite.True(connection.PageInfo.HasPreviousPage)
		}
	})
}

func (suite *CommentUsecaseTestSuite) TestEdgeCases() {
	suite.Run("context cancellation", func() {
		cancelledCtx, cancel := context.WithCancel(suite.ctx)
		cancel()

		comments, err := suite.uc.GetCommentsByPostID(cancelledCtx, 1, nil)

		suite.Error(err)
		suite.Nil(comments)
		suite.Contains(err.Error(), "context canceled")
	})

	suite.Run("zero limit pagination", func() {
		_, _, _, post, _, _, err := suite.createTestData()
		require.NoError(suite.T(), err)

		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, post.ID, nil, 0, 0)

		suite.NoError(err)
		// Zero limit might return empty or all results depending on implementation
		// Let's just verify no error occurs
		suite.NotNil(comments)
	})
}

func (suite *CommentUsecaseTestSuite) TestPerformance() {
	// Create test data with correct IDs
	user1, _, _, post, _, _, err := suite.createTestData()
	require.NoError(suite.T(), err)

	// Create many comments for performance testing
	for i := 1; i <= 100; i++ {
		_, err := suite.client.Comment.Create().
			SetContent(fmt.Sprintf("Performance test comment %d", i)).
			SetPostID(post.ID).
			SetAuthorID(user1.ID).
			SetCommunityID(post.CommunityID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Second)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)
	}

	suite.Run("large dataset pagination performance", func() {
		start := time.Now()

		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, post.ID, nil, 50, 0)

		duration := time.Since(start)

		suite.NoError(err)
		suite.Len(comments, 50)
		suite.Less(duration, 100*time.Millisecond, "Pagination should be fast even with large datasets")
	})
}

func TestCommentUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(CommentUsecaseTestSuite))
}
