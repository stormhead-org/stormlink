package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/graphql/models"
	"stormlink/server/usecase/post"
	"stormlink/tests/fixtures"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type PostUsecaseTestSuite struct {
	suite.Suite
	client *ent.Client
	uc     post.PostUsecase
	ctx    context.Context
}

func (suite *PostUsecaseTestSuite) SetupSuite() {
	suite.client = enttest.Open(suite.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	suite.uc = post.NewPostUsecase(suite.client)
	suite.ctx = context.Background()
}

func (suite *PostUsecaseTestSuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *PostUsecaseTestSuite) SetupTest() {
	// Clean up data before each test
	suite.client.Comment.Delete().ExecX(suite.ctx)
	suite.client.PostLike.Delete().ExecX(suite.ctx)
	suite.client.PostBookmark.Delete().ExecX(suite.ctx)
	suite.client.Post.Delete().ExecX(suite.ctx)
	suite.client.Community.Delete().ExecX(suite.ctx)
	suite.client.User.Delete().ExecX(suite.ctx)
	suite.client.Media.Delete().ExecX(suite.ctx)
}

func (suite *PostUsecaseTestSuite) TestGetPostByID() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("get existing post with basic data", func() {
		post, err := suite.uc.GetPostByID(suite.ctx, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(post)
		suite.Equal(fixtures.TestPost1.ID, post.ID)
		suite.Equal(fixtures.TestPost1.Title, post.Title)
		suite.Equal(fixtures.TestPost1.Content, post.Content)
		suite.Equal(fixtures.TestPost1.AuthorID, post.AuthorID)
		suite.Equal(fixtures.TestPost1.CommunityID, post.CommunityID)
	})

	suite.Run("get non-existing post", func() {
		post, err := suite.uc.GetPostByID(suite.ctx, 99999)

		suite.Error(err)
		suite.Nil(post)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("post with all relations loaded", func() {
		// Create hero image
		heroImage, err := fixtures.CreateTestMedia(suite.ctx, suite.client, "hero.jpg", "https://example.com/hero.jpg")
		suite.Require().NoError(err)

		// Create post with hero image
		postWithHero, err := suite.client.Post.Create().
			SetTitle("Post with Hero Image").
			SetContent("This post has a hero image").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHeroImageID(heroImage.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create some likes and bookmarks
		_, err = suite.client.PostLike.Create().
			SetPostID(postWithHero.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		_, err = suite.client.PostBookmark.Create().
			SetPostID(postWithHero.ID).
			SetUserID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Test retrieval with relations
		retrievedPost, err := suite.uc.GetPostByID(suite.ctx, postWithHero.ID)
		suite.NoError(err)
		suite.NotNil(retrievedPost)

		// Verify relations are loaded
		heroEdge, err := retrievedPost.QueryHeroImage().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(heroImage.ID, heroEdge.ID)

		authorEdge, err := retrievedPost.QueryAuthor().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(fixtures.TestUser1.ID, authorEdge.ID)

		communityEdge, err := retrievedPost.QueryCommunity().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(fixtures.TestCommunity1.ID, communityEdge.ID)

		likes, err := retrievedPost.QueryLikes().All(suite.ctx)
		suite.NoError(err)
		suite.Len(likes, 1)

		bookmarks, err := retrievedPost.QueryBookmarks().All(suite.ctx)
		suite.NoError(err)
		suite.Len(bookmarks, 1)

		comments, err := retrievedPost.QueryComments().All(suite.ctx)
		suite.NoError(err)
		suite.NotEmpty(comments) // Should have comments from fixtures
	})

	suite.Run("post with multiple comments", func() {
		// Create additional comments for the test post
		for i := 3; i <= 5; i++ {
			_, err := fixtures.CreateTestComment(suite.ctx, suite.client, fixtures.CommentFixture{
				ID:        i,
				Content:   fmt.Sprintf("Additional comment %d", i),
				PostID:    fixtures.TestPost1.ID,
				AuthorID:  fixtures.TestUser1.ID,
				CreatedAt: time.Now(),
			})
			suite.Require().NoError(err)
		}

		post, err := suite.uc.GetPostByID(suite.ctx, fixtures.TestPost1.ID)
		suite.NoError(err)
		suite.NotNil(post)

		comments, err := post.QueryComments().All(suite.ctx)
		suite.NoError(err)
		suite.Len(comments, 5) // 2 from fixtures + 3 additional
	})
}

func (suite *PostUsecaseTestSuite) TestGetPostStatus() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("user viewing own post", func() {
		status, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestPost1.AuthorID, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.True(status.IsOwn)
		suite.False(status.IsLiked)
		suite.False(status.IsBookmarked)
		suite.Equal(models.PostStatusRelationshipOwn, status.Relationship)
	})

	suite.Run("user viewing another user's post", func() {
		// Ensure TestPost1 is authored by TestUser1, and we're viewing as TestUser2
		otherUserID := fixtures.TestUser2.ID
		if fixtures.TestPost1.AuthorID == fixtures.TestUser2.ID {
			otherUserID = fixtures.TestUser1.ID
		}

		status, err := suite.uc.GetPostStatus(suite.ctx, otherUserID, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwn)
		suite.False(status.IsLiked)
		suite.False(status.IsBookmarked)
		suite.Equal(models.PostStatusRelationshipNone, status.Relationship)
	})

	suite.Run("user who liked the post", func() {
		// Create a like
		_, err := suite.client.PostLike.Create().
			SetPostID(fixtures.TestPost1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		status, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.True(status.IsLiked)
		suite.False(status.IsBookmarked)
		if fixtures.TestPost1.AuthorID == fixtures.TestUser2.ID {
			suite.True(status.IsOwn)
			suite.Equal(models.PostStatusRelationshipOwn, status.Relationship)
		} else {
			suite.False(status.IsOwn)
			suite.Equal(models.PostStatusRelationshipLiked, status.Relationship)
		}
	})

	suite.Run("user who bookmarked the post", func() {
		// Create a bookmark
		_, err := suite.client.PostBookmark.Create().
			SetPostID(fixtures.TestPost1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		status, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.True(status.IsBookmarked)
		// Check if user also liked (from previous test)
		suite.True(status.IsLiked) // Should still be liked from previous test
	})

	suite.Run("anonymous user viewing post", func() {
		status, err := suite.uc.GetPostStatus(suite.ctx, 0, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwn)
		suite.False(status.IsLiked)
		suite.False(status.IsBookmarked)
		suite.Equal(models.PostStatusRelationshipNone, status.Relationship)
	})

	suite.Run("non-existing post", func() {
		status, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser1.ID, 99999)

		suite.Error(err)
		suite.Nil(status)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("non-existing user viewing existing post", func() {
		status, err := suite.uc.GetPostStatus(suite.ctx, 99999, fixtures.TestPost1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwn)
		suite.False(status.IsLiked)
		suite.False(status.IsBookmarked)
		suite.Equal(models.PostStatusRelationshipNone, status.Relationship)
	})
}

func (suite *PostUsecaseTestSuite) TestPostWithComplexRelationships() {
	// Create a more complex test scenario
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	// Create hero image
	heroImage, err := fixtures.CreateTestMedia(suite.ctx, suite.client, "complex-hero.jpg", "https://example.com/complex-hero.jpg")
	suite.Require().NoError(err)

	// Create complex post
	complexPost, err := suite.client.Post.Create().
		SetTitle("Complex Post").
		SetContent("This is a complex post with many relationships").
		SetCommunityID(fixtures.TestCommunity1.ID).
		SetAuthorID(fixtures.TestUser1.ID).
		SetHeroImageID(heroImage.ID).
		SetCreatedAt(time.Now().Add(-1 * time.Hour)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Create multiple likes from different users
	for i, userID := range []int{fixtures.TestUser2.ID, fixtures.UnverifiedUser.ID} {
		_, err := suite.client.PostLike.Create().
			SetPostID(complexPost.ID).
			SetUserID(userID).
			SetCreatedAt(time.Now().Add(-time.Duration(i*10) * time.Minute)).
			Save(suite.ctx)
		suite.Require().NoError(err)
	}

	// Create multiple bookmarks
	_, err = suite.client.PostBookmark.Create().
		SetPostID(complexPost.ID).
		SetUserID(fixtures.TestUser2.ID).
		SetCreatedAt(time.Now().Add(-30 * time.Minute)).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Create nested comments
	parentComment, err := suite.client.Comment.Create().
		SetContent("Parent comment on complex post").
		SetPostID(complexPost.ID).
		SetAuthorID(fixtures.TestUser2.ID).
		SetCreatedAt(time.Now().Add(-20 * time.Minute)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	_, err = suite.client.Comment.Create().
		SetContent("Reply to parent comment").
		SetPostID(complexPost.ID).
		SetAuthorID(fixtures.TestUser1.ID).
		SetParentID(parentComment.ID).
		SetCreatedAt(time.Now().Add(-10 * time.Minute)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	suite.Run("retrieve complex post with all relations", func() {
		retrievedPost, err := suite.uc.GetPostByID(suite.ctx, complexPost.ID)

		suite.NoError(err)
		suite.NotNil(retrievedPost)
		suite.Equal("Complex Post", retrievedPost.Title)

		// Verify hero image
		heroEdge, err := retrievedPost.QueryHeroImage().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(heroImage.ID, heroEdge.ID)

		// Verify author
		authorEdge, err := retrievedPost.QueryAuthor().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(fixtures.TestUser1.ID, authorEdge.ID)

		// Verify community
		communityEdge, err := retrievedPost.QueryCommunity().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(fixtures.TestCommunity1.ID, communityEdge.ID)

		// Verify likes
		likes, err := retrievedPost.QueryLikes().All(suite.ctx)
		suite.NoError(err)
		suite.Len(likes, 2)

		// Verify bookmarks
		bookmarks, err := retrievedPost.QueryBookmarks().All(suite.ctx)
		suite.NoError(err)
		suite.Len(bookmarks, 1)

		// Verify comments (including nested)
		comments, err := retrievedPost.QueryComments().All(suite.ctx)
		suite.NoError(err)
		suite.Len(comments, 2) // Parent and reply
	})

	suite.Run("status for user who liked and bookmarked", func() {
		status, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, complexPost.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwn) // TestUser2 is not the author
		suite.True(status.IsLiked)
		suite.True(status.IsBookmarked)
		// The relationship should reflect both actions, but liked might take precedence
		suite.True(status.Relationship == models.PostStatusRelationshipLiked ||
			status.Relationship == models.PostStatusRelationshipBookmarked)
	})

	suite.Run("status for author", func() {
		status, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser1.ID, complexPost.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.True(status.IsOwn)
		suite.False(status.IsLiked)      // Author didn't like own post
		suite.False(status.IsBookmarked) // Author didn't bookmark own post
		suite.Equal(models.PostStatusRelationshipOwn, status.Relationship)
	})
}

func (suite *PostUsecaseTestSuite) TestEdgeCases() {
	suite.Run("context cancellation", func() {
		cancelledCtx, cancel := context.WithCancel(suite.ctx)
		cancel()

		post, err := suite.uc.GetPostByID(cancelledCtx, 1)

		suite.Error(err)
		suite.Nil(post)
		suite.Contains(err.Error(), "context canceled")
	})

	suite.Run("post with missing hero image reference", func() {
		err := fixtures.SeedBasicData(suite.ctx, suite.client)
		suite.Require().NoError(err)

		// Create post with non-existent hero image ID
		postWithBadHero, err := suite.client.Post.Create().
			SetTitle("Post with Bad Hero").
			SetContent("This post has a bad hero image reference").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHeroImageID(99999). // Non-existent image ID
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Should still retrieve the post, but hero image query will fail
		retrievedPost, err := suite.uc.GetPostByID(suite.ctx, postWithBadHero.ID)
		suite.NoError(err)
		suite.NotNil(retrievedPost)

		// Hero image query should fail gracefully
		_, err = retrievedPost.QueryHeroImage().Only(suite.ctx)
		suite.Error(err)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("post in deleted community", func() {
		err := fixtures.SeedBasicData(suite.ctx, suite.client)
		suite.Require().NoError(err)

		// Create a post
		post, err := suite.client.Post.Create().
			SetTitle("Post in Community").
			SetContent("This post is in a community").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Delete the community
		err = suite.client.Community.DeleteOneID(fixtures.TestCommunity1.ID).Exec(suite.ctx)
		suite.Require().NoError(err)

		// Post retrieval should still work, but community query will fail
		retrievedPost, err := suite.uc.GetPostByID(suite.ctx, post.ID)
		suite.NoError(err)
		suite.NotNil(retrievedPost)

		// Community query should fail
		_, err = retrievedPost.QueryCommunity().Only(suite.ctx)
		suite.Error(err)
		suite.True(ent.IsNotFound(err))
	})
}

func (suite *PostUsecaseTestSuite) TestPostStatusComplexScenarios() {
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("user interactions over time", func() {
		// User likes post
		_, err := suite.client.PostLike.Create().
			SetPostID(fixtures.TestPost1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now().Add(-2 * time.Hour)).
			Save(suite.ctx)
		suite.Require().NoError(err)

		status1, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)
		suite.NoError(err)
		suite.True(status1.IsLiked)
		suite.False(status1.IsBookmarked)

		// User bookmarks post later
		_, err = suite.client.PostBookmark.Create().
			SetPostID(fixtures.TestPost1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now().Add(-1 * time.Hour)).
			Save(suite.ctx)
		suite.Require().NoError(err)

		status2, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)
		suite.NoError(err)
		suite.True(status2.IsLiked)
		suite.True(status2.IsBookmarked)

		// User unlikes post (delete like)
		err = suite.client.PostLike.Delete().
			Where(func(s *ent.PostLikeSelect) {
				s.Where(
					s.PostIDEQ(fixtures.TestPost1.ID),
					s.UserIDEQ(fixtures.TestUser2.ID),
				)
			}).
			Exec(suite.ctx)
		suite.Require().NoError(err)

		status3, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)
		suite.NoError(err)
		suite.False(status3.IsLiked)
		suite.True(status3.IsBookmarked)
		suite.Equal(models.PostStatusRelationshipBookmarked, status3.Relationship)
	})
}

func (suite *PostUsecaseTestSuite) TestPerformance() {
	// Create test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	// Create additional test posts
	postIDs := make([]int, 50)
	for i := 0; i < 50; i++ {
		post, err := suite.client.Post.Create().
			SetTitle(fmt.Sprintf("Performance Test Post %d", i)).
			SetContent(fmt.Sprintf("Content for performance test post %d", i)).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Minute)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)
		postIDs[i] = post.ID

		// Add some likes and comments to make it realistic
		if i%3 == 0 {
			_, err = suite.client.PostLike.Create().
				SetPostID(post.ID).
				SetUserID(fixtures.TestUser2.ID).
				SetCreatedAt(time.Now()).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}

		if i%5 == 0 {
			_, err = suite.client.Comment.Create().
				SetContent(fmt.Sprintf("Comment on post %d", i)).
				SetPostID(post.ID).
				SetAuthorID(fixtures.TestUser2.ID).
				SetCreatedAt(time.Now()).
				SetUpdatedAt(time.Now()).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}
	}

	suite.Run("bulk post retrieval performance", func() {
		start := time.Now()

		for i := 0; i < 10; i++ {
			_, err := suite.uc.GetPostByID(suite.ctx, postIDs[i])
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / 10

		suite.Less(avgDuration, 20*time.Millisecond, "Average post retrieval should be fast")
	})

	suite.Run("status retrieval performance", func() {
		start := time.Now()

		for i := 0; i < 20; i++ {
			_, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser1.ID, postIDs[i])
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / 20

		suite.Less(avgDuration, 15*time.Millisecond, "Average status retrieval should be fast")
	})
}

// Benchmark tests
func (suite *PostUsecaseTestSuite) TestBenchmarkPostRetrieval() {
	// Setup data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("benchmark GetPostByID", func() {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := suite.uc.GetPostByID(suite.ctx, fixtures.TestPost1.ID)
			suite.NoError(err)
		}

		avgDuration := time.Since(start) / time.Duration(iterations)
		suite.Less(avgDuration, 10*time.Millisecond, "Average GetPostByID should be very fast")
	})

	suite.Run("benchmark GetPostStatus", func() {
		// Create some interactions to make it realistic
		_, err := suite.client.PostLike.Create().
			SetPostID(fixtures.TestPost1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := suite.uc.GetPostStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)
			suite.NoError(err)
		}

		avgDuration := time.Since(start) / time.Duration(iterations)
		suite.Less(avgDuration, 8*time.Millisecond, "Average GetPostStatus should be very fast")
	})
}

func (suite *PostUsecaseTestSuite) TestConcurrentAccess() {
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("concurrent post retrieval", func() {
		done := make(chan bool, 10)
		errors := make(chan error, 10)

		// Start 10 concurrent goroutines
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				post, err := suite.uc.GetPostByID(suite.ctx, fixtures.TestPost1.ID)
				if err != nil {
					errors <- err
					return
				}

				if post == nil || post.ID != fixtures.TestPost1.ID {
					errors <- fmt.Errorf("invalid post retrieved")
				}
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check for any errors
		select {
		case err := <-errors:
			suite.Fail("Concurrent access failed", err.Error())
		default:
			// No errors, test passed
		}
	})
}

func TestPostUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(PostUsecaseTestSuite))
}
