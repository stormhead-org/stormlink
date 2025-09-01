package service

import (
	"context"
	"testing"
	"time"

	"stormlink/server/ent/enttest"
	mailpb "stormlink/server/grpc/mail/protobuf"
	"stormlink/shared/jwt"
	"stormlink/tests/fixtures"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MailServiceTestSuite struct {
	suite.Suite
	containers *testcontainers.TestContainers
	service    *MailService
	ctx        context.Context
}

func (suite *MailServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create service
	suite.service = NewMailService(containers.EntClient)
}

func (suite *MailServiceTestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *MailServiceTestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *MailServiceTestSuite) TestVerifyEmail_Success() {
	// Create unverified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)
	suite.Assert().False(testUser.IsVerified)

	// Create email verification token
	token := "test-verification-token"
	_, err = fixtures.CreateTestEmailVerification(suite.ctx, suite.containers.EntClient, testUser.ID, token)
	suite.Require().NoError(err)

	req := &mailpb.VerifyEmailRequest{
		Token: token,
	}

	resp, err := suite.service.VerifyEmail(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().Contains(resp.Message, "успешно подтверждена")

	// Verify user is now verified
	updatedUser, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().True(updatedUser.IsVerified)

	// Verify verification token is deleted
	verifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.TokenEQ(token)).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Empty(verifications)
}

func (suite *MailServiceTestSuite) TestVerifyEmail_InvalidToken() {
	req := &mailpb.VerifyEmailRequest{
		Token: "invalid-token",
	}

	resp, err := suite.service.VerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.NotFound, st.Code())
	suite.Assert().Contains(st.Message(), "invalid or expired token")
}

func (suite *MailServiceTestSuite) TestVerifyEmail_EmptyToken() {
	req := &mailpb.VerifyEmailRequest{
		Token: "",
	}

	resp, err := suite.service.VerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
	suite.Assert().Contains(st.Message(), "token is required")
}

func (suite *MailServiceTestSuite) TestVerifyEmail_ExpiredToken() {
	// Create unverified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	// Create expired email verification token
	token := "expired-verification-token"
	expiredVerification, err := suite.containers.EntClient.EmailVerification.Create().
		SetUserID(testUser.ID).
		SetToken(token).
		SetExpiresAt(time.Now().Add(-1 * time.Hour)). // Expired
		SetCreatedAt(time.Now().Add(-2 * time.Hour)).
		Save(suite.ctx)
	suite.Require().NoError(err)

	req := &mailpb.VerifyEmailRequest{
		Token: token,
	}

	resp, err := suite.service.VerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.DeadlineExceeded, st.Code())
	suite.Assert().Contains(st.Message(), "verification token has expired")

	// Verify expired token is deleted
	verifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.IDEQ(expiredVerification.ID)).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Empty(verifications)

	// Verify user is still not verified
	updatedUser, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().False(updatedUser.IsVerified)
}

func (suite *MailServiceTestSuite) TestVerifyEmail_AlreadyVerifiedUser() {
	// Create verified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)
	suite.Assert().True(testUser.IsVerified)

	// Create verification token for already verified user
	token := "unnecessary-verification-token"
	_, err = fixtures.CreateTestEmailVerification(suite.ctx, suite.containers.EntClient, testUser.ID, token)
	suite.Require().NoError(err)

	req := &mailpb.VerifyEmailRequest{
		Token: token,
	}

	resp, err := suite.service.VerifyEmail(suite.ctx, req)

	// Should still succeed (idempotent operation)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().Contains(resp.Message, "успешно подтверждена")

	// Verify user is still verified
	updatedUser, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().True(updatedUser.IsVerified)
}

func (suite *MailServiceTestSuite) TestResendVerifyEmail_Success() {
	// Create unverified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	req := &mailpb.ResendVerifyEmailRequest{
		Email: fixtures.UnverifiedUser.Email,
	}

	resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().Contains(resp.Message, "Verification email sent successfully")

	// Verify new verification token was created
	verifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.HasUserWith(user.EmailEQ(fixtures.UnverifiedUser.Email))).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(verifications, 1)

	// Verify token properties
	verification := verifications[0]
	suite.Assert().NotEmpty(verification.Token)
	suite.Assert().True(verification.ExpiresAt.After(time.Now()))
	suite.Assert().True(verification.ExpiresAt.Before(time.Now().Add(25 * time.Hour))) // Should be ~24 hours
}

