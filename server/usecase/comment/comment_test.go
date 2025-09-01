package comment

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/ent/post"
	"stormlink/tests/fixtures"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func setupTestClient(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return client
}

func TestCommentUsecase_GetCommentsByPostID(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("get comments for existing post", func(t *testing.T) {
		comments, err := uc.GetCommentsByPostID(ctx, fixtures.TestPost1.ID, nil)

		assert.NoError(t, err)
		assert.NotNil(t, comments)
		assert.Len(t, comments, 2) // TestComment1 and TestReply1

		// Verify comments are ordered by creation time (ASC)
		if len(comments) > 1 {
			assert.True(t, comments[0].CreatedAt.Before(comments[1].CreatedAt) ||
				comments[0].CreatedAt.Equal(comments[1].CreatedAt))
		}
	})

	t.Run("get comments with deleted filter", func(t *testing.T) {
		// Create a deleted comment
		deletedComment := fixtures.CommentFixture{
			ID:        99,
			Content:   "This comment is deleted",
			PostID:    fixtures.TestPost1.ID,
			AuthorID:  fixtures.TestUser1.ID,
			ParentID:  nil,
			CreatedAt: time.Now(),
		}

		createdComment, err := fixtures.CreateTestComment(ctx, client, deletedComment)
		require.NoError(t, err)

		// Mark as deleted
		_, err = client.Comment.UpdateOneID(createdComment.ID).
			SetHasDeleted(true).
			Save(ctx)
		require.NoError(t, err)

		// Test with hasDeleted = false (should exclude deleted)
		hasDeleted := false
		comments, err := uc.GetCommentsByPostID(ctx, fixtures.TestPost1.ID, &hasDeleted)
		assert.NoError(t, err)
		assert.Len(t, comments, 2) // Only non-deleted comments

		// Test with hasDeleted = true (should include only deleted)
		hasDeleted = true
		deletedComments, err := uc.GetCommentsByPostID(ctx, fixtures.TestPost1.ID, &hasDeleted)
		assert.NoError(t, err)
		assert.Len(t, deletedComments, 1) // Only deleted comment
		assert.Equal(t, createdComment.ID, deletedComments[0].ID)
	})

	t.Run("non-existing post", func(t *testing.T) {
		comments, err := uc.GetCommentsByPostID(ctx, 99999, nil)

		assert.NoError(t, err)
		assert.Empty(t, comments)
	})
}

func TestCommentUsecase_GetCommentsByPostIDLight(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("light version returns comments without heavy relations", func(t *testing.T) {
		comments, err := uc.GetCommentsByPostIDLight(ctx, fixtures.TestPost1.ID, nil)

		assert.NoError(t, err)
		assert.NotNil(t, comments)
		assert.Len(t, comments, 2)

		// Comments should be ordered by creation time (ASC)
		if len(comments) > 1 {
			assert.True(t, comments[0].CreatedAt.Before(comments[1].CreatedAt) ||
				comments[0].CreatedAt.Equal(comments[1].CreatedAt))
		}
	})
}

