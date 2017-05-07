package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

var generateCmd = &cobra.Command{
	Use:     "generate [package_path] (grpc|thrift)",
	Aliases: []string{"g"},
	Short:   "Generate 'switcher.go' and '[service_name].pb.go' according to service.yaml and [service_name].proto",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("Usage: generate [package_path] (grpc|thrift)")
		}
		turbo.LoadServiceConfigWith(args[0])
		if len(args) == 1 || args[1] == "grpc" {
			turbo.GenerateGrpcSwitcher()
			turbo.GenerateProtobufStub()
		} else if args[1] == "thrift" {
			turbo.GenerateThriftSwitcher()
			turbo.GenerateThriftStub()
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
}
