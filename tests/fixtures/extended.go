package fixtures

import (
	"context"
	"time"

	"stormlink/server/ent"
)

// Extended fixtures for more complex test scenarios

// MediaFixture represents test media data
type MediaFixture struct {
	ID        int
	Filename  string
	URL       string
	Size      int64
	CreatedAt time.Time
}

// PostLikeFixture represents a post like
type PostLikeFixture struct {
	PostID    int
	UserID    int
	CreatedAt time.Time
}

// BookmarkFixture represents a post bookmark
type BookmarkFixture struct {
	PostID    int
	UserID    int
	CreatedAt time.Time
}

// CommunityFollowFixture represents community membership
type CommunityFollowFixture struct {
	CommunityID int
	UserID      int
	CreatedAt   time.Time
}

// EmailVerificationFixture represents email verification data
type EmailVerificationFixture struct {
	UserID    int
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Additional fixture for future use - RefreshToken entity doesn't exist yet
// type RefreshTokenFixture struct {
// 	UserID    int
// 	Token     string
// 	ExpiresAt time.Time
// 	CreatedAt time.Time
// }

// Default extended fixtures
var (
	TestMedia1 = MediaFixture{
		ID:        1,
		Filename:  "test-image.jpg",
		URL:       "https://example.com/test-image.jpg",
		Size:      1024,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	TestPostLike1 = PostLikeFixture{
		PostID:    1,
		UserID:    2,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}

	TestBookmark1 = BookmarkFixture{
		PostID:    1,
		UserID:    1,
		CreatedAt: time.Now().Add(-15 * time.Minute),
	}

	TestCommunityFollow1 = CommunityFollowFixture{
		CommunityID: 1,
		UserID:      2,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
	}
)

// CreateTestMedia creates a media record in the database
func CreateTestMedia(ctx context.Context, client *ent.Client, fixture MediaFixture) (*ent.Media, error) {
	return client.Media.Create().
		SetFilename(fixture.Filename).
		SetURL(fixture.URL).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestPostLike creates a post like in the database
func CreateTestPostLike(ctx context.Context, client *ent.Client, fixture PostLikeFixture) (*ent.PostLike, error) {
	return client.PostLike.Create().
		SetPostID(fixture.PostID).
		SetUserID(fixture.UserID).
		SetCreatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestBookmark creates a post bookmark in the database
func CreateTestBookmark(ctx context.Context, client *ent.Client, fixture BookmarkFixture) (*ent.Bookmark, error) {
	return client.Bookmark.Create().
		SetPostID(fixture.PostID).
		SetUserID(fixture.UserID).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestCommunityFollow creates a community follow in the database
func CreateTestCommunityFollow(ctx context.Context, client *ent.Client, fixture CommunityFollowFixture) (*ent.CommunityFollow, error) {
	return client.CommunityFollow.Create().
		SetCommunityID(fixture.CommunityID).
		SetUserID(fixture.UserID).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestEmailVerification creates an email verification in the database
func CreateTestEmailVerification(ctx context.Context, client *ent.Client, fixture EmailVerificationFixture) (*ent.EmailVerification, error) {
	return client.EmailVerification.Create().
		SetUserID(fixture.UserID).
		SetToken(fixture.Token).
		SetExpiresAt(fixture.ExpiresAt).
		SetCreatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestRefreshToken creates a refresh token in the database
// TODO: Implement when RefreshToken entity is added to the schema
// func CreateTestRefreshToken(ctx context.Context, client *ent.Client, fixture RefreshTokenFixture) (*ent.RefreshToken, error) {
// 	return client.RefreshToken.Create().
// 		SetUserID(fixture.UserID).
// 		SetToken(fixture.Token).
// 		SetExpiresAt(fixture.ExpiresAt).
// 		SetCreatedAt(fixture.CreatedAt).
// 		Save(ctx)
// }

// SeedExtendedData creates extended test data including likes, bookmarks, and memberships
func SeedExtendedData(ctx context.Context, client *ent.Client) error {
	// First seed basic data
	if err := SeedBasicData(ctx, client); err != nil {
		return err
	}

	// Create media
	_, err := CreateTestMedia(ctx, client, TestMedia1)
	if err != nil {
		return err
	}

	// Create post like
	_, err = CreateTestPostLike(ctx, client, TestPostLike1)
	if err != nil {
		return err
	}

	// Create post bookmark
	_, err = CreateTestBookmark(ctx, client, TestBookmark1)
	if err != nil {
		return err
	}

	// Create community follow
	_, err = CreateTestCommunityFollow(ctx, client, TestCommunityFollow1)
	if err != nil {
		return err
	}

	return nil
}

// CleanupTestData removes all test data from the database
func CleanupTestData(ctx context.Context, client *ent.Client) error {
	// Delete in reverse dependency order
	client.EmailVerification.Delete().ExecX(ctx)
	client.Bookmark.Delete().ExecX(ctx)
	client.PostLike.Delete().ExecX(ctx)
	client.CommunityFollow.Delete().ExecX(ctx)
	client.Comment.Delete().ExecX(ctx)
	client.Post.Delete().ExecX(ctx)
	client.Community.Delete().ExecX(ctx)
	client.Media.Delete().ExecX(ctx)
	client.User.Delete().ExecX(ctx)

	return nil
}

// GenerateTestData creates a large dataset for performance testing
func GenerateTestData(ctx context.Context, client *ent.Client, userCount, communityCount, postCount int) error {
	// Create users
	for i := 1; i <= userCount; i++ {
		userFixture := UserFixture{
			Name:       RandomString(),
			Slug:       RandomSlug(),
			Email:      RandomEmail(),
			Password:   "password123",
			Salt:       RandomString(),
			IsVerified: i%2 == 0, // Half verified
			CreatedAt:  time.Now().Add(-time.Duration(i) * time.Hour),
		}

		_, err := CreateTestUser(ctx, client, userFixture)
		if err != nil {
			return err
		}
	}

	// Create communities
	users, err := client.User.Query().All(ctx)
	if err != nil {
		return err
	}

	for i := 1; i <= communityCount; i++ {
		owner := users[i%len(users)]
		communityFixture := CommunityFixture{
			Name:        "Community " + RandomString(),
			Slug:        RandomSlug(),
			Description: "Generated test community",
			IsPrivate:   false, // Not used in schema
			OwnerID:     owner.ID,
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Hour),
		}

		_, err := CreateTestCommunity(ctx, client, communityFixture)
		if err != nil {
			return err
		}
	}

	// Create posts
	communities, err := client.Community.Query().All(ctx)
	if err != nil {
		return err
	}

	for i := 1; i <= postCount; i++ {
		author := users[i%len(users)]
		community := communities[i%len(communities)]

		postFixture := PostFixture{
			Title:       "Post " + RandomString(),
			Content:     "Generated test content " + RandomString(),
			CommunityID: community.ID,
			AuthorID:    author.ID,
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Minute),
		}

		_, err := CreateTestPost(ctx, client, postFixture)
		if err != nil {
			return err
		}
	}

	return nil
}
