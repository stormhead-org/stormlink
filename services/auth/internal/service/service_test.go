package service

import (
	"context"
	"testing"
	"time"

	"stormlink/server/ent/enttest"
	authpb "stormlink/server/grpc/auth/protobuf"
	useruc "stormlink/server/usecase/user"
	"stormlink/shared/jwt"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AuthServiceTestSuite struct {
	suite.Suite
	containers *testcontainers.TestContainers
	service    *AuthService
	ctx        context.Context
}

func (suite *AuthServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create service
	uc := useruc.NewUserUsecase(containers.EntClient)
	suite.service = NewAuthService(containers.EntClient, uc)
	suite.service.redis = containers.RedisClient
}

func (suite *AuthServiceTestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *AuthServiceTestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *AuthServiceTestSuite) TestLogin_Success() {
	// Create verified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	req := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: fixtures.TestUser1.Password,
	}

	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().NotEmpty(resp.AccessToken)
	suite.Assert().NotEmpty(resp.RefreshToken)
	suite.Assert().NotNil(resp.User)
	suite.Assert().Equal(testUser.ID, int(resp.User.Id))
	suite.Assert().Equal(testUser.Name, resp.User.Name)
	suite.Assert().Equal(testUser.Email, resp.User.Email)

	// Verify tokens are valid
	claims, err := jwt.ParseAccessToken(resp.AccessToken)
	suite.Assert().NoError(err)
	suite.Assert().Equal(testUser.ID, claims.UserID)

	refreshClaims, err := jwt.ParseRefreshToken(resp.RefreshToken)
	suite.Assert().NoError(err)
	suite.Assert().Equal(testUser.ID, refreshClaims.UserID)
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidCredentials() {
	// Create test user
	_, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	req := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: "wrong-password",
	}

	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.Unauthenticated, st.Code())
	suite.Assert().Contains(st.Message(), "invalid credentials")
}

func (suite *AuthServiceTestSuite) TestLogin_NonExistentUser() {
	req := &authpb.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.Unauthenticated, st.Code())
	suite.Assert().Contains(st.Message(), "invalid credentials")
}

func (suite *AuthServiceTestSuite) TestLogin_UnverifiedUser() {
	// Create unverified test user
	_, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	req := &authpb.LoginRequest{
		Email:    fixtures.UnverifiedUser.Email,
		Password: fixtures.UnverifiedUser.Password,
	}

	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.FailedPrecondition, st.Code())
	suite.Assert().Contains(st.Message(), "user email not verified")
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidEmail() {
	req := &authpb.LoginRequest{
		Email:    "invalid-email",
		Password: "password123",
	}

	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
}

func (suite *AuthServiceTestSuite) TestLogin_EmptyPassword() {
	req := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: "",
	}

	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
}

func (suite *AuthServiceTestSuite) TestLogin_WithRedisStorage() {
	// Create verified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	req := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: fixtures.TestUser1.Password,
	}

	resp, err := suite.service.Login(suite.ctx, req)
	suite.Require().NoError(err)

	// Verify refresh token is stored in Redis
	storedUserID, err := suite.containers.RedisClient.Get(suite.ctx, "refresh:"+resp.RefreshToken).Result()
	suite.Assert().NoError(err)
	suite.Assert().Equal(string(rune(testUser.ID)), storedUserID)
}

