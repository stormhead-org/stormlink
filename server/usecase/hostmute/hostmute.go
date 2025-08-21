package hostmute

import (
	"context"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/hostcommunitymute"
	"stormlink/server/ent/hostusermute"
	"stormlink/server/ent/user"
	sharedauth "stormlink/shared/auth"
	"strconv"
)

type HostMuteUsecase interface {
	MuteUser(ctx context.Context, userID string) (*ent.HostUserMute, error)
	UnmuteUser(ctx context.Context, muteID string) (bool, error)
	GetUserMutes(ctx context.Context) ([]*ent.HostUserMute, error)
	MuteCommunity(ctx context.Context, communityID string) (*ent.HostCommunityMute, error)
	UnmuteCommunity(ctx context.Context, muteID string) (bool, error)
	GetCommunityMutes(ctx context.Context) ([]*ent.HostCommunityMute, error)
}

type hostMuteUsecase struct {
	client *ent.Client
}

func NewHostMuteUsecase(client *ent.Client) HostMuteUsecase {
	return &hostMuteUsecase{client: client}
}

func (uc *hostMuteUsecase) MuteUser(ctx context.Context, userID string) (*ent.HostUserMute, error) {
	// Проверяем авторизацию
	currentUserID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Проверяем права на мут пользователей
	canMute, err := uc.canMuteUsers(ctx, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canMute {
		return nil, fmt.Errorf("insufficient permissions to mute users")
	}

	// Конвертируем userID в int
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Проверяем, не пытается ли пользователь замутить сам себя
	if userIDInt == currentUserID {
		return nil, fmt.Errorf("cannot mute yourself")
	}

	// Проверяем, не замучен ли уже пользователь
	_, err = uc.client.HostUserMute.
		Query().
		Where(hostusermute.HasUserWith(user.IDEQ(userIDInt))).
		Only(ctx)
	if err == nil {
		return nil, fmt.Errorf("user is already muted")
	}

	// Создаем мут
	mute, err := uc.client.HostUserMute.
		Create().
		SetUserID(userIDInt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to mute user: %w", err)
	}

	return mute, nil
}

func (uc *hostMuteUsecase) UnmuteUser(ctx context.Context, muteID string) (bool, error) {
	// Проверяем авторизацию
	currentUserID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	// Проверяем права на мут пользователей
	canMute, err := uc.canMuteUsers(ctx, currentUserID)
	if err != nil {
		return false, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canMute {
		return false, fmt.Errorf("insufficient permissions to unmute users")
	}

	// Конвертируем muteID в int
	muteIDInt, err := strconv.Atoi(muteID)
	if err != nil {
		return false, fmt.Errorf("invalid mute ID: %w", err)
	}

	// Удаляем мут
	err = uc.client.HostUserMute.DeleteOneID(muteIDInt).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to unmute user: %w", err)
	}

	return true, nil
}

func (uc *hostMuteUsecase) GetUserMutes(ctx context.Context) ([]*ent.HostUserMute, error) {
	mutes, err := uc.client.HostUserMute.
		Query().
		WithUser().
		Order(ent.Asc(hostusermute.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user mutes: %w", err)
	}

	return mutes, nil
}

func (uc *hostMuteUsecase) MuteCommunity(ctx context.Context, communityID string) (*ent.HostCommunityMute, error) {
	// Проверяем авторизацию
	currentUserID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Проверяем права на мут сообществ
	canMute, err := uc.canMuteCommunities(ctx, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canMute {
		return nil, fmt.Errorf("insufficient permissions to mute communities")
	}

	// Конвертируем communityID в int
	communityIDInt, err := strconv.Atoi(communityID)
	if err != nil {
		return nil, fmt.Errorf("invalid community ID: %w", err)
	}

	// Проверяем, не замучено ли уже сообщество
	_, err = uc.client.HostCommunityMute.
		Query().
		Where(hostcommunitymute.CommunityIDEQ(communityIDInt)).
		Only(ctx)
	if err == nil {
		return nil, fmt.Errorf("community is already muted")
	}

	// Создаем мут
	mute, err := uc.client.HostCommunityMute.
		Create().
		SetCommunityID(communityIDInt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to mute community: %w", err)
	}

	return mute, nil
}

func (uc *hostMuteUsecase) UnmuteCommunity(ctx context.Context, muteID string) (bool, error) {
	// Проверяем авторизацию
	currentUserID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	// Проверяем права на мут сообществ
	canMute, err := uc.canMuteCommunities(ctx, currentUserID)
	if err != nil {
		return false, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canMute {
		return false, fmt.Errorf("insufficient permissions to unmute communities")
	}

	// Конвертируем muteID в int
	muteIDInt, err := strconv.Atoi(muteID)
	if err != nil {
		return false, fmt.Errorf("invalid mute ID: %w", err)
	}

	// Удаляем мут
	err = uc.client.HostCommunityMute.DeleteOneID(muteIDInt).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to unmute community: %w", err)
	}

	return true, nil
}

func (uc *hostMuteUsecase) GetCommunityMutes(ctx context.Context) ([]*ent.HostCommunityMute, error) {
	mutes, err := uc.client.HostCommunityMute.
		Query().
		WithCommunity().
		Order(ent.Asc(hostcommunitymute.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get community mutes: %w", err)
	}

	return mutes, nil
}

// canMuteUsers проверяет, может ли пользователь мутить других пользователей
func (uc *hostMuteUsecase) canMuteUsers(ctx context.Context, userID int) (bool, error) {
	// 1. Проверяем, является ли пользователь владельцем платформы
	host, err := uc.client.Host.Get(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("host not found: %w", err)
	}
	
	if host.OwnerID != nil && *host.OwnerID == userID {
		return true, nil
	}
	
	// 2. Проверяем роли пользователя на платформе
	// TODO: Добавить проверку ролей когда будет реализована система ролей платформы
	// Пока что только владелец может мутить пользователей
	
	return false, nil
}

// canMuteCommunities проверяет, может ли пользователь мутить сообщества
func (uc *hostMuteUsecase) canMuteCommunities(ctx context.Context, userID int) (bool, error) {
	// 1. Проверяем, является ли пользователь владельцем платформы
	host, err := uc.client.Host.Get(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("host not found: %w", err)
	}
	
	if host.OwnerID != nil && *host.OwnerID == userID {
		return true, nil
	}
	
	// 2. Проверяем роли пользователя на платформе
	// TODO: Добавить проверку ролей когда будет реализована система ролей платформы
	// Пока что только владелец может мутить сообщества
	
	return false, nil
}
