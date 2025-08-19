package hostrole

import (
	"context"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/hostrole"
	"stormlink/server/graphql/models"
	"strconv"
)

type HostRoleUsecase interface {
	CreateHostRole(ctx context.Context, input *models.CreateHostRoleInput) (*ent.HostRole, error)
	UpdateHostRole(ctx context.Context, input *models.UpdateHostRoleInput) (*ent.HostRole, error)
	DeleteHostRole(ctx context.Context, id int) error
	GetHostRoles(ctx context.Context) ([]*ent.HostRole, error)
	GetHostRole(ctx context.Context, id int) (*ent.HostRole, error)
	AddUsersToRole(ctx context.Context, roleID int, userIDs []int) error
	RemoveUserFromRole(ctx context.Context, roleID int, userID int) error
}

type hostRoleUsecase struct {
	client *ent.Client
}

func NewHostRoleUsecase(client *ent.Client) HostRoleUsecase {
	return &hostRoleUsecase{client: client}
}

func (uc *hostRoleUsecase) CreateHostRole(ctx context.Context, input *models.CreateHostRoleInput) (*ent.HostRole, error) {
	create := uc.client.HostRole.Create().
		SetTitle(input.Title)

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
	if input.HostUserBan != nil {
		create = create.SetHostUserBan(*input.HostUserBan)
	}
	if input.HostUserMute != nil {
		create = create.SetHostUserMute(*input.HostUserMute)
	}
	if input.HostCommunityDeletePost != nil {
		create = create.SetHostCommunityDeletePost(*input.HostCommunityDeletePost)
	}
	if input.HostCommunityRemovePostFromPublication != nil {
		create = create.SetHostCommunityRemovePostFromPublication(*input.HostCommunityRemovePostFromPublication)
	}
	if input.HostCommunityDeleteComments != nil {
		create = create.SetHostCommunityDeleteComments(*input.HostCommunityDeleteComments)
	}

	role, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create host role: %w", err)
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

func (uc *hostRoleUsecase) UpdateHostRole(ctx context.Context, input *models.UpdateHostRoleInput) (*ent.HostRole, error) {
	roleID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	update := uc.client.HostRole.UpdateOneID(roleID)

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
	if input.HostUserBan != nil {
		update = update.SetHostUserBan(*input.HostUserBan)
	}
	if input.HostUserMute != nil {
		update = update.SetHostUserMute(*input.HostUserMute)
	}
	if input.HostCommunityDeletePost != nil {
		update = update.SetHostCommunityDeletePost(*input.HostCommunityDeletePost)
	}
	if input.HostCommunityRemovePostFromPublication != nil {
		update = update.SetHostCommunityRemovePostFromPublication(*input.HostCommunityRemovePostFromPublication)
	}
	if input.HostCommunityDeleteComments != nil {
		update = update.SetHostCommunityDeleteComments(*input.HostCommunityDeleteComments)
	}

	role, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update host role: %w", err)
	}

	// Обновляем пользователей если есть
	if input.UserIDs != nil {
		// Сначала очищаем всех пользователей
		_, err = uc.client.HostRole.UpdateOneID(roleID).ClearUsers().Save(ctx)
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

func (uc *hostRoleUsecase) DeleteHostRole(ctx context.Context, id int) error {
	// Проверяем, что роль существует
	_, err := uc.client.HostRole.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Удаляем роль
	err = uc.client.HostRole.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete host role: %w", err)
	}

	return nil
}

func (uc *hostRoleUsecase) GetHostRoles(ctx context.Context) ([]*ent.HostRole, error) {
	return uc.client.HostRole.Query().
		WithBadge().
		WithUsers().
		Order(ent.Asc("id")).
		All(ctx)
}

func (uc *hostRoleUsecase) GetHostRole(ctx context.Context, id int) (*ent.HostRole, error) {
	return uc.client.HostRole.Query().
		Where(hostrole.IDEQ(id)).
		WithBadge().
		WithUsers().
		Only(ctx)
}

func (uc *hostRoleUsecase) AddUsersToRole(ctx context.Context, roleID int, userIDs []int) error {
	// Получаем роль
	role, err := uc.client.HostRole.Get(ctx, roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Добавляем пользователей
	_, err = uc.client.HostRole.UpdateOne(role).AddUserIDs(userIDs...).Save(ctx)
	if err != nil {
		return fmt.Errorf("add users to role: %w", err)
	}

	return nil
}

func (uc *hostRoleUsecase) RemoveUserFromRole(ctx context.Context, roleID int, userID int) error {
	// Получаем роль
	role, err := uc.client.HostRole.Get(ctx, roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Удаляем пользователя
	_, err = uc.client.HostRole.UpdateOne(role).RemoveUserIDs(userID).Save(ctx)
	if err != nil {
		return fmt.Errorf("remove user from role: %w", err)
	}

	return nil
}
