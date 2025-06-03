package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"stormlink/server/ent/host"
	"stormlink/server/ent/hostrole"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"stormlink/server/ent/user"
	"stormlink/server/grpc/user/protobuf"
	"stormlink/server/pkg/jwt"
	"stormlink/server/pkg/rabbitmq"
)

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
	salt := jwt.GenerateSalt()
	rawHash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()+salt), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}
	passwordHash := base64.StdEncoding.EncodeToString(rawHash)

	// Создаем пользователя с is_verified = false
	newUser, err := s.client.User.
		Create().
		SetName(req.GetName()).
		SetSlug(req.GetName()).
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
		err = s.client.Host.
      UpdateOne(hostFirstSettingsRecord).
      SetOwnerID(newUser.ID).
      SetFirstSettings(false).
      Exec(ctx)
      if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to update host owner and first_settings: %v", err)
    }

		ownerRole, err := s.client.HostRole.
			Query().
			Where(hostrole.TitleEQ("owner")).
			Only(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find host owner role: %v", err)
		}

		err = s.client.User.
			UpdateOne(newUser).
			AddHostRoles(ownerRole).
			Exec(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to assign host owner role: %v", err)
		}
	}

	everyoneRole, err := s.client.HostRole.
		Query().
		Where(hostrole.TitleEQ("@everyone")).
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
	token, err := jwt.GenerateToken(16)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate verification token: %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = s.client.EmailVerification.
		Create().
		SetToken(token).
		SetExpiresAt(expiresAt).
		SetUser(newUser).
		Save(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create email verification record: %v", err)
	}

	// Публикуем задачу в RabbitMQ
	job := rabbitmq.EmailJob{
		To:    newUser.Email,
		Token: token,
	}
	if err := rabbitmq.PublishEmailJob(job); err != nil {
		return nil, status.Errorf(codes.Internal, "Не удалось поставить задачу на отправку письма: %v", err)
	}

	// Возвращаем успешный ответ
	return &protobuf.RegisterUserResponse{
		UserId:  strconv.Itoa(newUser.ID),
		Message: "User registered successfully. Please check your email to verify your account.",
	}, nil
}