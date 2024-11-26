/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package self

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

type RemoveCommand struct {
	*cobra.Command

	ModuleVersion string
}

// removeCmd represents the remove command
func NewRemoveCommand(ctx cli.Cli) *cobra.Command {
	command := &RemoveCommand{}
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Not supported",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			slog.Warn("Removing the main container is not supported")

			return cli.ExitCodeError{
				Err:  fmt.Errorf("not supported"),
				Code: 2,
			}
		},
	}
	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to remove")
	return cmd
}