func (suite *MailServiceTestSuite) TestResendVerifyEmail_UserNotFound() {
	req := &mailpb.ResendVerifyEmailRequest{
		Email: "nonexistent@example.com",
	}

	resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.NotFound, st.Code())
	suite.Assert().Contains(st.Message(), "user not found")
}

func (suite *MailServiceTestSuite) TestResendVerifyEmail_AlreadyVerified() {
	// Create verified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.TestUser1)
	suite.Require().NoError(err)

	req := &mailpb.ResendVerifyEmailRequest{
		Email: fixtures.TestUser1.Email,
	}

	resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.FailedPrecondition, st.Code())
	suite.Assert().Contains(st.Message(), "user already verified")
}

func (suite *MailServiceTestSuite) TestResendVerifyEmail_InvalidEmail() {
	req := &mailpb.ResendVerifyEmailRequest{
		Email: "invalid-email",
	}

	resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
	suite.Assert().Contains(st.Message(), "validation error")
}

func (suite *MailServiceTestSuite) TestResendVerifyEmail_EmptyEmail() {
	req := &mailpb.ResendVerifyEmailRequest{
		Email: "",
	}

	resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
}

func (suite *MailServiceTestSuite) TestResendVerifyEmail_ClearsOldTokens() {
	// Create unverified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	// Create old verification tokens
	oldToken1, err := fixtures.CreateTestEmailVerification(suite.ctx, suite.containers.EntClient, testUser.ID, "old-token-1")
	suite.Require().NoError(err)

	oldToken2, err := fixtures.CreateTestEmailVerification(suite.ctx, suite.containers.EntClient, testUser.ID, "old-token-2")
	suite.Require().NoError(err)

	req := &mailpb.ResendVerifyEmailRequest{
		Email: fixtures.UnverifiedUser.Email,
	}

	resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)

	// Verify old tokens are deleted
	oldVerifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.Or(
			emailverification.IDEQ(oldToken1.ID),
			emailverification.IDEQ(oldToken2.ID),
		)).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Empty(oldVerifications)

	// Verify new token exists
	newVerifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.HasUserWith(user.EmailEQ(fixtures.UnverifiedUser.Email))).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(newVerifications, 1)
	suite.Assert().NotEqual("old-token-1", newVerifications[0].Token)
	suite.Assert().NotEqual("old-token-2", newVerifications[0].Token)
}

func (suite *MailServiceTestSuite) TestVerifyEmail_WithUserEdge() {
	// Create unverified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	// Create email verification with user edge
	token := "token-with-user-edge"
	verification, err := suite.containers.EntClient.EmailVerification.Create().
		SetUserID(testUser.ID).
		SetToken(token).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		SetCreatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	req := &mailpb.VerifyEmailRequest{
		Token: token,
	}

	resp, err := suite.service.VerifyEmail(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)

	// Verify user is now verified
	updatedUser, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().True(updatedUser.IsVerified)

	// Verify verification record is deleted
	_, err = suite.containers.EntClient.EmailVerification.Get(suite.ctx, verification.ID)
	suite.Assert().Error(err)
	suite.Assert().True(suite.containers.EntClient.IsNotFound(err))
}

