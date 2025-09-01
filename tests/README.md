# Stormlink Backend Test Suite

Comprehensive testing suite for the Stormlink backend application built with Go, Ent ORM, GraphQL, and gRPC.

## 🎯 Overview

This test suite provides complete coverage for the Stormlink backend with multiple testing strategies:

- **Unit Tests** - Fast, isolated tests for individual components
- **Integration Tests** - Tests with real database and services
- **Performance Tests** - Benchmarks and load testing
- **End-to-End Tests** - Complete workflow testing

## 📁 Test Structure

```
tests/
├── fixtures/           # Test data and utilities
│   ├── user.go        # User test fixtures
│   └── extended.go    # Additional fixtures
├── integration/        # Integration tests
│   ├── auth_service_test.go
│   ├── e2e_test.go
│   ├── graphql_resolver_test.go
│   └── user_integration_test.go
├── unit/              # Unit tests
│   ├── comment_usecase_test.go
│   ├── community_usecase_test.go
│   ├── middleware_test.go
│   └── post_usecase_test.go
├── performance/       # Performance tests
│   ├── load_test.go
│   └── system_performance_test.go
├── testcontainers/    # Docker container setup
│   └── setup.go
├── mocks/             # Generated mocks
├── test_runner.go     # Test orchestration
└── README.md          # This file
```

## 🚀 Quick Start

### Prerequisites

```bash
# Install Go (1.21+)
go version

# Install Docker (for integration tests)
docker --version

# Install test dependencies
make install-tools
```

### Running Tests

```bash
# Quick unit tests
make test-unit

# All tests
make test

# With coverage
make test-coverage

# Performance tests
make test-performance
```

## 📋 Test Categories

### Unit Tests

Fast, isolated tests that don't require external dependencies.

```bash
# Run all unit tests
make test-unit

# Run specific test suite
go test ./tests/unit/user_usecase_test.go -v

# Run with race detection
make test-race
```

**Coverage:**
- User usecase logic
- Post operations
- Comment functionality
- Community management
- Middleware behavior
- JWT token handling

### Integration Tests

Tests with real databases and services using Docker containers.

```bash
# Run integration tests
make test-integration

# With test environment setup
make test-with-env
```

**Features:**
- Real PostgreSQL database
- Redis integration
- gRPC service communication
- GraphQL resolver testing
- Authentication flows

### Performance Tests

Benchmarks and load testing to ensure system performance.

```bash
# Run performance tests
make test-performance

# Run benchmarks only
make test-benchmark

# Memory profiling
make test-memory
```

**Metrics:**
- Request latency (< 50ms p95)
- Throughput (> 1000 RPS)
- Memory usage
- Database connection pooling
- Concurrent operation handling

### End-to-End Tests

Complete user journey testing.

```bash
# Run E2E tests
make test-e2e
```

**Scenarios:**
- User registration and verification
- Content creation workflow
- Community interactions
- Moderation processes
- Multi-user scenarios

## 🔧 Test Configuration

### Environment Variables

```bash
# Test database
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_NAME=stormlink_test
export TEST_DB_USER=test
export TEST_DB_PASSWORD=test

# Redis
export TEST_REDIS_URL=redis://localhost:6379

# Test settings
export GO_ENV=test
export LOG_LEVEL=error
```

### Test Fixtures

Test fixtures provide consistent test data:

```go
// Use predefined fixtures
user := fixtures.TestUser1
community := fixtures.TestCommunity1
post := fixtures.TestPost1

// Create custom fixtures
customUser := fixtures.UserFixture{
    Name:     "Custom User",
    Email:    "custom@test.com",
    Password: "password123",
}
```

### Test Containers

Integration tests use Docker containers for isolation:

```go
// Setup test containers
containers, err := testcontainers.Setup(ctx)
defer containers.Cleanup()

// Get database client
client := enttest.Open(t, "postgres", containers.PostgresDSN())
```

## 📊 Coverage Reports

### Generate Coverage

```bash
# Generate HTML coverage report
make test-coverage

# View coverage in browser
make coverage-html

# Show coverage summary
make coverage-summary
```

### Coverage Targets

- **Overall**: > 80%
- **Critical paths**: > 90%
- **New code**: > 85%

## 🎯 Testing Best Practices

### 1. Test Organization

```go
func (suite *TestSuite) TestFeatureName() {
    suite.Run("specific scenario", func() {
        // Arrange
        // Act  
        // Assert
    })
}
```

### 2. Test Data Management

```go
func (suite *TestSuite) SetupTest() {
    // Clean slate for each test
    suite.client.User.Delete().ExecX(suite.ctx)
    suite.client.Post.Delete().ExecX(suite.ctx)
}
```

### 3. Assertions

```go
// Use testify assertions
suite.NoError(err)
suite.NotNil(result)
suite.Equal(expected, actual)
suite.Contains(slice, item)
suite.True(condition)
```

