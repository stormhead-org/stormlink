package mapper

import (
	"fmt"
	"stormlink/server/graphql/models"
	authpb "stormlink/server/grpc/auth/protobuf"
)

// ProtoToGraphQLUser converts a proto User to a GraphQL UserResponse
func ProtoToGraphQLUser(protoUser *authpb.User) (*models.UserResponse, error) {
    if protoUser == nil {
        return nil, fmt.Errorf("proto user is nil")
    }

    // Маппинг аватара
    var avatar *models.UserAvatarResponse
    if protoUser.Avatar != nil {
        avatar = &models.UserAvatarResponse{
            ID:  protoUser.Avatar.Id,
            URL: protoUser.Avatar.Url,
        }
    }

    // Маппинг user info
    userInfo := make([]*models.UserInfoResponse, len(protoUser.UserInfo))
    for i, info := range protoUser.UserInfo {
        userInfo[i] = &models.UserInfoResponse{
            ID:    info.Id,
            Key:   info.Key,
            Value: info.Value,
        }
    }

    // Маппинг host roles
    hostRoles := make([]*models.UserHostRoleResponse, len(protoUser.HostRoles))
    for i, role := range protoUser.HostRoles {
        hostRoles[i] = &models.UserHostRoleResponse{
            ID:                                 role.Id,
            Title:                              role.Title,
            Color:                              role.Color,
            CommunityRolesManagement:           role.CommunityRolesManagement,
            HostUserBan:                        role.HostUserBan,
            HostUserMute:                      role.HostUserMute,
            HostCommunityDeletePost:            role.HostCommunityDeletePost,
            HostCommunityDeleteComments:          role.HostCommunityDeleteComments,
            HostCommunityRemovePostFromPublication: role.HostCommunityRemovePostFromPublication,
        }
    }

    // Маппинг community roles
    communityRoles := make([]*models.UserCommunityRoleResponse, len(protoUser.CommunitiesRoles))
    for i, role := range protoUser.CommunitiesRoles {
        communityRoles[i] = &models.UserCommunityRoleResponse{
            ID:                                 role.Id,
            Title:                              role.Title,
            Color:                              role.Color,
            CommunityRolesManagement:           role.CommunityRolesManagement,
            CommunityUserBan:                   role.CommunityUserBan,
            CommunityUserMute:                  role.CommunityUserMute,
            CommunityDeletePost:                role.CommunityDeletePost,
            CommunityDeleteComments:              role.CommunityDeleteComments,
            CommunityRemovePostFromPublication: role.CommunityRemovePostFromPublication,
        }
    }

    return &models.UserResponse{
        ID:              protoUser.Id,
        Name:            protoUser.Name,
        Slug:            protoUser.Slug,
        Avatar:          avatar,
        Email:           protoUser.Email,
        Description:     protoUser.Description,
        UserInfo:        userInfo,
        HostRoles:       hostRoles,
        CommunitiesRoles: communityRoles,
        IsVerified:      protoUser.IsVerified,
        CreatedAt:       protoUser.CreatedAt,
        UpdatedAt:       protoUser.UpdatedAt,
    }, nil
}