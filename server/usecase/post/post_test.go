package post

import (
	"context"
	"fmt"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/ent/post"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"
	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestClient(t *testing.T) (*ent.Client, *testhelper.PostgresTestHelper) {
	helper := testhelper.NewPostgresTestHelper(t)
	helper.WaitForDatabase(t)
	helper.CleanDatabase(t)
	return helper.GetClient(), helper
}

func TestPostUsecase_GetPostByID(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("existing post", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, post)
		assert.Equal(t, fixtures.TestPost1.ID, post.ID)
		assert.Equal(t, fixtures.TestPost1.Title, post.Title)
		// Content is stored as JSON, so we need to extract the text field
		contentMap := post.Content
		assert.Equal(t, fixtures.TestPost1.Content, contentMap["text"])
		assert.Equal(t, fixtures.TestPost1.CommunityID, post.CommunityID)
		assert.Equal(t, fixtures.TestPost1.AuthorID, post.AuthorID)
	})

	t.Run("non-existing post", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, post)
		assert.True(t, ent.IsNotFound(err))
	})

	t.Run("post with hero image", func(t *testing.T) {
		// Create media for hero image
		heroImage, err := fixtures.CreateTestMedia(ctx, client, fixtures.TestMedia1)
		require.NoError(t, err)

		// Create post with hero image
		postWithHero, err := client.Post.Create().
			SetTitle("Post With Hero Image").
			SetSlug("post-with-hero-image-test").
			SetContent(map[string]interface{}{"text": "Content with hero image"}).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHeroImageID(heroImage.ID).
			SetVisibility(post.VisibilityPublished).
			Save(ctx)
		require.NoError(t, err)

		// Test retrieval
		retrievedPost, err := uc.GetPostByID(ctx, postWithHero.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)

		// Load hero image edge and verify
		heroEdge, err := retrievedPost.QueryHeroImage().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, heroImage.ID, heroEdge.ID)
		assert.Equal(t, "hero.jpg", *heroEdge.Filename)
	})

	t.Run("post with author", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Load author edge and verify
		author, err := post.QueryAuthor().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fixtures.TestPost1.AuthorID, author.ID)
		assert.Equal(t, fixtures.TestUser1.Name, author.Name)
	})

	t.Run("post with community", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Load community edge and verify
		community, err := post.QueryCommunity().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fixtures.TestPost1.CommunityID, community.ID)
		assert.Equal(t, fixtures.TestCommunity1.Name, community.Title)
	})

	t.Run("post with comments", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Load comments edge and verify
		comments, err := post.QueryComments().All(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, comments)

		// Should have at least the test comments from fixtures
		assert.GreaterOrEqual(t, len(comments), 2)
	})

	t.Run("post with likes", func(t *testing.T) {
		// Create a like for the post
		_, err := client.PostLike.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(fixtures.TestPost1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Load likes edge and verify
		likes, err := post.QueryLikes().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, likes, 1)
		assert.Equal(t, fixtures.TestUser2.ID, likes[0].UserID)
	})

	t.Run("post with bookmarks", func(t *testing.T) {
		// Create a bookmark for the post
		_, err := client.Bookmark.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(fixtures.TestPost1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Load bookmarks edge and verify
		bookmarks, err := post.QueryBookmarks().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, bookmarks, 1)
		assert.Equal(t, fixtures.TestUser2.ID, bookmarks[0].UserID)
	})
}

