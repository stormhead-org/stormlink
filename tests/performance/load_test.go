package performance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"stormlink/server/usecase/comment"
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/post"
	"stormlink/server/usecase/user"
	"stormlink/services/auth/internal/service"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	authpb "stormlink/server/grpc/auth/protobuf"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LoadTestSuite struct {
	suite.Suite
	containers    *testcontainers.TestContainers
	userUC        user.UserUsecase
	communityUC   community.CommunityUsecase
	postUC        post.PostUsecase
	commentUC     comment.CommentUsecase
	authService   *service.AuthService
	ctx           context.Context
}

func (suite *LoadTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create usecases
	suite.userUC = user.NewUserUsecase(containers.EntClient)
	suite.communityUC = community.NewCommunityUsecase(containers.EntClient)
	suite.postUC = post.NewPostUsecase(containers.EntClient)
	suite.commentUC = comment.NewCommentUsecase(containers.EntClient)

	// Create auth service
	suite.authService = service.NewAuthService(containers.EntClient, suite.userUC)
	suite.authService.SetRedisClient(containers.RedisClient)
}

func (suite *LoadTestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *LoadTestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *LoadTestSuite) TestHighConcurrencyUserOperations() {
	// Test system under high concurrency load

	// Create base test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	concurrency := 100
	operationsPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*operationsPerGoroutine)

	start := time.Now()

	// Spawn concurrent goroutines performing user operations
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of different operations
				switch j % 4 {
				case 0:
					// User retrieval
					_, err := suite.userUC.GetUserByID(suite.ctx, fixtures.TestUser1.ID)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d, op %d: GetUserByID failed: %w", goroutineID, j, err)
						return
					}

				case 1:
					// Permission check
					_, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, fixtures.TestUser1.ID, []int{fixtures.TestCommunity1.ID})
					if err != nil {
						errors <- fmt.Errorf("goroutine %d, op %d: GetPermissionsByCommunities failed: %w", goroutineID, j, err)
						return
					}

				case 2:
					// User status
					_, err := suite.userUC.GetUserStatus(suite.ctx, fixtures.TestUser1.ID, fixtures.TestUser2.ID)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d, op %d: GetUserStatus failed: %w", goroutineID, j, err)
						return
					}

				case 3:
					// Community retrieval
					_, err := suite.communityUC.GetCommunityByID(suite.ctx, fixtures.TestCommunity1.ID)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d, op %d: GetCommunityByID failed: %w", goroutineID, j, err)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No errors should occur during load test")

	totalOperations := concurrency * operationsPerGoroutine
	opsPerSecond := float64(totalOperations) / duration.Seconds()

	suite.T().Logf("Completed %d operations in %v (%.2f ops/sec)", totalOperations, duration, opsPerSecond)
	suite.Assert().Greater(opsPerSecond, 100.0, "Should handle at least 100 operations per second")
}

func (suite *LoadTestSuite) TestAuthenticationLoad() {
	// Test authentication system under load

	// Create multiple test users
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 50)
	suite.Require().NoError(err)

	concurrency := 20
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	start := time.Now()

	// Concurrent login attempts
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()

			user := users[userIndex%len(users)]

			// Login
			loginReq := &authpb.LoginRequest{
				Email:    user.Email,
				Password: "password123", // Default password from CreateRandomUser
			}

			loginResp, err := suite.authService.Login(suite.ctx, loginReq)
			if err != nil {
				errors <- fmt.Errorf("login failed for user %d: %w", user.ID, err)
				return
			}

			// Validate token
			validateReq := &authpb.ValidateTokenRequest{
				Token: loginResp.AccessToken,
			}

			validateResp, err := suite.authService.ValidateToken(suite.ctx, validateReq)
			if err != nil {
				errors <- fmt.Errorf("token validation failed for user %d: %w", user.ID, err)
				return
			}

			if !validateResp.IsValid {
				errors <- fmt.Errorf("token is invalid for user %d", user.ID)
				return
			}

			// Refresh token
			refreshReq := &authpb.RefreshTokenRequest{
				RefreshToken: loginResp.RefreshToken,
			}

			_, err = suite.authService.RefreshToken(suite.ctx, refreshReq)
			if err != nil {
				errors <- fmt.Errorf("token refresh failed for user %d: %w", user.ID, err)
				return
			}

		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No auth errors should occur during load test")

	authOpsPerSecond := float64(concurrency*3) / duration.Seconds() // 3 ops per goroutine
	suite.T().Logf("Completed %d auth operations in %v (%.2f auth ops/sec)", concurrency*3, duration, authOpsPerSecond)
	suite.Assert().Greater(authOpsPerSecond, 50.0, "Should handle at least 50 auth operations per second")
}

