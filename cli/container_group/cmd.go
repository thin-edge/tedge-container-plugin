package container_group

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewContainerGroupCommand returns a cobra command for `container-group` subcommands
func NewContainerGroupCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container-group",
		Short: "container-group software management plugin",
	}
	cmd.AddCommand(
		NewPrepareCommand(cmdCli),
		NewInstallCommand(cmdCli),
		NewRemoveCommand(cmdCli),
		NewUpdateListCommand(cmdCli),
		NewListCommand(cmdCli),
		NewFinalizeCommand(cmdCli),
	)
	return cmd
}
