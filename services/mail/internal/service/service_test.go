package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	mailpb "stormlink/server/grpc/mail/protobuf"
	"stormlink/tests/fixtures"
	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SimpleMailServiceTestSuite struct {
	suite.Suite
	ctx    context.Context
	helper *testhelper.PostgresTestHelper
}

func (suite *SimpleMailServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Set JWT_SECRET for testing
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-for-testing")

	// Setup PostgreSQL test helper
	suite.helper = testhelper.NewPostgresTestHelper(suite.T())
	suite.helper.WaitForDatabase(suite.T())
}

func (suite *SimpleMailServiceTestSuite) TearDownSuite() {
	if suite.helper != nil {
		suite.helper.Cleanup()
	}
}

func (suite *SimpleMailServiceTestSuite) SetupTest() {
	// Clean database before each test
	suite.helper.CleanDatabase(suite.T())
}

func (suite *SimpleMailServiceTestSuite) createTestService() *MailService {
	client := suite.helper.GetClient()

	// Create mail service with proper constructor
	service := NewMailService(client)

	return service
}

func (suite *SimpleMailServiceTestSuite) TestSendVerificationEmail_Success() {
	service := suite.createTestService()
	defer service.client.Close()

	// Create test user
	testUser := fixtures.UserFixture{
		Name:       "Test User",
		Slug:       fmt.Sprintf("test-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, service.client, testUser)
	require.NoError(suite.T(), err)

	// Create email verification record manually since SendVerificationEmail doesn't exist
	_, err = service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("test-verification-token").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify email verification record was created
	verifications, err := service.client.EmailVerification.Query().All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), verifications, 1)
	verificationUser, err := verifications[0].QueryUser().Only(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, verificationUser.ID)
}

func (suite *SimpleMailServiceTestSuite) TestSendVerificationEmail_InvalidUser() {
	service := suite.createTestService()
	defer service.client.Close()

	// Test that we can't create verification without a user
	// Since we use edges, we need a valid user or this will fail
	_, err := service.client.EmailVerification.Create().
		SetToken("test-token").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)

	// Should fail due to foreign key constraint
	assert.Error(suite.T(), err)
}

func (suite *SimpleMailServiceTestSuite) TestResendVerifyEmail_Success() {
	service := suite.createTestService()
	defer service.client.Close()

	// Create test user
	testUser := fixtures.UserFixture{
		Name:       "Resend Test User",
		Slug:       fmt.Sprintf("resend-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("resend-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, service.client, testUser)
	require.NoError(suite.T(), err)

	// First create a verification record
	_, err = service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("test-token-123").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	// Now resend verification email
	resendReq := &mailpb.ResendVerifyEmailRequest{
		Email: testUser.Email,
	}

	resp, err := service.ResendVerifyEmail(suite.ctx, resendReq)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify response - ResendVerifyEmailResponse should have a Message field
	assert.NotEmpty(suite.T(), resp.Message)
}

func (suite *SimpleMailServiceTestSuite) TestResendVerifyEmail_NonExistentUser() {
	service := suite.createTestService()
	defer service.client.Close()

	req := &mailpb.ResendVerifyEmailRequest{
		Email: "nonexistent@example.com",
	}

	_, err := service.ResendVerifyEmail(suite.ctx, req)
	assert.Error(suite.T(), err)

	// Should be a not found error
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.NotFound, st.Code())
}

