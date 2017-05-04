package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

var generateCmd = &cobra.Command{
	Use:     "generate [package_path]",
	Aliases: []string{"g"},
	Short:   "Generate 'switcher.go' and '[service_name].pb.go' according to service.yaml and [service_name].proto",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("Usage: generate [package_path]")
		}
		turbo.LoadServiceConfigWith(args[0])
		turbo.GenerateSwitcher()
		turbo.GenerateProtobufStub()
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
}
