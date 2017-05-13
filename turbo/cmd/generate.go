package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

var generateCmd = &cobra.Command{
	Use:     "generate package_path",
	Aliases: []string{"g"},
	Example: "turbo generate package/path/to/yourservice -r grpc \n" +
		"        -I (absolute_paths_to_proto/thrift_files) -I ... -I ...\n",
	Short:   "Generate '[gprc|thrift]switcher.go' and grpc|thrift generated codes \n" +
		"according to service.yaml and .proto|.thrift files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("Usage: generate [package_path]")
		}
		if g_rpcType == "" {
			return errors.New("missing rpctype (-r)")
		}
		if g_rpcType != "grpc" && g_rpcType != "thrift" {
			return errors.New("invalid rpctype")
		}
		if g_rpcType == "grpc" && len(filePaths) == 0 {
			return errors.New("missing .proto file path (-I)")
		}
		var options string
		if g_rpcType == "grpc" {
			for _, p := range filePaths {
				options = options + " -I " + p + " " + p + "/*.proto "
			}
		} else if g_rpcType == "thrift" {
			for _, p := range filePaths {
				options = options + " -I " + p + " "
			}
		}
		turbo.InitRpcType(g_rpcType)
		turbo.LoadServiceConfigWith(args[0])
		if g_rpcType == "grpc" {
			turbo.GenerateGrpcSwitcher()
			turbo.GenerateProtobufStub(options)
		} else if g_rpcType == "thrift" {
			turbo.GenerateThriftSwitcher()
			turbo.GenerateThriftStub(options)
		} else {
			return errors.New("Invalid server type, should be (grpc|thrift)")
		}
		return nil
	},
}

var filePaths []string
var g_rpcType string

func init() {
	RootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&g_rpcType, "rpctype", "r", "", "required, (grpc|thrift)")
	generateCmd.Flags().StringArrayVarP(&filePaths, "include-path", "I", []string{}, "required for grpc, .proto|.thrift file paths(absolute path)")
}
