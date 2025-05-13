package auth

import (
	"context"
	"stormlink/server/ent/emailverification"
	"stormlink/server/ent/user"
	"stormlink/server/utils"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"stormlink/server/grpc/auth/protobuf"
)

func (s *AuthService) Login(ctx context.Context, req *protobuf.LoginRequest) (*protobuf.LoginResponse, error) {
	// ✅ Проверяем входные данные
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	email := req.GetEmail()
	password := req.GetPassword()

	// Ищем пользователя по email
	user, err := s.client.User.
		Query().
		Where(user.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// Проверяем пароль
	err = utils.ComparePassword(user.PasswordHash, password, user.Salt)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// TODO раскоментить при реализации верификации на фронте
	//Проверяем, верифицирован ли email
	if !user.IsVerified {
		return nil, status.Errorf(codes.FailedPrecondition, "user email not verified")
	}

	// Генерируем токены
	accessToken, err := utils.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error generating access token: %v", err)
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error generating access token: %v", err)
	}

	return &protobuf.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) ResendVerificationEmail(ctx context.Context, req *protobuf.ResendVerificationRequest) (*protobuf.ResendVerificationResponse, error) {
	// ✅ Проверяем входные данные
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	email := req.GetEmail()

	// 🔍 Ищем пользователя по email
	u, err := s.client.User.
		Query().
		Where(user.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	// ⛔ Проверяем, что пользователь ещё не верифицирован
	if u.IsVerified {
		return nil, status.Errorf(codes.FailedPrecondition, "user already verified")
	}

	// Удаляем предыдущие токены
	_, err = s.client.EmailVerification.
		Delete().
		Where(emailverification.HasUserWith(user.EmailEQ(email))).
		Exec(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clear old verification tokens: %v", err)
	}

	// 🔐 Генерируем новый токен
	token, err := utils.GenerateToken(16)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate verification token: %v", err)
	}

	// 🕓 Сохраняем новый токен
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = s.client.EmailVerification.
		Create().
		SetToken(token).
		SetExpiresAt(expiresAt).
		SetUser(u).
		Save(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save verification token: %v", err)
	}

	// 📧 Отправляем письмо
	err = utils.SendVerificationEmail(u.Email, token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send email: %v", err)
	}

	return &protobuf.ResendVerificationResponse{
		Message: "Verification email sent successfully.",
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req *protobuf.VerifyEmailRequest) (*protobuf.VerifyEmailResponse, error) {
	token := req.GetToken()
	if token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "token is required")
	}

	// 🔍 Ищем запись по токену
	ev, err := s.client.EmailVerification.
		Query().
		Where(emailverification.TokenEQ(token)).
		WithUser().
		Only(ctx)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "invalid or expired token")
	}

	// ⏰ Проверяем срок действия токена
	if time.Now().After(ev.ExpiresAt) {
		// Удаляем истёкший токен
		_ = s.client.EmailVerification.DeleteOne(ev).Exec(ctx)
		return nil, status.Errorf(codes.DeadlineExceeded, "verification token has expired")
	}

	// ✅ Обновляем пользователя
	_, err = s.client.User.
		UpdateOne(ev.Edges.User).
		SetIsVerified(true).
		Save(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify user: %v", err)
	}

	// 🧹 Удаляем использованный токен
	_ = s.client.EmailVerification.DeleteOne(ev).Exec(ctx)

	return &protobuf.VerifyEmailResponse{
		Message: "Email verified successfully.",
	}, nil
}
