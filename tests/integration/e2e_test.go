package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"stormlink/server/usecase/comment"
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/post"
	"stormlink/server/usecase/user"
	"stormlink/services/auth/internal/service"
	mailservice "stormlink/services/mail/internal/service"
	mediaservice "stormlink/services/media/internal/service"
	"stormlink/shared/context"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	containers    *testcontainers.TestContainers
	userUC        user.UserUsecase
	communityUC   community.CommunityUsecase
	postUC        post.PostUsecase
	commentUC     comment.CommentUsecase
	authService   *service.AuthService
	mailService   *mailservice.MailService
	mediaService  *mediaservice.MediaService
	ctx           context.Context
}

func (suite *E2ETestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers (PostgreSQL + Redis)
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create all usecases
	suite.userUC = user.NewUserUsecase(containers.EntClient)
	suite.communityUC = community.NewCommunityUsecase(containers.EntClient)
	suite.postUC = post.NewPostUsecase(containers.EntClient)
	suite.commentUC = comment.NewCommentUsecase(containers.EntClient)

	// Create services
	suite.authService = service.NewAuthService(containers.EntClient, suite.userUC)
	suite.authService.SetRedisClient(containers.RedisClient)
	suite.mailService = mailservice.NewMailService(containers.EntClient)

	// Create mock S3 client for media service
	mockS3 := &MockS3Client{
		uploads: make(map[string][]byte),
		errors:  make(map[string]error),
	}
	suite.mediaService = mediaservice.NewMediaServiceWithClient(mockS3, containers.EntClient)
}