func TestCommentUsecase_GetCommentsByPostIDLightPaginated(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Create multiple comments for pagination testing
	postID := 1
	var commentFixtures []fixtures.CommentFixture

	// Create test post first
	testPost := fixtures.PostFixture{
		ID:          postID,
		Title:       "Test Post for Pagination",
		Content:     "Content",
		CommunityID: 1,
		AuthorID:    1,
		CreatedAt:   time.Now().Add(-1 * time.Hour),
	}

	// Seed basic data and create additional post
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	_, err = fixtures.CreateTestPost(ctx, client, testPost)
	require.NoError(t, err)

	// Create 10 comments for pagination testing
	for i := 0; i < 10; i++ {
		commentFixture := fixtures.CommentFixture{
			ID:        100 + i,
			Content:   fmt.Sprintf("Comment %d", i),
			PostID:    postID,
			AuthorID:  1,
			ParentID:  nil,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		commentFixtures = append(commentFixtures, commentFixture)
		_, err := fixtures.CreateTestComment(ctx, client, commentFixture)
		require.NoError(t, err)
	}

	t.Run("pagination with limit", func(t *testing.T) {
		comments, err := uc.GetCommentsByPostIDLightPaginated(ctx, postID, nil, 5, 0)

		assert.NoError(t, err)
		assert.Len(t, comments, 5)

		// Should be ordered by creation time ASC
		for i := 1; i < len(comments); i++ {
			assert.True(t, comments[i-1].CreatedAt.Before(comments[i].CreatedAt) ||
				comments[i-1].CreatedAt.Equal(comments[i].CreatedAt))
		}
	})

	t.Run("pagination with offset", func(t *testing.T) {
		// Get first page
		firstPage, err := uc.GetCommentsByPostIDLightPaginated(ctx, postID, nil, 3, 0)
		require.NoError(t, err)
		require.Len(t, firstPage, 3)

		// Get second page
		secondPage, err := uc.GetCommentsByPostIDLightPaginated(ctx, postID, nil, 3, 3)
		assert.NoError(t, err)
		assert.Len(t, secondPage, 3)

		// Verify no overlap
		for _, first := range firstPage {
			for _, second := range secondPage {
				assert.NotEqual(t, first.ID, second.ID)
			}
		}
	})

	t.Run("zero limit returns all", func(t *testing.T) {
		comments, err := uc.GetCommentsByPostIDLightPaginated(ctx, postID, nil, 0, 0)

		assert.NoError(t, err)
		assert.Len(t, comments, 10) // All comments
	})
}

func TestCommentUsecase_GetCommentsFeed(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	// Update posts to be published
	_, err = client.Post.Update().
		SetVisibility(post.VisibilityPublished).
		Save(ctx)
	require.NoError(t, err)

	t.Run("get recent comments feed", func(t *testing.T) {
		comments, err := uc.GetCommentsFeed(ctx, 10)

		assert.NoError(t, err)
		assert.NotNil(t, comments)
		assert.LessOrEqual(t, len(comments), 10)

		// Should be ordered by creation time DESC (newest first)
		for i := 1; i < len(comments); i++ {
			assert.True(t, comments[i-1].CreatedAt.After(comments[i].CreatedAt) ||
				comments[i-1].CreatedAt.Equal(comments[i].CreatedAt))
		}
	})

	t.Run("feed with limit", func(t *testing.T) {
		comments, err := uc.GetCommentsFeed(ctx, 1)

		assert.NoError(t, err)
		assert.LessOrEqual(t, len(comments), 1)
	})

	t.Run("feed excludes deleted comments", func(t *testing.T) {
		// Create and mark a comment as deleted
		deletedComment := fixtures.CommentFixture{
			ID:        199,
			Content:   "Deleted comment",
			PostID:    fixtures.TestPost1.ID,
			AuthorID:  fixtures.TestUser1.ID,
			CreatedAt: time.Now(),
		}

		created, err := fixtures.CreateTestComment(ctx, client, deletedComment)
		require.NoError(t, err)

		_, err = client.Comment.UpdateOneID(created.ID).
			SetHasDeleted(true).
			Save(ctx)
		require.NoError(t, err)

		comments, err := uc.GetCommentsFeed(ctx, 100)
		assert.NoError(t, err)

		// Verify deleted comment is not in feed
		for _, comment := range comments {
			assert.NotEqual(t, created.ID, comment.ID)
		}
	})
}

func TestCommentUsecase_CommentByID(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("existing comment", func(t *testing.T) {
		// Get the first test comment
		allComments, err := client.Comment.Query().All(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, allComments)

		targetComment := allComments[0]

		comment, err := uc.CommentByID(ctx, targetComment.ID)

		assert.NoError(t, err)
		assert.NotNil(t, comment)
		assert.Equal(t, targetComment.ID, comment.ID)
		assert.Equal(t, targetComment.Content, comment.Content)
	})

	t.Run("non-existing comment", func(t *testing.T) {
		comment, err := uc.CommentByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, comment)
		assert.True(t, ent.IsNotFound(err))
	})
}

