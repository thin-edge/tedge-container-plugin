package engine

import (
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// NewCliCommand returns a cobra command for `cli` subcommands
func NewCliCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Run container engine commands",
	}
	cmd.AddCommand(
		NewRunCommand(cmdCli),
		NewContainerCloneCommand(cmdCli),
	)
	return cmd
}
