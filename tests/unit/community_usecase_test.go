package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/server/graphql/models"
	"stormlink/server/usecase/community"
	"stormlink/tests/fixtures"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type CommunityUsecaseTestSuite struct {
	suite.Suite
	client *ent.Client
	uc     community.CommunityUsecase
	ctx    context.Context
}

func (suite *CommunityUsecaseTestSuite) SetupSuite() {
	suite.client = enttest.Open(suite.T(), "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	suite.uc = community.NewCommunityUsecase(suite.client)
	suite.ctx = context.Background()
}

func (suite *CommunityUsecaseTestSuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *CommunityUsecaseTestSuite) SetupTest() {
	// Clean up data before each test
	suite.client.Role.Delete().ExecX(suite.ctx)
	suite.client.CommunityFollow.Delete().ExecX(suite.ctx)
	suite.client.Post.Delete().ExecX(suite.ctx)
	suite.client.ProfileTableInfoItem.Delete().ExecX(suite.ctx)
	suite.client.Community.Delete().ExecX(suite.ctx)
	suite.client.User.Delete().ExecX(suite.ctx)
	suite.client.Media.Delete().ExecX(suite.ctx)
}

func (suite *CommunityUsecaseTestSuite) TestGetCommunityByID() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("get existing community with basic data", func() {
		community, err := suite.uc.GetCommunityByID(suite.ctx, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(community)
		suite.Equal(fixtures.TestCommunity1.ID, community.ID)
		suite.Equal(fixtures.TestCommunity1.Name, community.Title)
		suite.Equal(fixtures.TestCommunity1.Slug, community.Slug)
		suite.Equal(fixtures.TestCommunity1.Description, community.Description)
		// IsPrivate field doesn't exist in the current schema
		// suite.Equal(fixtures.TestCommunity1.IsPrivate, community.IsPrivate)
		suite.Equal(fixtures.TestCommunity1.OwnerID, community.OwnerID)
	})

	suite.Run("get non-existing community", func() {
		community, err := suite.uc.GetCommunityByID(suite.ctx, 99999)

		suite.Error(err)
		suite.Nil(community)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("community with logo", func() {
		// Create logo image
		media, err := fixtures.CreateTestMedia(suite.ctx, suite.client, fixtures.TestMedia1)
		suite.Require().NoError(err)

		// Create community with logo
		communityWithLogo, err := suite.client.Community.Create().
			SetTitle("Community with Logo").
			SetSlug("community-with-logo").
			SetDescription("A community with a logo").
			SetOwnerID(fixtures.TestUser1.ID).
			SetLogoID(media.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Test retrieval
		retrievedCommunity, err := suite.uc.GetCommunityByID(suite.ctx, communityWithLogo.ID)
		suite.NoError(err)
		suite.NotNil(retrievedCommunity)

		// Verify logo edge is loaded
		logoEdge, err := retrievedCommunity.QueryLogo().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(media.ID, logoEdge.ID)
		suite.Equal(fixtures.TestMedia1.Filename, logoEdge.Filename)
	})

	suite.Run("community with community info", func() {
		// Create community info
		communityInfo, err := suite.client.ProfileTableInfoItem.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetKey("description").
			SetValue("This is a longer description for the community").
			SetType("community").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Test retrieval
		retrievedCommunity, err := suite.uc.GetCommunityByID(suite.ctx, fixtures.TestCommunity1.ID)
		suite.NoError(err)
		suite.NotNil(retrievedCommunity)

		// Verify community info edge is loaded
		// Verify info edge is loaded
		infoEdges, err := retrievedCommunity.QueryCommunityInfo().All(suite.ctx)
		suite.NoError(err)
		suite.NotEmpty(infoEdges)
		// Check if our description info item exists
		found := false
		for _, info := range infoEdges {
			if info.Key == "description" {
				suite.Equal("This is a longer description for the community", info.Value)
				found = true
				break
			}
		}
		suite.True(found, "Description info item should be found")
	})

	suite.Run("community with roles", func() {
		// Create community roles
		adminRole, err := suite.client.CommunityRole.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetName("Admin").
			SetDescription("Community administrator").
			SetPermissions("admin,moderate,post,comment").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		moderatorRole, err := suite.client.CommunityRole.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetName("Moderator").
			SetDescription("Community moderator").
			SetPermissions("moderate,post,comment").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Test retrieval
		retrievedCommunity, err := suite.uc.GetCommunityByID(suite.ctx, fixtures.TestCommunity1.ID)
		suite.NoError(err)
		suite.NotNil(retrievedCommunity)

		// Verify roles are loaded
		roles, err := retrievedCommunity.QueryRoles().All(suite.ctx)
		suite.NoError(err)
		suite.Len(roles, 2)

		// Verify role details
		roleNames := make(map[string]bool)
		for _, role := range roles {
			roleNames[role.Name] = true
			if role.ID == adminRole.ID {
				suite.Equal("Community administrator", role.Description)
				suite.Equal("admin,moderate,post,comment", role.Permissions)
			}
			if role.ID == moderatorRole.ID {
				suite.Equal("Community moderator", role.Description)
				suite.Equal("moderate,post,comment", role.Permissions)
			}
		}
		suite.True(roleNames["Admin"])
		suite.True(roleNames["Moderator"])
	})
}

func (suite *CommunityUsecaseTestSuite) TestGetCommunityStatus() {
	// Seed test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("owner viewing own community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestCommunity1.OwnerID, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.True(status.IsOwner)
		suite.False(status.IsMember)    // Owner is not counted as regular member
		suite.False(status.IsModerator) // Owner is not counted as moderator (has higher privileges)
		suite.Equal(models.CommunityStatusRelationshipOwner, status.Relationship)
	})

	suite.Run("user viewing another user's community", func() {
		otherUserID := fixtures.TestUser2.ID
		if fixtures.TestCommunity1.OwnerID == fixtures.TestUser2.ID {
			otherUserID = fixtures.TestUser1.ID
		}

		status, err := suite.uc.GetCommunityStatus(suite.ctx, otherUserID, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.False(status.IsMember)
		suite.False(status.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipNone, status.Relationship)
	})

	suite.Run("user who is a member", func() {
		// Create membership
		_, err := suite.client.CommunityMember.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetJoinedAt(time.Now().Add(-24 * time.Hour)).
			Save(suite.ctx)
		suite.Require().NoError(err)

		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.True(status.IsMember)
		suite.False(status.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipMember, status.Relationship)
	})

	suite.Run("user who is a moderator", func() {
		// First create a role
		moderatorRole, err := suite.client.CommunityRole.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetName("Moderator").
			SetDescription("Community moderator").
			SetPermissions("moderate,post,comment").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create membership with moderator role
		_, err = suite.client.CommunityMember.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetUserID(fixtures.UnverifiedUser.ID).
			SetRoleID(moderatorRole.ID).
			SetJoinedAt(time.Now().Add(-48 * time.Hour)).
			Save(suite.ctx)
		suite.Require().NoError(err)

		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.UnverifiedUser.ID, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.True(status.IsMember)
		suite.True(status.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipModerator, status.Relationship)
	})

	suite.Run("anonymous user viewing community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, 0, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.False(status.IsMember)
		suite.False(status.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipNone, status.Relationship)
	})

	suite.Run("private community status", func() {
		// Test status for private community
		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser1.ID, fixtures.PrivateCommunity.ID)

		suite.NoError(err)
		suite.NotNil(status)

		if fixtures.PrivateCommunity.OwnerID == fixtures.TestUser1.ID {
			suite.True(status.IsOwner)
			suite.Equal(models.CommunityStatusRelationshipOwner, status.Relationship)
		} else {
			suite.False(status.IsOwner)
			suite.False(status.IsMember)
			suite.Equal(models.CommunityStatusRelationshipNone, status.Relationship)
		}
	})

	suite.Run("non-existing community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser1.ID, 99999)

		suite.Error(err)
		suite.Nil(status)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("non-existing user viewing existing community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, 99999, fixtures.TestCommunity1.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.False(status.IsMember)
		suite.False(status.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipNone, status.Relationship)
	})
}

func (suite *CommunityUsecaseTestSuite) TestCommunityWithComplexRelationships() {
	// Create a more complex test scenario
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	// Create logo
	logo, err := fixtures.CreateTestMedia(suite.ctx, suite.client, "complex-logo.png", "https://example.com/complex-logo.png")
	suite.Require().NoError(err)

	// Create complex community
	complexCommunity, err := suite.client.Community.Create().
		SetName("Complex Community").
		SetSlug("complex-community").
		SetDescription("A complex community with many relationships").
		SetIsPrivate(false).
		SetOwnerID(fixtures.TestUser1.ID).
		SetLogoID(logo.ID).
		SetCreatedAt(time.Now().Add(-30 * 24 * time.Hour)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Create community info
	_, err = suite.client.CommunityInfo.Create().
		SetCommunityID(complexCommunity.ID).
		SetLongDescription("This is a very detailed description of our complex community with multiple paragraphs and detailed information about what we do and how we operate.").
		SetRules("1. Be respectful to all members\n2. No spam or self-promotion\n3. Stay on topic\n4. Use appropriate tags\n5. No hate speech").
		SetMemberCount(1500).
		SetPostCount(2300).
		SetCreatedAt(time.Now().Add(-25 * 24 * time.Hour)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Create multiple roles with different permissions
	adminRole, err := suite.client.CommunityRole.Create().
		SetCommunityID(complexCommunity.ID).
		SetName("Administrator").
		SetDescription("Full administrative privileges").
		SetPermissions("admin,moderate,delete,ban,post,comment,manage_roles").
		SetCreatedAt(time.Now().Add(-20 * 24 * time.Hour)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	moderatorRole, err := suite.client.CommunityRole.Create().
		SetCommunityID(complexCommunity.ID).
		SetName("Moderator").
		SetDescription("Content moderation privileges").
		SetPermissions("moderate,delete,post,comment").
		SetCreatedAt(time.Now().Add(-20 * 24 * time.Hour)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	contributorRole, err := suite.client.CommunityRole.Create().
		SetCommunityID(complexCommunity.ID).
		SetName("Contributor").
		SetDescription("Active community contributor").
		SetPermissions("post,comment,feature_content").
		SetCreatedAt(time.Now().Add(-15 * 24 * time.Hour)).
		SetUpdatedAt(time.Now()).
		Save(suite.ctx)
	suite.Require().NoError(err)

	// Create members with different roles
	_, err = suite.client.CommunityMember.Create().
		SetCommunityID(complexCommunity.ID).
		SetUserID(fixtures.TestUser2.ID).
		SetRoleID(moderatorRole.ID).
		SetJoinedAt(time.Now().Add(-10 * 24 * time.Hour)).
		Save(suite.ctx)
	suite.Require().NoError(err)

	_, err = suite.client.CommunityMember.Create().
		SetCommunityID(complexCommunity.ID).
		SetUserID(fixtures.UnverifiedUser.ID).
		SetRoleID(contributorRole.ID).
		SetJoinedAt(time.Now().Add(-5 * 24 * time.Hour)).
		Save(suite.ctx)
	suite.Require().NoError(err)

	suite.Run("retrieve complex community with all relations", func() {
		retrievedCommunity, err := suite.uc.GetCommunityByID(suite.ctx, complexCommunity.ID)

		suite.NoError(err)
		suite.NotNil(retrievedCommunity)
		suite.Equal("Complex Community", retrievedCommunity.Name)

		// Verify logo
		logoEdge, err := retrievedCommunity.QueryLogo().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(logo.ID, logoEdge.ID)

		// Verify community info
		infoEdge, err := retrievedCommunity.QueryCommunityInfo().Only(suite.ctx)
		suite.NoError(err)
		suite.Equal(1500, infoEdge.MemberCount)
		suite.Equal(2300, infoEdge.PostCount)
		suite.Contains(infoEdge.Rules, "Be respectful")

		// Verify roles
		roles, err := retrievedCommunity.QueryRoles().All(suite.ctx)
		suite.NoError(err)
		suite.Len(roles, 3)

		roleNames := make(map[string]string)
		for _, role := range roles {
			roleNames[role.Name] = role.Permissions
		}
		suite.Contains(roleNames["Administrator"], "admin")
		suite.Contains(roleNames["Moderator"], "moderate")
		suite.Contains(roleNames["Contributor"], "feature_content")
	})

	suite.Run("owner status for complex community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser1.ID, complexCommunity.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.True(status.IsOwner)
		suite.False(status.IsMember)    // Owner is not regular member
		suite.False(status.IsModerator) // Owner has higher privileges
		suite.Equal(models.CommunityStatusRelationshipOwner, status.Relationship)
	})

	suite.Run("moderator status for complex community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser2.ID, complexCommunity.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.True(status.IsMember)
		suite.True(status.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipModerator, status.Relationship)
	})

	suite.Run("contributor status for complex community", func() {
		status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.UnverifiedUser.ID, complexCommunity.ID)

		suite.NoError(err)
		suite.NotNil(status)
		suite.False(status.IsOwner)
		suite.True(status.IsMember)
		suite.False(status.IsModerator) // Contributor is not moderator
		suite.Equal(models.CommunityStatusRelationshipMember, status.Relationship)
	})
}

func (suite *CommunityUsecaseTestSuite) TestEdgeCases() {
	suite.Run("context cancellation", func() {
		cancelledCtx, cancel := context.WithCancel(suite.ctx)
		cancel()

		community, err := suite.uc.GetCommunityByID(cancelledCtx, 1)

		suite.Error(err)
		suite.Nil(community)
		suite.Contains(err.Error(), "context canceled")
	})

	suite.Run("community with missing logo reference", func() {
		err := fixtures.SeedBasicData(suite.ctx, suite.client)
		suite.Require().NoError(err)

		// Create community with non-existent logo ID
		communityWithBadLogo, err := suite.client.Community.Create().
			SetName("Community with Bad Logo").
			SetSlug("bad-logo-community").
			SetDescription("This community has a bad logo reference").
			SetIsPrivate(false).
			SetOwnerID(fixtures.TestUser1.ID).
			SetLogoID(99999). // Non-existent logo ID
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Should still retrieve the community, but logo query will fail
		retrievedCommunity, err := suite.uc.GetCommunityByID(suite.ctx, communityWithBadLogo.ID)
		suite.NoError(err)
		suite.NotNil(retrievedCommunity)

		// Logo query should fail gracefully
		_, err = retrievedCommunity.QueryLogo().Only(suite.ctx)
		suite.Error(err)
		suite.True(ent.IsNotFound(err))
	})

	suite.Run("community with deleted owner", func() {
		err := fixtures.SeedBasicData(suite.ctx, suite.client)
		suite.Require().NoError(err)

		// Create a community
		community, err := suite.client.Community.Create().
			SetName("Community with Deleted Owner").
			SetSlug("deleted-owner-community").
			SetDescription("This community's owner will be deleted").
			SetIsPrivate(false).
			SetOwnerID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Delete the owner
		err = suite.client.User.DeleteOneID(fixtures.TestUser1.ID).Exec(suite.ctx)
		suite.Require().NoError(err)

		// Community retrieval should still work
		retrievedCommunity, err := suite.uc.GetCommunityByID(suite.ctx, community.ID)
		suite.NoError(err)
		suite.NotNil(retrievedCommunity)
		suite.Equal(fixtures.TestUser1.ID, retrievedCommunity.OwnerID) // ID still stored

		// Owner query would fail, but that's expected
	})

	suite.Run("orphaned community info", func() {
		// Create community info without a community (edge case)
		_, err := suite.client.CommunityInfo.Create().
			SetCommunityID(99999). // Non-existent community
			SetLongDescription("Orphaned info").
			SetRules("No rules").
			SetMemberCount(0).
			SetPostCount(0).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		// This should fail due to foreign key constraints
		suite.Error(err)
	})
}

func (suite *CommunityUsecaseTestSuite) TestPermissionScenarios() {
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("role permission changes over time", func() {
		// Create a role
		role, err := suite.client.CommunityRole.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetName("Evolving Role").
			SetDescription("A role that changes over time").
			SetPermissions("post,comment").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create member with this role
		_, err = suite.client.CommunityMember.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetRoleID(role.ID).
			SetJoinedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Check initial status
		status1, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)
		suite.NoError(err)
		suite.True(status1.IsMember)
		suite.False(status1.IsModerator) // No moderate permission yet

		// Update role to include moderate permission
		_, err = role.Update().
			SetPermissions("post,comment,moderate").
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Check updated status
		status2, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)
		suite.NoError(err)
		suite.True(status2.IsMember)
		suite.True(status2.IsModerator) // Now has moderate permission
		suite.Equal(models.CommunityStatusRelationshipModerator, status2.Relationship)
	})

	suite.Run("member role changes", func() {
		// Create roles
		memberRole, err := suite.client.CommunityRole.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetName("Member").
			SetDescription("Regular member").
			SetPermissions("post,comment").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		moderatorRole, err := suite.client.CommunityRole.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetName("Moderator").
			SetDescription("Community moderator").
			SetPermissions("post,comment,moderate,delete").
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Create member with regular role
		membership, err := suite.client.CommunityMember.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetUserID(fixtures.UnverifiedUser.ID).
			SetRoleID(memberRole.ID).
			SetJoinedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Check initial status
		status1, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.UnverifiedUser.ID, fixtures.TestCommunity1.ID)
		suite.NoError(err)
		suite.True(status1.IsMember)
		suite.False(status1.IsModerator)

		// Promote to moderator
		_, err = membership.Update().
			SetRoleID(moderatorRole.ID).
			Save(suite.ctx)
		suite.Require().NoError(err)

		// Check updated status
		status2, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.UnverifiedUser.ID, fixtures.TestCommunity1.ID)
		suite.NoError(err)
		suite.True(status2.IsMember)
		suite.True(status2.IsModerator)
		suite.Equal(models.CommunityStatusRelationshipModerator, status2.Relationship)
	})
}

