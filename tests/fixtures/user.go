package fixtures

import (
	"context"
	"time"

	"stormlink/server/ent"
	"stormlink/shared/jwt"

	"github.com/google/uuid"
)

// UserFixture represents a test user with all associated data
type UserFixture struct {
	ID           int
	Name         string
	Slug         string
	Email        string
	Password     string
	Salt         string
	PasswordHash string
	IsVerified   bool
	CreatedAt    time.Time
}

// CommunityFixture represents a test community
type CommunityFixture struct {
	ID          int
	Name        string
	Slug        string
	Description string
	IsPrivate   bool
	OwnerID     int
	CreatedAt   time.Time
}

// PostFixture represents a test post
type PostFixture struct {
	ID          int
	Title       string
	Content     string
	CommunityID int
	AuthorID    int
	CreatedAt   time.Time
}

// CommentFixture represents a test comment
type CommentFixture struct {
	ID        int
	Content   string
	PostID    int
	AuthorID  int
	ParentID  *int
	CreatedAt time.Time
}

// Default test users
var (
	TestUser1 = UserFixture{
		ID:         1,
		Name:       "Test User 1",
		Slug:       "test-user-1",
		Email:      "test1@example.com",
		Password:   "password123",
		Salt:       "test-salt-1",
		IsVerified: true,
		CreatedAt:  time.Now().Add(-24 * time.Hour),
	}

	TestUser2 = UserFixture{
		ID:         2,
		Name:       "Test User 2",
		Slug:       "test-user-2",
		Email:      "test2@example.com",
		Password:   "password456",
		Salt:       "test-salt-2",
		IsVerified: true,
		CreatedAt:  time.Now().Add(-12 * time.Hour),
	}

	UnverifiedUser = UserFixture{
		ID:         3,
		Name:       "Unverified User",
		Slug:       "unverified-user",
		Email:      "unverified@example.com",
		Password:   "password789",
		Salt:       "test-salt-3",
		IsVerified: false,
		CreatedAt:  time.Now().Add(-6 * time.Hour),
	}

	// Test communities
	TestCommunity1 = CommunityFixture{
		ID:          1,
		Name:        "Test Community 1",
		Slug:        "test-community-1",
		Description: "A test community for testing purposes",
		IsPrivate:   false,
		OwnerID:     1,
		CreatedAt:   time.Now().Add(-18 * time.Hour),
	}

	PrivateCommunity = CommunityFixture{
		ID:          2,
		Name:        "Private Community",
		Slug:        "private-community",
		Description: "A private test community",
		IsPrivate:   true,
		OwnerID:     2,
		CreatedAt:   time.Now().Add(-10 * time.Hour),
	}

	// Test posts
	TestPost1 = PostFixture{
		ID:          1,
		Title:       "Test Post 1",
		Content:     "This is the content of test post 1",
		CommunityID: 1,
		AuthorID:    1,
		CreatedAt:   time.Now().Add(-8 * time.Hour),
	}

	TestPost2 = PostFixture{
		ID:          2,
		Title:       "Test Post 2",
		Content:     "This is the content of test post 2",
		CommunityID: 1,
		AuthorID:    2,
		CreatedAt:   time.Now().Add(-4 * time.Hour),
	}

	// Test comments
	TestComment1 = CommentFixture{
		ID:        1,
		Content:   "This is a test comment",
		PostID:    1,
		AuthorID:  2,
		ParentID:  nil,
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	TestReply1 = CommentFixture{
		ID:        2,
		Content:   "This is a reply to the test comment",
		PostID:    1,
		AuthorID:  1,
		ParentID:  &TestComment1.ID,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
)

// CreateTestUser creates a user in the database using the fixture data
func CreateTestUser(ctx context.Context, client *ent.Client, fixture UserFixture) (*ent.User, error) {
	// Generate password hash if not provided
	passwordHash := fixture.PasswordHash
	if passwordHash == "" {
		var err error
		passwordHash, err = jwt.HashPassword(fixture.Password, fixture.Salt)
		if err != nil {
			return nil, err
		}
	}

	return client.User.Create().
		SetName(fixture.Name).
		SetSlug(fixture.Slug).
		SetEmail(fixture.Email).
		SetPasswordHash(passwordHash).
		SetSalt(fixture.Salt).
		SetIsVerified(fixture.IsVerified).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestCommunity creates a community in the database using the fixture data
func CreateTestCommunity(ctx context.Context, client *ent.Client, fixture CommunityFixture) (*ent.Community, error) {
	return client.Community.Create().
		SetTitle(fixture.Name).
		SetSlug(fixture.Slug).
		SetDescription(fixture.Description).
		SetOwnerID(fixture.OwnerID).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestPost creates a post in the database using the fixture data
func CreateTestPost(ctx context.Context, client *ent.Client, fixture PostFixture) (*ent.Post, error) {
	// Convert content string to JSON format expected by the entity
	contentMap := map[string]interface{}{
		"text": fixture.Content,
	}

	return client.Post.Create().
		SetTitle(fixture.Title).
		SetSlug(fixture.Title + "-" + RandomString()).
		SetContent(contentMap).
		SetCommunityID(fixture.CommunityID).
		SetAuthorID(fixture.AuthorID).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt).
		Save(ctx)
}

// CreateTestComment creates a comment in the database using the fixture data
func CreateTestComment(ctx context.Context, client *ent.Client, fixture CommentFixture) (*ent.Comment, error) {
	// Get the community ID from the post
	post, err := client.Post.Get(ctx, fixture.PostID)
	if err != nil {
		return nil, err
	}

	creator := client.Comment.Create().
		SetContent(fixture.Content).
		SetPostID(fixture.PostID).
		SetCommunityID(post.CommunityID).
		SetAuthorID(fixture.AuthorID).
		SetCreatedAt(fixture.CreatedAt).
		SetUpdatedAt(fixture.CreatedAt)

	if fixture.ParentID != nil {
		creator = creator.SetParentCommentID(*fixture.ParentID)
	}

	return creator.Save(ctx)
}

// SeedBasicData creates a basic set of test data (users, communities, posts, comments)
func SeedBasicData(ctx context.Context, client *ent.Client) error {
	// Create users
	_, err := CreateTestUser(ctx, client, TestUser1)
	if err != nil {
		return err
	}

	_, err = CreateTestUser(ctx, client, TestUser2)
	if err != nil {
		return err
	}

	_, err = CreateTestUser(ctx, client, UnverifiedUser)
	if err != nil {
		return err
	}

	// Create communities
	_, err = CreateTestCommunity(ctx, client, TestCommunity1)
	if err != nil {
		return err
	}

	_, err = CreateTestCommunity(ctx, client, PrivateCommunity)
	if err != nil {
		return err
	}

	// Create posts
	_, err = CreateTestPost(ctx, client, TestPost1)
	if err != nil {
		return err
	}

	_, err = CreateTestPost(ctx, client, TestPost2)
	if err != nil {
		return err
	}

	// Create comments
	_, err = CreateTestComment(ctx, client, TestComment1)
	if err != nil {
		return err
	}

	_, err = CreateTestComment(ctx, client, TestReply1)
	if err != nil {
		return err
	}

	return nil
}

// GenerateTestJWT creates a valid JWT token for testing
func GenerateTestJWT(userID int) (string, error) {
	return jwt.GenerateAccessToken(userID)
}

// GenerateTestRefreshToken creates a valid refresh token for testing
func GenerateTestRefreshToken(userID int) (string, error) {
	return jwt.GenerateRefreshToken(userID)
}

// RandomString generates a random string for testing
func RandomString() string {
	return uuid.New().String()[:8]
}

// RandomEmail generates a random email for testing
func RandomEmail() string {
	return RandomString() + "@test.com"
}

// RandomSlug generates a random slug for testing
func RandomSlug() string {
	return "test-" + RandomString()
}
