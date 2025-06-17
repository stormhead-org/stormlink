package user

import (
	"context"
	"stormlink/server/ent/community"
	"stormlink/server/ent/hostrole"
	"stormlink/server/ent/role"
	"stormlink/server/ent/user"

	"stormlink/server/model"
)

func (uc *userUsecase) GetPermissionsByCommunities(
  ctx context.Context,
  userID int,
  communityIDs []int,
) (map[int]*model.CommunityPermissions, error) {
  // 1) Получаем роли пользователя во всех нужных сообществах
  roles, _ := uc.client.Role.
    Query().
    Where(
      role.CommunityIDIn(communityIDs...),
      role.HasUsersWith(user.IDEQ(userID)),
    ).
    All(ctx)

  // 3) Проверяем host-роли
  hostOwner := false
  hr, _ := uc.client.User.
    Query().
    Where(user.IDEQ(userID)).
    QueryHostRoles().
    Where(hostrole.TitleEQ("owner")).
    Exist(ctx)
  hostOwner = hr

  // 4) Собираем результат по каждому communityID
  res := make(map[int]*model.CommunityPermissions, len(communityIDs))
  for _, cid := range communityIDs {
    perms := &model.CommunityPermissions{}
    // communityOwner?
    owner, _ := uc.client.Community.
      Query().
      Where(community.IDEQ(cid), community.OwnerIDEQ(userID)).
      Exist(ctx)
    perms.CommunityOwner = owner

    // hostOwner
    perms.HostOwner = hostOwner

    // роли
    for _, r := range roles {
      if r.CommunityID == cid {
        if r.CommunityRolesManagement {
					perms.CommunityRolesManagement = true
				}
				if r.CommunityUserBan {
					perms.CommunityUserBan = true
				}
				if r.CommunityUserMute {
					perms.CommunityUserMute = true
				}
        if r.CommunityDeletePost {
					perms.CommunityDeletePost = true
				}
        if r.CommunityDeleteComments {
					perms.CommunityDeleteComments = true
				}
				if r.CommunityRemovePostFromPublication {
					perms.CommunityRemovePostFromPublication = true
				}
      }
    }
    res[cid] = perms
  }
  return res, nil
}