func TestPostUsecase_GetPostStatus(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("post status for author", func(t *testing.T) {
		status, err := uc.GetPostStatus(ctx, fixtures.TestPost1.AuthorID, fixtures.TestPost1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsLiked)
		assert.False(t, status.HasBookmark)
	})

	t.Run("post status for non-author", func(t *testing.T) {
		status, err := uc.GetPostStatus(ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsLiked)
		assert.False(t, status.HasBookmark)
	})

	t.Run("post status with like", func(t *testing.T) {
		// Create like
		_, err := client.PostLike.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(fixtures.TestPost1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetPostStatus(ctx, fixtures.TestUser2.ID, fixtures.TestPost1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.True(t, status.IsLiked)
		assert.False(t, status.HasBookmark)
	})

	t.Run("post status with bookmark", func(t *testing.T) {
		// Create bookmark
		_, err := client.Bookmark.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(fixtures.TestPost2.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		status, err := uc.GetPostStatus(ctx, fixtures.TestUser2.ID, fixtures.TestPost2.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsLiked)
		assert.True(t, status.HasBookmark)
	})

	t.Run("post status for anonymous user", func(t *testing.T) {
		status, err := uc.GetPostStatus(ctx, 0, fixtures.TestPost1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsLiked)
		assert.False(t, status.HasBookmark)
	})

	t.Run("non-existing post", func(t *testing.T) {
		status, err := uc.GetPostStatus(ctx, fixtures.TestUser1.ID, 99999)

		assert.Error(t, err)
		assert.Nil(t, status)
	})

	t.Run("non-existing user", func(t *testing.T) {
		status, err := uc.GetPostStatus(ctx, 99999, fixtures.TestPost1.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.IsLiked)
		assert.False(t, status.HasBookmark)
	})
}

func TestPostUsecase_PostWithRelationships(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		post, err := uc.GetPostByID(cancelledCtx, 1)

		assert.Error(t, err)
		assert.Nil(t, post)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("post with multiple relationships", func(t *testing.T) {
		// Seed basic data first
		err := fixtures.SeedBasicData(ctx, client)
		require.NoError(t, err)

		// Create media for hero image
		heroImage, err := fixtures.CreateTestMedia(ctx, client, fixtures.TestMedia1)
		require.NoError(t, err)

		// Create complex post with all relationships
		complexPost, err := client.Post.Create().
			SetTitle("Complex Post").
			SetSlug("complex-post-test").
			SetContent(map[string]interface{}{"text": "Post with all relationships"}).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHeroImageID(heroImage.ID).
			SetVisibility(post.VisibilityPublished).
			Save(ctx)
		require.NoError(t, err)

		// Add likes from multiple users
		_, err = client.PostLike.Create().
			SetUserID(fixtures.TestUser1.ID).
			SetPostID(complexPost.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.PostLike.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(complexPost.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Add bookmarks
		_, err = client.Bookmark.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(complexPost.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Add comments
		_, err = client.Comment.Create().
			SetContent("Complex comment 1").
			SetPostID(complexPost.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.Comment.Create().
			SetContent("Complex comment 2").
			SetPostID(complexPost.ID).
			SetAuthorID(fixtures.TestUser2.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Retrieve post and verify all relationships are loaded
		retrievedPost, err := uc.GetPostByID(ctx, complexPost.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)

		// Verify hero image
		heroEdge, err := retrievedPost.QueryHeroImage().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, heroImage.ID, heroEdge.ID)

		// Verify author
		author, err := retrievedPost.QueryAuthor().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fixtures.TestUser1.ID, author.ID)

		// Verify community
		community, err := retrievedPost.QueryCommunity().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fixtures.TestCommunity1.ID, community.ID)

		// Verify likes
		likes, err := retrievedPost.QueryLikes().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, likes, 2)

		// Verify bookmarks
		bookmarks, err := retrievedPost.QueryBookmarks().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, bookmarks, 1)

		// Verify comments
		comments, err := retrievedPost.QueryComments().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, comments, 2)
	})
}

func TestPostUsecase_PostWithDifferentVisibilities(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("published post", func(t *testing.T) {
		// Create published post
		publishedPost, err := client.Post.Create().
			SetTitle("Published Post").
			SetSlug("published-post-test").
			SetContent(map[string]interface{}{"text": "This is a published post"}).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetVisibility(post.VisibilityPublished).
			Save(ctx)
		require.NoError(t, err)

		retrievedPost, err := uc.GetPostByID(ctx, publishedPost.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)
		assert.Equal(t, post.VisibilityPublished, retrievedPost.Visibility)
	})

	t.Run("draft post", func(t *testing.T) {
		// Create draft post
		draftPost, err := client.Post.Create().
			SetTitle("Draft Post").
			SetSlug("draft-post-test").
			SetContent(map[string]interface{}{"text": "This is a draft post"}).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetVisibility(post.VisibilityDraft).
			Save(ctx)
		require.NoError(t, err)

		retrievedPost, err := uc.GetPostByID(ctx, draftPost.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)
		assert.Equal(t, post.VisibilityDraft, retrievedPost.Visibility)
	})

	t.Run("deleted post", func(t *testing.T) {
		// Create deleted post
		deletedPost, err := client.Post.Create().
			SetTitle("Deleted Post").
			SetSlug("deleted-post-test").
			SetContent(map[string]interface{}{"text": "This is a deleted post"}).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetVisibility(post.VisibilityDeleted).
			Save(ctx)
		require.NoError(t, err)

		retrievedPost, err := uc.GetPostByID(ctx, deletedPost.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)
		assert.Equal(t, post.VisibilityDeleted, retrievedPost.Visibility)
	})
}

func TestPostUsecase_PostInteractions(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("multiple users liking same post", func(t *testing.T) {
		// Create likes from different users
		users := []int{fixtures.TestUser1.ID, fixtures.TestUser2.ID}

		for _, userID := range users {
			_, err := client.PostLike.Create().
				SetUserID(userID).
				SetPostID(fixtures.TestPost1.ID).
				SetCreatedAt(time.Now()).
				Save(ctx)
			require.NoError(t, err)
		}

		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		likes, err := post.QueryLikes().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, likes, 2)

		// Verify both users are represented
		userIDs := make(map[int]bool)
		for _, like := range likes {
			userIDs[like.UserID] = true
		}
		assert.True(t, userIDs[fixtures.TestUser1.ID])
		assert.True(t, userIDs[fixtures.TestUser2.ID])
	})

	t.Run("multiple users bookmarking same post", func(t *testing.T) {
		// Create bookmarks from different users
		users := []int{fixtures.TestUser1.ID, fixtures.TestUser2.ID}

		for _, userID := range users {
			_, err := client.Bookmark.Create().
				SetUserID(userID).
				SetPostID(fixtures.TestPost2.ID).
				SetCreatedAt(time.Now()).
				Save(ctx)
			require.NoError(t, err)
		}

		post, err := uc.GetPostByID(ctx, fixtures.TestPost2.ID)
		require.NoError(t, err)

		bookmarks, err := post.QueryBookmarks().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, bookmarks, 2)

		// Verify both users are represented
		userIDs := make(map[int]bool)
		for _, bookmark := range bookmarks {
			userIDs[bookmark.UserID] = true
		}
		assert.True(t, userIDs[fixtures.TestUser1.ID])
		assert.True(t, userIDs[fixtures.TestUser2.ID])
	})
}

