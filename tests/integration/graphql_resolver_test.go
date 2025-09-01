package integration

import (
	"context"
	"testing"
	"time"

	"stormlink/server/graphql"
	"stormlink/server/graphql/models"
	"stormlink/server/usecase/comment"
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/post"
	"stormlink/server/usecase/user"
	"stormlink/shared/context"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GraphQLResolverTestSuite struct {
	suite.Suite
	containers *testcontainers.TestContainers
	resolver   *graphql.Resolver
	client     *client.Client
	ctx        context.Context
}

func (suite *GraphQLResolverTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create usecases
	userUC := user.NewUserUsecase(containers.EntClient)
	communityUC := community.NewCommunityUsecase(containers.EntClient)
	postUC := post.NewPostUsecase(containers.EntClient)
	commentUC := comment.NewCommentUsecase(containers.EntClient)

	// Create resolver
	suite.resolver = &graphql.Resolver{
		Client:      containers.EntClient,
		UserUC:      userUC,
		CommunityUC: communityUC,
		PostUC:      postUC,
		CommentUC:   commentUC,
		// Note: Add other dependencies as needed (AuthClient, UserClient, etc.)
	}

	// Create GraphQL client for testing
	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{Resolvers: suite.resolver}))
	suite.client = client.New(srv)
}

func (suite *GraphQLResolverTestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *GraphQLResolverTestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *GraphQLResolverTestSuite) TestUserQueries() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("user by ID query", func() {
		var resp struct {
			User *models.User `json:"user"`
		}

		query := `
			query($id: Int!) {
				user(id: $id) {
					id
					name
					slug
					email
					isVerified
					createdAt
				}
			}
		`

		err := suite.client.Post(query, &resp, client.Var("id", fixtures.TestUser1.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.User)
		suite.Assert().Equal(fixtures.TestUser1.ID, int(resp.User.ID))
		suite.Assert().Equal(fixtures.TestUser1.Name, resp.User.Name)
		suite.Assert().Equal(fixtures.TestUser1.Slug, resp.User.Slug)
		suite.Assert().Equal(fixtures.TestUser1.Email, resp.User.Email)
		suite.Assert().True(resp.User.IsVerified)
	})

	suite.Run("user with avatar query", func() {
		// Create user with avatar
		avatar, err := fixtures.CreateTestMedia(suite.ctx, suite.containers.EntClient, "avatar.jpg", "https://example.com/avatar.jpg")
		suite.Require().NoError(err)

		userWithAvatar, err := suite.containers.EntClient.User.Create().
			SetName("User With Avatar").
			SetSlug("user-with-avatar").
			SetEmail("avatar@test.com").
			SetPasswordHash("hash").
			SetSalt("salt").
			SetAvatarID(avatar.ID).
			Save(suite.ctx)
		suite.Require().NoError(err)

		var resp struct {
			User *models.User `json:"user"`
		}

		query := `
			query($id: Int!) {
				user(id: $id) {
					id
					name
					avatar {
						id
						filename
						url
					}
				}
			}
		`

		err = suite.client.Post(query, &resp, client.Var("id", userWithAvatar.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.User)
		suite.Assert().NotNil(resp.User.Avatar)
		suite.Assert().Equal(avatar.ID, int(resp.User.Avatar.ID))
		suite.Assert().Equal("avatar.jpg", resp.User.Avatar.Filename)
	})

	suite.Run("non-existing user query", func() {
		var resp struct {
			User *models.User `json:"user"`
		}

		query := `
			query($id: Int!) {
				user(id: $id) {
					id
					name
				}
			}
		`

		err := suite.client.Post(query, &resp, client.Var("id", 99999))

		// Should handle gracefully (return null or error depending on schema)
		suite.Assert().Error(err)
	})

	suite.Run("user status query", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser1.ID)

		var resp struct {
			UserStatus *models.UserStatus `json:"userStatus"`
		}

		query := `
			query($targetUserId: Int!) {
				userStatus(targetUserId: $targetUserId) {
					isOwn
					isFollowing
					isBlocked
					relationship
				}
			}
		`

		// Mock client with context - this would need proper implementation
		// For now, just test the direct resolver method
		status, err := suite.resolver.Query().UserStatus(authCtx, fixtures.TestUser2.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)
		suite.Assert().False(status.IsOwn)
		suite.Assert().False(status.IsFollowing)
		suite.Assert().Equal(models.UserStatusRelationshipNone, status.Relationship)
	})
}

