package mapper

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/auth/protobuf"
	"strconv"
	"time"
)

func UserToProto(u *ent.User) *protobuf.User {
    // Маппим UserInfo
    var userInfo []*protobuf.UserInfo
    if u.Edges.UserInfo != nil {
        for _, info := range u.Edges.UserInfo {
            userInfo = append(userInfo, &protobuf.UserInfo{
                Id:    strconv.Itoa(info.ID),
                Key:   info.Key,
                Value: info.Value,
            })
        }
    }

    // Маппим Avatar
    var avatar *protobuf.Avatar
    if u.Edges.Avatar != nil && u.Edges.Avatar.URL != nil {
        avatar = &protobuf.Avatar{
						Id: strconv.Itoa(u.Edges.Avatar.ID),
            Url: *u.Edges.Avatar.URL,
        }
    }

    // Маппим HostRoles
    var hostRoles []*protobuf.HostRole
    if u.Edges.HostRoles != nil {
        for _, role := range u.Edges.HostRoles {
            hostRoles = append(hostRoles, &protobuf.HostRole{
                Id:                                strconv.Itoa(role.ID),
                Title:                             role.Title,
                Color:                             *role.Color,
                CommunityRolesManagement:          role.CommunityRolesManagement,
                HostUserBan:                       role.HostUserBan,
                HostUserMute:                      role.HostUserMute,
                HostCommunityDeletePost:           role.HostCommunityDeletePost,
                HostCommunityDeleteComments:       role.HostCommunityDeleteComments,
                HostCommunityRemovePostFromPublication: role.HostCommunityRemovePostFromPublication,
            })
        }
    }

    // Маппим CommunitiesRoles
    var communitiesRoles []*protobuf.CommunityRole
    if u.Edges.CommunitiesRoles != nil {
        for _, role := range u.Edges.CommunitiesRoles {
            communitiesRoles = append(communitiesRoles, &protobuf.CommunityRole{
                Id:                                strconv.Itoa(role.ID),
                Title:                             role.Title,
                Color:                             *role.Color,
                CommunityRolesManagement:          role.CommunityRolesManagement,
                CommunityUserBan:                  role.CommunityUserBan,
                CommunityUserMute:                 role.CommunityUserMute,
                CommunityDeletePost:               role.CommunityDeletePost,
                CommunityDeleteComments:           role.CommunityDeleteComments,
                CommunityRemovePostFromPublication: role.CommunityRemovePostFromPublication,
            })
        }
    }

    return &protobuf.User{
        Id:              strconv.Itoa(u.ID),
        Name:            u.Name,
        Slug:            u.Slug,
        Avatar:          avatar,
        Email:           u.Email,
        Description:     u.Description,
        UserInfo:        userInfo,
        HostRoles:       hostRoles,
        CommunitiesRoles: communitiesRoles,
        IsVerified:      u.IsVerified,
        CreatedAt:       u.CreatedAt.Format(time.RFC3339),
        UpdatedAt:       u.UpdatedAt.Format(time.RFC3339),
    }
}