package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/vaporz/turbo"
)

var generateCmd = &cobra.Command{
	Use:     "generate package_path",
	Aliases: []string{"g"},
	Example: "turbo generate package/path/to/yourservice -r grpc \n" +
		"        -I (absolute_paths_to_proto/thrift_files) -I ... -I ...\n",
	Short: "Generate '[gprc|thrift]switcher.go' and grpc|thrift generated codes \n" +
		"according to service.yaml and .proto|.thrift files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("Usage: generate [package_path]")
		}
		if RpcType == "" {
			return errors.New("missing rpctype (-r)")
		}
		if RpcType != "grpc" && RpcType != "thrift" {
			return errors.New("invalid rpctype")
		}
		if RpcType == "grpc" && len(FilePaths) == 0 {
			return errors.New("missing .proto file path (-I)")
		}
		var options string
		if RpcType == "grpc" {
			for _, p := range FilePaths {
				options = options + " -I " + p + " " + p + "/*.proto "
			}
		} else if RpcType == "thrift" {
			for _, p := range FilePaths {
				options = options + " -I " + p + " "
			}
		}

		g := turbo.Generator{
			RpcType:        RpcType,
			PkgPath:        args[0],
			ConfigFileName: "service",
			Options:        options,
		}
		g.Generate()
		return nil
	},
}

// FilePaths is a list of paths to proto files
var FilePaths []string

// RpcType should be either "grpc" or "thrift"
var RpcType string

func init() {
	RootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&RpcType, "rpctype", "r", "", "required, (grpc|thrift)")
	generateCmd.Flags().StringArrayVarP(&FilePaths, "include-path", "I", []string{}, "required for grpc, .proto|.thrift file paths(absolute path)")
}
