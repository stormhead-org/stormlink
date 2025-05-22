package converter

import (
	"stormlink/server/model"
)

func ConvertPermissionsToCommunityPermissions(p *model.CommunityPermissions) *model.CommunityPermissions {
	if p == nil {
			return &model.CommunityPermissions{}
	}
	return &model.CommunityPermissions{
			CommunityRolesManagement:            p.CommunityRolesManagement,
			CommunityUserBan:                    p.CommunityUserBan,
			CommunityUserMute:                   p.CommunityUserMute,
			CommunityDeletePost:                 p.CommunityDeletePost,
			CommunityDeleteComments:             p.CommunityDeleteComments,
			CommunityRemovePostFromPublication: p.CommunityRemovePostFromPublication,
			CommunityOwner:                      p.CommunityOwner,
			HostOwner:                          p.HostOwner,
			CommunityUserHasBanned:              p.CommunityUserHasBanned,
			CommunityUserHasMuted:               p.CommunityUserHasMuted,
	}
}