func TestCommentUsecase_CommentsByPostConnection(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Create test data with multiple comments
	postID := 1
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	// Create additional comments with specific timestamps
	baseTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 5; i++ {
		commentFixture := fixtures.CommentFixture{
			ID:        200 + i,
			Content:   fmt.Sprintf("Test comment %d", i),
			PostID:    fixtures.TestPost1.ID,
			AuthorID:  fixtures.TestUser1.ID,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}
		_, err := fixtures.CreateTestComment(ctx, client, commentFixture)
		require.NoError(t, err)
	}

	t.Run("forward pagination (first/after)", func(t *testing.T) {
		first := 3
		connection, err := uc.CommentsByPostConnection(ctx, fixtures.TestPost1.ID, nil, &first, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.LessOrEqual(t, len(connection.Edges), 3)
		assert.NotNil(t, connection.PageInfo)

		// Verify ordering (ASC by created_at, then ID)
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			assert.True(t, prev.CreatedAt.Before(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID < curr.ID))
		}

		if len(connection.Edges) > 0 {
			assert.NotNil(t, connection.PageInfo.StartCursor)
			assert.NotNil(t, connection.PageInfo.EndCursor)
		}
	})

	t.Run("backward pagination (last/before)", func(t *testing.T) {
		last := 2
		connection, err := uc.CommentsByPostConnection(ctx, fixtures.TestPost1.ID, nil, nil, nil, &last, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.LessOrEqual(t, len(connection.Edges), 2)

		// Should still be ordered ASC for client consumption
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			assert.True(t, prev.CreatedAt.Before(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID < curr.ID))
		}
	})

	t.Run("pagination with after cursor", func(t *testing.T) {
		// Get first page
		first := 2
		firstPage, err := uc.CommentsByPostConnection(ctx, fixtures.TestPost1.ID, nil, &first, nil, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, firstPage.Edges)

		// Use end cursor for next page
		endCursor := *firstPage.PageInfo.EndCursor
		secondPage, err := uc.CommentsByPostConnection(ctx, fixtures.TestPost1.ID, nil, &first, &endCursor, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, secondPage)

		// Verify no overlap between pages
		if len(secondPage.Edges) > 0 {
			lastFromFirst := firstPage.Edges[len(firstPage.Edges)-1].Node
			firstFromSecond := secondPage.Edges[0].Node
			assert.True(t, lastFromFirst.CreatedAt.Before(firstFromSecond.CreatedAt) ||
				(lastFromFirst.CreatedAt.Equal(firstFromSecond.CreatedAt) && lastFromFirst.ID < firstFromSecond.ID))
		}
	})

	t.Run("empty result when no first or last specified", func(t *testing.T) {
		connection, err := uc.CommentsByPostConnection(ctx, fixtures.TestPost1.ID, nil, nil, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Empty(t, connection.Edges)
		assert.False(t, connection.PageInfo.HasNextPage)
		assert.False(t, connection.PageInfo.HasPreviousPage)
	})

	t.Run("invalid cursor", func(t *testing.T) {
		invalidCursor := "invalid-cursor"
		first := 3
		connection, err := uc.CommentsByPostConnection(ctx, fixtures.TestPost1.ID, nil, &first, &invalidCursor, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, connection)
	})
}

func TestCommentUsecase_CommentsWindow(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	// Create additional comments around anchor
	baseTime := time.Now().Add(-1 * time.Hour)
	var createdComments []*ent.Comment

	for i := 0; i < 10; i++ {
		commentFixture := fixtures.CommentFixture{
			ID:        300 + i,
			Content:   fmt.Sprintf("Window comment %d", i),
			PostID:    fixtures.TestPost1.ID,
			AuthorID:  fixtures.TestUser1.ID,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}
		created, err := fixtures.CreateTestComment(ctx, client, commentFixture)
		require.NoError(t, err)
		createdComments = append(createdComments, created)
	}

	t.Run("window around middle comment", func(t *testing.T) {
		// Use middle comment as anchor
		anchorComment := createdComments[5]

		connection, err := uc.CommentsWindow(ctx, fixtures.TestPost1.ID, anchorComment.ID, 2, 2, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.LessOrEqual(t, len(connection.Edges), 5) // 2 before + anchor + 2 after

		// Find anchor in results
		anchorFound := false
		for _, edge := range connection.Edges {
			if edge.Node.ID == anchorComment.ID {
				anchorFound = true
				break
			}
		}
		assert.True(t, anchorFound, "Anchor comment should be included in window")

		// Verify ordering
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			assert.True(t, prev.CreatedAt.Before(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID < curr.ID))
		}
	})

	t.Run("window at beginning", func(t *testing.T) {
		// Use first comment as anchor
		anchorComment := createdComments[0]

		connection, err := uc.CommentsWindow(ctx, fixtures.TestPost1.ID, anchorComment.ID, 5, 3, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		// Should have anchor + up to 3 after (no before since it's first)
		assert.LessOrEqual(t, len(connection.Edges), 4)
		assert.False(t, connection.PageInfo.HasPreviousPage)
	})

	t.Run("window at end", func(t *testing.T) {
		// Use last comment as anchor
		anchorComment := createdComments[len(createdComments)-1]

		connection, err := uc.CommentsWindow(ctx, fixtures.TestPost1.ID, anchorComment.ID, 3, 5, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		// Should have up to 3 before + anchor (no after since it's last)
		assert.LessOrEqual(t, len(connection.Edges), 4)
		assert.False(t, connection.PageInfo.HasNextPage)
	})

	t.Run("anchor not belonging to post", func(t *testing.T) {
		// Create comment for different post
		differentPost := fixtures.PostFixture{
			ID:          999,
			Title:       "Different Post",
			Content:     "Different content",
			CommunityID: 1,
			AuthorID:    1,
			CreatedAt:   time.Now(),
		}
		_, err := fixtures.CreateTestPost(ctx, client, differentPost)
		require.NoError(t, err)

		differentComment := fixtures.CommentFixture{
			ID:        999,
			Content:   "Comment in different post",
			PostID:    999,
			AuthorID:  1,
			CreatedAt: time.Now(),
		}
		differentCommentEnt, err := fixtures.CreateTestComment(ctx, client, differentComment)
		require.NoError(t, err)

		// Try to use this comment as anchor for our original post
		connection, err := uc.CommentsWindow(ctx, fixtures.TestPost1.ID, differentCommentEnt.ID, 2, 2, nil)

		assert.Error(t, err)
		assert.Nil(t, connection)
		assert.Contains(t, err.Error(), "anchor does not belong to post")
	})

	t.Run("non-existing anchor", func(t *testing.T) {
		connection, err := uc.CommentsWindow(ctx, fixtures.TestPost1.ID, 99999, 2, 2, nil)

		assert.Error(t, err)
		assert.Nil(t, connection)
		assert.True(t, ent.IsNotFound(err))
	})
}

func TestCommentUsecase_CommentsFeedConnection(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	// Seed basic test data and ensure posts are published
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	_, err = client.Post.Update().
		SetVisibility(post.VisibilityPublished).
		Save(ctx)
	require.NoError(t, err)

	// Create additional comments for testing
	baseTime := time.Now().Add(-2 * time.Hour)
	for i := 0; i < 8; i++ {
		commentFixture := fixtures.CommentFixture{
			ID:        400 + i,
			Content:   fmt.Sprintf("Feed comment %d", i),
			PostID:    fixtures.TestPost1.ID,
			AuthorID:  fixtures.TestUser1.ID,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}
		_, err := fixtures.CreateTestComment(ctx, client, commentFixture)
		require.NoError(t, err)
	}

	t.Run("forward pagination in feed", func(t *testing.T) {
		first := 5
		connection, err := uc.CommentsFeedConnection(ctx, nil, &first, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.LessOrEqual(t, len(connection.Edges), 5)

		// Should be ordered DESC (newest first)
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			assert.True(t, prev.CreatedAt.After(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID > curr.ID))
		}
	})

	t.Run("backward pagination in feed", func(t *testing.T) {
		last := 3
		connection, err := uc.CommentsFeedConnection(ctx, nil, nil, nil, &last, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.LessOrEqual(t, len(connection.Edges), 3)

		// Should still be ordered DESC for client
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			assert.True(t, prev.CreatedAt.After(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID > curr.ID))
		}
	})

	t.Run("pagination with cursors", func(t *testing.T) {
		// Get first page
		first := 3
		firstPage, err := uc.CommentsFeedConnection(ctx, nil, &first, nil, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, firstPage.Edges)

		// Get next page using end cursor
		endCursor := *firstPage.PageInfo.EndCursor
		secondPage, err := uc.CommentsFeedConnection(ctx, nil, &first, &endCursor, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, secondPage)

		// Verify no overlap and correct ordering
		if len(secondPage.Edges) > 0 {
			lastFromFirst := firstPage.Edges[len(firstPage.Edges)-1].Node
			firstFromSecond := secondPage.Edges[0].Node

			// In DESC order, second page should have older comments
			assert.True(t, lastFromFirst.CreatedAt.After(firstFromSecond.CreatedAt) ||
				(lastFromFirst.CreatedAt.Equal(firstFromSecond.CreatedAt) && lastFromFirst.ID > firstFromSecond.ID))
		}
	})

	t.Run("empty parameters returns empty connection", func(t *testing.T) {
		connection, err := uc.CommentsFeedConnection(ctx, nil, nil, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, connection)
		assert.Empty(t, connection.Edges)
		assert.False(t, connection.PageInfo.HasNextPage)
		assert.False(t, connection.PageInfo.HasPreviousPage)
	})

	t.Run("filter by deleted status", func(t *testing.T) {
		// Create deleted comment
		deletedComment := fixtures.CommentFixture{
			ID:        500,
			Content:   "Deleted feed comment",
			PostID:    fixtures.TestPost1.ID,
			AuthorID:  fixtures.TestUser1.ID,
			CreatedAt: time.Now(),
		}
		created, err := fixtures.CreateTestComment(ctx, client, deletedComment)
		require.NoError(t, err)

		_, err = client.Comment.UpdateOneID(created.ID).
			SetHasDeleted(true).
			Save(ctx)
		require.NoError(t, err)

		// Test with hasDeleted = false (default behavior)
		hasDeleted := false
		first := 20
		connection, err := uc.CommentsFeedConnection(ctx, &hasDeleted, &first, nil, nil, nil)
		assert.NoError(t, err)

		// Verify deleted comment is not included
		for _, edge := range connection.Edges {
			assert.NotEqual(t, created.ID, edge.Node.ID)
		}

		// Test with hasDeleted = true (should include deleted)
		hasDeleted = true
		deletedConnection, err := uc.CommentsFeedConnection(ctx, &hasDeleted, &first, nil, nil, nil)
		assert.NoError(t, err)

		// Should find the deleted comment
		found := false
		for _, edge := range deletedConnection.Edges {
			if edge.Node.ID == created.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Deleted comment should be found when hasDeleted=true")
	})
}

func TestCommentUsecase_CursorEncoding(t *testing.T) {
	// Test cursor encoding/decoding functions
	testTime := time.Now().UTC()
	testID := 123

	t.Run("encode and decode cursor", func(t *testing.T) {
		// Create mock comment for cursor generation
		comment := &ent.Comment{
			ID:        testID,
			CreatedAt: testTime,
		}

		// Generate cursor
		key := cursorKey(comment)
		cursor := encodeCursor(key)

		// Decode cursor
		decodedTime, decodedID, err := decodeCursor(cursor)

		assert.NoError(t, err)
		assert.True(t, testTime.Equal(decodedTime))
		assert.Equal(t, testID, decodedID)
	})

	t.Run("invalid cursor formats", func(t *testing.T) {
		// Test invalid base64
		_, _, err := decodeCursor("invalid-base64!")
		assert.Error(t, err)

		// Test invalid cursor structure
		invalidCursor := base64.StdEncoding.EncodeToString([]byte("invalid"))
		_, _, err = decodeCursor(invalidCursor)
		assert.Error(t, err)

		// Test invalid time format
		invalidTimeCursor := base64.StdEncoding.EncodeToString([]byte("invalid-time|123"))
		_, _, err = decodeCursor(invalidTimeCursor)
		assert.Error(t, err)

		// Test invalid ID format
		invalidIDCursor := base64.StdEncoding.EncodeToString([]byte("2023-01-01T00:00:00Z|invalid-id"))
		_, _, err = decodeCursor(invalidIDCursor)
		assert.Error(t, err)
	})
}

func TestCommentUsecase_EdgeCases(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	uc := NewCommentUsecase(client)
	ctx := context.Background()

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		comments, err := uc.GetCommentsByPostID(cancelledCtx, 1, nil)

		assert.Error(t, err)
		assert.Nil(t, comments)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("zero limit in pagination", func(t *testing.T) {
		comments, err := uc.GetCommentsByPostIDLightPaginated(ctx, 1, nil, 0, 0)

		assert.NoError(t, err)
		assert.NotNil(t, comments)
	})

	t.Run("negative values in pagination", func(t *testing.T) {
		// Should handle negative values gracefully
		comments, err := uc.GetCommentsByPostIDLightPaginated(ctx, 1, nil, -1, -1)

		assert.NoError(t, err)
