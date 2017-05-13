package cmd

import (
	"github.com/spf13/cobra"
	"errors"
	"turbo"
)

var createCmd = &cobra.Command{
	Use:   "create package_path ServiceName",
	Aliases: []string{"c"},
	Short: "Create a project with runnable HTTP server and gRPC/thrift server",
	Example: "turbo create package/path/to/yourservice YourService -p grpc\n" +
		"'ServiceName' must be a CamelCase string",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("invalid args")
		}
		// TODO assert that args[1] must be a CamelCase string
		turbo.CreateProject(args[0], args[1], c_rpcType)
		return nil
	},
}

var c_rpcType string

func init() {
	createCmd.Flags().StringVarP(&c_rpcType, "rpctype", "r", "grpc", "[grpc|thrift]")
	RootCmd.AddCommand(createCmd)
}
