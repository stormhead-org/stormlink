package mapper

import (
	"strconv"
	"time"

	"stormlink/server/ent"
	authpb "stormlink/server/grpc/auth/protobuf"
)

func UserToProto(u *ent.User) *authpb.User {
    var userInfo []*authpb.UserInfo
    if u.Edges.UserInfo != nil {
        for _, info := range u.Edges.UserInfo {
            userInfo = append(userInfo, &authpb.UserInfo{Id: strconv.Itoa(info.ID), Key: info.Key, Value: info.Value})
        }
    }
    var avatar *authpb.Avatar
    if u.Edges.Avatar != nil && u.Edges.Avatar.URL != nil {
        avatar = &authpb.Avatar{Id: strconv.Itoa(u.Edges.Avatar.ID), Url: *u.Edges.Avatar.URL}
    }
    var hostRoles []*authpb.HostRole
    if u.Edges.HostRoles != nil {
        for _, role := range u.Edges.HostRoles {
            hostRoles = append(hostRoles, &authpb.HostRole{
                Id:   strconv.Itoa(role.ID),
                Title: role.Title,
                Color: *role.Color,
                CommunityRolesManagement: role.CommunityRolesManagement,
                HostUserBan: role.HostUserBan,
                HostUserMute: role.HostUserMute,
                HostCommunityDeletePost: role.HostCommunityDeletePost,
                HostCommunityDeleteComments: role.HostCommunityDeleteComments,
                HostCommunityRemovePostFromPublication: role.HostCommunityRemovePostFromPublication,
            })
        }
    }
    var communitiesRoles []*authpb.CommunityRole
    if u.Edges.CommunitiesRoles != nil {
        for _, role := range u.Edges.CommunitiesRoles {
            communitiesRoles = append(communitiesRoles, &authpb.CommunityRole{
                Id:   strconv.Itoa(role.ID),
                Title: role.Title,
                Color: *role.Color,
                CommunityRolesManagement: role.CommunityRolesManagement,
                CommunityUserBan: role.CommunityUserBan,
                CommunityUserMute: role.CommunityUserMute,
                CommunityDeletePost: role.CommunityDeletePost,
                CommunityDeleteComments: role.CommunityDeleteComments,
                CommunityRemovePostFromPublication: role.CommunityRemovePostFromPublication,
            })
        }
    }
    return &authpb.User{
        Id: strconv.Itoa(u.ID),
        Name: u.Name,
        Slug: u.Slug,
        Avatar: avatar,
        Email: u.Email,
        Description: u.Description,
        UserInfo: userInfo,
        HostRoles: hostRoles,
        CommunitiesRoles: communitiesRoles,
        IsVerified: u.IsVerified,
        CreatedAt: u.CreatedAt.Format(time.RFC3339),
        UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
    }
}


