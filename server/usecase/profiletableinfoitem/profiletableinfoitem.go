package profiletableinfoitem

import (
	"context"
	"fmt"
	"strconv"

	"stormlink/server/ent"
	"stormlink/server/ent/profiletableinfoitem"
	"stormlink/server/graphql/models"
)

type ProfileTableInfoItemUsecase interface {
	CreateProfileTableInfoItem(ctx context.Context, input *models.CreateProfileTableInfoItemInput) (*ent.ProfileTableInfoItem, error)
	UpdateProfileTableInfoItem(ctx context.Context, input *models.UpdateProfileTableInfoItemInput) (*ent.ProfileTableInfoItem, error)
	DeleteProfileTableInfoItem(ctx context.Context, id int) error
	GetProfileTableInfoItem(ctx context.Context, id int) (*ent.ProfileTableInfoItem, error)
	GetProfileTableInfoItems(ctx context.Context, entityID int, itemType profiletableinfoitem.Type) ([]*ent.ProfileTableInfoItem, error)
}

type profileTableInfoItemUsecase struct {
	client *ent.Client
}

func NewProfileTableInfoItemUsecase(client *ent.Client) ProfileTableInfoItemUsecase {
	return &profileTableInfoItemUsecase{client: client}
}

func (uc *profileTableInfoItemUsecase) CreateProfileTableInfoItem(ctx context.Context, input *models.CreateProfileTableInfoItemInput) (*ent.ProfileTableInfoItem, error) {
	create := uc.client.ProfileTableInfoItem.Create().
		SetKey(input.Key).
		SetValue(input.Value).
		SetType(input.Type)

	// Устанавливаем связь с сообществом или пользователем
	if input.CommunityID != nil {
		cid, err := strconv.Atoi(*input.CommunityID)
		if err != nil {
			return nil, fmt.Errorf("invalid communityID: %w", err)
		}
		create = create.SetCommunityID(cid)
	}

	if input.UserID != nil {
		uid, err := strconv.Atoi(*input.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid userID: %w", err)
		}
		create = create.SetUserID(uid)
	}

	return create.Save(ctx)
}

func (uc *profileTableInfoItemUsecase) UpdateProfileTableInfoItem(ctx context.Context, input *models.UpdateProfileTableInfoItemInput) (*ent.ProfileTableInfoItem, error) {
	id, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	
	update := uc.client.ProfileTableInfoItem.UpdateOneID(id)

	if input.Key != nil {
		update = update.SetKey(*input.Key)
	}

	if input.Value != nil {
		update = update.SetValue(*input.Value)
	}

	return update.Save(ctx)
}

func (uc *profileTableInfoItemUsecase) DeleteProfileTableInfoItem(ctx context.Context, id int) error {
	return uc.client.ProfileTableInfoItem.DeleteOneID(id).Exec(ctx)
}

func (uc *profileTableInfoItemUsecase) GetProfileTableInfoItem(ctx context.Context, id int) (*ent.ProfileTableInfoItem, error) {
	return uc.client.ProfileTableInfoItem.Get(ctx, id)
}

func (uc *profileTableInfoItemUsecase) GetProfileTableInfoItems(ctx context.Context, entityID int, itemType profiletableinfoitem.Type) ([]*ent.ProfileTableInfoItem, error) {
	query := uc.client.ProfileTableInfoItem.Query().Where(profiletableinfoitem.TypeEQ(itemType))

	// Фильтруем по типу сущности
	switch itemType {
	case profiletableinfoitem.TypeCommunity:
		query = query.Where(profiletableinfoitem.CommunityIDEQ(entityID))
	case profiletableinfoitem.TypeUser:
		query = query.Where(profiletableinfoitem.UserIDEQ(entityID))
	}

	return query.All(ctx)
}