func (suite *CommunityUsecaseTestSuite) TestPerformance() {
	// Create test data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	// Create additional test communities
	communityIDs := make([]int, 20)
	for i := 0; i < 20; i++ {
		community, err := suite.client.Community.Create().
			SetName(fmt.Sprintf("Performance Test Community %d", i)).
			SetSlug(fmt.Sprintf("perf-community-%d", i)).
			SetDescription(fmt.Sprintf("Performance test community %d", i)).
			SetIsPrivate(i%3 == 0). // Make every 3rd community private
			SetOwnerID(fixtures.TestUser1.ID).
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Minute)).
			SetUpdatedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)
		communityIDs[i] = community.ID

		// Add community info to some communities
		if i%2 == 0 {
			_, err = suite.client.CommunityInfo.Create().
				SetCommunityID(community.ID).
				SetLongDescription(fmt.Sprintf("Long description for community %d", i)).
				SetRules("Standard rules").
				SetMemberCount(i * 10).
				SetPostCount(i * 5).
				SetCreatedAt(time.Now()).
				SetUpdatedAt(time.Now()).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}

		// Add roles to some communities
		if i%3 == 0 {
			_, err = suite.client.CommunityRole.Create().
				SetCommunityID(community.ID).
				SetName("Admin").
				SetDescription("Administrator").
				SetPermissions("admin,moderate,post,comment").
				SetCreatedAt(time.Now()).
				SetUpdatedAt(time.Now()).
				Save(suite.ctx)
			suite.Require().NoError(err)
		}
	}

	suite.Run("bulk community retrieval performance", func() {
		start := time.Now()

		for i := 0; i < 10; i++ {
			_, err := suite.uc.GetCommunityByID(suite.ctx, communityIDs[i])
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / 10

		suite.Less(avgDuration, 30*time.Millisecond, "Average community retrieval should be fast")
	})

	suite.Run("status retrieval performance", func() {
		start := time.Now()

		for i := 0; i < 15; i++ {
			_, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser1.ID, communityIDs[i])
			suite.NoError(err)
		}

		duration := time.Since(start)
		avgDuration := duration / 15

		suite.Less(avgDuration, 25*time.Millisecond, "Average status retrieval should be fast")
	})
}

