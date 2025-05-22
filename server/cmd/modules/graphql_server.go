package modules

import (
	"log"
	"net/http"

	"stormlink/server/ent"
	"stormlink/server/graphql"
	"stormlink/server/usecase"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func StartGraphQLServer(client *ent.Client) {
	uc := usecase.NewUserUsecase(client)

	resolver := &graphql.Resolver{
		Client: client,
		UC:     uc,
	}

	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("GraphQL", "/query"))
	mux.Handle("/query", srv)

	log.Println("🚀 GraphQL-сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("ошибка при запуске GraphQL-сервера: %v", err)
	}
}
