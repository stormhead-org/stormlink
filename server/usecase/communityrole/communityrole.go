package communityrole

import (
	"context"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/role"
	"stormlink/server/graphql/models"
	"strconv"
)

type CommunityRoleUsecase interface {
	CreateCommunityRole(ctx context.Context, input *models.CreateCommunityRoleInput) (*ent.Role, error)
	UpdateCommunityRole(ctx context.Context, input *models.UpdateCommunityRoleInput) (*ent.Role, error)
	DeleteCommunityRole(ctx context.Context, id int) error
	GetCommunityRoles(ctx context.Context, communityID int) ([]*ent.Role, error)
	GetCommunityRole(ctx context.Context, id int) (*ent.Role, error)
	AddUsersToRole(ctx context.Context, roleID int, userIDs []int) error
	RemoveUserFromRole(ctx context.Context, roleID int, userID int) error
}

type communityRoleUsecase struct {
	client *ent.Client
}

func NewCommunityRoleUsecase(client *ent.Client) CommunityRoleUsecase {
	return &communityRoleUsecase{client: client}
}

func (uc *communityRoleUsecase) CreateCommunityRole(ctx context.Context, input *models.CreateCommunityRoleInput) (*ent.Role, error) {
	communityID, err := strconv.Atoi(input.CommunityID)
	if err != nil {
		return nil, fmt.Errorf("invalid communityID: %w", err)
	}

	create := uc.client.Role.Create().
		SetTitle(input.Title).
		SetCommunityID(communityID)

	if input.Color != nil {
		create = create.SetColor(*input.Color)
	}
	if input.BadgeID != nil {
		badgeID, err := strconv.Atoi(*input.BadgeID)
		if err != nil {
			return nil, fmt.Errorf("invalid badgeID: %w", err)
		}
		create = create.SetBadgeID(badgeID)
	}
	if input.CommunityRolesManagement != nil {
		create = create.SetCommunityRolesManagement(*input.CommunityRolesManagement)
	}
	if input.CommunityUserBan != nil {
		create = create.SetCommunityUserBan(*input.CommunityUserBan)
	}
	if input.CommunityUserMute != nil {
		create = create.SetCommunityUserMute(*input.CommunityUserMute)
	}
	if input.CommunityDeletePost != nil {
		create = create.SetCommunityDeletePost(*input.CommunityDeletePost)
	}
	if input.CommunityDeleteComments != nil {
		create = create.SetCommunityDeleteComments(*input.CommunityDeleteComments)
	}
	if input.CommunityRemovePostFromPublication != nil {
		create = create.SetCommunityRemovePostFromPublication(*input.CommunityRemovePostFromPublication)
	}

	role, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create community role: %w", err)
	}

	// Добавляем пользователей если есть
	if len(input.UserIDs) > 0 {
		userIDs := make([]int, len(input.UserIDs))
		for i, userIDStr := range input.UserIDs {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid userID %s: %w", userIDStr, err)
			}
			userIDs[i] = userID
		}
		err = uc.AddUsersToRole(ctx, role.ID, userIDs)
		if err != nil {
			return nil, fmt.Errorf("add users to role: %w", err)
		}
	}

	return role, nil
}

func (uc *communityRoleUsecase) UpdateCommunityRole(ctx context.Context, input *models.UpdateCommunityRoleInput) (*ent.Role, error) {
	roleID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	update := uc.client.Role.UpdateOneID(roleID)

	if input.Title != nil {
		update = update.SetTitle(*input.Title)
	}
	if input.Color != nil {
		update = update.SetColor(*input.Color)
	}
	if input.BadgeID != nil {
		badgeID, err := strconv.Atoi(*input.BadgeID)
		if err != nil {
			return nil, fmt.Errorf("invalid badgeID: %w", err)
		}
		update = update.SetBadgeID(badgeID)
	}
	if input.CommunityRolesManagement != nil {
		update = update.SetCommunityRolesManagement(*input.CommunityRolesManagement)
	}
	if input.CommunityUserBan != nil {
		update = update.SetCommunityUserBan(*input.CommunityUserBan)
	}
	if input.CommunityUserMute != nil {
		update = update.SetCommunityUserMute(*input.CommunityUserMute)
	}
	if input.CommunityDeletePost != nil {
		update = update.SetCommunityDeletePost(*input.CommunityDeletePost)
	}
	if input.CommunityDeleteComments != nil {
		update = update.SetCommunityDeleteComments(*input.CommunityDeleteComments)
	}
	if input.CommunityRemovePostFromPublication != nil {
		update = update.SetCommunityRemovePostFromPublication(*input.CommunityRemovePostFromPublication)
	}

	role, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update community role: %w", err)
	}

	// Обновляем пользователей если есть
	if input.UserIDs != nil {
		// Сначала очищаем всех пользователей
		_, err = uc.client.Role.UpdateOneID(roleID).ClearUsers().Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("clear users from role: %w", err)
		}

		// Добавляем новых пользователей
		if len(input.UserIDs) > 0 {
			userIDs := make([]int, len(input.UserIDs))
			for i, userIDStr := range input.UserIDs {
				userID, err := strconv.Atoi(userIDStr)
				if err != nil {
					return nil, fmt.Errorf("invalid userID %s: %w", userIDStr, err)
				}
				userIDs[i] = userID
			}
			err = uc.AddUsersToRole(ctx, roleID, userIDs)
			if err != nil {
				return nil, fmt.Errorf("add users to role: %w", err)
			}
		}
	}

	return role, nil
}

func (uc *communityRoleUsecase) DeleteCommunityRole(ctx context.Context, id int) error {
	// Проверяем, что роль существует
	_, err := uc.client.Role.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Удаляем роль
	err = uc.client.Role.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete community role: %w", err)
	}

	return nil
}

func (uc *communityRoleUsecase) GetCommunityRoles(ctx context.Context, communityID int) ([]*ent.Role, error) {
	fmt.Printf("🔍 GetCommunityRoles: communityID=%d\n", communityID)

	roles, err := uc.client.Role.Query().
		Where(role.CommunityIDEQ(communityID)).
		WithBadge().
		WithUsers().
		Order(ent.Asc("id")).
		All(ctx)

	if err != nil {
		fmt.Printf("❌ GetCommunityRoles: error - %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ GetCommunityRoles: found %d roles\n", len(roles))
	return roles, nil
}

func (uc *communityRoleUsecase) GetCommunityRole(ctx context.Context, id int) (*ent.Role, error) {
	return uc.client.Role.Query().
		Where(role.IDEQ(id)).
		WithBadge().
		WithUsers().
		WithCommunity().
		Only(ctx)
}

func (uc *communityRoleUsecase) AddUsersToRole(ctx context.Context, roleID int, userIDs []int) error {
	// Получаем роль
	role, err := uc.client.Role.Get(ctx, roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Добавляем пользователей
	_, err = uc.client.Role.UpdateOne(role).AddUserIDs(userIDs...).Save(ctx)
	if err != nil {
		return fmt.Errorf("add users to role: %w", err)
	}

	return nil
}

func (uc *communityRoleUsecase) RemoveUserFromRole(ctx context.Context, roleID int, userID int) error {
	// Получаем роль
	role, err := uc.client.Role.Get(ctx, roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Удаляем пользователя
	_, err = uc.client.Role.UpdateOne(role).RemoveUserIDs(userID).Save(ctx)
	if err != nil {
		return fmt.Errorf("remove user from role: %w", err)
	}

	return nil
}
