package self

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

func CalledAsSMPlugin() bool {
	args := os.Args
	name := filepath.Base(args[0])
	return name == "self"
}

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
