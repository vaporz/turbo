package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

var createCmd = &cobra.Command{
	Use:   "create [package_name] [service_name]",
	Short: "Create a project with runnable HTTP server and gRPC server",
	Long:  "Create a project with runnable HTTP server and gRPC server, " +
		"'turbo create' = 'turbo init' + 'turbo generate'",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("Usage: create [package_name] [service_name]")
		}
		turbo.Init(args[0], args[1])
		turbo.Generate(args[0], args[1])
		return nil
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
}
