package container_logs

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewContainerLogsCommand returns a cobra command for `container-logs` subcommands
func NewContainerLogsCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container-logs",
		Short: "container log plugin",
	}
	cmd.AddCommand(
		NewListCommand(cmdCli),
		NewGetCommand(cmdCli),
	)
	return cmd
}