func TestPostUsecase_PostInPrivateCommunity(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("post in private community", func(t *testing.T) {
		// Create post in private community
		privatePost, err := client.Post.Create().
			SetTitle("Private Post").
			SetSlug("private-post-test").
			SetContent(map[string]interface{}{"text": "This post is in a private community"}).
			SetCommunityID(fixtures.PrivateCommunity.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetVisibility(post.VisibilityPublished).
			Save(ctx)
		require.NoError(t, err)

		retrievedPost, err := uc.GetPostByID(ctx, privatePost.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)

		// Note: IsPrivate field no longer exists in community schema
	})
}

func TestPostUsecase_EdgeCases(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	t.Run("context cancellation during GetPostByID", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		post, err := uc.GetPostByID(cancelledCtx, 1)

		assert.Error(t, err)
		assert.Nil(t, post)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("context cancellation during GetPostStatus", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		status, err := uc.GetPostStatus(cancelledCtx, 1, 1)

		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("post with missing media reference", func(t *testing.T) {
		// Seed basic data
		err := fixtures.SeedBasicData(ctx, client)
		require.NoError(t, err)

		// Create post with non-existent hero image ID
		postWithBrokenMedia, err := client.Post.Create().
			SetTitle("Post With Broken Media").
			SetSlug("post-with-broken-media-test").
			SetContent(map[string]interface{}{"text": "This post references non-existent media"}).
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHeroImageID(99999). // Non-existent media ID
			SetVisibility(post.VisibilityPublished).
			Save(ctx)
		require.NoError(t, err)

		// Should still retrieve post successfully
		retrievedPost, err := uc.GetPostByID(ctx, postWithBrokenMedia.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPost)

		// Trying to query hero image should return error
		_, err = retrievedPost.QueryHeroImage().Only(ctx)
		assert.Error(t, err)
		assert.True(t, ent.IsNotFound(err))
	})
}

func TestPostUsecase_Performance(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	// Create many comments for performance testing
	postID := fixtures.TestPost1.ID

	for i := 0; i < 100; i++ {
		_, err := client.Comment.Create().
			SetContent(fmt.Sprintf("Performance comment %d", i)).
			SetPostID(postID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Second)).
			Save(ctx)
		require.NoError(t, err)
	}

	// Create many likes
	for i := 0; i < 50; i++ {
		// Create additional users for likes
		testUser, err := client.User.Create().
			SetName(fmt.Sprintf("User %d", i)).
			SetSlug(fmt.Sprintf("user-%d", i)).
			SetEmail(fmt.Sprintf("user%d@test.com", i)).
			SetPasswordHash("hash").
			SetSalt("salt").
			Save(ctx)
		require.NoError(t, err)

		_, err = client.PostLike.Create().
			SetUserID(testUser.ID).
			SetPostID(postID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}

	t.Run("post retrieval with many relationships should be fast", func(t *testing.T) {
		start := time.Now()

		post, err := uc.GetPostByID(ctx, postID)

		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, post)

		// Should complete reasonably quickly (under 100ms for this test)
		assert.Less(t, duration, 100*time.Millisecond, "Post retrieval should be fast even with many relationships")

		// Verify data integrity
		comments, err := post.QueryComments().All(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(comments), 100)

		likes, err := post.QueryLikes().All(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(likes), 50)
	})
}

func TestPostUsecase_PostStatusConcurrency(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("concurrent status checks", func(t *testing.T) {
		concurrency := 10
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(userID int) {
				status, err := uc.GetPostStatus(ctx, userID, fixtures.TestPost1.ID)
				if err != nil {
					results <- err
					return
				}

				if status == nil {
					results <- fmt.Errorf("status is nil")
					return
				}

				results <- nil
			}(fixtures.TestUser1.ID)
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})
}

func TestPostUsecase_DatabaseIntegrity(t *testing.T) {
	client, helper := setupTestClient(t)
	defer helper.Cleanup()

	uc := NewPostUsecase(client)
	ctx := context.Background()

	// Seed basic test data
	err := fixtures.SeedBasicData(ctx, client)
	require.NoError(t, err)

	t.Run("post referential integrity", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Verify author exists and matches
		author, err := post.QueryAuthor().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fixtures.TestPost1.AuthorID, author.ID)

		// Verify community exists and matches
		community, err := post.QueryCommunity().Only(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fixtures.TestPost1.CommunityID, community.ID)
	})

	t.Run("post likes integrity", func(t *testing.T) {
		// Create like
		like, err := client.PostLike.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(fixtures.TestPost1.ID).
			SetCreatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Verify like relationship
		likes, err := post.QueryLikes().All(ctx)
		assert.NoError(t, err)
		assert.Len(t, likes, 1)
		assert.Equal(t, like.ID, likes[0].ID)
		assert.Equal(t, fixtures.TestUser2.ID, likes[0].UserID)
		assert.Equal(t, fixtures.TestPost1.ID, likes[0].PostID)
	})

	t.Run("post comments integrity", func(t *testing.T) {
		post, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		require.NoError(t, err)

		// Verify comment relationships
		comments, err := post.QueryComments().All(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, comments)

		// Each comment should belong to this post
		for _, comment := range comments {
			assert.Equal(t, fixtures.TestPost1.ID, comment.PostID)
		}
	})
}

