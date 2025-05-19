package graphql

import (
	"stormlink/server/ent"
)

type Resolver struct {
	Client *ent.Client
}