func (suite *E2ETestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *E2ETestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *E2ETestSuite) TestCompleteUserJourney() {
	// Test a complete user journey from registration to creating content

	// Step 1: Create unverified user
	unverifiedUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)
	suite.Assert().False(unverifiedUser.IsVerified)

	// Step 2: Request email verification
	resendReq := &mailpb.ResendVerifyEmailRequest{
		Email: fixtures.UnverifiedUser.Email,
	}
	resendResp, err := suite.mailService.ResendVerifyEmail(suite.ctx, resendReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(resendResp)

	// Get verification token from database
	verifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.HasUserWith(user.EmailEQ(fixtures.UnverifiedUser.Email))).
		All(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Len(verifications, 1)
	verificationToken := verifications[0].Token

	// Step 3: Verify email
	verifyReq := &mailpb.VerifyEmailRequest{Token: verificationToken}
	verifyResp, err := suite.mailService.VerifyEmail(suite.ctx, verifyReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(verifyResp)

	// Verify user is now verified
	verifiedUser, err := suite.userUC.GetUserByID(suite.ctx, unverifiedUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().True(verifiedUser.IsVerified)

	// Step 4: User login
	loginReq := &authpb.LoginRequest{
		Email:    fixtures.UnverifiedUser.Email,
		Password: fixtures.UnverifiedUser.Password,
	}
	loginResp, err := suite.authService.Login(suite.ctx, loginReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(loginResp)
	suite.Assert().NotEmpty(loginResp.AccessToken)
	suite.Assert().NotEmpty(loginResp.RefreshToken)

	// Step 5: Create authenticated context
	authCtx := context.WithUserID(suite.ctx, verifiedUser.ID)

	// Step 6: Create community
	community, err := fixtures.CreateTestCommunity(suite.ctx, suite.containers.EntClient, fixtures.CommunityFixture{
		Name:        "User Journey Community",
		Slug:        "user-journey-community",
		Description: "Community created during user journey test",
		IsPrivate:   false,
		OwnerID:     verifiedUser.ID,
		CreatedAt:   time.Now(),
	})
	suite.Assert().NoError(err)
	suite.Assert().NotNil(community)

	// Step 7: Upload media for post
	mediaContent := []byte("test image content for user journey")
	uploadReq := &mediapb.UploadMediaRequest{
		Filename:    "journey-hero.jpg",
		FileContent: mediaContent,
		Dir:         "heroes",
	}
	uploadResp, err := suite.mediaService.UploadMedia(authCtx, uploadReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(uploadResp)

	// Step 8: Create post with hero image
	post, err := suite.containers.EntClient.Post.Create().
		SetTitle("My First Post").
		SetContent("This is my first post after registration!").
		SetCommunityID(community.ID).
		SetAuthorID(verifiedUser.ID).
		SetHeroImageID(int(uploadResp.Id)).
		Save(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(post)

	// Step 9: Retrieve post with all relationships
	retrievedPost, err := suite.postUC.GetPostByID(suite.ctx, post.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(retrievedPost)

	// Verify all relationships are loaded correctly
	author, err := retrievedPost.QueryAuthor().Only(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Equal(verifiedUser.ID, author.ID)

	postCommunity, err := retrievedPost.QueryCommunity().Only(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Equal(community.ID, postCommunity.ID)

	heroImage, err := retrievedPost.QueryHeroImage().Only(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Equal(int(uploadResp.Id), heroImage.ID)

	// Step 10: Create comment on post
	comment, err := fixtures.CreateTestComment(suite.ctx, suite.containers.EntClient, fixtures.CommentFixture{
		Content:   "Great first post!",
		PostID:    post.ID,
		AuthorID:  verifiedUser.ID,
		CreatedAt: time.Now(),
	})
	suite.Assert().NoError(err)
	suite.Assert().NotNil(comment)

	// Step 11: Test comment retrieval
	comments, err := suite.commentUC.GetCommentsByPostID(suite.ctx, post.ID, nil)
	suite.Assert().NoError(err)
	suite.Assert().Len(comments, 1)
	suite.Assert().Equal(comment.ID, comments[0].ID)

	// Step 12: Test user status and permissions
	userStatus, err := suite.userUC.GetUserStatus(authCtx, verifiedUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(userStatus)
	suite.Assert().True(userStatus.IsOwn)

	permissions, err := suite.userUC.GetPermissionsByCommunities(authCtx, verifiedUser.ID, []int{community.ID})
	suite.Assert().NoError(err)
	suite.Assert().NotEmpty(permissions)
	suite.Assert().Contains(permissions, community.ID)

	// User should be owner of their community
	communityPermissions := permissions[community.ID]
	suite.Assert().NotNil(communityPermissions)
	suite.Assert().True(communityPermissions.CanDeletePosts)
	suite.Assert().True(communityPermissions.CanBanUsers)
	suite.Assert().True(communityPermissions.CanManageRoles)
}

func (suite *E2ETestSuite) TestMultiUserInteractions() {
	// Test interactions between multiple users

	// Create multiple users
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 5)
	suite.Require().NoError(err)
	suite.Require().Len(users, 5)

	owner := users[0]
	moderator := users[1]
	member1 := users[2]
	member2 := users[3]
	lurker := users[4]

	// Create community
	community, err := fixtures.CreateRandomCommunity(suite.ctx, suite.containers.EntClient, owner.ID)
	suite.Require().NoError(err)

	// Create roles
	moderatorRole, err := fixtures.CreateTestRole(suite.ctx, suite.containers.EntClient, fixtures.RoleFixture{
		Name:        "Moderator",
		CommunityID: community.ID,
		Permissions: []string{"delete_posts", "ban_users", "mute_users"},
		CreatedAt:   time.Now(),
	})
	suite.Require().NoError(err)

	// Assign moderator role
	_, err = fixtures.CreateTestCommunityModerator(suite.ctx, suite.containers.EntClient, fixtures.CommunityModeratorFixture{
		UserID:      moderator.ID,
		CommunityID: community.ID,
		RoleID:      moderatorRole.ID,
		CreatedAt:   time.Now(),
	})
	suite.Require().NoError(err)

	// Members follow the community
	for _, user := range []*ent.User{member1, member2} {
		_, err := fixtures.CreateTestCommunityFollow(suite.ctx, suite.containers.EntClient, fixtures.CommunityFollowFixture{
			UserID:      user.ID,
			CommunityID: community.ID,
			CreatedAt:   time.Now(),
		})
		suite.Require().NoError(err)
	}

	// Member1 creates a post
	post, err := fixtures.CreateRandomPost(suite.ctx, suite.containers.EntClient, community.ID, member1.ID)
	suite.Require().NoError(err)

	// Multiple users interact with the post
	interactions := []struct {
		userID int
		action string
	}{
		{member2.ID, "like"},
		{moderator.ID, "like"},
		{lurker.ID, "like"},
		{member2.ID, "bookmark"},
		{moderator.ID, "bookmark"},
	}

	for _, interaction := range interactions {
		switch interaction.action {
		case "like":
			_, err := fixtures.CreateTestPostLike(suite.ctx, suite.containers.EntClient, fixtures.PostLikeFixture{
				UserID:    interaction.userID,
				PostID:    post.ID,
				CreatedAt: time.Now(),
			})
			suite.Assert().NoError(err)
		case "bookmark":
			_, err := fixtures.CreateTestBookmark(suite.ctx, suite.containers.EntClient, fixtures.BookmarkFixture{
				UserID:    interaction.userID,
				PostID:    post.ID,
				CreatedAt: time.Now(),
			})
			suite.Assert().NoError(err)
		}
	}

	// Verify post has all interactions
	retrievedPost, err := suite.postUC.GetPostByID(suite.ctx, post.ID)
	suite.Assert().NoError(err)

	likes, err := retrievedPost.QueryLikes().All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(likes, 3) // member2, moderator, lurker

	bookmarks, err := retrievedPost.QueryBookmarks().All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(bookmarks, 2) // member2, moderator

	// Test post status for different users
	for _, user := range users {
		status, err := suite.postUC.GetPostStatus(suite.ctx, user.ID, post.ID)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)

		if user.ID == member1.ID {
			suite.Assert().True(status.IsOwn)
		} else {
			suite.Assert().False(status.IsOwn)
		}

		// Check if user liked the post
		expectedLiked := false
		for _, interaction := range interactions {
			if interaction.userID == user.ID && interaction.action == "like" {
				expectedLiked = true
				break
			}
		}
		suite.Assert().Equal(expectedLiked, status.IsLiked)

		// Check if user bookmarked the post
		expectedBookmarked := false
		for _, interaction := range interactions {
			if interaction.userID == user.ID && interaction.action == "bookmark" {
				expectedBookmarked = true
				break
			}
		}
		suite.Assert().Equal(expectedBookmarked, status.IsBookmarked)
	}

	// Test community status for different users
	for _, user := range users {
		communityStatus, err := suite.communityUC.GetCommunityStatus(suite.ctx, user.ID, community.ID)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(communityStatus)

		if user.ID == owner.ID {
			suite.Assert().True(communityStatus.IsOwn)
			suite.Assert().Equal(models.CommunityStatusRelationshipOwner, communityStatus.Relationship)
		} else if user.ID == moderator.ID {
			suite.Assert().False(communityStatus.IsOwn)
			suite.Assert().Equal(models.CommunityStatusRelationshipModerator, communityStatus.Relationship)
		} else if user.ID == member1.ID || user.ID == member2.ID {
			suite.Assert().False(communityStatus.IsOwn)
			suite.Assert().True(communityStatus.IsFollowing)
			suite.Assert().Equal(models.CommunityStatusRelationshipFollowing, communityStatus.Relationship)
		} else { // lurker
			suite.Assert().False(communityStatus.IsOwn)
			suite.Assert().False(communityStatus.IsFollowing)
			suite.Assert().Equal(models.CommunityStatusRelationshipNone, communityStatus.Relationship)
		}
	}
}

func (suite *E2ETestSuite) TestContentModerationWorkflow() {
	// Test complete content moderation workflow

	// Setup users
	err := fixtures.SeedExtendedData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	// Create community with moderation setup
	community, err := fixtures.CreateTestCommunity(suite.ctx, suite.containers.EntClient, fixtures.LargeCommunity)
	suite.Require().NoError(err)

	// Create moderator role
	moderatorRole, err := fixtures.CreateTestRole(suite.ctx, suite.containers.EntClient, fixtures.ModeratorRole)
	suite.Require().NoError(err)

	// Assign moderator
	_, err = fixtures.CreateTestCommunityModerator(suite.ctx, suite.containers.EntClient, fixtures.CommunityModeratorFixture{
		UserID:      fixtures.ModeratorUser.ID,
		CommunityID: community.ID,
		RoleID:      moderatorRole.ID,
		CreatedAt:   time.Now(),
	})
	suite.Require().NoError(err)

	// User creates problematic content
	problematicPost, err := suite.containers.EntClient.Post.Create().
		SetTitle("Spam Post").
		SetContent("This is spam content that should be moderated").
		SetCommunityID(community.ID).
		SetAuthorID(fixtures.BannedUser.ID).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Moderator reviews content
	moderatorCtx := context.WithUserID(suite.ctx, fixtures.ModeratorUser.ID)

	// Check moderator permissions
	permissions, err := suite.userUC.GetPermissionsByCommunities(moderatorCtx, fixtures.ModeratorUser.ID, []int{community.ID})
	suite.Assert().NoError(err)
	suite.Assert().Contains(permissions, community.ID)

	modPermissions := permissions[community.ID]
	suite.Assert().NotNil(modPermissions)
	suite.Assert().True(modPermissions.CanDeletePosts)
	suite.Assert().True(modPermissions.CanBanUsers)

	// Moderator bans the problematic user
	_, err = fixtures.CreateTestBan(suite.ctx, suite.containers.EntClient, fixtures.BanFixture{
		UserID:      fixtures.BannedUser.ID,
		CommunityID: community.ID,
		Reason:      "Spam and inappropriate content",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // 7 days
	})
	suite.Assert().NoError(err)

	// Verify banned user status
	bannedUserStatus, err := suite.communityUC.GetCommunityStatus(suite.ctx, fixtures.BannedUser.ID, community.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(bannedUserStatus)
	suite.Assert().True(bannedUserStatus.IsBanned)
	suite.Assert().Equal(models.CommunityStatusRelationshipBanned, bannedUserStatus.Relationship)

	// Moderator can still access the post for review
	moderatorPostStatus, err := suite.postUC.GetPostStatus(moderatorCtx, problematicPost.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(moderatorPostStatus)
	suite.Assert().False(moderatorPostStatus.IsOwn)

	// Regular user should see normal community status
	regularUserStatus, err := suite.communityUC.GetCommunityStatus(suite.ctx, fixtures.TestUser1.ID, community.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(regularUserStatus)
	suite.Assert().False(regularUserStatus.IsBanned)
	suite.Assert().Equal(models.CommunityStatusRelationshipNone, regularUserStatus.Relationship)
}

func (suite *E2ETestSuite) TestContentDiscoveryWorkflow() {
	// Test content discovery across multiple communities and posts

	// Create test ecosystem
	err := fixtures.SeedComplexScenario(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	// Get all communities
	communities, err := suite.containers.EntClient.Community.Query().All(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(communities)

	// Create posts in different communities
	var allPosts []*ent.Post
	for i, community := range communities {
		// Create multiple posts per community
		for j := 0; j < 3; j++ {
			post, err := suite.containers.EntClient.Post.Create().
				SetTitle(fmt.Sprintf("Discovery Post %d-%d", i, j)).
				SetContent(fmt.Sprintf("Content for discovery testing in community %d, post %d", i, j)).
				SetCommunityID(community.ID).
				SetAuthorID(fixtures.AdminUser.ID).
				SetVisibility(post.VisibilityPublished).
				SetCreatedAt(time.Now().Add(-time.Duration(i*3+j) * time.Hour)).
				Save(suite.ctx)
			suite.Require().NoError(err)
			allPosts = append(allPosts, post)
		}
	}

	// Create comments on posts for content discovery
	for i, post := range allPosts[:5] { // Comment on first 5 posts
		for j := 0; j < 2; j++ { // 2 comments per post
			_, err := suite.containers.EntClient.Comment.Create().
				SetContent(fmt.Sprintf("Discovery comment %d-%d", i, j)).
				SetPostID(post.ID).
				SetAuthorID(fixtures.TestUser1.ID).
				SetCreatedAt(time.Now().Add(-time.Duration(i*2+j) * time.Minute)).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}
	}

	// Test comments feed discovery
	feedComments, err := suite.commentUC.GetCommentsFeed(suite.ctx, 20)
	suite.Assert().NoError(err)
	suite.Assert().NotEmpty(feedComments)
	suite.Assert().LessOrEqual(len(feedComments), 20)

	// Verify feed is ordered by creation time DESC (newest first)
	for i := 1; i < len(feedComments); i++ {
		prev := feedComments[i-1]
		curr := feedComments[i]
		suite.Assert().True(prev.CreatedAt.After(curr.CreatedAt) ||
			prev.CreatedAt.Equal(curr.CreatedAt))
	}

	// Test paginated comments feed
	first := 5
	connection, err := suite.commentUC.CommentsFeedConnection(suite.ctx, nil, &first, nil, nil, nil)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(connection)
	suite.Assert().LessOrEqual(len(connection.Edges), 5)

	// Test navigation through pages
	if connection.PageInfo.HasNextPage {
		endCursor := *connection.PageInfo.EndCursor
		nextPage, err := suite.commentUC.CommentsFeedConnection(suite.ctx, nil, &first, &endCursor, nil, nil)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(nextPage)

		// Verify no overlap between pages
		if len(nextPage.Edges) > 0 {
			lastFromFirst := connection.Edges[len(connection.Edges)-1].Node
			firstFromNext := nextPage.Edges[0].Node
			suite.Assert().True(lastFromFirst.CreatedAt.After(firstFromNext.CreatedAt) ||
				(lastFromFirst.CreatedAt.Equal(firstFromNext.CreatedAt) && lastFromFirst.ID > firstFromNext.ID))
		}
	}
}

func (suite *E2ETestSuite) TestAuthenticationFlow() {
	// Test complete authentication flow with token refresh

	// Create verified user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Step 1: Login
	loginReq := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: fixtures.TestUser1.Password,
	}
	loginResp, err := suite.authService.Login(suite.ctx, loginReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(loginResp)
	suite.Assert().NotEmpty(loginResp.AccessToken)
	suite.Assert().NotEmpty(loginResp.RefreshToken)

	// Step 2: Validate access token
	validateReq := &authpb.ValidateTokenRequest{
		Token: loginResp.AccessToken,
	}
	validateResp, err := suite.authService.ValidateToken(suite.ctx, validateReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(validateResp)
	suite.Assert().True(validateResp.IsValid)
	suite.Assert().Equal(int32(testUser.ID), validateResp.UserId)

	// Step 3: Use refresh token to get new tokens
	refreshReq := &authpb.RefreshTokenRequest{
		RefreshToken: loginResp.RefreshToken,
	}
	refreshResp, err := suite.authService.RefreshToken(suite.ctx, refreshReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(refreshResp)
	suite.Assert().NotEmpty(refreshResp.AccessToken)
	suite.Assert().NotEmpty(refreshResp.RefreshToken)

	// Verify old and new tokens are different (token rotation)
	suite.Assert().NotEqual(loginResp.AccessToken, refreshResp.AccessToken)
	suite.Assert().NotEqual(loginResp.RefreshToken, refreshResp.RefreshToken)

	// Step 4: Validate new access token
	newValidateReq := &authpb.ValidateTokenRequest{
		Token: refreshResp.AccessToken,
	}
	newValidateResp, err := suite.authService.ValidateToken(suite.ctx, newValidateReq)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(newValidateResp)
	suite.Assert().True(newValidateResp.IsValid)
	suite.Assert().Equal(int32(testUser.ID), newValidateResp.UserId)

	// Step 5: Logout
	authCtx := context.WithUserID(suite.ctx, testUser.ID)
	logoutResp, err := suite.authService.Logout(authCtx, &emptypb.Empty{})
	suite.Assert().NoError(err)
	suite.Assert().NotNil(logoutResp)
	suite.Assert().True(logoutResp.Success)
}

func (suite *E2ETestSuite) TestCommunityLifecycle() {
	// Test complete community lifecycle

	// Create community owner
	owner, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.AdminUser)
	suite.Require().NoError(err)

	// Upload community logo
	logoContent := []byte("community logo content")
	uploadReq := &mediapb.UploadMediaRequest{
		Filename:    "community-logo.png",
		FileContent: logoContent,
		Dir:         "logos",
	}
	uploadResp, err := suite.mediaService.UploadMedia(suite.ctx, uploadReq)
	suite.Assert().NoError(err)

	// Create community with logo
	community, err := suite.containers.EntClient.Community.Create().
		SetName("E2E Test Community").
		SetSlug("e2e-test-community").
		SetDescription("Community for end-to-end testing").
		SetIsPrivate(false).
		SetOwnerID(owner.ID).
		SetLogoID(int(uploadResp.Id)).
		Save(suite.ctx)
	suite.Assert().NoError(err)

	// Create community roles
	roles := []fixtures.RoleFixture{
		{
			Name:        "Admin",
			CommunityID: community.ID,
			Permissions: []string{"all", "manage_community"},
			CreatedAt:   time.Now(),
		},
		{
			Name:        "Moderator",
			CommunityID: community.ID,
			Permissions: []string{"delete_posts", "ban_users", "mute_users"},
			CreatedAt:   time.Now(),
		},
		{
			Name:        "Member",
			CommunityID: community.ID,
			Permissions: []string{"create_posts", "comment", "like"},
			CreatedAt:   time.Now(),
		},
	}

	var createdRoles []*ent.Role
	for _, roleFixture := range roles {
		role, err := fixtures.CreateTestRole(suite.ctx, suite.containers.EntClient, roleFixture)
		suite.Assert().NoError(err)
		createdRoles = append(createdRoles, role)
	}

	// Create members
	members, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 10)
	suite.Require().NoError(err)

	// Assign roles to members
	for i, member := range members {
		roleIndex := i % len(createdRoles)
		_, err := fixtures.CreateTestCommunityModerator(suite.ctx, suite.containers.EntClient, fixtures.CommunityModeratorFixture{
			UserID:      member.ID,
			CommunityID: community.ID,
			RoleID:      createdRoles[roleIndex].ID,
			CreatedAt:   time.Now(),
		})
		suite.Assert().NoError(err)
	}

	// Members follow community
	for _, member := range members {
		_, err := fixtures.CreateTestCommunityFollow(suite.ctx, suite.containers.EntClient, fixtures.CommunityFollowFixture{
			UserID:      member.ID,
			CommunityID: community.ID,
			CreatedAt:   time.Now(),
		})
		suite.Assert().NoError(err)
	}

	// Create posts by different members
	for i, member := range members[:5] { // First 5 members create posts
		post, err := suite.containers.EntClient.Post.Create().
			SetTitle(fmt.Sprintf("Community Post %d", i)).
			SetContent(fmt.Sprintf("Post content from member %d", i)).
			SetCommunityID(community.ID).
			SetAuthorID(member.ID).
			SetVisibility(post.VisibilityPublished).
			Save(suite.ctx)
		suite.Assert().NoError(err)

		// Other members comment on posts
		for j, commenter := range members[5:] { // Other 5 members comment
			_, err := suite.containers.EntClient.Comment.Create().
				SetContent(fmt.Sprintf("Comment from member %d on post %d", j+5, i)).
				SetPostID(post.ID).
				SetAuthorID(commenter.ID).
				Save(suite.ctx)
			suite.Assert().NoError(err)
		}
	}

	// Test community retrieval with all relationships
	retrievedCommunity, err := suite.communityUC.GetCommunityByID(suite.ctx, community.ID)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(retrievedCommunity)

	// Verify logo
	logo, err := retrievedCommunity.QueryLogo().Only(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Equal(int(uploadResp.Id), logo.ID)

	// Verify roles
	communityRoles, err := retrievedC
