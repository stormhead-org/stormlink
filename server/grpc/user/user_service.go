package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"stormlink/server/ent"
	"stormlink/server/grpc/user/protobuf"
	"stormlink/server/utils"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	protobuf.UnimplementedUserServiceServer
	client *ent.Client
}

func NewUserService(client *ent.Client) *UserService {
	return &UserService{client: client}
}

func (s *UserService) RegisterUser(ctx context.Context, req *protobuf.RegisterUserRequest) (*protobuf.RegisterUserResponse, error) {
	// Хешируем пароль
	salt := utils.GenerateSalt()
	rawHash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()+salt), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}
	passwordHash := base64.StdEncoding.EncodeToString(rawHash)

	// Создаем пользователя
	newUser, err := s.client.User.
		Create().
		SetName(req.GetName()).
		SetEmail(req.GetEmail()).
		SetPasswordHash(passwordHash).
		SetSalt(salt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %v", err)
	}

	// Возвращаем успешный ответ
	return &protobuf.RegisterUserResponse{
		UserId:  strconv.Itoa(newUser.ID),
		Message: "User registered successfully",
	}, nil
}
