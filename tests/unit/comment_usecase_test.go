package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/usecase/comment"
	"stormlink/tests/fixtures"
)

type CommentUsecaseTestSuite struct {
	suite.Suite
	client *ent.Client
	uc     comment.CommentUsecase
	ctx    context.Context
}

func (suite *CommentUsecaseTestSuite) SetupSuite() {
	suite.client = enttest.Open(suite.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	suite.uc = comment.NewCommentUsecase(suite.client)
	suite.ctx = context.Background()
}

func (suite *CommentUsecaseTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *CommentUsecaseTestSuite) SetupTest() {
	// Clean up data before each test
	suite.client.Comment.Delete().ExecX(suite.ctx)
	suite.client.Post.Delete().ExecX(suite.ctx)
	suite.client.Community.Delete().ExecX(suite.ctx)
	suite.client.User.Delete().ExecX(suite.ctx)
}

func (suite *CommentUsecaseTestSuite) TestGetCommentsByPostID() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	suite.Run("get comments for existing post", func() {
		comments, err := suite.uc.GetCommentsByPostID(suite.ctx, fixtures.TestPost1.ID, nil)

		suite.NoError(err)
		suite.NotNil(comments)
		suite.Len(comments, 2) // TestComment1 and TestReply1

		// Verify comment content
		foundComment := false
		foundReply := false
		for _, comment := range comments {
			if comment.ID == fixtures.TestComment1.ID {
				suite.Equal(fixtures.TestComment1.Content, comment.Content)
				suite.Equal(fixtures.TestComment1.AuthorID, comment.AuthorID)
				foundComment = true
			}
			if comment.ID == fixtures.TestReply1.ID {
				suite.Equal(fixtures.TestReply1.Content, comment.Content)
				suite.Equal(fixtures.TestReply1.AuthorID, comment.AuthorID)
				suite.NotNil(comment.ParentCommentID)
				suite.Equal(fixtures.TestComment1.ID, *comment.ParentCommentID)
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
			SetPostID(fixtures.TestPost1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHasDeleted(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)

		// Test filtering for non-deleted comments
		hasDeleted := false
		comments, err := suite.uc.GetCommentsByPostID(suite.ctx, fixtures.TestPost1.ID, &hasDeleted)
		suite.NoError(err)
		suite.Len(comments, 2) // Should only get non-deleted comments

		// Test filtering for deleted comments
		hasDeleted = true
		deletedComments, err := suite.uc.GetCommentsByPostID(suite.ctx, fixtures.TestPost1.ID, &hasDeleted)
		suite.NoError(err)
		suite.Len(deletedComments, 1)
		suite.Equal(deletedComment.ID, deletedComments[0].ID)
	})
}

func (suite *CommentUsecaseTestSuite) TestGetCommentsByPostIDLight() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	suite.Run("light version returns comments without heavy relations", func() {
		comments, err := suite.uc.GetCommentsByPostIDLight(suite.ctx, fixtures.TestPost1.ID, nil)

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
	// Seed test data and create more comments for pagination testing
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	// Create additional comments for pagination testing
	for i := 3; i <= 10; i++ {
		_, err := suite.client.Comment.Create().
			SetContent(fmt.Sprintf("Test comment %d", i)).
			SetPostID(fixtures.TestPost1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Minute)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)
	}

	suite.Run("paginated results with limit and offset", func() {
		// Test first page
		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, fixtures.TestPost1.ID, nil, 5, 0)
		suite.NoError(err)
		suite.Len(comments, 5)

		// Test second page
		comments2, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, fixtures.TestPost1.ID, nil, 5, 5)
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
		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, fixtures.TestPost1.ID, nil, 100, 0)
		suite.NoError(err)
		suite.Len(comments, 10) // Should return all 10 comments
	})

	suite.Run("offset beyond available comments", func() {
		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, fixtures.TestPost1.ID, nil, 5, 100)
		suite.NoError(err)
		suite.Empty(comments)
	})
}

func (suite *CommentUsecaseTestSuite) TestCommentByID() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	suite.Run("get existing comment", func() {
		comment, err := suite.uc.CommentByID(suite.ctx, fixtures.TestComment1.ID)

		suite.NoError(err)
		suite.NotNil(comment)
		suite.Equal(fixtures.TestComment1.ID, comment.ID)
		suite.Equal(fixtures.TestComment1.Content, comment.Content)
		suite.Equal(fixtures.TestComment1.AuthorID, comment.AuthorID)
		suite.Equal(fixtures.TestComment1.PostID, comment.PostID)
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
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	suite.Run("user viewing own comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, fixtures.TestComment1.AuthorID, fixtures.TestComment1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsLiked)
		// For now, we'll check basic fields available in CommentStatus
		// The owner check would need to be implemented in the usecase logic
	})

	suite.Run("user viewing another user's comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, fixtures.TestUser1.ID, fixtures.TestComment1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		// Check that the user can see the comment status
		suite.False(status.IsLiked)
		// Additional ownership/permission checks would be handled by business logic
	})

	suite.Run("anonymous user viewing comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, 0, fixtures.TestComment1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsLiked)
		// Anonymous users should see basic comment status without ownership info
	})

	suite.Run("non-existing comment", func() {
		status, err := suite.uc.GetCommentStatus(suite.ctx, fixtures.TestUser1.ID, 99999)

		suite.Error(err)
		suite.Nil(status)
	})
}

func (suite *CommentUsecaseTestSuite) TestCommentsFeedConnection() {
	// Create multiple comments across different posts for feed testing
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	// Create additional comments for feed
	for i := 3; i <= 8; i++ {
		_, err := suite.client.Comment.Create().
			SetContent(fmt.Sprintf("Feed comment %d", i)).
			SetPostID(fixtures.TestPost1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
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
		suite.Len(connection.Edges, 3)
		suite.NotNil(connection.PageInfo)
		suite.True(connection.PageInfo.HasNextPage)
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
		suite.Len(connection.Edges, 3)
		suite.NotNil(connection.PageInfo)
		suite.False(connection.PageInfo.HasNextPage)
		suite.True(connection.PageInfo.HasPreviousPage)
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
		err := fixtures.SeedBasicData(suite.ctx, suite.client)
		require.NoError(suite.T(), err)

		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, fixtures.TestPost1.ID, nil, 0, 0)

		suite.NoError(err)
		suite.Empty(comments)
	})
}

func (suite *CommentUsecaseTestSuite) TestPerformance() {
	// Create a large number of comments for performance testing
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	require.NoError(suite.T(), err)

	// Create 1000 comments
	for i := 0; i < 1000; i++ {
		_, err := suite.client.Comment.Create().
			SetContent(fmt.Sprintf("Performance comment %d", i)).
			SetPostID(fixtures.TestPost1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Second)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)
	}

	suite.Run("large dataset pagination performance", func() {
		start := time.Now()

		comments, err := suite.uc.GetCommentsByPostIDLightPaginated(suite.ctx, fixtures.TestPost1.ID, nil, 50, 0)

		duration := time.Since(start)

		suite.NoError(err)
		suite.Len(comments, 50)
		suite.Less(duration, 100*time.Millisecond, "Pagination should be fast even with large datasets")
	})
}

func TestCommentUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(CommentUsecaseTestSuite))
}
