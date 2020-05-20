/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/vaporz/turbo"
	"path/filepath"
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
		if len(RpcType) == 0 || (RpcType != "grpc" && RpcType != "thrift") {
			return errors.New("invalid value for -r, should be grpc or thrift")
		}
		p, e := filepath.Abs(FileRootPath)
		if e != nil {
			panic(e)
		}
		g := turbo.Creator{
			RpcType:      RpcType,
			PkgPath:      args[0],
			FileRootPath: p,
		}
		g.CreateProject(args[1], force)
		return nil
	},
}

var force bool

// FileRootPath is the root folder of the created package, "." by default(current directory)
var FileRootPath string

func init() {
	createCmd.Flags().StringVarP(&RpcType, "rpctype", "r", "grpc", "[grpc|thrift]")
	createCmd.Flags().BoolVarP(&force, "force", "f", false, "create service and override existing files")
	createCmd.Flags().StringVarP(&FileRootPath, "rootpath", "p", ".", "the path to the directory where new packages will be created in")
	RootCmd.AddCommand(createCmd)
}