func (suite *MailServiceTestSuite) TestTokenGeneration_Uniqueness() {
	// Create multiple users
	users := []fixtures.UserFixture{
		{
			Name:       "User 1",
			Slug:       "user-1",
			Email:      "user1@test.com",
			Password:   "password",
			Salt:       "salt1",
			IsVerified: false,
			CreatedAt:  time.Now(),
		},
		{
			Name:       "User 2",
			Slug:       "user-2",
			Email:      "user2@test.com",
			Password:   "password",
			Salt:       "salt2",
			IsVerified: false,
			CreatedAt:  time.Now(),
		},
		{
			Name:       "User 3",
			Slug:       "user-3",
			Email:      "user3@test.com",
			Password:   "password",
			Salt:       "salt3",
			IsVerified: false,
			CreatedAt:  time.Now(),
		},
	}

	var createdUsers []*ent.User
	for _, userFixture := range users {
		user, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, userFixture)
		suite.Require().NoError(err)
		createdUsers = append(createdUsers, user)
	}

	// Generate verification tokens for all users
	var tokens []string
	for _, user := range createdUsers {
		req := &mailpb.ResendVerifyEmailRequest{
			Email: user.Email,
		}

		resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp)

		// Get the generated token
		verifications, err := suite.containers.EntClient.EmailVerification.Query().
			Where(emailverification.HasUserWith(user.EmailEQ(user.Email))).
			All(suite.ctx)
		suite.Assert().NoError(err)
		suite.Assert().Len(verifications, 1)

		tokens = append(tokens, verifications[0].Token)
	}

	// Verify all tokens are unique
	tokenSet := make(map[string]bool)
	for _, token := range tokens {
		suite.Assert().False(tokenSet[token], "Token should be unique: %s", token)
		tokenSet[token] = true
		suite.Assert().NotEmpty(token)
		suite.Assert().GreaterOrEqual(len(token), 16) // Should have reasonable length
	}
}

func (suite *MailServiceTestSuite) TestConcurrentVerification() {
	// Test concurrent verification attempts
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	// Create verification token
	token := "concurrent-test-token"
	_, err = fixtures.CreateTestEmailVerification(suite.ctx, suite.containers.EntClient, testUser.ID, token)
	suite.Require().NoError(err)

	concurrency := 5
	results := make(chan error, concurrency)

	// Attempt concurrent verifications with same token
	for i := 0; i < concurrency; i++ {
		go func() {
			req := &mailpb.VerifyEmailRequest{
				Token: token,
			}

			resp, err := suite.service.VerifyEmail(suite.ctx, req)
			if err != nil {
				results <- err
				return
			}

			if resp == nil {
				results <- assert.AnError
				return
			}

			results <- nil
		}()
	}

	// Wait for all goroutines to complete
	successCount := 0
	errorCount := 0
	for i := 0; i < concurrency; i++ {
		err := <-results
		if err != nil {
			errorCount++
			// Errors should be "not found" after first successful verification
			st, ok := status.FromError(err)
			if ok {
				suite.Assert().Equal(codes.NotFound, st.Code())
			}
		} else {
			successCount++
		}
	}

	// Only one should succeed, others should fail with "not found"
	suite.Assert().Equal(1, successCount, "Only one verification should succeed")
	suite.Assert().Equal(concurrency-1, errorCount, "Other attempts should fail")

	// Verify user is verified
	updatedUser, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
	suite.Assert().NoError(err)
	suite.Assert().True(updatedUser.IsVerified)
}

func (suite *MailServiceTestSuite) TestConcurrentResendRequests() {
	// Test concurrent resend requests for same user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	concurrency := 3
	results := make(chan error, concurrency)

	// Attempt concurrent resend requests
	for i := 0; i < concurrency; i++ {
		go func() {
			req := &mailpb.ResendVerifyEmailRequest{
				Email: fixtures.UnverifiedUser.Email,
			}

			resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)
			if err != nil {
				results <- err
				return
			}

			if resp == nil {
				results <- assert.AnError
				return
			}

			results <- nil
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		suite.Assert().NoError(err, "All resend requests should succeed")
	}

	// Verify only one verification token exists (last one wins)
	verifications, err := suite.containers.EntClient.EmailVerification.Query().
		Where(emailverification.HasUserWith(user.EmailEQ(fixtures.UnverifiedUser.Email))).
		All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(verifications, 1, "Should have exactly one verification token after concurrent resends")
}

