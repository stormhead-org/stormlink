package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/usecase/comment"
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/post"
	"stormlink/server/usecase/user"
	"stormlink/services/auth/internal/service"
	"stormlink/shared/jwt"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type SystemPerformanceTestSuite struct {
	suite.Suite
	containers  *testcontainers.TestContainers
	client      *ent.Client
	userUC      user.UserUsecase
	communityUC community.CommunityUsecase
	postUC      post.PostUsecase
	commentUC   comment.CommentUsecase
	authService *service.AuthService
	ctx         context.Context

	// Performance tracking
	testUsers       []*ent.User
	testCommunities []*ent.Community
	testPosts       []*ent.Post
	testComments    []*ent.Comment
}

func (suite *SystemPerformanceTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers for realistic performance testing
	containers, err := testcontainers.Setup(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Use PostgreSQL for more realistic performance testing
	suite.client = enttest.Open(suite.T(), "postgres", containers.PostgresDSN())

	// Initialize use cases
	suite.userUC = user.NewUserUsecase(suite.client)
	suite.communityUC = community.NewCommunityUsecase(suite.client)
	suite.postUC = post.NewPostUsecase(suite.client)
	suite.commentUC = comment.NewCommentUsecase(suite.client)
	suite.authService = service.NewAuthService(suite.client, suite.userUC)

	// Create large dataset for performance testing
	suite.createLargeDataset()
}

func (suite *SystemPerformanceTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
	if suite.containers != nil {
		suite.containers.Cleanup()
	}
}

func (suite *SystemPerformanceTestSuite) createLargeDataset() {
	suite.T().Log("Creating large dataset for performance testing...")
	start := time.Now()

	// Create 1000 users
	suite.testUsers = make([]*ent.User, 1000)
	for i := 0; i < 1000; i++ {
		userFixture := fixtures.UserFixture{
			Name:       fmt.Sprintf("PerfUser_%d", i),
			Slug:       fmt.Sprintf("perf-user-%d", i),
			Email:      fmt.Sprintf("perfuser%d@test.com", i),
			Password:   "password123",
			Salt:       fmt.Sprintf("salt-%d", i),
			IsVerified: true,
			CreatedAt:  time.Now().Add(time.Duration(-i) * time.Hour),
		}

		user, err := fixtures.CreateTestUser(suite.ctx, suite.client, userFixture)
		suite.Require().NoError(err)
		suite.testUsers[i] = user

		// Progress indication
		if i%100 == 0 {
			suite.T().Logf("Created %d users", i)
		}
	}

	// Create 100 communities
	suite.testCommunities = make([]*ent.Community, 100)
	for i := 0; i < 100; i++ {
		communityFixture := fixtures.CommunityFixture{
			Name:        fmt.Sprintf("PerfCommunity_%d", i),
			Slug:        fmt.Sprintf("perf-community-%d", i),
			Description: fmt.Sprintf("Performance test community %d with longer description", i),
			IsPrivate:   i%10 == 0, // 10% private communities
			OwnerID:     suite.testUsers[i%len(suite.testUsers)].ID,
			CreatedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
		}

		community, err := fixtures.CreateTestCommunity(suite.ctx, suite.client, communityFixture)
		suite.Require().NoError(err)
		suite.testCommunities[i] = community

		// Create community info for some communities
		if i%2 == 0 {
			_, err = suite.client.CommunityInfo.Create().
				SetCommunityID(community.ID).
				SetLongDescription(fmt.Sprintf("Long description for performance community %d", i)).
				SetRules("1. Be respectful\n2. No spam\n3. Stay on topic").
				SetMemberCount(i * 10).
				SetPostCount(i * 5).
				SetCreatedAt(time.Now()).
				SetUpdatedAt(time.Now()).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}
	}

	// Create 5000 posts
	suite.testPosts = make([]*ent.Post, 5000)
	for i := 0; i < 5000; i++ {
		postFixture := fixtures.PostFixture{
			Title:       fmt.Sprintf("Performance Test Post %d", i),
			Content:     fmt.Sprintf("This is the content for performance test post %d. It contains some text to simulate real posts.", i),
			CommunityID: suite.testCommunities[i%len(suite.testCommunities)].ID,
			AuthorID:    suite.testUsers[i%len(suite.testUsers)].ID,
			CreatedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
		}

		post, err := fixtures.CreateTestPost(suite.ctx, suite.client, postFixture)
		suite.Require().NoError(err)
		suite.testPosts[i] = post

		if i%500 == 0 {
			suite.T().Logf("Created %d posts", i)
		}
	}

	// Create 20000 comments
	suite.testComments = make([]*ent.Comment, 20000)
	for i := 0; i < 20000; i++ {
		commentFixture := fixtures.CommentFixture{
			Content:   fmt.Sprintf("Performance test comment %d with some content", i),
			PostID:    suite.testPosts[i%len(suite.testPosts)].ID,
			AuthorID:  suite.testUsers[i%len(suite.testUsers)].ID,
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Second),
		}

		// 20% of comments are replies
		if i > 0 && i%5 == 0 {
			parentComment := suite.testComments[i-1]
			commentFixture.ParentID = &parentComment.ID
		}

		comment, err := fixtures.CreateTestComment(suite.ctx, suite.client, commentFixture)
		suite.Require().NoError(err)
		suite.testComments[i] = comment

		if i%2000 == 0 {
			suite.T().Logf("Created %d comments", i)
		}
	}

	// Create some likes and bookmarks for realistic load
	for i := 0; i < 1000; i++ {
		// Create post likes
		_, err := suite.client.PostLike.Create().
			SetPostID(suite.testPosts[i%len(suite.testPosts)].ID).
			SetUserID(suite.testUsers[(i*2)%len(suite.testUsers)].ID).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create post bookmarks
		if i%3 == 0 {
			_, err = suite.client.PostBookmark.Create().
				SetPostID(suite.testPosts[i%len(suite.testPosts)].ID).
				SetUserID(suite.testUsers[(i*3)%len(suite.testUsers)].ID).
				SetCreatedAt(time.Now()).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}
	}

	// Create community memberships
	for i := 0; i < 2000; i++ {
		_, err := suite.client.CommunityMember.Create().
			SetCommunityID(suite.testCommunities[i%len(suite.testCommunities)].ID).
			SetUserID(suite.testUsers[i%len(suite.testUsers)].ID).
			SetJoinedAt(time.Now().Add(time.Duration(-i) * time.Hour)).
			Save(suite.ctx)
		suite.Require().NoError(err)
	}

	suite.T().Logf("Dataset creation completed in %v", time.Since(start))
}

