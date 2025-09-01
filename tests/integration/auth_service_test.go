package integration

import (
	"context"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	authpb "stormlink/server/grpc/auth/protobuf"
	"stormlink/server/usecase/user"
	"stormlink/services/auth/internal/service"
	"stormlink/shared/jwt"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServiceIntegrationTestSuite struct {
	suite.Suite
	containers  *testcontainers.TestContainers
	client      *ent.Client
	authService *service.AuthService
	userUC      user.UserUsecase
	ctx         context.Context
}

func (suite *AuthServiceIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers
	containers, err := testcontainers.Setup(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Setup database client
	suite.client = enttest.Open(suite.T(), "postgres", containers.PostgresDSN())

	// Setup user usecase
	suite.userUC = user.NewUserUsecase(suite.client)

	// Setup auth service
	suite.authService = service.NewAuthService(suite.client, suite.userUC)
}

func (suite *AuthServiceIntegrationTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
	if suite.containers != nil {
		suite.containers.Cleanup()
	}
}

func (suite *AuthServiceIntegrationTestSuite) SetupTest() {
	// Clean up data before each test
	suite.client.EmailVerification.Delete().ExecX(suite.ctx)
	suite.client.RefreshToken.Delete().ExecX(suite.ctx)
	suite.client.User.Delete().ExecX(suite.ctx)
}

func (suite *AuthServiceIntegrationTestSuite) TestLogin() {
	// Create test user with properly hashed password
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)

	testUser.PasswordHash = hashedPassword
	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("successful login with valid credentials", func() {
		req := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		resp, err := suite.authService.Login(suite.ctx, req)

		suite.NoError(err)
		suite.NotNil(resp)
		suite.NotEmpty(resp.AccessToken)
		suite.NotEmpty(resp.RefreshToken)
		suite.Equal(user.ID, int(resp.User.Id))
		suite.Equal(user.Email, resp.User.Email)
		suite.Equal(user.Name, resp.User.Name)

		// Verify token is valid
		userID, err := jwt.ParseAccessToken(resp.AccessToken)
		suite.NoError(err)
		suite.Equal(user.ID, userID)
	})

	suite.Run("login with invalid email", func() {
		req := &authpb.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: testUser.Password,
		}

		resp, err := suite.authService.Login(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.NotFound, grpcErr.Code())
		suite.Contains(grpcErr.Message(), "user not found")
	})

	suite.Run("login with invalid password", func() {
		req := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: "wrongpassword",
		}

		resp, err := suite.authService.Login(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
		suite.Contains(grpcErr.Message(), "invalid credentials")
	})

	suite.Run("login with unverified user", func() {
		// Create unverified user
		unverifiedUser := fixtures.UnverifiedUser
		unverifiedUser.PasswordHash, err = jwt.HashPassword(unverifiedUser.Password, unverifiedUser.Salt)
		suite.Require().NoError(err)

		_, err := fixtures.CreateTestUser(suite.ctx, suite.client, unverifiedUser)
		suite.Require().NoError(err)

		req := &authpb.LoginRequest{
			Email:    unverifiedUser.Email,
			Password: unverifiedUser.Password,
		}

		resp, err := suite.authService.Login(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.PermissionDenied, grpcErr.Code())
		suite.Contains(grpcErr.Message(), "email not verified")
	})

	suite.Run("login with empty email", func() {
		req := &authpb.LoginRequest{
			Email:    "",
			Password: testUser.Password,
		}

		resp, err := suite.authService.Login(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.InvalidArgument, grpcErr.Code())
	})

	suite.Run("login with empty password", func() {
		req := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: "",
		}

		resp, err := suite.authService.Login(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.InvalidArgument, grpcErr.Code())
	})
}

