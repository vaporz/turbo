package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

var generateCmd = &cobra.Command{
	Use:     "generate [package_path] [service_name]",
	Aliases: []string{"g"},
	Short: "Generate Golang codes according to service.yaml and [service_name].proto",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("Usage: generate [package_path] [service_name]")
		}
		turbo.Generate(args[0], args[1])
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
}