func (suite *MailServiceTestSuite) TestEmailValidation() {
	// Test email validation in ResendVerifyEmail
	testCases := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"invalid email no @", "invalid-email", false},
		{"invalid email no domain", "user@", false},
		{"invalid email no user", "@example.com", false},
		{"invalid email multiple @", "user@@example.com", false},
		{"empty email", "", false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req := &mailpb.ResendVerifyEmailRequest{
				Email: tc.email,
			}

			resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)

			if tc.valid {
				// For valid emails, we expect "user not found" error since we didn't create users
				suite.Assert().Error(err)
				st, ok := status.FromError(err)
				suite.Assert().True(ok)
				suite.Assert().Equal(codes.NotFound, st.Code())
			} else {
				// For invalid emails, we expect validation error
				suite.Assert().Error(err)
				suite.Assert().Nil(resp)
				st, ok := status.FromError(err)
				suite.Assert().True(ok)
				suite.Assert().Equal(codes.InvalidArgument, st.Code())
			}
		})
	}
}

func (suite *MailServiceTestSuite) TestTokenExpiry_EdgeCases() {
	// Create unverified test user
	testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
	suite.Require().NoError(err)

	suite.Run("token expires exactly at verification time", func() {
		// Create token that expires very soon
		token := "about-to-expire-token"
		verification, err := suite.containers.EntClient.EmailVerification.Create().
			SetUserID(testUser.ID).
			SetToken(token).
			SetExpiresAt(time.Now().Add(1 * time.Millisecond)). // Expires very soon
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Wait for token to expire
		time.Sleep(5 * time.Millisecond)

		req := &mailpb.VerifyEmailRequest{
			Token: token,
		}

		resp, err := suite.service.VerifyEmail(suite.ctx, req)

		suite.Assert().Error(err)
		suite.Assert().Nil(resp)

		st, ok := status.FromError(err)
		suite.Assert().True(ok)
		suite.Assert().Equal(codes.DeadlineExceeded, st.Code())

		// Verify token is deleted
		_, err = suite.containers.EntClient.EmailVerification.Get(suite.ctx, verification.ID)
		suite.Assert().Error(err)
		suite.Assert().True(suite.containers.EntClient.IsNotFound(err))
	})

	suite.Run("resend creates token with correct expiry", func() {
		req := &mailpb.ResendVerifyEmailRequest{
			Email: fixtures.UnverifiedUser.Email,
		}

		beforeSend := time.Now()
		resp, err := suite.service.ResendVerifyEmail(suite.ctx, req)
		afterSend := time.Now()

		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp)

		// Verify token expiry is approximately 24 hours from now
		verifications, err := suite.containers.EntClient.EmailVerification.Query().
			Where(emailverification.HasUserWith(user.EmailEQ(fixtures.UnverifiedUser.Email))).
			All(suite.ctx)
		suite.Assert().NoError(err)
		suite.Assert().Len(verifications, 1)

		verification := verifications[0]
		expectedMinExpiry := beforeSend.Add(23*time.Hour + 59*time.Minute) // ~24h - 1min
		expectedMaxExpiry := afterSend.Add(24*time.Hour + 1*time.Minute)   // ~24h + 1min

		suite.Assert().True(verification.ExpiresAt.After(expectedMinExpiry))
		suite.Assert().True(verification.ExpiresAt.Before(expectedMaxExpiry))
	})
}

func (suite *MailServiceTestSuite) TestDatabaseIntegrity() {
	// Test database integrity during mail operations

	suite.Run("verify email maintains data integrity", func() {
		// Create unverified user
		testUser, err := fixtures.CreateTestUser(suite.ctx, suite.containers.EntClient, fixtures.UnverifiedUser)
		suite.Require().NoError(err)

		// Create verification token
		token := "integrity-test-token"
		verification, err := fixtures.CreateTestEmailVerification(suite.ctx, suite.containers.EntClient, testUser.ID, token)
		suite.Require().NoError(err)

		// Verify before operation
		userBefore, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
		suite.Require().NoError(err)
		suite.Assert().False(userBefore.IsVerified)

		// Perform verification
		req := &mailpb.VerifyEmailRequest{Token: token}
		resp, err := suite.service.VerifyEmail(suite.ctx, req)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp)

		// Verify after operation
		userAfter, err := suite.containers.EntClient.User.Get(suite.ctx, testUser.ID)
		suite.Assert().NoError(err)
		suite.Assert().True(userAfter.IsVerified)

		// Verify token is deleted
		_, err = suite.containers.EntClient.EmailVerification.Get(suite.ctx, verification.ID)
		suite.Assert().Error(err)
		suite.Assert().True(suite.containers.EntClient.IsNotFound(err))

		// Verify other user data is unchanged
		suite.Assert().Equal(userBefore.Name, userAfter.Name)
		suite.Assert().Equal(userBefore.Email, userAfter.Email)
		suite.Assert().Equal(userBefore.Slug, userAfter.Slug)
		suite.Assert().Equal(userBefore.CreatedAt, userAfter.CreatedAt)
	})
}