func (suite *LoadTestSuite) TestCommentPaginationLoad() {
	// Test comment pagination under load with large datasets

	// Create large dataset
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 20)
	suite.Require().NoError(err)

	community, err := fixtures.CreateRandomCommunity(suite.ctx, suite.containers.EntClient, users[0].ID)
	suite.Require().NoError(err)

	testPost, err := fixtures.CreateRandomPost(suite.ctx, suite.containers.EntClient, community.ID, users[0].ID)
	suite.Require().NoError(err)

	// Create large number of comments
	commentCount := 1000
	suite.T().Logf("Creating %d comments for pagination load test...", commentCount)

	_, err = fixtures.CreateBulkComments(suite.ctx, suite.containers.EntClient, commentCount, testPost.ID, users[0].ID)
	suite.Require().NoError(err)

	concurrency := 10
	pagesPerGoroutine := 5
	pageSize := 20

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*pagesPerGoroutine)

	start := time.Now()

	// Concurrent pagination requests
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			var cursor *string

			for page := 0; page < pagesPerGoroutine; page++ {
				first := pageSize
				connection, err := suite.commentUC.CommentsByPostConnection(
					suite.ctx, testPost.ID, nil, &first, cursor, nil, nil)

				if err != nil {
					errors <- fmt.Errorf("goroutine %d, page %d: pagination failed: %w", goroutineID, page, err)
					return
				}

				if connection == nil {
					errors <- fmt.Errorf("goroutine %d, page %d: connection is nil", goroutineID, page)
					return
				}

				// Verify page has reasonable number of comments
				if len(connection.Edges) == 0 && page == 0 {
					errors <- fmt.Errorf("goroutine %d: first page should have comments", goroutineID)
					return
				}

				// Verify ordering
				for i := 1; i < len(connection.Edges); i++ {
					prev := connection.Edges[i-1].Node
					curr := connection.Edges[i].Node
					if prev.CreatedAt.After(curr.CreatedAt) ||
						(prev.CreatedAt.Equal(curr.CreatedAt) && prev.ID >= curr.ID) {
						errors <- fmt.Errorf("goroutine %d, page %d: comments not properly ordered", goroutineID, page)
						return
					}
				}

				// Move to next page
				if connection.PageInfo.HasNextPage {
					cursor = connection.PageInfo.EndCursor
				} else {
					break
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No pagination errors should occur during load test")

	totalPaginationOps := concurrency * pagesPerGoroutine
	paginationOpsPerSecond := float64(totalPaginationOps) / duration.Seconds()

	suite.T().Logf("Completed %d pagination operations in %v (%.2f pagination ops/sec)",
		totalPaginationOps, duration, paginationOpsPerSecond)
	suite.Assert().Greater(paginationOpsPerSecond, 20.0, "Should handle at least 20 pagination operations per second")
}

func (suite *LoadTestSuite) TestDatabaseConnectionPooling() {
	// Test database connection pooling under load

	// Create test data
	err := fixtures.SeedBasicData(suite.ctx, suite.containers.EntClient)
	suite.Require().NoError(err)

	concurrency := 50 // More than typical connection pool size
	operationsPerGoroutine := 20

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*operationsPerGoroutine)
	durations := make(chan time.Duration, concurrency*operationsPerGoroutine)

	start := time.Now()

	// Spawn many concurrent database operations
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				opStart := time.Now()

				// Perform database-intensive operation
				_, err := suite.userUC.GetUserByID(suite.ctx, fixtures.TestUser1.ID)
				opDuration := time.Since(opStart)

				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: database operation failed: %w", goroutineID, j, err)
					return
				}

				durations <- opDuration
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)
	close(errors)
	close(durations)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No database errors should occur under connection pool load")

	// Analyze operation durations
	var totalOpDuration time.Duration
	var maxOpDuration time.Duration
	opCount := 0

	for duration := range durations {
		totalOpDuration += duration
		if duration > maxOpDuration {
			maxOpDuration = duration
		}
		opCount++
	}

	avgOpDuration := totalOpDuration / time.Duration(opCount)

	suite.T().Logf("Database pool test: %d operations in %v", opCount, totalDuration)
	suite.T().Logf("Average operation duration: %v", avgOpDuration)
	suite.T().Logf("Maximum operation duration: %v", maxOpDuration)

	// Performance assertions
	suite.Assert().Less(avgOpDuration, 50*time.Millisecond, "Average database operation should be fast")
	suite.Assert().Less(maxOpDuration, 200*time.Millisecond, "No operation should take too long")
}