func (suite *AuthServiceIntegrationTestSuite) TestRefreshToken() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("successful token refresh", func() {
		// First, login to get tokens
		loginReq := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		loginResp, err := suite.authService.Login(suite.ctx, loginReq)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(loginResp.RefreshToken)

		// Wait a moment to ensure different timestamps
		time.Sleep(time.Millisecond * 100)

		// Now refresh the token
		refreshReq := &authpb.RefreshTokenRequest{
			RefreshToken: loginResp.RefreshToken,
		}

		refreshResp, err := suite.authService.RefreshToken(suite.ctx, refreshReq)

		suite.NoError(err)
		suite.NotNil(refreshResp)
		suite.NotEmpty(refreshResp.AccessToken)
		suite.NotEmpty(refreshResp.RefreshToken)

		// New tokens should be different
		suite.NotEqual(loginResp.AccessToken, refreshResp.AccessToken)
		suite.NotEqual(loginResp.RefreshToken, refreshResp.RefreshToken)

		// Verify new access token is valid
		userID, err := jwt.ParseAccessToken(refreshResp.AccessToken)
		suite.NoError(err)
		suite.Equal(user.ID, userID)

		// Old refresh token should be invalid now
		oldRefreshReq := &authpb.RefreshTokenRequest{
			RefreshToken: loginResp.RefreshToken,
		}

		_, err = suite.authService.RefreshToken(suite.ctx, oldRefreshReq)
		suite.Error(err)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})

	suite.Run("refresh with invalid token", func() {
		req := &authpb.RefreshTokenRequest{
			RefreshToken: "invalid-refresh-token",
		}

		resp, err := suite.authService.RefreshToken(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
		suite.Contains(grpcErr.Message(), "invalid refresh token")
	})

	suite.Run("refresh with empty token", func() {
		req := &authpb.RefreshTokenRequest{
			RefreshToken: "",
		}

		resp, err := suite.authService.RefreshToken(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.InvalidArgument, grpcErr.Code())
	})

	suite.Run("refresh with expired token", func() {
		// Create an expired refresh token manually
		expiredToken, err := jwt.GenerateRefreshTokenWithExpiry(user.ID, time.Now().Add(-time.Hour))
		suite.Require().NoError(err)

		req := &authpb.RefreshTokenRequest{
			RefreshToken: expiredToken,
		}

		resp, err := suite.authService.RefreshToken(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
		suite.Contains(grpcErr.Message(), "token expired")
	})
}

func (suite *AuthServiceIntegrationTestSuite) TestValidateToken() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("validate valid access token", func() {
		// Generate valid token
		accessToken, err := jwt.GenerateAccessToken(user.ID)
		suite.Require().NoError(err)

		req := &authpb.ValidateTokenRequest{
			AccessToken: accessToken,
		}

		resp, err := suite.authService.ValidateToken(suite.ctx, req)

		suite.NoError(err)
		suite.NotNil(resp)
		suite.True(resp.Valid)
		suite.Equal(user.ID, int(resp.UserId))
		suite.NotNil(resp.User)
		suite.Equal(user.Email, resp.User.Email)
		suite.Equal(user.Name, resp.User.Name)
	})

	suite.Run("validate invalid access token", func() {
		req := &authpb.ValidateTokenRequest{
			AccessToken: "invalid-access-token",
		}

		resp, err := suite.authService.ValidateToken(suite.ctx, req)

		suite.NoError(err)
		suite.NotNil(resp)
		suite.False(resp.Valid)
		suite.Equal(int32(0), resp.UserId)
		suite.Nil(resp.User)
	})

	suite.Run("validate empty access token", func() {
		req := &authpb.ValidateTokenRequest{
			AccessToken: "",
		}

		resp, err := suite.authService.ValidateToken(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.InvalidArgument, grpcErr.Code())
	})

	suite.Run("validate token for deleted user", func() {
		// Create another user and generate token
		deletedUser := fixtures.TestUser2
		deletedUser.PasswordHash, err = jwt.HashPassword(deletedUser.Password, deletedUser.Salt)
		suite.Require().NoError(err)

		createdUser, err := fixtures.CreateTestUser(suite.ctx, suite.client, deletedUser)
		suite.Require().NoError(err)

		// Generate token
		accessToken, err := jwt.GenerateAccessToken(createdUser.ID)
		suite.Require().NoError(err)

		// Delete the user
		err = suite.client.User.DeleteOneID(createdUser.ID).Exec(suite.ctx)
		suite.Require().NoError(err)

		req := &authpb.ValidateTokenRequest{
			AccessToken: accessToken,
		}

		resp, err := suite.authService.ValidateToken(suite.ctx, req)

		suite.NoError(err)
		suite.NotNil(resp)
		suite.False(resp.Valid)
		suite.Equal(int32(0), resp.UserId)
		suite.Nil(resp.User)
	})

	suite.Run("validate expired access token", func() {
		// Create an expired access token manually
		expiredToken, err := jwt.GenerateAccessTokenWithExpiry(user.ID, time.Now().Add(-time.Hour))
		suite.Require().NoError(err)

		req := &authpb.ValidateTokenRequest{
			AccessToken: expiredToken,
		}

		resp, err := suite.authService.ValidateToken(suite.ctx, req)

		suite.NoError(err)
		suite.NotNil(resp)
		suite.False(resp.Valid)
		suite.Equal(int32(0), resp.UserId)
		suite.Nil(resp.User)
	})
}