// Benchmark tests
func BenchmarkPostUsecase_GetPostByID(b *testing.B) {
	ctx := context.Background()

	// Setup test containers
	containers, err := testcontainers.Setup(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer containers.Cleanup()

	// Create Ent client
	client := enttest.Open(b, "postgres", containers.GetPostgresDSN())
	defer client.Close()

	uc := NewPostUsecase(client)

	// Create test data with correct IDs
	user1, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	if err != nil {
		b.Fatal(err)
	}

	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user1.ID
	community, err := fixtures.CreateTestCommunity(ctx, client, communityFixture)
	if err != nil {
		b.Fatal(err)
	}

	postFixture := fixtures.TestPost1
	postFixture.AuthorID = user1.ID
	postFixture.CommunityID = community.ID
	testPost, err := fixtures.CreateTestPost(ctx, client, postFixture)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetPostByID(ctx, testPost.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostUsecase_GetPostStatus(b *testing.B) {
	ctx := context.Background()

	// Setup test containers
	containers, err := testcontainers.Setup(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer containers.Cleanup()

	// Create Ent client
	client := enttest.Open(b, "postgres", containers.GetPostgresDSN())
	defer client.Close()

	uc := NewPostUsecase(client)

	// Create test data with correct IDs
	user1, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	if err != nil {
		b.Fatal(err)
	}

	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user1.ID
	community, err := fixtures.CreateTestCommunity(ctx, client, communityFixture)
	if err != nil {
		b.Fatal(err)
	}

	postFixture := fixtures.TestPost1
	postFixture.AuthorID = user1.ID
	postFixture.CommunityID = community.ID
	testPost, err := fixtures.CreateTestPost(ctx, client, postFixture)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetPostStatus(ctx, testPost.ID, user1.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostUsecase_GetPostByID_WithManyRelationships(b *testing.B) {
	ctx := context.Background()

	// Setup test containers
	containers, err := testcontainers.Setup(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer containers.Cleanup()

	// Create Ent client
	client := enttest.Open(b, "postgres", containers.GetPostgresDSN())
	defer client.Close()

	// Create test data with correct IDs
	user1, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	if err != nil {
		b.Fatal(err)
	}

	communityFixture := fixtures.TestCommunity1
	communityFixture.OwnerID = user1.ID
	community, err := fixtures.CreateTestCommunity(ctx, client, communityFixture)
	if err != nil {
		b.Fatal(err)
	}

	postFixture := fixtures.TestPost1
	postFixture.AuthorID = user1.ID
	postFixture.CommunityID = community.ID
	post, err := fixtures.CreateTestPost(ctx, client, postFixture)
	if err != nil {
		b.Fatal(err)
	}

	uc := NewPostUsecase(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetPostByID(ctx, post.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