func (suite *LoadTestSuite) TestRedisPerformance() {
	// Test Redis performance under load

	concurrency := 30
	operationsPerGoroutine := 100

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*operationsPerGoroutine)

	start := time.Now()

	// Concurrent Redis operations
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("load_test:%d:%d", goroutineID, j)
				value := fmt.Sprintf("value_%d_%d", goroutineID, j)

				// Set
				err := suite.containers.RedisClient.Set(suite.ctx, key, value, time.Hour).Err()
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: Redis SET failed: %w", goroutineID, j, err)
					return
				}

				// Get
				retrievedValue, err := suite.containers.RedisClient.Get(suite.ctx, key).Result()
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: Redis GET failed: %w", goroutineID, j, err)
					return
				}

				if retrievedValue != value {
					errors <- fmt.Errorf("goroutine %d, op %d: Redis value mismatch: expected %s, got %s", goroutineID, j, value, retrievedValue)
					return
				}

				// Delete
				err = suite.containers.RedisClient.Del(suite.ctx, key).Err()
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: Redis DEL failed: %w", goroutineID, j, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No Redis errors should occur during load test")

	totalRedisOps := concurrency * operationsPerGoroutine * 3 // SET, GET, DEL per iteration
	redisOpsPerSecond := float64(totalRedisOps) / duration.Seconds()

	suite.T().Logf("Completed %d Redis operations in %v (%.2f Redis ops/sec)", totalRedisOps, duration, redisOpsPerSecond)
	suite.Assert().Greater(redisOpsPerSecond, 1000.0, "Should handle at least 1000 Redis operations per second")
}

func (suite *LoadTestSuite) TestDataConsistencyUnderLoad() {
	// Test that data remains consistent under concurrent modifications

	// Create initial test data
	initialUsers, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 10)
	suite.Require().NoError(err)

	community, err := fixtures.CreateRandomCommunity(suite.ctx, suite.containers.EntClient, initialUsers[0].ID)
	suite.Require().NoError(err)

	testPost, err := fixtures.CreateRandomPost(suite.ctx, suite.containers.EntClient, community.ID, initialUsers[0].ID)
	suite.Require().NoError(err)

	concurrency := 20
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*operationsPerGoroutine)

	// Concurrent operations that modify relationships
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Create unique user for each operation to avoid conflicts
				user, err := fixtures.CreateRandomUser(suite.ctx, suite.containers.EntClient)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: CreateRandomUser failed: %w", goroutineID, j, err)
					return
				}

				// Create follow relationship
				_, err = fixtures.CreateTestFollow(suite.ctx, suite.containers.EntClient, fixtures.FollowFixture{
					FollowerID:  user.ID,
					FollowingID: initialUsers[0].ID,
					CreatedAt:   time.Now(),
				})
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: CreateTestFollow failed: %w", goroutineID, j, err)
					return
				}

				// Create post like
				_, err = fixtures.CreateTestPostLike(suite.ctx, suite.containers.EntClient, fixtures.PostLikeFixture{
					UserID:    user.ID,
					PostID:    testPost.ID,
					CreatedAt: time.Now(),
				})
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: CreateTestPostLike failed: %w", goroutineID, j, err)
					return
				}

				// Create comment
				_, err = fixtures.CreateRandomComment(suite.ctx, suite.containers.EntClient, testPost.ID, user.ID, nil)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: CreateRandomComment failed: %w", goroutineID, j, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No consistency errors should occur during concurrent modifications")

	// Verify final data integrity
	finalUserCount, err := suite.containers.EntClient.User.Query().Count(suite.ctx)
	suite.Assert().NoError(err)
	expectedUserCount := len(initialUsers) + (concurrency * operationsPerGoroutine)
	suite.Assert().Equal(expectedUserCount, finalUserCount, "User count should match expected")

	finalFollowCount, err := suite.containers.EntClient.UserFollow.Query().Count(suite.ctx)
	suite.Assert().NoError(err)
	expectedFollowCount := concurrency * operationsPerGoroutine
	suite.Assert().Equal(expectedFollowCount, finalFollowCount, "Follow count should match expected")

	finalLikeCount, err := suite.containers.EntClient.PostLike.Query().Count(suite.ctx)
	suite.Assert().NoError(err)
	expectedLikeCount := concurrency * operationsPerGoroutine
	suite.Assert().Equal(expectedLikeCount, finalLikeCount, "Like count should match expected")

	finalCommentCount, err := suite.containers.EntClient.Comment.Query().Count(suite.ctx)
	suite.Assert().NoError(err)
	expectedCommentCount := concurrency * operationsPerGoroutine
	suite.Assert().Equal(expectedCommentCount, finalCommentCount, "Comment count should match expected")
}