func (suite *AuthServiceTestSuite) TestRefreshToken_Success() {
	// Create test user and login first
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Generate valid refresh token
	refreshToken, err := fixtures.GenerateTestRefreshToken(testUser.ID)
	suite.Require().NoError(err)

	// Store in Redis if available
	if suite.service.redis != nil {
		err = suite.service.redis.Set(suite.ctx, "refresh:"+refreshToken, testUser.ID, 7*24*time.Hour).Err()
		suite.Require().NoError(err)
	}

	req := &authpb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	resp, err := suite.service.RefreshToken(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().NotEmpty(resp.AccessToken)
	suite.Assert().NotEmpty(resp.RefreshToken)

	// Verify new tokens are valid
	claims, err := jwt.ParseAccessToken(resp.AccessToken)
	suite.Assert().NoError(err)
	suite.Assert().Equal(testUser.ID, claims.UserID)

	newRefreshClaims, err := jwt.ParseRefreshToken(resp.RefreshToken)
	suite.Assert().NoError(err)
	suite.Assert().Equal(testUser.ID, newRefreshClaims.UserID)

	// Verify new refresh token is different (rotation)
	suite.Assert().NotEqual(refreshToken, resp.RefreshToken)
}

func (suite *AuthServiceTestSuite) TestRefreshToken_InvalidToken() {
	req := &authpb.RefreshTokenRequest{
		RefreshToken: "invalid-token",
	}

	resp, err := suite.service.RefreshToken(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.Unauthenticated, st.Code())
}

func (suite *AuthServiceTestSuite) TestRefreshToken_ExpiredToken() {
	// Create test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Generate expired refresh token (manually create with past expiry)
	claims := map[string]interface{}{
		"user_id": string(rune(testUser.ID)),
		"exp":     time.Now().Add(-24 * time.Hour).Unix(), // Expired
		"type":    "refresh",
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, err := expiredToken.SignedString([]byte("test-secret"))
	suite.Require().NoError(err)

	req := &authpb.RefreshTokenRequest{
		RefreshToken: expiredTokenString,
	}

	resp, err := suite.service.RefreshToken(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.Unauthenticated, st.Code())
}

func (suite *AuthServiceTestSuite) TestValidateToken_Success() {
	// Create test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Generate valid access token
	accessToken, err := fixtures.GenerateTestJWT(testUser.ID)
	suite.Require().NoError(err)

	req := &authpb.ValidateTokenRequest{
		Token: accessToken,
	}

	resp, err := suite.service.ValidateToken(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().True(resp.IsValid)
	suite.Assert().Equal(int32(testUser.ID), resp.UserId)
}

func (suite *AuthServiceTestSuite) TestValidateToken_InvalidToken() {
	req := &authpb.ValidateTokenRequest{
		Token: "invalid-token",
	}

	resp, err := suite.service.ValidateToken(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().False(resp.IsValid)
	suite.Assert().Equal(int32(0), resp.UserId)
}

func (suite *AuthServiceTestSuite) TestValidateToken_ExpiredToken() {
	// Create test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Generate expired access token
	claims := map[string]interface{}{
		"user_id": string(rune(testUser.ID)),
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Expired
		"type":    "access",
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, err := expiredToken.SignedString([]byte("test-secret"))
	suite.Require().NoError(err)

	req := &authpb.ValidateTokenRequest{
		Token: expiredTokenString,
	}

	resp, err := suite.service.ValidateToken(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().False(resp.IsValid)
	suite.Assert().Equal(int32(0), resp.UserId)
}

func (suite *AuthServiceTestSuite) TestLogout_Success() {
	// Create test user and simulate login
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Set user in context (simulate authenticated request)
	ctx := context.WithValue(suite.ctx, "user_id", testUser.ID)

	resp, err := suite.service.Logout(ctx, &emptypb.Empty{})

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().True(resp.Success)
}

func (suite *AuthServiceTestSuite) TestLogout_UnauthenticatedUser() {
	resp, err := suite.service.Logout(suite.ctx, &emptypb.Empty{})

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.Unauthenticated, st.Code())
}

func (suite *AuthServiceTestSuite) TestConcurrentLogins() {
	// Create test user
	_, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Perform concurrent logins
	concurrency := 10
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req := &authpb.LoginRequest{
				Email:    fixtures.TestUser1.Email,
				Password: fixtures.TestUser1.Password,
			}

			resp, err := suite.service.Login(suite.ctx, req)
			if err != nil {
				results <- err
				return
			}

			// Verify response
			if resp == nil || resp.AccessToken == "" || resp.RefreshToken == "" {
				results <- assert.AnError
				return
			}

			results <- nil
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		suite.Assert().NoError(err)
	}
}

func (suite *AuthServiceTestSuite) TestRedisFailover() {
	// Create test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	// Temporarily disable Redis to simulate failure
	originalRedis := suite.service.redis
	suite.service.redis = nil

	req := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: fixtures.TestUser1.Password,
	}

	// Login should still work without Redis
	resp, err := suite.service.Login(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().NotEmpty(resp.AccessToken)
	suite.Assert().NotEmpty(resp.RefreshToken)

	// Restore Redis
	suite.service.redis = originalRedis
}

// Test with SQLite for faster unit tests
func TestAuthService_Unit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	uc := useruc.NewUserUsecase(client)
	service := NewAuthService(client, uc)
	ctx := context.Background()

	t.Run("login with valid credentials", func(t *testing.T) {
		// Create test user
		_, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
		require.NoError(t, err)

		req := &authpb.LoginRequest{
			Email:    fixtures.TestUser1.Email,
			Password: fixtures.TestUser1.Password,
		}

		resp, err := service.Login(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})
}

// Run integration test suite
func TestAuthServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(AuthServiceTestSuite))
}

// Benchmarks
func BenchmarkAuthService_Login(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	uc := useruc.NewUserUsecase(client)
	service := NewAuthService(client, uc)
	ctx := context.Background()

	// Create test user
	_, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	require.NoError(b, err)

	req := &authpb.LoginRequest{
		Email:    fixtures.TestUser1.Email,
		Password: fixtures.TestUser1.Password,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Login(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuthService_ValidateToken(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	uc := useruc.NewUserUsecase(client)
	service := NewAuthService(client, uc)
	ctx := context.Background()

	// Create test user and token
	testUser, err := fixtures.CreateTestUser(ctx, client, fixtures.TestUser1)
	require.NoError(b, err)

	accessToken, err := fixtures.GenerateTestJWT(testUser.ID)
	require.NoError(b, err)

	req := &authpb.ValidateTokenRequest{
		Token: accessToken,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ValidateToken(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
