/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package tools

import (
	"context"
	"errors"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type ContainerRemoveCommand struct {
	*cobra.Command

	CommandContext cli.Cli

	// Options
	Tail       string
	Since      string
	Until      string
	Timestamps bool
	Follow     bool
	Details    bool
}

// NewContainerLogsCommand creates a new container remove command
func NewContainerRemoveCommand(ctx cli.Cli) *cobra.Command {
	command := &ContainerLogsCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:          "container-remove [OPTIONS] CONTAINER...",
		Short:        "Remove a container (stopping if necessary)",
		RunE:         command.RunE,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}
	command.Command = cmd
	return cmd
}

func (c *ContainerRemoveCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)

	containerCli, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	errs := make([]error, 0)
	for _, name := range args {
		errs = append(errs, containerCli.StopRemoveContainer(ctx, name))
	}

	return errors.Join(errs...)
}