func (suite *GraphQLResolverTestSuite) TestCommunityQueries() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("community by ID query", func() {
		var resp struct {
			Community *models.Community `json:"community"`
		}

		query := `
			query($id: Int!) {
				community(id: $id) {
					id
					name
					slug
					description
					isPrivate
					ownerId
					createdAt
				}
			}
		`

		err := suite.client.Post(query, &resp, client.Var("id", fixtures.TestCommunity1.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.Community)
		suite.Assert().Equal(fixtures.TestCommunity1.ID, int(resp.Community.ID))
		suite.Assert().Equal(fixtures.TestCommunity1.Name, resp.Community.Name)
		suite.Assert().Equal(fixtures.TestCommunity1.Slug, resp.Community.Slug)
		suite.Assert().Equal(fixtures.TestCommunity1.IsPrivate, resp.Community.IsPrivate)
	})

	suite.Run("community with logo query", func() {
		// Create community with logo
		logo, err := fixtures.CreateTestMedia(suite.ctx, suite.containers.EntClient, "logo.png", "https://example.com/logo.png")
		suite.Require().NoError(err)

		communityWithLogo, err := suite.containers.EntClient.Community.Create().
			SetName("Community With Logo").
			SetSlug("community-with-logo").
			SetDescription("Test community").
			SetIsPrivate(false).
			SetOwnerID(fixtures.TestUser1.ID).
			SetLogoID(logo.ID).
			Save(suite.ctx)
		suite.Require().NoError(err)

		var resp struct {
			Community *models.Community `json:"community"`
		}

		query := `
			query($id: Int!) {
				community(id: $id) {
					id
					name
					logo {
						id
						filename
						url
					}
				}
			}
		`

		err = suite.client.Post(query, &resp, client.Var("id", communityWithLogo.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.Community)
		suite.Assert().NotNil(resp.Community.Logo)
		suite.Assert().Equal(logo.ID, int(resp.Community.Logo.ID))
		suite.Assert().Equal("logo.png", resp.Community.Logo.Filename)
	})

	suite.Run("community status query", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser2.ID)

		// Test direct resolver method
		status, err := suite.resolver.Query().CommunityStatus(authCtx, fixtures.TestCommunity1.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)
		suite.Assert().False(status.IsOwn)
		suite.Assert().False(status.IsFollowing)
		suite.Assert().Equal(models.CommunityStatusRelationshipNone, status.Relationship)
	})
}

