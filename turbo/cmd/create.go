package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/vaporz/turbo"
)

var createCmd = &cobra.Command{
	Use:     "create package_path ServiceName",
	Aliases: []string{"c"},
	Short:   "Create a project with runnable HTTP server and gRPC/thrift server",
	Example: "turbo create package/path/to/yourservice YourService -r grpc\n" +
		"'ServiceName' *MUST* be a CamelCase string",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("invalid args")
		}
		if turbo.IsNotCamelCase(args[1]) {
			return errors.New("[" + args[1] + "] is not a CamelCase string")
		}
		g := turbo.Creator{
			RpcType: RpcType,
			PkgPath: args[0],
		}
		g.CreateProject(args[1], force)
		return nil
	},
}

var force bool

func init() {
	createCmd.Flags().StringVarP(&RpcType, "rpctype", "r", "grpc", "[grpc|thrift]")
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "create service and override existing files")
	RootCmd.AddCommand(createCmd)
}
