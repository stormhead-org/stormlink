package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"

	"stormlink/server/ent"
	enth "stormlink/server/ent/host"
	enthr "stormlink/server/ent/hostrole"
	entu "stormlink/server/ent/user"
	userpb "stormlink/server/grpc/user/protobuf"
	useruc "stormlink/server/usecase/user"
	errorsx "stormlink/shared/errors"
	"stormlink/shared/jwt"
	"stormlink/shared/rabbitmq"
)

type UserService struct {
    userpb.UnimplementedUserServiceServer
    client *ent.Client
    uc     useruc.UserUsecase
}

func NewUserService(client *ent.Client, uc useruc.UserUsecase) *UserService {
    return &UserService{client: client, uc: uc}
}

func (s *UserService) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.RegisterUserResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, errorsx.FromGRPCCode(codes.InvalidArgument, "validation error", err)
    }
    exists, err := s.client.User.Query().Where(entu.EmailEQ(req.GetEmail())).Exist(ctx)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "failed to check existing email", err) }
    if exists { return nil, errorsx.FromGRPCCode(codes.AlreadyExists, "email already in use", nil) }

    salt := jwt.GenerateSalt()
    rawHash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()+salt), bcrypt.DefaultCost)
    if err != nil { return nil, fmt.Errorf("error hashing password: %v", err) }
    passwordHash := base64.StdEncoding.EncodeToString(rawHash)

    newUser, err := s.client.User.Create().
        SetName(req.GetName()).
        SetSlug(req.GetName()).
        SetEmail(req.GetEmail()).
        SetPasswordHash(passwordHash).
        SetSalt(salt).
        SetIsVerified(false).
        Save(ctx)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "error creating user", err) }

    hostFirst, err := s.client.Host.Query().Where(enth.IDEQ(1)).Only(ctx)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "failed to get host", err) }
    if hostFirst.FirstSettings {
        if err := s.client.Host.UpdateOne(hostFirst).SetOwnerID(newUser.ID).SetFirstSettings(false).Exec(ctx); err != nil {
            return nil, errorsx.FromGRPCCode(codes.Internal, "failed to update host owner and first_settings", err)
        }
        ownerRole, err := s.client.HostRole.Query().Where(enthr.TitleEQ("owner")).Only(ctx)
        if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "failed to find host owner role", err) }
        if err := s.client.User.UpdateOne(newUser).AddHostRoles(ownerRole).Exec(ctx); err != nil {
            return nil, errorsx.FromGRPCCode(codes.Internal, "failed to assign host owner role", err)
        }
    }

    everyone, err := s.client.HostRole.Query().Where(enthr.TitleEQ("@everyone")).Only(ctx)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "failed to find @everyone role", err) }
    if err := s.client.User.UpdateOne(newUser).AddHostRoles(everyone).Exec(ctx); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "failed to assign @everyone role", err)
    }

    token, err := jwt.GenerateToken(16)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "failed to generate verification token", err) }
    expiresAt := time.Now().Add(24 * time.Hour)
    if _, err := s.client.EmailVerification.Create().SetToken(token).SetExpiresAt(expiresAt).SetUser(newUser).Save(ctx); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "failed to create email verification record", err)
    }

    job := rabbitmq.EmailJob{To: newUser.Email, Token: token}
    if err := rabbitmq.PublishEmailJob(job); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "Не удалось поставить задачу на отправку письма", err)
    }

    return &userpb.RegisterUserResponse{UserId: fmt.Sprint(newUser.ID), Message: "User registered successfully. Please check your email to verify your account."}, nil
}