func (suite *AuthServiceIntegrationTestSuite) TestLogout() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("successful logout", func() {
		// First login to get refresh token
		loginReq := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		loginResp, err := suite.authService.Login(suite.ctx, loginReq)
		suite.Require().NoError(err)

		// Logout
		logoutReq := &authpb.LogoutRequest{
			RefreshToken: loginResp.RefreshToken,
		}

		resp, err := suite.authService.Logout(suite.ctx, logoutReq)

		suite.NoError(err)
		suite.NotNil(resp)

		// Try to use the refresh token again - should fail
		refreshReq := &authpb.RefreshTokenRequest{
			RefreshToken: loginResp.RefreshToken,
		}

		_, err = suite.authService.RefreshToken(suite.ctx, refreshReq)
		suite.Error(err)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})

	suite.Run("logout with invalid refresh token", func() {
		req := &authpb.LogoutRequest{
			RefreshToken: "invalid-refresh-token",
		}

		resp, err := suite.authService.Logout(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})

	suite.Run("logout with empty refresh token", func() {
		req := &authpb.LogoutRequest{
			RefreshToken: "",
		}

		resp, err := suite.authService.Logout(suite.ctx, req)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.InvalidArgument, grpcErr.Code())
	})

	suite.Run("logout twice with same token", func() {
		// Login first
		loginReq := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		loginResp, err := suite.authService.Login(suite.ctx, loginReq)
		suite.Require().NoError(err)

		// First logout
		logoutReq := &authpb.LogoutRequest{
			RefreshToken: loginResp.RefreshToken,
		}

		_, err = suite.authService.Logout(suite.ctx, logoutReq)
		suite.Require().NoError(err)

		// Second logout with same token should fail
		_, err = suite.authService.Logout(suite.ctx, logoutReq)
		suite.Error(err)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})
}

func (suite *AuthServiceIntegrationTestSuite) TestConcurrentOperations() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	_, err = fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("concurrent login attempts", func() {
		done := make(chan bool, 5)
		errors := make(chan error, 5)
		responses := make(chan *authpb.LoginResponse, 5)

		// Start 5 concurrent login requests
		for i := 0; i < 5; i++ {
			go func() {
				defer func() { done <- true }()

				req := &authpb.LoginRequest{
					Email:    testUser.Email,
					Password: testUser.Password,
				}

				resp, err := suite.authService.Login(suite.ctx, req)
				if err != nil {
					errors <- err
					return
				}

				responses <- resp
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}

		// Check that no errors occurred
		select {
		case err := <-errors:
			suite.Fail("Concurrent login failed", err.Error())
		default:
			// No errors, test passed
		}

		// Verify we got 5 successful responses
		suite.Len(responses, 5)

		// Verify all tokens are different (no collision)
		tokens := make(map[string]bool)
		for i := 0; i < 5; i++ {
			resp := <-responses
			suite.False(tokens[resp.AccessToken], "Access tokens should be unique")
			suite.False(tokens[resp.RefreshToken], "Refresh tokens should be unique")
			tokens[resp.AccessToken] = true
			tokens[resp.RefreshToken] = true
		}
	})

	suite.Run("concurrent token validation", func() {
		// First get a valid token
		loginReq := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		loginResp, err := suite.authService.Login(suite.ctx, loginReq)
		suite.Require().NoError(err)

		done := make(chan bool, 10)
		errors := make(chan error, 10)

		// Start 10 concurrent validation requests
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				req := &authpb.ValidateTokenRequest{
					AccessToken: loginResp.AccessToken,
				}

				resp, err := suite.authService.ValidateToken(suite.ctx, req)
				if err != nil {
					errors <- err
					return
				}

				if !resp.Valid {
					errors <- suite.T().Errorf("token validation failed")
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
			suite.Fail("Concurrent validation failed", err.Error())
		default:
			// No errors, test passed
		}
	})
}

