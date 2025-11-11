package log_plugins

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/cli/log_plugins/container_group_plugin"
	"github.com/thin-edge/tedge-container-plugin/cli/log_plugins/container_plugin"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewCommand returns a cobra command for `logs` subcommands
func NewCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log-plugins",
		Short: "log plugins",
	}
	cmd.AddCommand(
		container_group_plugin.NewCommand(cmdCli),
		container_plugin.NewCommand(cmdCli),
	)
	return cmd
}
