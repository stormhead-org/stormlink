package model

type CommunityPermissions struct {
	CommunityRolesManagement           bool `json:"communityRolesManagement"`
	CommunityUserBan                   bool `json:"communityUserBan"`
	CommunityUserMute                  bool `json:"communityUserMute"`
	CommunityDeletePost                bool `json:"communityDeletePost"`
	CommunityDeleteComments            bool `json:"communityDeleteComments"`
	CommunityRemovePostFromPublication bool `json:"communityRemovePostFromPublication"`
	CommunityOwner                     bool `json:"communityOwner"`
	HostOwner                          bool `json:"hostOwner"`
}