func (suite *AuthServiceIntegrationTestSuite) TestRateLimitingScenarios() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	_, err = fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("multiple failed login attempts", func() {
		// Make multiple failed login attempts
		for i := 0; i < 5; i++ {
			req := &authpb.LoginRequest{
				Email:    testUser.Email,
				Password: "wrongpassword",
			}

			resp, err := suite.authService.Login(suite.ctx, req)
			suite.Error(err)
			suite.Nil(resp)

			grpcErr, ok := status.FromError(err)
			suite.True(ok)
			suite.Equal(codes.Unauthenticated, grpcErr.Code())
		}

		// After multiple failed attempts, even correct password might be rate limited
		// This depends on implementation - this test documents the expected behavior
		correctReq := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		_, err := suite.authService.Login(suite.ctx, correctReq)
		// This might still succeed or might be rate limited depending on implementation
		// We just document the behavior here
		suite.T().Logf("Login after failed attempts result: %v", err)
	})
}

func (suite *AuthServiceIntegrationTestSuite) TestTokenLifecycle() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	suite.Run("complete token lifecycle", func() {
		// Step 1: Login
		loginReq := &authpb.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		loginResp, err := suite.authService.Login(suite.ctx, loginReq)
		suite.Require().NoError(err)
		suite.NotEmpty(loginResp.AccessToken)
		suite.NotEmpty(loginResp.RefreshToken)

		originalAccessToken := loginResp.AccessToken
		originalRefreshToken := loginResp.RefreshToken

		// Step 2: Validate access token
		validateReq := &authpb.ValidateTokenRequest{
			AccessToken: originalAccessToken,
		}

		validateResp, err := suite.authService.ValidateToken(suite.ctx, validateReq)
		suite.Require().NoError(err)
		suite.True(validateResp.Valid)
		suite.Equal(user.ID, int(validateResp.UserId))

		// Step 3: Refresh tokens
		refreshReq := &authpb.RefreshTokenRequest{
			RefreshToken: originalRefreshToken,
		}

		refreshResp, err := suite.authService.RefreshToken(suite.ctx, refreshReq)
		suite.Require().NoError(err)
		suite.NotEmpty(refreshResp.AccessToken)
		suite.NotEmpty(refreshResp.RefreshToken)

		newAccessToken := refreshResp.AccessToken
		newRefreshToken := refreshResp.RefreshToken

		// Verify new tokens are different
		suite.NotEqual(originalAccessToken, newAccessToken)
		suite.NotEqual(originalRefreshToken, newRefreshToken)

		// Step 4: Validate new access token
		validateNewReq := &authpb.ValidateTokenRequest{
			AccessToken: newAccessToken,
		}

		validateNewResp, err := suite.authService.ValidateToken(suite.ctx, validateNewReq)
		suite.Require().NoError(err)
		suite.True(validateNewResp.Valid)
		suite.Equal(user.ID, int(validateNewResp.UserId))

		// Step 5: Old tokens should be invalid
		validateOldReq := &authpb.ValidateTokenRequest{
			AccessToken: originalAccessToken,
		}

		validateOldResp, err := suite.authService.ValidateToken(suite.ctx, validateOldReq)
		suite.NoError(err) // ValidateToken doesn't return error for invalid tokens
		suite.False(validateOldResp.Valid)

		// Step 6: Logout with new refresh token
		logoutReq := &authpb.LogoutRequest{
			RefreshToken: newRefreshToken,
		}

		_, err = suite.authService.Logout(suite.ctx, logoutReq)
		suite.Require().NoError(err)

		// Step 7: All tokens should now be invalid
		finalValidateReq := &authpb.ValidateTokenRequest{
			AccessToken: newAccessToken,
		}

		finalValidateResp, err := suite.authService.ValidateToken(suite.ctx, finalValidateReq)
		suite.NoError(err)
		// After logout, access token might still be valid until expiry (stateless)
		// or might be invalid if using Redis blacklist
		suite.T().Logf("Access token valid after logout: %v", finalValidateResp.Valid)

		// Trying to refresh with logged out token should fail
		finalRefreshReq := &authpb.RefreshTokenRequest{
			RefreshToken: newRefreshToken,
		}

		_, err = suite.authService.RefreshToken(suite.ctx, finalRefreshReq)
		suite.Error(err)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})
}

func TestAuthServiceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceIntegrationTestSuite))
}