func (suite *MailServiceTestSuite) TestErrorRecovery() {
	// Test error recovery scenarios

	suite.Run("verify email with missing user edge", func() {
		// Create verification token without proper user relationship
		token := "orphaned-token"
		_, err := suite.containers.EntClient.EmailVerification.Create().
			SetUserID(99999). // Non-existent user
			SetToken(token).
			SetExpiresAt(time.Now().Add(24 * time.Hour)).
			SetCreatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		req := &mailpb.VerifyEmailRequest{Token: token}
		resp, err := suite.service.VerifyEmail(suite.ctx, req)

		suite.Assert().Error(err)
		suite.Assert().Nil(resp)

		st, ok := status.FromError(err)
		suite.Assert().True(ok)
		suite.Assert().Equal(codes.NotFound, st.Code())
	})
}

func (suite *MailServiceTestSuite) TestContextCancellation() {
	// Test behavior with cancelled context

	suite.Run("verify email with cancelled context", func() {
		cancelledCtx, cancel := context.WithCancel(suite.ctx)
		cancel()

		req := &mailpb.VerifyEmailRequest{
			Token: "test-token",
		}

		resp, err := suite.service.VerifyEmail(cancelledCtx, req)

		suite.Assert().Error(err)
		suite.Assert().Nil(resp)
		suite.Assert().Contains(err.Error(), "context canceled")
	})

	suite.Run("resend verify email with cancelled context", func() {
		cancelledCtx, cancel := context.WithCancel(suite.ctx)
		cancel()

		req := &mailpb.ResendVerifyEmailRequest{
			Email: "test@example.com",
		}

		resp, err := suite.service.ResendVerifyEmail(cancelledCtx, req)

		suite.Assert().Error(err)
		suite.Assert().Nil(resp)
		suite.Assert().Contains(err.Error(), "context canceled")
	})
}

// Test with SQLite for faster unit tests
func TestMailService_Unit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	service := NewMailService(client)
	ctx := context.Background()

	t.Run("verify email with valid token", func(t *testing.T) {
		// Create test user
		testUser, err := fixtures.CreateTestUser(ctx, client, fixtures.UnverifiedUser)
		require.NoError(t, err)

		// Create verification token
		token := "unit-test-token"
		_, err = fixtures.CreateTestEmailVerification(ctx, client, testUser.ID, token)
		require.NoError(t, err)

		req := &mailpb.VerifyEmailRequest{Token: token}
		resp, err := service.VerifyEmail(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "успешно подтверждена")

		// Verify user is now verified
		updatedUser, err := client.User.Get(ctx, testUser.ID)
		assert.NoError(t, err)
		assert.True(t, updatedUser.IsVerified)
	})

	t.Run("resend verify email for unverified user", func(t *testing.T) {
		// Create test user
		testUser, err := fixtures.CreateTestUser(ctx, client, fixtures.UserFixture{
			Name:       "Unit Test User",
			Slug:       "unit-test-user",
			Email:      "unittest@example.com",
			Password:   "password",
			Salt:       "salt",
			IsVerified: false,
			CreatedAt:  time.Now(),
		})
		require.NoError(t, err)

		req := &mailpb.ResendVerifyEmailRequest{
			Email: testUser.Email,
		}

		resp, err := service.ResendVerify
