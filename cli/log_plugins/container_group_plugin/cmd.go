/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group_plugin

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewCommand returns a cobra command for `container-group` subcommands
func NewCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container-group",
		Short: "container-group logs plugin",
	}
	cmd.AddCommand(
		NewListCommand(cmdCli),
		NewGetCommand(cmdCli),
	)
	return cmd
}
