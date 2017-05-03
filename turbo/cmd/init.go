package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

// init creates folder "[service_name]", "gen",  file "service.yaml", "[service_name].proto"
var initCmd = &cobra.Command{
	Use:   "init [package_path] [service_name]",
	Short: "create an empty project",
	Long:  `Create folders, service.yaml, [service_name].proto. You can edit these
	config files, and then run "turbo generate [package_path] [service_name]" to generate Golang codes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("Usage: init [package_path] [service_name]")
		}
		turbo.Init(args[0], args[1])
		return nil
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
