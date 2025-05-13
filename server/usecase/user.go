package usecase

import (
	"context"
	"github.com/google/uuid"
	"stormlink/server/ent"
)

type UserUsecase interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*ent.User, error)
}

type userUsecase struct {
	client *ent.Client
}

func NewUserUsecase(client *ent.Client) UserUsecase {
	return &userUsecase{client: client}
}

func (uc *userUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	return uc.client.User.Get(ctx, id)
}
