package ban

import (
	"context"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/community"
	"stormlink/server/ent/communityuserban"
	"stormlink/server/ent/communityusermute"
	"stormlink/server/ent/hostcommunityban"
	"stormlink/server/ent/hostuserban"
	"stormlink/server/ent/user"
)

type BanUsecase interface {
	// Host bans
	BanUserFromHost(ctx context.Context, userID, hostID int) (*ent.HostUserBan, error)
	UnbanUserFromHost(ctx context.Context, banID int) error
	BanCommunityFromHost(ctx context.Context, communityID, hostID int) (*ent.HostCommunityBan, error)
	UnbanCommunityFromHost(ctx context.Context, banID int) error

	// Community bans/mutes
	BanUserFromCommunity(ctx context.Context, userID, communityID int) (*ent.CommunityUserBan, error)
	UnbanUserFromCommunity(ctx context.Context, banID int) error
	MuteUserInCommunity(ctx context.Context, userID, communityID int) (*ent.CommunityUserMute, error)
	UnmuteUserInCommunity(ctx context.Context, muteID int) error

	// Queries
	GetHostUserBans(ctx context.Context) ([]*ent.HostUserBan, error)
	GetHostCommunityBans(ctx context.Context) ([]*ent.HostCommunityBan, error)
	GetCommunityUserBans(ctx context.Context, communityID int) ([]*ent.CommunityUserBan, error)
	GetCommunityUserMutes(ctx context.Context, communityID int) ([]*ent.CommunityUserMute, error)
}

type banUsecase struct {
	client *ent.Client
}

func NewBanUsecase(client *ent.Client) BanUsecase {
	return &banUsecase{client: client}
}

func (uc *banUsecase) BanUserFromHost(ctx context.Context, userID, hostID int) (*ent.HostUserBan, error) {
	// Проверяем, что пользователь не забанен уже
	exists, err := uc.client.HostUserBan.Query().
		Where(hostuserban.HasUserWith(user.IDEQ(userID))).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing ban: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user already banned from host")
	}

	// Создаем бан
	ban, err := uc.client.HostUserBan.Create().
		SetUserID(userID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create host user ban: %w", err)
	}

	return ban, nil
}

func (uc *banUsecase) UnbanUserFromHost(ctx context.Context, banID int) error {
	// Проверяем, что бан существует
	ban, err := uc.client.HostUserBan.Get(ctx, banID)
	if err != nil {
		return fmt.Errorf("ban not found: %w", err)
	}

	// Удаляем бан
	err = uc.client.HostUserBan.DeleteOne(ban).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete host user ban: %w", err)
	}

	return nil
}

func (uc *banUsecase) BanCommunityFromHost(ctx context.Context, communityID, hostID int) (*ent.HostCommunityBan, error) {
	// Проверяем, что сообщество не забанено уже
	exists, err := uc.client.HostCommunityBan.Query().
		Where(hostcommunityban.HasCommunityWith(community.IDEQ(communityID))).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing ban: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("community already banned from host")
	}

	// Создаем бан
	ban, err := uc.client.HostCommunityBan.Create().
		SetCommunityID(communityID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create host community ban: %w", err)
	}

	return ban, nil
}

func (uc *banUsecase) UnbanCommunityFromHost(ctx context.Context, banID int) error {
	// Проверяем, что бан существует
	ban, err := uc.client.HostCommunityBan.Get(ctx, banID)
	if err != nil {
		return fmt.Errorf("ban not found: %w", err)
	}

	// Удаляем бан
	err = uc.client.HostCommunityBan.DeleteOne(ban).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete host community ban: %w", err)
	}

	return nil
}

func (uc *banUsecase) BanUserFromCommunity(ctx context.Context, userID, communityID int) (*ent.CommunityUserBan, error) {
	// Проверяем, что пользователь не забанен уже
	exists, err := uc.client.CommunityUserBan.Query().
		Where(
			communityuserban.UserIDEQ(userID),
			communityuserban.CommunityIDEQ(communityID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing ban: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user already banned from community")
	}

	// Создаем бан
	ban, err := uc.client.CommunityUserBan.Create().
		SetUserID(userID).
		SetCommunityID(communityID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create community user ban: %w", err)
	}

	return ban, nil
}

func (uc *banUsecase) UnbanUserFromCommunity(ctx context.Context, banID int) error {
	// Проверяем, что бан существует
	ban, err := uc.client.CommunityUserBan.Get(ctx, banID)
	if err != nil {
		return fmt.Errorf("ban not found: %w", err)
	}

	// Удаляем бан
	err = uc.client.CommunityUserBan.DeleteOne(ban).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete community user ban: %w", err)
	}

	return nil
}

func (uc *banUsecase) MuteUserInCommunity(ctx context.Context, userID, communityID int) (*ent.CommunityUserMute, error) {
	// Проверяем, что пользователь не замьючен уже
	exists, err := uc.client.CommunityUserMute.Query().
		Where(
			communityusermute.UserIDEQ(userID),
			communityusermute.CommunityIDEQ(communityID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing mute: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user already muted in community")
	}

	// Создаем мут
	mute, err := uc.client.CommunityUserMute.Create().
		SetUserID(userID).
		SetCommunityID(communityID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create community user mute: %w", err)
	}

	return mute, nil
}

func (uc *banUsecase) UnmuteUserInCommunity(ctx context.Context, muteID int) error {
	// Проверяем, что мут существует
	mute, err := uc.client.CommunityUserMute.Get(ctx, muteID)
	if err != nil {
		return fmt.Errorf("mute not found: %w", err)
	}

	// Удаляем мут
	err = uc.client.CommunityUserMute.DeleteOne(mute).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete community user mute: %w", err)
	}

	return nil
}

func (uc *banUsecase) GetHostUserBans(ctx context.Context) ([]*ent.HostUserBan, error) {
	return uc.client.HostUserBan.Query().
		WithUser().
		Order(ent.Desc("created_at")).
		All(ctx)
}

func (uc *banUsecase) GetHostCommunityBans(ctx context.Context) ([]*ent.HostCommunityBan, error) {
	return uc.client.HostCommunityBan.Query().
		WithCommunity().
		Order(ent.Desc("created_at")).
		All(ctx)
}

func (uc *banUsecase) GetCommunityUserBans(ctx context.Context, communityID int) ([]*ent.CommunityUserBan, error) {
	return uc.client.CommunityUserBan.Query().
		Where(communityuserban.CommunityIDEQ(communityID)).
		WithUser().
		Order(ent.Desc("created_at")).
		All(ctx)
}

func (uc *banUsecase) GetCommunityUserMutes(ctx context.Context, communityID int) ([]*ent.CommunityUserMute, error) {
	return uc.client.CommunityUserMute.Query().
		Where(communityusermute.CommunityIDEQ(communityID)).
		WithUser().
		Order(ent.Desc("created_at")).
		All(ctx)
}
