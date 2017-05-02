package cmd

import (
	"github.com/spf13/cobra"
	"turbo"
	"errors"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:     "generate [package_name]",
	Aliases: []string{"g"},
	Short:   "generate switcher.go",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("package_name missing")
		}
		turbo.GenerateHandler(args[0])
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
}
