//go:build ignore
// +build ignore

package main

import (
	"log"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	ex, err := entgql.NewExtension(
		entgql.WithSchemaGenerator(),                    // генерируем SDL
		entgql.WithSchemaPath("../graphql/ent.graphql"), // сохраняем его в server/graphql/ent.graphql
		entgql.WithWhereInputs(true),                    // опционально: фильтры
		//entgql.WithNodeDescriptor(true),                         // Relay-дескрипторы
	)
	if err != nil {
		log.Fatalf("failed creating entgql extension: %v", err)
	}
	if err := entc.Generate("./schema", &gen.Config{},
		entc.Extensions(ex),
	); err != nil {
		log.Fatalf("ent codegen failed: %v", err)
	}
}