func (suite *GraphQLResolverTestSuite) TestPostQueries() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("post by ID query", func() {
		var resp struct {
			Post *models.Post `json:"post"`
		}

		query := `
			query($id: Int!) {
				post(id: $id) {
					id
					title
					content
					communityId
					authorId
					createdAt
				}
			}
		`

		err := suite.client.Post(query, &resp, client.Var("id", fixtures.TestPost1.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.Post)
		suite.Assert().Equal(fixtures.TestPost1.ID, int(resp.Post.ID))
		suite.Assert().Equal(fixtures.TestPost1.Title, resp.Post.Title)
		suite.Assert().Equal(fixtures.TestPost1.Content, resp.Post.Content)
	})

	suite.Run("post with hero image query", func() {
		// Create post with hero image
		heroImage, err := fixtures.CreateTestMedia(suite.ctx, suite.containers.EntClient, "hero.jpg", "https://example.com/hero.jpg")
		suite.Require().NoError(err)

		postWithHero, err := suite.containers.EntClient.Post.Create().
			SetTitle("Post With Hero").
			SetContent("Content").
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetAuthorID(fixtures.TestUser1.ID).
			SetHeroImageID(heroImage.ID).
			Save(suite.ctx)
		suite.Require().NoError(err)

		var resp struct {
			Post *models.Post `json:"post"`
		}

		query := `
			query($id: Int!) {
				post(id: $id) {
					id
					title
					heroImage {
						id
						filename
						url
					}
				}
			}
		`

		err = suite.client.Post(query, &resp, client.Var("id", postWithHero.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.Post)
		suite.Assert().NotNil(resp.Post.HeroImage)
		suite.Assert().Equal(heroImage.ID, int(resp.Post.HeroImage.ID))
	})

	suite.Run("post with author and community query", func() {
		var resp struct {
			Post *models.Post `json:"post"`
		}

		query := `
			query($id: Int!) {
				post(id: $id) {
					id
					title
					author {
						id
						name
						slug
					}
					community {
						id
						name
						slug
					}
				}
			}
		`

		err := suite.client.Post(query, &resp, client.Var("id", fixtures.TestPost1.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.Post)
		suite.Assert().NotNil(resp.Post.Author)
		suite.Assert().NotNil(resp.Post.Community)
		suite.Assert().Equal(fixtures.TestUser1.Name, resp.Post.Author.Name)
		suite.Assert().Equal(fixtures.TestCommunity1.Name, resp.Post.Community.Name)
	})

	suite.Run("post status query", func() {
		// Create like for testing
		_, err := suite.containers.EntClient.PostLike.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetPostID(fixtures.TestPost1.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser2.ID)

		// Test direct resolver method
		status, err := suite.resolver.Query().PostStatus(authCtx, fixtures.TestPost1.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)
		suite.Assert().False(status.IsOwn)
		suite.Assert().True(status.IsLiked)
		suite.Assert().False(status.IsBookmarked)
	})
}

func (suite *GraphQLResolverTestSuite) TestCommentQueries() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("comments by post connection query", func() {
		// Create additional comments for pagination testing
		baseTime := time.Now().Add(-1 * time.Hour)
		for i := 0; i < 5; i++ {
			commentFixture := fixtures.CommentFixture{
				ID:        1000 + i,
				Content:   fmt.Sprintf("Pagination comment %d", i),
				PostID:    fixtures.TestPost1.ID,
				AuthorID:  fixtures.TestUser1.ID,
				CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
			}
			_, err := fixtures.CreateTestComment(suite.ctx, suite.containers.EntClient, commentFixture)
			suite.Require().NoError(err)
		}

		// Test direct resolver method for pagination
		first := 3
		connection, err := suite.resolver.Query().CommentsByPost(suite.ctx, fixtures.TestPost1.ID, &first, nil, nil, nil)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(connection)
		suite.Assert().LessOrEqual(len(connection.Edges), 3)
		suite.Assert().NotNil(connection.PageInfo)

		// Verify ordering
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			suite.Assert().True(prev.CreatedAt.Before(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID < curr.ID))
		}
	})

	suite.Run("comments feed connection query", func() {
		// Update posts to be published
		_, err := suite.containers.EntClient.Post.Update().
			SetVisibility(post.VisibilityPublished).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Test direct resolver method
		first := 5
		connection, err := suite.resolver.Query().CommentsFeed(suite.ctx, &first, nil, nil, nil)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(connection)
		suite.Assert().LessOrEqual(len(connection.Edges), 5)

		// Should be ordered DESC (newest first)
		for i := 1; i < len(connection.Edges); i++ {
			prev := connection.Edges[i-1].Node
			curr := connection.Edges[i].Node
			suite.Assert().True(prev.CreatedAt.After(curr.CreatedAt) ||
				(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID > curr.ID))
		}
	})

	suite.Run("comments window query", func() {
		// Get existing comment as anchor
		comments, err := suite.containers.EntClient.Comment.Query().All(suite.ctx)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(comments)

		anchorComment := comments[0]

		// Test direct resolver method
		connection, err := suite.resolver.Query().CommentsWindow(suite.ctx, anchorComment.PostID, anchorComment.ID, 2, 2)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(connection)

		// Anchor should be included
		anchorFound := false
		for _, edge := range connection.Edges {
			if edge.Node.ID == anchorComment.ID {
				anchorFound = true
				break
			}
		}
		suite.Assert().True(anchorFound, "Anchor comment should be included in window")
	})

	suite.Run("comment status query", func() {
		// Create comment like
		comments, err := suite.containers.EntClient.Comment.Query().All(suite.ctx)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(comments)

		targetComment := comments[0]

		_, err = suite.containers.EntClient.CommentLike.Create().
			SetUserID(fixtures.TestUser2.ID).
			SetCommentID(targetComment.ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser2.ID)

		// Test direct resolver method
		status, err := suite.resolver.Query().CommentStatus(authCtx, targetComment.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(status)
		suite.Assert().False(status.IsOwn)
		suite.Assert().True(status.IsLiked)
	})
}

func (suite *GraphQLResolverTestSuite) TestMutations() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("like post mutation", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser2.ID)

		// Test direct resolver method
		result, err := suite.resolver.Mutation().LikePost(authCtx, fixtures.TestPost1.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(result)
		suite.Assert().True(result.Success)

		// Verify like was created in database
		likes, err := suite.containers.EntClient.PostLike.Query().
			Where(post.like.UserIDEQ(fixtures.TestUser2.ID)).
			Where(post.like.PostIDEQ(fixtures.TestPost1.ID)).
			All(suite.ctx)
		suite.Assert().NoError(err)
		suite.Assert().Len(likes, 1)
	})

	suite.Run("bookmark post mutation", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser2.ID)

		// Test direct resolver method
		result, err := suite.resolver.Mutation().BookmarkPost(authCtx, fixtures.TestPost1.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(result)
		suite.Assert().True(result.Success)

		// Verify bookmark was created in database
		bookmarks, err := suite.containers.EntClient.Bookmark.Query().
			Where(bookmark.UserIDEQ(fixtures.TestUser2.ID)).
			Where(bookmark.PostIDEQ(fixtures.TestPost1.ID)).
			All(suite.ctx)
		suite.Assert().NoError(err)
		suite.Assert().Len(bookmarks, 1)
	})

	suite.Run("follow user mutation", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser1.ID)

		// Test direct resolver method
		result, err := suite.resolver.Mutation().FollowUser(authCtx, fixtures.TestUser2.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(result)
		suite.Assert().True(result.Success)

		// Verify follow was created in database
		follows, err := suite.containers.EntClient.UserFollow.Query().
			Where(user.follow.FollowerIDEQ(fixtures.TestUser1.ID)).
			Where(user.follow.FollowingIDEQ(fixtures.TestUser2.ID)).
			All(suite.ctx)
		suite.Assert().NoError(err)
		suite.Assert().Len(follows, 1)
	})

	suite.Run("follow community mutation", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser1.ID)

		// Test direct resolver method
		result, err := suite.resolver.Mutation().FollowCommunity(authCtx, fixtures.TestCommunity1.ID)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(result)
		suite.Assert().True(result.Success)

		// Verify follow was created in database
		follows, err := suite.containers.EntClient.CommunityFollow.Query().
			Where(community.follow.UserIDEQ(fixtures.TestUser1.ID)).
			Where(community.follow.CommunityIDEQ(fixtures.TestCommunity1.ID)).
			All(suite.ctx)
		suite.Assert().NoError(err)
		suite.Assert().Len(follows, 1)
	})
}

