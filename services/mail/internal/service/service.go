package service

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"

	"stormlink/server/ent"
	entev "stormlink/server/ent/emailverification"
	entu "stormlink/server/ent/user"
	mailpb "stormlink/server/grpc/mail/protobuf"
	errorsx "stormlink/shared/errors"
	"stormlink/shared/jwt"
	sharedmail "stormlink/shared/mail"
)

type MailService struct {
    mailpb.UnimplementedMailServiceServer
    client *ent.Client
}

func NewMailService(client *ent.Client) *MailService {
    return &MailService{client: client}
}

func (s *MailService) VerifyEmail(ctx context.Context, req *mailpb.VerifyEmailRequest) (*mailpb.VerifyEmailResponse, error) {
    token := req.GetToken()
    if token == "" {
        return nil, errorsx.FromGRPCCode(codes.InvalidArgument, "token is required", nil)
    }
    ev, err := s.client.EmailVerification.Query().Where(entev.TokenEQ(token)).WithUser().Only(ctx)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.NotFound, "invalid or expired token", nil) }
    if time.Now().After(ev.ExpiresAt) {
        _ = s.client.EmailVerification.DeleteOne(ev).Exec(ctx)
        return nil, errorsx.FromGRPCCode(codes.DeadlineExceeded, "verification token has expired", nil)
    }
    if _, err := s.client.User.UpdateOne(ev.Edges.User).SetIsVerified(true).Save(ctx); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "failed to verify user", err)
    }
    _ = s.client.EmailVerification.DeleteOne(ev).Exec(ctx)
    return &mailpb.VerifyEmailResponse{Message: "Почта успешно подтверждена."}, nil
}

func (s *MailService) ResendVerifyEmail(ctx context.Context, req *mailpb.ResendVerifyEmailRequest) (*mailpb.ResendVerifyEmailResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, errorsx.FromGRPCCode(codes.InvalidArgument, "validation error", err)
    }
    email := req.GetEmail()
    u, err := s.client.User.Query().Where(entu.EmailEQ(email)).Only(ctx)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.NotFound, "user not found", nil) }
    if u.IsVerified { return nil, errorsx.FromGRPCCode(codes.FailedPrecondition, "user already verified", nil) }
    if _, err := s.client.EmailVerification.Delete().Where(entev.HasUserWith(entu.EmailEQ(email))).Exec(ctx); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "failed to clear old verification tokens", err)
    }
    token, err := jwt.GenerateToken(16)
    if err != nil { return nil, errorsx.FromGRPCCode(codes.Internal, "failed to generate verification token", err) }
    expiresAt := time.Now().Add(24 * time.Hour)
    if _, err := s.client.EmailVerification.Create().SetToken(token).SetExpiresAt(expiresAt).SetUser(u).Save(ctx); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "failed to save verification token", err)
    }
    if err := sharedmail.SendVerifyEmail(u.Email, token); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Internal, "failed to send email", err)
    }
    return &mailpb.ResendVerifyEmailResponse{Message: "Verification email sent successfully."}, nil
}


