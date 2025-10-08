package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/stormhead-org/stormic-backend/iam/internal/ent"
)

var debugCommand = &cobra.Command{
	Use:   "debug",
	Short: "debug",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return debugCommandImpl()
	},
}

func debugCommandImpl() error {
	// os.Getenv("DEBUG") == "1"
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("can't load .env file: %w", err)
	}

	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "127.0.0.1"
	}

	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}

	postgresUser := os.Getenv("POSTGRES_USER")
	if postgresUser == "" {
		postgresUser = "postgres"
	}

	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	if postgresPassword == "" {
		postgresPassword = ""
	}

	postgresDatabase := os.Getenv("POSTGRES_DATABASE")
	if postgresDatabase == "" {
		postgresDatabase = "postgres"
	}

	database, err := sql.Open(
		"pgx",
		fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s",
			postgresUser,
			postgresPassword,
			postgresHost,
			postgresPort,
			postgresDatabase,
		),
	)
	if err != nil {
		return err
	}

	client := ent.NewClient(
		ent.Driver(
			entsql.OpenDB(dialect.Postgres, database),
		),
	)

	err = client.Schema.Create(context.Background())
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = client.Schema.Create(ctx)
	if err != nil {
		return err
	}

	user, err := client.User.Create().SetAge(30).SetName("Ivan").Save(ctx)
	if err != nil {
		return err
	}

	fmt.Println(user)

	users, err := client.User.
		Query().
		All(ctx)

	if err != nil {
		return err
	}

	fmt.Println(users)

	return nil
}

func init() {
	rootCommand.AddCommand(debugCommand)
}
