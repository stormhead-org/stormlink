package mapper

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/user/protobuf"
	"strconv"
	"time"
)

func UserToProto(u *ent.User) *protobuf.User {
	return &protobuf.User{
		Id:         strconv.Itoa(u.ID),
		Name:       u.Name,
		Email:      u.Email,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  u.UpdatedAt.Format(time.RFC3339),
	}
}
