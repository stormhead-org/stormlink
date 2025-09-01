package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/tests/fixtures"

	_ "github.com/lib/pq"
)

type SimpleTestSuite struct {
	suite.Suite
	client *ent.Client
	ctx    context.Context
}

func (suite *SimpleTestSuite) SetupSuite() {
	suite.client = enttest.Open(suite.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	suite.ctx = context.Background()
}

func (suite *SimpleTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *SimpleTestSuite) SetupTest() {
	// Clean up data before each test
	suite.client.User.Delete().ExecX(suite.ctx)
	suite.client.Community.Delete().ExecX(suite.ctx)
}

func (suite *SimpleTestSuite) TestBasicUserCreation() {
	suite.Run("create user with fixtures", func() {
		// Test creating a user using fixtures
		testUser := fixtures.UserFixture{
			Name:       "Test User",
			Slug:       "test-user",
			Email:      "test@example.com",
			Password:   "password123",
			Salt:       "salt123",
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		user, err := fixtures.CreateTestUser(suite.ctx, suite.client, testUser)

		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), user)
		assert.Equal(suite.T(), testUser.Name, user.Name)
		assert.Equal(suite.T(), testUser.Email, user.Email)
		assert.Equal(suite.T(), testUser.Slug, user.Slug)
		assert.Equal(suite.T(), testUser.IsVerified, user.IsVerified)
	})

	suite.Run("create multiple users", func() {
		users := []fixtures.UserFixture{
			{
				Name:       "User 1",
				Slug:       "user-1",
				Email:      "user1@test.com",
				Password:   "pass1",
				Salt:       "salt1",
				IsVerified: true,
				CreatedAt:  time.Now(),
			},
			{
				Name:       "User 2",
				Slug:       "user-2",
				Email:      "user2@test.com",
				Password:   "pass2",
				Salt:       "salt2",
				IsVerified: false,
				CreatedAt:  time.Now(),
			},
		}

		for _, userFixture := range users {
			user, err := fixtures.CreateTestUser(suite.ctx, suite.client, userFixture)
			require.NoError(suite.T(), err)
			assert.NotNil(suite.T(), user)
			assert.Equal(suite.T(), userFixture.Email, user.Email)
		}

		// Verify both users exist
		allUsers, err := suite.client.User.Query().All(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), allUsers, 2)
	})
}

func (suite *SimpleTestSuite) TestBasicCommunityCreation() {
	suite.Run("create community with owner", func() {
		// First create a user to be the owner
		userFixture := fixtures.UserFixture{
			Name:       "Owner User",
			Slug:       "owner-user",
			Email:      "owner@test.com",
			Password:   "password",
			Salt:       "salt",
			IsVerified: true,
			CreatedAt:  time.Now(),
		}

		owner, err := fixtures.CreateTestUser(suite.ctx, suite.client, userFixture)
		require.NoError(suite.T(), err)

		// Create community
		communityFixture := fixtures.CommunityFixture{
			Name:        "Test Community",
			Slug:        "test-community",
			Description: "A test community",
			IsPrivate:   false,
			OwnerID:     owner.ID,
			CreatedAt:   time.Now(),
		}

		community, err := fixtures.CreateTestCommunity(suite.ctx, suite.client, communityFixture)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), community)
		assert.Equal(suite.T(), communityFixture.Name, community.Name)
		assert.Equal(suite.T(), communityFixture.Slug, community.Slug)
		assert.Equal(suite.T(), owner.ID, community.OwnerID)
	})
}

func (suite *SimpleTestSuite) TestDatabaseOperations() {
	suite.Run("basic database queries", func() {
		// Test counting users
		count, err := suite.client.User.Query().Count(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 0, count)

		// Create a user
		user, err := suite.client.User.Create().
			SetName("DB Test User").
			SetSlug("db-test-user").
			SetEmail("dbtest@example.com").
			SetPasswordHash("hash").
			SetSalt("salt").
			SetIsVerified(true).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)

		// Test counting again
		count, err = suite.client.User.Query().Count(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 1, count)

		// Test querying by ID
		queriedUser, err := suite.client.User.Get(suite.ctx, user.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.Email, queriedUser.Email)

		// Test querying by email
		userByEmail, err := suite.client.User.Query().
			Where(func(s *ent.UserSelect) {
				s.Where(s.EmailEQ("dbtest@example.com"))
			}).
			Only(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.ID, userByEmail.ID)
	})

	suite.Run("test error handling", func() {
		// Test querying non-existent user
		_, err := suite.client.User.Get(suite.ctx, 99999)
		assert.Error(suite.T(), err)
		assert.True(suite.T(), ent.IsNotFound(err))

		// Test duplicate email constraint
		_, err = suite.client.User.Create().
			SetName("User 1").
			SetSlug("user-1").
			SetEmail("duplicate@test.com").
			SetPasswordHash("hash").
			SetSalt("salt").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		require.NoError(suite.T(), err)

		// Try to create another user with same email
		_, err = suite.client.User.Create().
			SetName("User 2").
			SetSlug("user-2").
			SetEmail("duplicate@test.com").
			SetPasswordHash("hash").
			SetSalt("salt").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		assert.Error(suite.T(), err)
	})
}

func (suite *SimpleTestSuite) TestFixtureHelpers() {
	suite.Run("test fixture helper functions", func() {
		// Test random string generation
		str1 := fixtures.RandomString()
		str2 := fixtures.RandomString()
		assert.NotEqual(suite.T(), str1, str2)
		assert.Len(suite.T(), str1, 8)

		// Test random email generation
		email1 := fixtures.RandomEmail()
		email2 := fixtures.RandomEmail()
		assert.NotEqual(suite.T(), email1, email2)
		assert.Contains(suite.T(), email1, "@test.com")

		// Test random slug generation
		slug1 := fixtures.RandomSlug()
		slug2 := fixtures.RandomSlug()
		assert.NotEqual(suite.T(), slug1, slug2)
		assert.Contains(suite.T(), slug1, "test-")
	})
}

func TestSimpleTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleTestSuite))
}

// Standard Go tests (without suite)
func TestBasicAssertions(t *testing.T) {
	t.Run("basic testify assertions", func(t *testing.T) {
		// Test basic assertions
		assert.True(t, true)
		assert.False(t, false)
		assert.Equal(t, 1, 1)
		assert.NotEqual(t, 1, 2)
		assert.Nil(t, nil)

		var notNil = "something"
		assert.NotNil(t, notNil)

		slice := []string{"a", "b", "c"}
		assert.Len(t, slice, 3)
		assert.Contains(t, slice, "b")
		assert.NotContains(t, slice, "d")

		// Test with require (fails immediately on error)
		require.True(t, true)
		require.Equal(t, "hello", "hello")
	})

	t.Run("string operations", func(t *testing.T) {
		str := "Hello, World!"
		assert.Contains(t, str, "World")
		assert.True(t, len(str) > 5)
		assert.Equal(t, "Hello, World!", str)
	})

	t.Run("numeric operations", func(t *testing.T) {
		assert.Greater(t, 10, 5)
		assert.Less(t, 3, 7)
		assert.GreaterOrEqual(t, 5, 5)
		assert.LessOrEqual(t, 4, 4)

		floatVal := 3.14159
		assert.InDelta(t, 3.14, floatVal, 0.01)
	})
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	assert.NotNil(t, ctx)

	// Test context with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Error("Context should not be done immediately")
	default:
		// Context is not done, which is expected
	}
}