### 4. Mocking

```go
// Mock external dependencies
mockClient := &MockAuthClient{}
mockClient.On("ValidateToken", token).Return(userID, nil)
```

## 🚀 Performance Benchmarks

### Running Benchmarks

```bash
# All benchmarks
go test ./tests/performance/... -bench=. -benchmem

# Specific benchmark
go test -bench=BenchmarkUserRetrieval -benchmem

# Compare with previous results
make benchmark-compare
```

### Performance Targets

| Operation | Target | Current |
|-----------|--------|---------|
| User retrieval | < 10ms | ~5ms |
| Post with relations | < 25ms | ~15ms |
| Comment pagination | < 30ms | ~20ms |
| GraphQL queries | < 50ms | ~35ms |
| Authentication | < 5ms | ~2ms |

## 🔍 Debugging Tests

### Verbose Output

```bash
# Maximum verbosity
make test-verbose

# Debug specific test
go test ./tests/unit/... -v -run=TestSpecificCase
```

### Test Debugging

```go
func TestDebugExample(t *testing.T) {
    // Add debug logging
    t.Logf("Debug info: %+v", data)
    
    // Use debugger breakpoints
    _ = data // Set breakpoint here
}
```

### Common Issues

1. **Database connection failures**
   ```bash
   make setup-test-env  # Ensure test containers are running
   ```

2. **Test data conflicts**
   ```bash
   make cleanup-test-env  # Reset test environment
   ```

3. **Timeout issues**
   ```bash
   go test -timeout=60m  # Increase timeout
   ```

## 🔄 Continuous Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: make ci
```

### Pre-commit Hooks

```bash
# Install pre-commit hooks
make pre-commit

# Manual pre-commit check
make dev-check
```

## 📈 Test Metrics

### Automated Reporting

```bash
# Generate test report
go test -json ./... | tee test-results.json

# Coverage badge
make coverage-badge
```

### Key Metrics

- **Test execution time**: < 10 minutes
- **Flaky test rate**: < 1%
- **Code coverage**: > 80%
- **Performance regression**: < 5%

## 🛠️ Development Workflow

### Adding New Tests

1. **Create test file**
   ```bash
   touch tests/unit/new_feature_test.go
   ```

2. **Follow naming convention**
   ```go
   func TestNewFeature_SpecificBehavior(t *testing.T) {
       // Test implementation
   }
   ```

3. **Add to test suite**
   ```go
   type NewFeatureTestSuite struct {
       suite.Suite
       // Test dependencies
   }
   ```

4. **Run tests**
   ```bash
   make test-unit
   ```

### Test-Driven Development

1. Write failing test
2. Implement minimal code to pass
3. Refactor while keeping tests green
4. Add edge cases and error scenarios

## 🎭 Mock Generation

### Generate Mocks

```bash
# Install mockery
go install github.com/vektra/mockery/v2@latest

# Generate mocks
mockery --all --dir=./server/usecase --output=./tests/mocks
```

### Using Mocks

```go
// tests/mocks/UserUsecase.go (generated)
mockUC := mocks.NewUserUsecase(t)
mockUC.On("GetUserByID", ctx, 1).Return(user, nil)
```

## 🔧 Troubleshooting

### Common Issues

**1. Test Containers Not Starting**
```bash
# Check Docker
docker ps

# Restart containers
make cleanup-test-env
make setup-test-env
```

**2. Database Connection Errors**
```bash
# Check connection string
echo $TEST_DB_DSN

# Verify database is running
psql $TEST_DB_DSN -c "SELECT 1"
```

**3. Test Timeouts**
```bash
# Increase timeout
go test -timeout=30m ./...

# Check for infinite loops or deadlocks
go test -race ./...
```

### Debug Commands

```bash
# Show environment info
make env-info

# Run with debugging
make debug-test

# Check test coverage
make coverage-summary
```

## 📚 Additional Resources

### Documentation

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Ent Testing Guide](https://entgo.io/docs/testing)
- [Test Containers Go](https://golang.testcontainers.org/)

### Tools

- **Testing**: `go test`, `testify`
- **Mocking**: `mockery`, `gomock`
- **Coverage**: `go tool cover`
- **Benchmarking**: `benchstat`
- **Containers**: `testcontainers-go`

## 🤝 Contributing

### Test Guidelines

1. Write tests for all new features
2. Maintain > 80% coverage
3. Follow naming conventions
4. Add integration tests for complex flows
5. Include performance tests for critical paths

### Code Review Checklist

- [ ] Tests cover happy path and edge cases
- [ ] Test names are descriptive
- [ ] No flaky tests
- [ ] Performance tests pass
- [ ] Coverage maintains target levels

---

**Happy Testing!** 🚀

For questions or issues, please check the [troubleshooting section](#-troubleshooting) or create an issue.