package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/middleware"
	"stormlink/shared/jwt"
	"stormlink/tests/fixtures"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MiddlewareTestSuite struct {
	suite.Suite
	client *ent.Client
	ctx    context.Context
}

func (suite *MiddlewareTestSuite) SetupSuite() {
	suite.client = enttest.Open(suite.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	suite.ctx = context.Background()
}

func (suite *MiddlewareTestSuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *MiddlewareTestSuite) SetupTest() {
	// Clean up data before each test
	suite.client.User.Delete().ExecX(suite.ctx)
}

// HTTP Auth Middleware Tests
func (suite *MiddlewareTestSuite) TestHTTPAuthMiddleware() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	// Generate valid access token
	validToken, err := jwt.GenerateAccessToken(user.ID)
	suite.Require().NoError(err)

	suite.Run("valid bearer token", func() {
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, exists := r.Context().Value("userID").(int)
			suite.True(exists)
			suite.Equal(user.ID, userID)
			w.WriteHeader(http.StatusOK)
		})

		// Wrap with auth middleware
		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		// Create request with valid token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
	})

	suite.Run("invalid bearer token", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			suite.Fail("Handler should not be called with invalid token")
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusUnauthorized, w.Code)
	})

	suite.Run("missing authorization header", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Should be called but without userID in context
			userID, exists := r.Context().Value("userID").(int)
			suite.False(exists)
			suite.Equal(0, userID)
			w.WriteHeader(http.StatusOK)
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
	})

	suite.Run("malformed authorization header", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, exists := r.Context().Value("userID").(int)
			suite.False(exists)
			suite.Equal(0, userID)
			w.WriteHeader(http.StatusOK)
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
	})

	suite.Run("expired token", func() {
		// Generate expired token
		expiredToken, err := jwt.GenerateAccessTokenWithExpiry(user.ID, time.Now().Add(-time.Hour))
		suite.Require().NoError(err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			suite.Fail("Handler should not be called with expired token")
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusUnauthorized, w.Code)
	})

	suite.Run("token for deleted user", func() {
		// Create another user
		deletedUser := fixtures.TestUser2
		deletedUser.PasswordHash, err = jwt.HashPassword(deletedUser.Password, deletedUser.Salt)
		suite.Require().NoError(err)

		createdUser, err := fixtures.CreateTestUser(suite.ctx, suite.client, deletedUser)
		suite.Require().NoError(err)

		// Generate token for this user
		tokenForDeletedUser, err := jwt.GenerateAccessToken(createdUser.ID)
		suite.Require().NoError(err)

		// Delete the user
		err = suite.client.User.DeleteOneID(createdUser.ID).Exec(suite.ctx)
		suite.Require().NoError(err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			suite.Fail("Handler should not be called for deleted user")
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenForDeletedUser)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusUnauthorized, w.Code)
	})

	suite.Run("cookie-based authentication", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, exists := r.Context().Value("userID").(int)
			suite.True(exists)
			suite.Equal(user.ID, userID)
			w.WriteHeader(http.StatusOK)
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		cookie := &http.Cookie{
			Name:  "access_token",
			Value: validToken,
		}
		req.AddCookie(cookie)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
	})
}

