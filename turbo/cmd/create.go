package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vaporz/turbo"
	"errors"
)

var createCmd = &cobra.Command{
	Use:   "create [package_name] [service_name] (grpc|thrift)",
	Short: "Create a project with runnable HTTP server and gRPC/thrift server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 3 {
			return errors.New("Usage: create [package_name] [service_name] (grpc|thrift)")
		}
		turbo.CreateProject(args[0], args[1], args[2])
		return nil
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
}
