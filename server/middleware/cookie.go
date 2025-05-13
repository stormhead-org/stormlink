package middleware

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	authpb "stormlink/server/grpc/auth/protobuf"
)

func CookieInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, err
		}

		// Устанавливаем куки только для Login
		if info.FullMethod == "/auth.AuthService/Login" {
			if loginResp, ok := resp.(*authpb.LoginResponse); ok {
				cookie := fmt.Sprintf(
					"auth-token=%s; HttpOnly; Path=/; Max-Age=900; SameSite=Strict",
					loginResp.AccessToken,
				)
				if err := grpc.SetHeader(ctx, metadata.Pairs("Set-Cookie", cookie)); err != nil {
					return nil, err
				}
			}
		}

		return resp, err
	}
}