func (suite *GraphQLResolverTestSuite) TestAuthenticatedQueries() {
	// Test queries that require authentication

	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("me query with authenticated user", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser1.ID)

		// Test direct resolver method
		user, err := suite.resolver.Query().Me(authCtx)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(user)
		suite.Assert().Equal(fixtures.TestUser1.ID, user.ID)
		suite.Assert().Equal(fixtures.TestUser1.Name, user.Name)
		suite.Assert().Equal(fixtures.TestUser1.Email, user.Email)
	})

	suite.Run("me query without authentication", func() {
		// Test without user context
		user, err := suite.resolver.Query().Me(suite.ctx)

		suite.Assert().Error(err)
		suite.Assert().Nil(user)
	})

	suite.Run("my permissions query", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.TestUser1.ID)

		// Test direct resolver method
		permissions, err := suite.resolver.Query().MyPermissions(authCtx, []int{fixtures.TestCommunity1.ID})

		suite.Assert().NoError(err)
		suite.Assert().NotNil(permissions)
		suite.Assert().Contains(permissions, fixtures.TestCommunity1.ID)
	})
}

func (suite *GraphQLResolverTestSuite) TestErrorHandling() {
	// Test GraphQL error handling scenarios

	suite.Run("invalid user ID", func() {
		// Test with negative ID
		user, err := suite.resolver.Query().User(suite.ctx, -1)

		suite.Assert().Error(err)
		suite.Assert().Nil(user)
	})

	suite.Run("invalid community ID", func() {
		// Test with negative ID
		community, err := suite.resolver.Query().Community(suite.ctx, -1)

		suite.Assert().Error(err)
		suite.Assert().Nil(community)
	})

	suite.Run("invalid post ID", func() {
		// Test with negative ID
		post, err := suite.resolver.Query().Post(suite.ctx, -1)

		suite.Assert().Error(err)
		suite.Assert().Nil(post)
	})

	suite.Run("unauthorized mutation", func() {
		// Try to like post without authentication
		result, err := suite.resolver.Mutation().LikePost(suite.ctx, fixtures.TestPost1.ID)

		suite.Assert().Error(err)
		suite.Assert().Nil(result)
	})
}

