package permissionloader

import (
	"context"
	"fmt"
	"stormlink/server/model"
	"stormlink/server/usecase"
)

type contextKey string

const ContextKey contextKey = "perm_loader"

// PermLoader держит в себе кэш Permissions по communityID
type PermLoader struct {
  uc usecase.UserUsecase
  cache map[int]*model.CommunityPermissions
}

// NewPermLoader создаёт новый загрузчик
func NewPermLoader(uc usecase.UserUsecase) *PermLoader {
  return &PermLoader{
    uc:    uc,
    cache: make(map[int]*model.CommunityPermissions),
  }
}

// LoadAll заполняет loader.cache правами во всех нужных сообществах
func (l *PermLoader) LoadAll(ctx context.Context, userID int, communityIDs []int) error {
  // вызываем единый usecase, который вернёт map[communityID]*model.Permissions
  permsMap, err := l.uc.GetPermissionsByCommunities(ctx, userID, communityIDs)
  if err != nil {
    return fmt.Errorf("failed to batch-load permissions: %w", err)
  }
  // сохраняем в кэш
  for cid, perms := range permsMap {
    l.cache[cid] = perms
  }
  // для сообществ без записи — возвращаем пустые права
  empty := &model.CommunityPermissions{}
  for _, cid := range communityIDs {
    if _, ok := l.cache[cid]; !ok {
      l.cache[cid] = empty
    }
  }
  return nil
}

// ForCommunity возвращает права для одного communityID
func (l *PermLoader) ForCommunity(cid int) *model.CommunityPermissions {
  return l.cache[cid]
}
