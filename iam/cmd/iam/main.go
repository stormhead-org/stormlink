package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:   "iam",
	Short: "iam",
	Long:  "iam",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCommandImplementation()
	},
}

func rootCommandImplementation() error {
	var logo []string = []string{
		"Stormic IAM microservice",
	}
	for _, l := range logo {
		fmt.Println(l)
	}

	return nil
}

func main() {
	err := rootCommand.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