// gRPC Auth Middleware Tests
func (suite *MiddlewareTestSuite) TestGRPCAuthMiddleware() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	// Generate valid access token
	validToken, err := jwt.GenerateAccessToken(user.ID)
	suite.Require().NoError(err)

	suite.Run("valid grpc auth with metadata", func() {
		// Create test handler
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, exists := ctx.Value("userID").(int)
			suite.True(exists)
			suite.Equal(user.ID, userID)
			return "success", nil
		}

		// Create auth interceptor
		interceptor := middleware.GRPCAuthInterceptor(suite.client)

		// Create context with auth metadata
		md := metadata.Pairs("authorization", "Bearer "+validToken)
		ctx := metadata.NewIncomingContext(suite.ctx, md)

		info := &grpc.UnaryServerInfo{
			Server:     nil,
			FullMethod: "/test.Service/TestMethod",
		}

		resp, err := interceptor(ctx, "test-request", info, testHandler)

		suite.NoError(err)
		suite.Equal("success", resp)
	})

	suite.Run("invalid grpc token", func() {
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			suite.Fail("Handler should not be called with invalid token")
			return nil, nil
		}

		interceptor := middleware.GRPCAuthInterceptor(suite.client)

		md := metadata.Pairs("authorization", "Bearer invalid-token")
		ctx := metadata.NewIncomingContext(suite.ctx, md)

		info := &grpc.UnaryServerInfo{
			Server:     nil,
			FullMethod: "/test.Service/TestMethod",
		}

		resp, err := interceptor(ctx, "test-request", info, testHandler)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})

	suite.Run("missing grpc auth metadata", func() {
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			// Should be called but without userID in context for public methods
			userID, exists := ctx.Value("userID").(int)
			suite.False(exists)
			suite.Equal(0, userID)
			return "success", nil
		}

		interceptor := middleware.GRPCAuthInterceptor(suite.client)

		// Context without metadata
		ctx := suite.ctx

		info := &grpc.UnaryServerInfo{
			Server:     nil,
			FullMethod: "/test.Service/PublicMethod",
		}

		resp, err := interceptor(ctx, "test-request", info, testHandler)

		// For public methods, should succeed without auth
		suite.NoError(err)
		suite.Equal("success", resp)
	})

	suite.Run("protected grpc method without auth", func() {
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			suite.Fail("Handler should not be called for protected method without auth")
			return nil, nil
		}

		interceptor := middleware.GRPCAuthInterceptor(suite.client)

		ctx := suite.ctx

		info := &grpc.UnaryServerInfo{
			Server:     nil,
			FullMethod: "/auth.AuthService/Login", // Protected method
		}

		resp, err := interceptor(ctx, "test-request", info, testHandler)

		suite.Error(err)
		suite.Nil(resp)

		grpcErr, ok := status.FromError(err)
		suite.True(ok)
		suite.Equal(codes.Unauthenticated, grpcErr.Code())
	})
}

// Rate Limiting Tests
func (suite *MiddlewareTestSuite) TestRateLimitMiddleware() {
	suite.Run("http rate limiting", func() {
		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Create rate limit middleware with very low limit for testing
		rateLimitMiddleware := middleware.RateLimitMiddleware(2, time.Minute) // 2 requests per minute
		handler := rateLimitMiddleware(testHandler)

		// Make first request - should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "127.0.0.1:12345"
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		suite.Equal(http.StatusOK, w1.Code)

		// Make second request - should succeed
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "127.0.0.1:12345"
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		suite.Equal(http.StatusOK, w2.Code)

		// Make third request - should be rate limited
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.RemoteAddr = "127.0.0.1:12345"
		w3 := httptest.NewRecorder()
		handler.ServeHTTP(w3, req3)
		suite.Equal(http.StatusTooManyRequests, w3.Code)
	})

	suite.Run("rate limiting per IP", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		rateLimitMiddleware := middleware.RateLimitMiddleware(1, time.Minute)
		handler := rateLimitMiddleware(testHandler)

		// Request from first IP - should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "127.0.0.1:12345"
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		suite.Equal(http.StatusOK, w1.Code)

		// Request from second IP - should succeed (different IP)
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.1:12345"
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		suite.Equal(http.StatusOK, w2.Code)

		// Second request from first IP - should be rate limited
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.RemoteAddr = "127.0.0.1:12345"
		w3 := httptest.NewRecorder()
		handler.ServeHTTP(w3, req3)
		suite.Equal(http.StatusTooManyRequests, w3.Code)
	})

	suite.Run("grpc rate limiting", func() {
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}

		rateLimitInterceptor := middleware.GRPCRateLimitInterceptor(2, time.Minute)

		info := &grpc.UnaryServerInfo{
			Server:     nil,
			FullMethod: "/test.Service/TestMethod",
		}

		// First request - should succeed
		resp1, err1 := rateLimitInterceptor(suite.ctx, "test-req", info, testHandler)
		suite.NoError(err1)
		suite.Equal("success", resp1)

		// Second request - should succeed
		resp2, err2 := rateLimitInterceptor(suite.ctx, "test-req", info, testHandler)
		suite.NoError(err2)
		suite.Equal("success", resp2)

		// Third request - should be rate limited
		resp3, err3 := rateLimitInterceptor(suite.ctx, "test-req", info, testHandler)
		suite.Error(err3)
		suite.Nil(resp3)

		grpcErr, ok := status.FromError(err3)
		suite.True(ok)
		suite.Equal(codes.ResourceExhausted, grpcErr.Code())
	})
}