func (suite *LoadTestSuite) TestMemoryUsage() {
	// Test memory usage patterns during intensive operations

	// Create large dataset
	userCount := 100
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, userCount)
	suite.Require().NoError(err)

	communityCount := 20
	communities, err := fixtures.CreateBulkCommunities(suite.ctx, suite.containers.EntClient, communityCount, users[0].ID)
	suite.Require().NoError(err)

	// Create posts across communities
	postsPerCommunity := 10
	for _, community := range communities {
		_, err := fixtures.CreateBulkPosts(suite.ctx, suite.containers.EntClient, postsPerCommunity, community.ID, users[0].ID)
		suite.Require().NoError(err)
	}

	// Test memory-intensive operations
	suite.Run("bulk user retrieval", func() {
		start := time.Now()

		var retrievedUsers []*ent.User
		for _, user := range users {
			retrievedUser, err := suite.userUC.GetUserByID(suite.ctx, user.ID)
			suite.Assert().NoError(err)
			retrievedUsers = append(retrievedUsers, retrievedUser)
		}

		duration := time.Since(start)
		suite.T().Logf("Retrieved %d users in %v", len(retrievedUsers), duration)
		suite.Assert().Len(retrievedUsers, userCount)
	})

	suite.Run("bulk community retrieval", func() {
		start := time.Now()

		var retrievedCommunities []*ent.Community
		for _, community := range communities {
			retrievedCommunity, err := suite.communityUC.GetCommunityByID(suite.ctx, community.ID)
			suite.Assert().NoError(err)
			retrievedCommunities = append(retrievedCommunities, retrievedCommunity)
		}

		duration := time.Since(start)
		suite.T().Logf("Retrieved %d communities in %v", len(retrievedCommunities), duration)
		suite.Assert().Len(retrievedCommunities, communityCount)
	})

	suite.Run("bulk permission checks", func() {
		start := time.Now()

		var communityIDs []int
		for _, community := range communities {
			communityIDs = append(communityIDs, community.ID)
		}

		for _, user := range users[:10] { // Test first 10 users
			permissions, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, user.ID, communityIDs)
			suite.Assert().NoError(err)
			suite.Assert().Len(permissions, len(communityIDs))
		}

		duration := time.Since(start)
		suite.T().Logf("Checked permissions for 10 users across %d communities in %v", len(communityIDs), duration)
		suite.Assert().Less(duration, 5*time.Second, "Bulk permission checks should complete within 5 seconds")
	})
}

