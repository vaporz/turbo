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
		if rpcType == "" {
			return errors.New("missing rpctype (-r)")
		}
		if rpcType != "grpc" && rpcType != "thrift" {
			return errors.New("invalid rpctype")
		}
		if rpcType == "grpc" && len(filePaths) == 0 {
			return errors.New("missing .proto file path (-I)")
		}
		var options string
		if rpcType == "grpc" {
			for _, p := range filePaths {
				options = options + " -I " + p + " " + p + "/*.proto "
			}
		} else if rpcType == "thrift" {
			for _, p := range filePaths {
				options = options + " -I " + p + " "
			}
		}
		turbo.Generate(rpcType, args[0], "service", options)
		return nil
	},
}

var filePaths []string
var rpcType string

func init() {
	RootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&rpcType, "rpctype", "r", "", "required, (grpc|thrift)")
	generateCmd.Flags().StringArrayVarP(&filePaths, "include-path", "I", []string{}, "required for grpc, .proto|.thrift file paths(absolute path)")
}
