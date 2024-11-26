/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package self

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

type InstallCommand struct {
	*cobra.Command

	CommandContext cli.Cli
	ModuleVersion  string
	File           string
}

// installCmd represents the install command
func NewInstallCommand(ctx cli.Cli) *cobra.Command {
	command := &InstallCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:   "install <MODULE_NAME>",
		Short: "No-op command. self updates are handled by a dedicated workflow",
		Args:  cobra.ExactArgs(1),
		RunE:  command.RunE,
	}

	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to install")
	cmd.Flags().StringVar(&command.File, "file", "", "File")
	viper.SetDefault("container.alwaysPull", false)
	command.Command = cmd
	return cmd
}

func (c *InstallCommand) RunE(cmd *cobra.Command, args []string) error {
	// don't do anything, just return without an error
	return nil
}