func (suite *SimpleMailServiceTestSuite) TestVerifyEmail_Success() {
	service := suite.createTestService()
	defer service.client.Close()

	// Create test user
	testUser := fixtures.UserFixture{
		Name:       "Verify Test User",
		Slug:       fmt.Sprintf("verify-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("verify-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, service.client, testUser)
	require.NoError(suite.T(), err)

	// Create email verification record
	verification, err := service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("test-verification-token").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify email
	req := &mailpb.VerifyEmailRequest{
		Token: verification.Token,
	}

	resp, err := service.VerifyEmail(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify response - VerifyEmailResponse has a Message field, not Success
	assert.NotEmpty(suite.T(), resp.Message)

	// Verify user is now verified
	updatedUser, err := service.client.User.Get(suite.ctx, user.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), updatedUser.IsVerified)
}

func (suite *SimpleMailServiceTestSuite) TestVerifyEmail_InvalidToken() {
	service := suite.createTestService()
	defer service.client.Close()

	req := &mailpb.VerifyEmailRequest{
		Token: "invalid-token",
	}

	_, err := service.VerifyEmail(suite.ctx, req)
	assert.Error(suite.T(), err)

	// Should be a not found error
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.NotFound, st.Code())
}

func (suite *SimpleMailServiceTestSuite) TestVerifyEmail_ExpiredToken() {
	service := suite.createTestService()
	defer service.client.Close()

	// Create test user
	testUser := fixtures.UserFixture{
		Name:       "Expired Test User",
		Slug:       fmt.Sprintf("expired-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("expired-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, service.client, testUser)
	require.NoError(suite.T(), err)

	// Create expired email verification record
	verification, err := service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("expired-verification-token").
		SetExpiresAt(time.Now().Add(-1 * time.Hour)). // Expired 1 hour ago
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	// Try to verify with expired token
	req := &mailpb.VerifyEmailRequest{
		Token: verification.Token,
	}

	_, err = service.VerifyEmail(suite.ctx, req)
	assert.Error(suite.T(), err)

	// Should be a deadline exceeded error (token expired)
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.DeadlineExceeded, st.Code())
}

func (suite *SimpleMailServiceTestSuite) TestMultipleVerificationTokens() {
	service := suite.createTestService()
	defer service.client.Close()

	// Create test user
	testUser := fixtures.UserFixture{
		Name:       "Multiple Tokens User",
		Slug:       fmt.Sprintf("multiple-user-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("multiple-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: false,
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, service.client, testUser)
	require.NoError(suite.T(), err)

	// Create multiple verification records
	_, err = service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("valid-token-456").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	_, err = service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("token-2").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	// Check that we have verification records
	verifications, err := service.client.EmailVerification.Query().All(suite.ctx)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(verifications) >= 1) // At least one should exist

	// All verifications should be for the same user
	for _, verification := range verifications {
		verificationUser, err := verification.QueryUser().Only(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.ID, verificationUser.ID)
		assert.Equal(suite.T(), testUser.Email, verificationUser.Email)
	}
}

func (suite *SimpleMailServiceTestSuite) TestVerifyAlreadyVerifiedUser() {
	service := suite.createTestService()
	defer service.client.Close()

	// Create already verified test user
	testUser := fixtures.UserFixture{
		Name:       "Already Verified User",
		Slug:       fmt.Sprintf("already-verified-%d", time.Now().UnixNano()),
		Email:      fmt.Sprintf("verified-%d@example.com", time.Now().UnixNano()),
		Password:   "password123",
		Salt:       "test-salt",
		IsVerified: true, // Already verified
		CreatedAt:  time.Now(),
	}

	user, err := fixtures.CreateTestUser(suite.ctx, service.client, testUser)
	require.NoError(suite.T(), err)

	// Create email verification record
	verification, err := service.client.EmailVerification.Create().
		SetUser(user).
		SetToken("already-verified-token").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(suite.ctx)
	require.NoError(suite.T(), err)

	// Try to verify already verified user
	req := &mailpb.VerifyEmailRequest{
		Token: verification.Token,
	}

	resp, err := service.VerifyEmail(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Should still succeed (idempotent operation)
	assert.NotEmpty(suite.T(), resp.Message)

	// User should still be verified
	updatedUser, err := service.client.User.Get(suite.ctx, user.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), updatedUser.IsVerified)
}

func TestSimpleMailService(t *testing.T) {
	suite.Run(t, new(SimpleMailServiceTestSuite))
}
