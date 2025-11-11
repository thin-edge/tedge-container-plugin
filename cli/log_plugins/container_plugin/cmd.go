package container_plugin

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewCommand returns a cobra command for `container-logs` subcommands
func NewCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "container logs plugin",
	}
	cmd.AddCommand(
		NewListCommand(cmdCli),
		NewGetCommand(cmdCli),
	)
	return cmd
}