func (suite *LoadTestSuite) TestLongRunningOperations() {
	// Test system stability during long-running operations

	// Create substantial test data
	users, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 30)
	suite.Require().NoError(err)

	communities, err := fixtures.CreateBulkCommunities(suite.ctx, suite.containers.EntClient, 10, users[0].ID)
	suite.Require().NoError(err)

	// Create posts and comments
	for _, community := range communities {
		posts, err := fixtures.CreateBulkPosts(suite.ctx, suite.containers.EntClient, 5, community.ID, users[0].ID)
		suite.Require().NoError(err)

		for _, post := range posts {
			_, err := fixtures.CreateBulkComments(suite.ctx, suite.containers.EntClient, 20, post.ID, users[1].ID)
			suite.Require().NoError(err)
		}
	}

	// Run operations for extended period
	testDuration := 30 * time.Second
	operationInterval := 100 * time.Millisecond

	var wg sync.WaitGroup
	errors := make(chan error, 1000) // Large buffer for errors
	stopChan := make(chan struct{})

	// Start long-running operation goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ticker := time.NewTicker(operationInterval)
			defer ticker.Stop()

			operationCount := 0
			for {
				select {
				case <-stopChan:
					suite.T().Logf("Worker %d completed %d operations", workerID, operationCount)
					return
				case <-ticker.C:
					// Rotate through different operations
					switch operationCount % 4 {
					case 0:
						_, err := suite.userUC.GetUserByID(suite.ctx, users[operationCount%len(users)].ID)
						if err != nil {
							errors <- fmt.Errorf("worker %d: GetUserByID failed: %w", workerID, err)
						}
					case 1:
						_, err := suite.communityUC.GetCommunityByID(suite.ctx, communities[operationCount%len(communities)].ID)
						if err != nil {
							errors <- fmt.Errorf("worker %d: GetCommunityByID failed: %w", workerID, err)
						}
					case 2:
						first := 10
						_, err := suite.commentUC.CommentsFeedConnection(suite.ctx, nil, &first, nil, nil, nil)
						if err != nil {
							errors <- fmt.Errorf("worker %d: CommentsFeedConnection failed: %w", workerID, err)
						}
					case 3:
						// Permission check
						communityIDs := []int{communities[0].ID, communities[1].ID}
						_, err := suite.userUC.GetPermissionsByCommunities(suite.ctx, users[0].ID, communityIDs)
						if err != nil {
							errors <- fmt.Errorf("worker %d: GetPermissionsByCommunities failed: %w", workerID, err)
						}
					}
					operationCount++
				}
			}
		}(i)
	}

	// Let it run for the test duration
	time.Sleep(testDuration)
	close(stopChan)
	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	suite.Assert().Empty(errorList, "No errors should occur during long-running operations")

	// Verify system is still responsive after long-running test
	_, err = suite.userUC.GetUserByID(suite.ctx, users[0].ID)
	suite.Assert().NoError(err, "System should remain responsive after load test")
}

func (suite *LoadTestSuite) TestScalabilityLimits() {
	// Test system behavior at scalability limits

	suite.Run("large comment pagination", func() {
		// Create very large comment dataset
		largeUsers, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 5)
		suite.Require().NoError(err)

		largeCommunity, err := fixtures.CreateRandomCommunity(suite.ctx, suite.containers.EntClient, largeUsers[0].ID)
		suite.Require().NoError(err)

		largePost, err := fixtures.CreateRandomPost(suite.ctx, suite.containers.EntClient, largeCommunity.ID, largeUsers[0].ID)
		suite.Require().NoError(err)

		// Create 5000 comments
		largeCommentCount := 5000
		suite.T().Logf("Creating %d comments for scalability test...", largeCommentCount)

		batchSize := 500
		for batch := 0; batch < largeCommentCount/batchSize; batch++ {
			_, err := fixtures.CreateBulkComments(suite.ctx, suite.containers.EntClient, batchSize, largePost.ID, largeUsers[0].ID)
			suite.Require().NoError(err)
		}

		// Test pagination performance with large dataset
		start := time.Now()

		first := 50
		connection, err := suite.commentUC.CommentsByPostConnection(suite.ctx, largePost.ID, nil, &first, nil, nil, nil)

		duration := time.Since(start)

		suite.Assert().NoError(err)
		suite.Assert().NotNil(connection)
		suite.Assert().LessOrEqual(len(connection.Edges), 50)
		suite.Assert().Less(duration, 500*time.Millisecond, "Large dataset pagination should complete within 500ms")

		suite.T().Logf("Paginated %d comments from %d total in %v", len(connection.Edges), largeCommentCount, duration)
	})

	suite.Run("deep comment thread", func() {
		// Test performance with deeply nested comment threads
		threadUsers, err := fixtures.CreateBulkUsers(suite.ctx, suite.containers.EntClient, 3)
		suite.Require().NoError(err)

		threadCommunity, err := fixtures.CreateRandomCommunity(suite.ctx, suite.containers.EntClient, threadUsers[0].ID)
		suite.Require().No
