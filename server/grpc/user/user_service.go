package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"
	"stormlink/server/ent/host"
	"stormlink/server/ent/hostrole"
	"stormlink/server/pkg/auth"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"stormlink/server/ent"
	"stormlink/server/ent/user"
	"stormlink/server/grpc/user/protobuf"
	"stormlink/server/pkg/mapper"
	"stormlink/server/usecase"
	"stormlink/server/utils"
)

type UserService struct {
	protobuf.UnimplementedUserServiceServer
	client *ent.Client
	uc     usecase.UserUsecase
}

func NewUserService(client *ent.Client, uc usecase.UserUsecase) *UserService {
	return &UserService{
		client: client,
		uc:     uc,
	}
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

	hostFirstSettingsRecord, err := s.client.Host.
		Query().
		Where(host.IDEQ(1)).
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get host: %v", err)
	}

	firstSettings := hostFirstSettingsRecord.FirstSettings

	if firstSettings {
		ownerRole, err := s.client.HostRole.
			Query().
			Where(hostrole.NameEQ("owner")).
			Only(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find @everyone role: %v", err)
		}

		err = s.client.User.
			UpdateOne(newUser).
			AddHostRoles(ownerRole).
			Exec(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to assign @everyone role: %v", err)
		}
	}

	everyoneRole, err := s.client.HostRole.
		Query().
		Where(hostrole.NameEQ("@everyone")).
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find @everyone role: %v", err)
	}

	err = s.client.User.
		UpdateOne(newUser).
		AddHostRoles(everyoneRole).
		Exec(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign @everyone role: %v", err)
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

func (s *UserService) GetMe(ctx context.Context, _ *emptypb.Empty) (*protobuf.UserResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated: %v", err)
	}

	user, err := s.uc.GetUserByID(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &protobuf.UserResponse{
		User: mapper.UserToProto(user),
	}, nil
}
