/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type RemoveCommand struct {
	*cobra.Command

	ModuleVersion string
}

// removeCmd represents the remove command
func NewRemoveCommand(cliContext cli.Cli) *cobra.Command {
	command := &RemoveCommand{}
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a container",
		Example: `
Example 1: Remove a container

	$ tedge-container container remove myapp1
				`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			ctx := context.Background()
			containerName := args[0]

			cli, err := container.NewContainerClient(ctx, cliContext.GetContainerClientOptions()...)
			if err != nil {
				return err
			}

			return cli.StopRemoveContainer(ctx, containerName)
		},
	}
	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to remove")
	return cmd
}
