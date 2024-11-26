package self

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewSoftwareManagementSelfCommand returns a cobra command for `self` subcommands
func NewSoftwareManagementSelfCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "self",
		Short:  "self software management plugin",
		Hidden: true,
	}
	cmd.AddCommand(
		NewPrepareCommand(cmdCli),
		NewInstallCommand(cmdCli),
		NewRemoveCommand(cmdCli),
		NewUpdateListCommand(cmdCli),
		NewListCommand(cmdCli),
		NewFinalizeCommand(cmdCli),
		NewCheckCommand(cmdCli),
	)
	return cmd
}