// User Performance Tests
func (suite *SystemPerformanceTestSuite) TestUserPerformance() {
	suite.Run("user retrieval performance", func() {
		// Warm up
		for i := 0; i < 100; i++ {
			_, err := suite.userUC.GetUserByID(suite.ctx, suite.testUsers[i].ID)
			suite.NoError(err)
		}

		// Measure performance
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			userID := suite.testUsers[i%len(suite.testUsers)].ID
			_, err := suite.userUC.GetUserByID(suite.ctx, userID)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("User retrieval: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 10*time.Millisecond, "Average user retrieval should be under 10ms")
	})

	suite.Run("user permissions performance", func() {
		iterations := 500
		communityIDs := make([]int, 10)
		for i := 0; i < 10; i++ {
			communityIDs[i] = suite.testCommunities[i].ID
		}

		start := time.Now()

		for i := 0; i < iterations; i++ {
			userID := suite.testUsers[i%len(suite.testUsers)].ID
			_, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, userID, communityIDs)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("User permissions: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 20*time.Millisecond, "Average permissions check should be under 20ms")
	})

	suite.Run("concurrent user operations", func() {
		concurrency := 50
		requestsPerWorker := 100

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					userID := suite.testUsers[(workerID*requestsPerWorker+j)%len(suite.testUsers)].ID
					_, err := suite.userUC.GetUserByID(suite.ctx, userID)
					if err != nil {
						suite.T().Errorf("Worker %d request %d failed: %v", workerID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		totalRequests := concurrency * requestsPerWorker

		suite.T().Logf("Concurrent users: %d requests with %d workers in %v", totalRequests, concurrency, duration)
		avgDuration := duration / time.Duration(totalRequests)
		suite.Less(avgDuration, 15*time.Millisecond, "Average concurrent request should be under 15ms")
	})
}

// Post Performance Tests
func (suite *SystemPerformanceTestSuite) TestPostPerformance() {
	suite.Run("post retrieval with relations performance", func() {
		iterations := 500
		start := time.Now()

		for i := 0; i < iterations; i++ {
			postID := suite.testPosts[i%len(suite.testPosts)].ID
			_, err := suite.postUC.GetPostByID(suite.ctx, postID)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Post retrieval: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 25*time.Millisecond, "Average post retrieval should be under 25ms")
	})

	suite.Run("post status performance", func() {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			postID := suite.testPosts[i%len(suite.testPosts)].ID
			userID := suite.testUsers[i%len(suite.testUsers)].ID
			_, err := suite.postUC.GetPostStatus(suite.ctx, userID, postID)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Post status: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 15*time.Millisecond, "Average post status should be under 15ms")
	})
}

// Comment Performance Tests
func (suite *SystemPerformanceTestSuite) TestCommentPerformance() {
	suite.Run("comment retrieval performance", func() {
		iterations := 500
		start := time.Now()

		for i := 0; i < iterations; i++ {
			postID := suite.testPosts[i%len(suite.testPosts)].ID
			_, err := suite.commentUC.GetCommentsByPostIDLight(suite.ctx, postID, nil)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Comment retrieval: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 30*time.Millisecond, "Average comment retrieval should be under 30ms")
	})

	suite.Run("paginated comment performance", func() {
		iterations := 300
		start := time.Now()

		for i := 0; i < iterations; i++ {
			postID := suite.testPosts[i%len(suite.testPosts)].ID
			_, err := suite.commentUC.GetCommentsByPostIDLightPaginated(suite.ctx, postID, nil, 20, int32(i%10)*20)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Paginated comments: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 25*time.Millisecond, "Average paginated comment retrieval should be under 25ms")
	})

	suite.Run("comment connection performance", func() {
		iterations := 200
		first := 25
		start := time.Now()

		for i := 0; i < iterations; i++ {
			postID := suite.testPosts[i%len(suite.testPosts)].ID
			_, err := suite.commentUC.CommentsByPostConnection(suite.ctx, postID, nil, &first, nil, nil, nil)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Comment connections: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 35*time.Millisecond, "Average comment connection should be under 35ms")
	})
}

// Community Performance Tests
func (suite *SystemPerformanceTestSuite) TestCommunityPerformance() {
	suite.Run("community retrieval performance", func() {
		iterations := 500
		start := time.Now()

		for i := 0; i < iterations; i++ {
			communityID := suite.testCommunities[i%len(suite.testCommunities)].ID
			_, err := suite.communityUC.GetCommunityByID(suite.ctx, communityID)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Community retrieval: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 20*time.Millisecond, "Average community retrieval should be under 20ms")
	})

	suite.Run("community status performance", func() {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			communityID := suite.testCommunities[i%len(suite.testCommunities)].ID
			userID := suite.testUsers[i%len(suite.testUsers)].ID
			_, err := suite.communityUC.GetCommunityStatus(suite.ctx, userID, communityID)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Community status: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 15*time.Millisecond, "Average community status should be under 15ms")
	})
}

// Authentication Performance Tests
func (suite *SystemPerformanceTestSuite) TestAuthenticationPerformance() {
	suite.Run("token validation performance", func() {
		// Generate tokens for testing
		tokens := make([]string, 100)
		for i := 0; i < 100; i++ {
			token, err := jwt.GenerateAccessToken(suite.testUsers[i].ID)
			suite.Require().NoError(err)
			tokens[i] = token
		}

		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			token := tokens[i%len(tokens)]
			_, err := jwt.ParseAccessToken(token)
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		suite.T().Logf("Token validation: %d requests in %v (avg: %v per request)", iterations, duration, avgDuration)
		suite.Less(avgDuration, 1*time.Millisecond, "Average token validation should be under 1ms")
	})

	suite.Run("concurrent authentication", func() {
		concurrency := 20
		requestsPerWorker := 50

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					userIndex := (workerID*requestsPerWorker + j) % len(suite.testUsers)
					token, err := jwt.GenerateAccessToken(suite.testUsers[userIndex].ID)
					if err != nil {
						suite.T().Errorf("Token generation failed: %v", err)
						continue
					}

					_, err = jwt.ParseAccessToken(token)
					if err != nil {
						suite.T().Errorf("Token validation failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		totalRequests := concurrency * requestsPerWorker

		suite.T().Logf("Concurrent auth: %d requests with %d workers in %v", totalRequests, concurrency, duration)
		avgDuration := duration / time.Duration(totalRequests)
		suite.Less(avgDuration, 2*time.Millisecond, "Average concurrent auth should be under 2ms")
	})
}

// Memory and Resource Usage Tests
func (suite *SystemPerformanceTestSuite) TestResourceUsage() {
	suite.Run("memory usage during heavy operations", func() {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Perform memory-intensive operations
		for i := 0; i < 100; i++ {
			// Get posts with all relations
			_, err := suite.postUC.GetPostByID(suite.ctx, suite.testPosts[i].ID)
			suite.NoError(err)

			// Get comments for posts
			_, err = suite.commentUC.GetCommentsByPostID(suite.ctx, suite.testPosts[i].ID, nil)
			suite.NoError(err)

			// Get community with relations
			communityID := suite.testCommunities[i%len(suite.testCommunities)].ID
			_, err = suite.communityUC.GetCommunityByID(suite.ctx, communityID)
			suite.NoError(err)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		allocDiff := m2.TotalAlloc - m1.TotalAlloc
		heapDiff := m2.HeapAlloc - m1.HeapAlloc

		suite.T().Logf("Memory usage - Total allocated: %d bytes, Heap difference: %d bytes", allocDiff, heapDiff)

		// Memory usage should be reasonable (less than 100MB for this test)
		suite.Less(heapDiff, uint64(100*1024*1024), "Heap usage should be under 100MB")
	})

	suite.Run("database connection pool performance", func() {
		concurrency := 100
		requestsPerWorker := 20

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					userID := suite.testUsers[(workerID*requestsPerWorker+j)%len(suite.testUsers)].ID
					_, err := suite.userUC.GetUserByID(suite.ctx, userID)
					if err != nil {
						suite.T().Errorf("DB pool test failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		totalRequests := concurrency * requestsPerWorker

		suite.T().Logf("DB pool test: %d requests with %d workers in %v", totalRequests, concurrency, duration)
		avgDuration := duration / time.Duration(totalRequests)
		suite.Less(avgDuration, 20*time.Millisecond, "Average DB pool request should be under 20ms")
	})
}

// Stress Tests
func (suite *SystemPerformanceTestSuite) TestStressScenarios() {
	suite.Run("high load mixed operations", func() {
		duration := 30 * time.Second
		concurrency := 50

		ctx, cancel := context.WithTimeout(suite.ctx, duration)
		defer cancel()

		var wg sync.WaitGroup
		requestCounts := make([]int, concurrency)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				count := 0

				for {
					select {
					case <-ctx.Done():
						requestCounts[workerID] = count
						return
					default:
						// Mix of different operations
						switch count % 4 {
						case 0:
							userID := suite.testUsers[count%len(suite.testUsers)].ID
							_, err := suite.userUC.GetUserByID(suite.ctx, userID)
							if err != nil {
								suite.T().Logf("User retrieval error: %v", err)
							}
						case 1:
							postID := suite.testPosts[count%len(suite.testPosts)].ID
							_, err := suite.postUC.GetPostByID(suite.ctx, postID)
							if err != nil {
								suite.T().Logf("Post retrieval error: %v", err)
							}
						case 2:
							communityID := suite.testCommunities[count%len(suite.testCommunities)].ID
							_, err := suite.communityUC.GetCommunityByID(suite.ctx, communityID)
							if err != nil {
								suite.T().Logf("Community retrieval error: %v", err)
							}
						case 3:
							postID := suite.testPosts[count%len(suite.testPosts)].ID
							_, err := suite.commentUC.GetCommentsByPostIDLight(suite.ctx, postID, nil)
							if err != nil {
								suite.T().Logf("Comment retrieval error: %v", err)
							}
						}
						count++
					}
				}
			}(i)
		}

		wg.Wait()
		actualDuration := time.Since(start)

		totalRequests := 0
		for _, count := range requestCounts {
			totalRequests += count
		}

		rps := float64(totalRequests) / actualDuration.Seconds()
		suite.T().Logf("Stress test: %d requests in %v with %d workers (%.2f RPS)",
			totalRequests, actualDuration, concurrency, rps)

		// Should handle at least 100 RPS under stress
		suite.Greater(rps, 100.0, "Should handle at least 100 requests per second")
	})
}

// Benchmark functions for go test -bench
func BenchmarkUserRetrieval(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	uc := user.NewUserUsecase(client)

	// Create test user
	testUser, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetUserByID(ctx, testUser.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostRetrieval(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	uc := post.NewPostUsecase(client)

	// Create test data
	err := fixtures.SeedBasicData(ctx, client)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetPostByID(ctx, fixtures.TestPost1.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenValidation(b *testing.B) {
	// Generate test token
	token, err := jwt.GenerateAccessToken(1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jwt.ParseAccessToken(token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSystemPerformanceTestSuite(t *testing.T) {
	// Skip performance tests in short mode
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite.Run(t, new(SystemPerformanceTestSuite))
}