// Benchmark tests
func (suite *CommunityUsecaseTestSuite) TestBenchmarkCommunityRetrieval() {
	// Setup data
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("benchmark GetCommunityByID", func() {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := suite.uc.GetCommunityByID(suite.ctx, fixtures.TestCommunity1.ID)
			suite.NoError(err)
		}

		avgDuration := time.Since(start) / time.Duration(iterations)
		suite.Less(avgDuration, 15*time.Millisecond, "Average GetCommunityByID should be very fast")
	})

	suite.Run("benchmark GetCommunityStatus", func() {
		// Create some membership to make it realistic
		_, err := suite.client.CommunityMember.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetJoinedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)
			suite.NoError(err)
		}

		avgDuration := time.Since(start) / time.Duration(iterations)
		suite.Less(avgDuration, 12*time.Millisecond, "Average GetCommunityStatus should be very fast")
	})
}

func (suite *CommunityUsecaseTestSuite) TestConcurrentAccess() {
	err := fixtures.SeedBasicData(suite.ctx, suite.client)
	suite.Require().NoError(err)

	suite.Run("concurrent community retrieval", func() {
		done := make(chan bool, 10)
		errors := make(chan error, 10)

		// Start 10 concurrent goroutines
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				community, err := suite.uc.GetCommunityByID(suite.ctx, fixtures.TestCommunity1.ID)
				if err != nil {
					errors <- err
					return
				}

				if community == nil || community.ID != fixtures.TestCommunity1.ID {
					errors <- fmt.Errorf("invalid community retrieved")
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
			suite.Fail("Concurrent access failed", err.Error())
		default:
			// No errors, test passed
		}
	})

	suite.Run("concurrent status retrieval", func() {
		// Create membership for testing
		_, err := suite.client.CommunityMember.Create().
			SetCommunityID(fixtures.TestCommunity1.ID).
			SetUserID(fixtures.TestUser2.ID).
			SetJoinedAt(time.Now()).
			Save(suite.ctx)
		suite.Require().NoError(err)

		done := make(chan bool, 10)
		errors := make(chan error, 10)

		// Start 10 concurrent goroutines
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				status, err := suite.uc.GetCommunityStatus(suite.ctx, fixtures.TestUser2.ID, fixtures.TestCommunity1.ID)
				if err != nil {
					errors <- err
					return
				}

				if status == nil || !status.IsMember {
					errors <- fmt.Errorf("invalid status retrieved")
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
			suite.Fail("Concurrent status access failed", err.Error())
		default:
			// No errors, test passed
		}
	})
}

func TestCommunityUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(CommunityUsecaseTestSuite))
}