// Audit Middleware Tests
func (suite *MiddlewareTestSuite) TestAuditMiddleware() {
	suite.Run("http audit logging", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		auditMiddleware := middleware.AuditMiddleware()
		handler := auditMiddleware(testHandler)

		req := httptest.NewRequest("POST", "/api/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
		suite.Equal("success", w.Body.String())

		// Audit should log the request/response details
		// In a real implementation, we'd check logs or audit database
	})

	suite.Run("grpc audit logging", func() {
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "audit-success", nil
		}

		auditInterceptor := middleware.AuditInterceptor()

		info := &grpc.UnaryServerInfo{
			Server:     nil,
			FullMethod: "/test.Service/AuditMethod",
		}

		resp, err := auditInterceptor(suite.ctx, "audit-request", info, testHandler)

		suite.NoError(err)
		suite.Equal("audit-success", resp)

		// Audit should log the gRPC call details
		// In a real implementation, we'd check logs or audit database
	})
}

// Middleware Chain Tests
func (suite *MiddlewareTestSuite) TestMiddlewareChain() {
	// Create test user
	testUser := fixtures.TestUser1
	hashedPassword, err := jwt.HashPassword(testUser.Password, testUser.Salt)
	suite.Require().NoError(err)
	testUser.PasswordHash = hashedPassword

	user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)
	suite.Require().NoError(err)

	validToken, err := jwt.GenerateAccessToken(user.ID)
	suite.Require().NoError(err)

	suite.Run("http middleware chain", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, exists := r.Context().Value("userID").(int)
			suite.True(exists)
			suite.Equal(user.ID, userID)
			w.WriteHeader(http.StatusOK)
		})

		// Chain multiple middlewares
		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		rateLimitMiddleware := middleware.RateLimitMiddleware(10, time.Minute)
		auditMiddleware := middleware.AuditMiddleware()

		// Apply middleware chain
		handler := auditMiddleware(rateLimitMiddleware(authMiddleware(testHandler)))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
	})

	suite.Run("grpc middleware chain", func() {
		testHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, exists := ctx.Value("userID").(int)
			suite.True(exists)
			suite.Equal(user.ID, userID)
			return "chain-success", nil
		}

		// Create interceptor chain
		authInterceptor := middleware.GRPCAuthInterceptor(suite.client)
		rateLimitInterceptor := middleware.GRPCRateLimitInterceptor(10, time.Minute)
		auditInterceptor := middleware.AuditInterceptor()

		// Chain interceptors manually for testing
		chainedHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return auditInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/test.Service/ChainTest"},
				func(ctx context.Context, req interface{}) (interface{}, error) {
					return rateLimitInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/test.Service/ChainTest"},
						func(ctx context.Context, req interface{}) (interface{}, error) {
							return authInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/test.Service/ChainTest"}, testHandler)
						})
				})
		}

		md := metadata.Pairs("authorization", "Bearer "+validToken)
		ctx := metadata.NewIncomingContext(suite.ctx, md)

		resp, err := chainedHandler(ctx, "chain-request")

		suite.NoError(err)
		suite.Equal("chain-success", resp)
	})
}

// Edge Cases and Error Scenarios
func (suite *MiddlewareTestSuite) TestMiddlewareEdgeCases() {
	suite.Run("concurrent rate limiting", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		rateLimitMiddleware := middleware.RateLimitMiddleware(5, time.Minute)
		handler := rateLimitMiddleware(testHandler)

		// Make concurrent requests
		done := make(chan bool, 10)
		results := make(chan int, 10)

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "127.0.0.1:12345"
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check results
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 10; i++ {
			code := <-results
			if code == http.StatusOK {
				successCount++
			} else if code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		suite.Equal(5, successCount, "Should have 5 successful requests")
		suite.Equal(5, rateLimitedCount, "Should have 5 rate-limited requests")
	})

	suite.Run("malformed auth headers", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		// Test various malformed headers
		malformedHeaders := []string{
			"Bearer",                   // Missing token
			"Bearer ",                  // Empty token
			"InvalidScheme token",      // Wrong scheme
			"Bearer token with spaces", // Invalid token format
		}

		for _, header := range malformedHeaders {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", header)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should handle gracefully - either 401 or continue without auth
			suite.True(w.Code == http.StatusUnauthorized || w.Code == http.StatusOK,
				"Should handle malformed header gracefully, got: %d for header: %s", w.Code, header)
		}
	})

	suite.Run("context cancellation during middleware", func() {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow handler
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		authMiddleware := middleware.HTTPAuthMiddleware(suite.client)
		handler := authMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(req.Context(), 50*time.Millisecond)
		defer cancel()

		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should handle context cancellation appropriately
		// The exact behavior depends on implementation
		suite.T().Logf("Response code with cancelled context: %d", w.Code)
	})
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