func (suite *GraphQLResolverTestSuite) TestComplexQueries() {
	// Test complex queries with multiple relationships

	// Seed extended data
	err := fixtures.SeedComplexScenario(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	suite.Run("nested user relationships", func() {
		// Create a complete post with all relationships
		completePost, err := fixtures.CreateCompletePost(suite.ctx, suite.containers.EntClient, fixtures.LargeCommunity.ID, fixtures.AdminUser.ID)
		suite.Require().NoError(err)

		var resp struct {
			Post *models.Post `json:"post"`
		}

		query := `
			query($id: Int!) {
				post(id: $id) {
					id
					title
					author {
						id
						name
						avatar {
							id
							url
						}
					}
					community {
						id
						name
						logo {
							id
							url
						}
					}
					heroImage {
						id
						url
					}
					likes {
						id
						userId
					}
					bookmarks {
						id
						userId
					}
					comments {
						id
						content
						author {
							id
							name
						}
					}
				}
			}
		`

		err = suite.client.Post(query, &resp, client.Var("id", completePost.ID))

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp.Post)
		suite.Assert().Equal(completePost.ID, int(resp.Post.ID))
		suite.Assert().NotNil(resp.Post.Author)
		suite.Assert().NotNil(resp.Post.Community)
		suite.Assert().NotNil(resp.Post.HeroImage)
		suite.Assert().NotEmpty(resp.Post.Likes)
		suite.Assert().NotEmpty(resp.Post.Bookmarks)
		suite.Assert().NotEmpty(resp.Post.Comments)
	})

	suite.Run("user with permissions across communities", func() {
		// Create context with authenticated user
		authCtx := context.WithUserID(suite.ctx, fixtures.AdminUser.ID)

		// Get communities
		communities, err := suite.containers.EntClient.Community.Query().All(suite.ctx)
		suite.Require().NoError(err)

		var communityIDs []int
		for _, community := range communities {
			communityIDs = append(communityIDs, community.ID)
		}

		// Test permissions
		permissions, err := suite.resolver.Query().MyPermissions(authCtx, communityIDs)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(permissions)
		suite.Assert().Len(permissions, len(communityIDs))

		// Admin user should be owner of LargeCommunity
		if adminPermissions, exists := permissions[fixtures.LargeCommunity.ID]; exists {
			suite.Assert().NotNil(adminPermissions)
			suite.Assert().True(adminPermissions.CanDeletePosts)
			suite.Assert().True(adminPermissions.CanBanUsers)
			suite.Assert().True(adminPermissions.CanManageRoles)
		}
	})
}

func (suite *GraphQLResolverTestSuite) TestPaginationPerformance() {
	// Test pagination performance with large datasets

	// Create large dataset
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 10)
	suite.Require().NoError(err)

	community, err := fixtures.CreateRandomCommunity(suite.ctx, suite.containers.EntClient, users[0].ID)
	suite.Require().NoError(err)

	post, err := fixtures.CreateRandomPost(suite.ctx, suite.containers.EntClient, community.ID, users[0].ID)
	suite.Require().NoError(err)

	// Create many comments
	_, err = fixtures.CreateBulkComments(suite.ctx, suite.containers.EntClient, 100, post.ID, users[0].ID)
	suite.Require().NoError(err)

	suite.Run("large comment pagination performance", func() {
		start := time.Now()

		first := 20
		connection, err := suite.resolver.Query().CommentsByPost(suite.ctx, post.ID, &first, nil, nil, nil)

		duration := time.Since(start)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(connection)
		suite.Assert().LessOrEqual(len(connection.Edges), 20)
		suite.Assert().Less(duration, 100*time.Millisecond, "Large comment pagination should be fast")
	})

	suite.Run("deep comment pagination", func() {
		// Test pagination through multiple pages
		pageSize := 10
		var allComments []*models.Comment
		var cursor *string

		for page := 0; page < 5; page++ { // Get 5 pages
			first := pageSize
			connection, err := suite.resolver.Query().CommentsByPost(suite.ctx, post.ID, &first, cursor, nil, nil)
			suite.Assert().NoError(err)
			suite.Assert().NotNil(connection)

			for _, edge := range connection.Edges {
				allComments = append(allComments, edge.Node)
			}

			if !connection.PageInfo.HasNextPage {
				break
			}

			cursor = connection.PageInfo.EndCursor
		}

		// Verify we got data and it's properly ordered
		suite.Assert().NotEmpty(allComments)
		for i := 1; i < len(allComments); i++ {
			prev := allComments[i-1]
			curr := allCom
