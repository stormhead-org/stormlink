package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"stormlink/server/ent"
	"stormlink/server/ent/user"
	"stormlink/server/grpc/user/protobuf"
	"stormlink/server/utils"
)

type UserService struct {
	protobuf.UnimplementedUserServiceServer
	client *ent.Client
}

func NewUserService(client *ent.Client) *UserService {
	return &UserService{client: client}
}

func (s *UserService) RegisterUser(ctx context.Context, req *protobuf.RegisterUserRequest) (*protobuf.RegisterUserResponse, error) {
	// Валидация входных данных
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	// Проверка: существует ли уже пользователь с таким email
	exists, err := s.client.User.
		Query().
		Where(user.EmailEQ(req.GetEmail())).
		Exist(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check existing email: %v", err)
	}
	if exists {
		return nil, status.Errorf(codes.AlreadyExists, "email already in use")
	}

	// Хешируем пароль
	salt := utils.GenerateSalt()
	rawHash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()+salt), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}
	passwordHash := base64.StdEncoding.EncodeToString(rawHash)

	// Создаем пользователя с is_verified = false
	newUser, err := s.client.User.
		Create().
		SetName(req.GetName()).
		SetEmail(req.GetEmail()).
		SetPasswordHash(passwordHash).
		SetSalt(salt).
		SetIsVerified(false).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %v", err)
	}

	// Генерируем токен верификации
	token, err := utils.GenerateToken(16) // генерируем шестнадцатеричный токен нужной длины
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate verification token: %v", err)
	}

	// Определяем время истечения (например, 24 часа)
	expiresAt := time.Now().Add(24 * time.Hour)
	// Создаем запись в таблице EmailVerification
	_, err = s.client.EmailVerification.
		Create().
		SetToken(token).
		SetExpiresAt(expiresAt).
		SetUser(newUser).
		Save(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create email verification record: %v", err)
	}

	// Отправляем письмо с верификацией
	err = utils.SendVerificationEmail(newUser.Email, token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send verification email: %v", err)
	}

	// Возвращаем успешный ответ
	return &protobuf.RegisterUserResponse{
		UserId:  strconv.Itoa(newUser.ID),
		Message: "User registered successfully. Please check your email to verify your account.",
	}, nil
}
