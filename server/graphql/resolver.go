package graphql

import (
	"stormlink/server/ent"
	"stormlink/server/usecase"
)

type Resolver struct {
	Client *ent.Client
	UC     usecase.UserUsecase
}