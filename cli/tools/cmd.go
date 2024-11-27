package tools

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewToolsCommand returns a cobra command for `cli` subcommands
func NewToolsCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tools",
		Short:  "Container tools/utilities",
		Hidden: true,
	}
	cmd.AddCommand(
		NewContainerCloneCommand(cmdCli),
		NewContainerLogsCommand(cmdCli),
	)
	return cmd
}
