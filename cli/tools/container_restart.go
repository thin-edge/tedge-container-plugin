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

type ContainerRestartCommand struct {
	*cobra.Command

	CommandContext cli.Cli
}

// NewContainerRestartCommand restarts a container
func NewContainerRestartCommand(ctx cli.Cli) *cobra.Command {
	command := &ContainerRestartCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:          "container-restart [OPTIONS] [CONTAINER]",
		Short:        "Restart a container",
		Long:         "If this command is called with no arguments, then it will try to detect the container that the command is running under (if running inside a container)",
		RunE:         command.RunE,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}

	command.Command = cmd
	return cmd
}

func (c *ContainerRestartCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)

	containerCli, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if len(args) == 0 {
		slog.Info("Restarting current container")
		con, err := containerCli.Self(ctx)
		if err != nil {
			return err
		}
		args = append(args, con.ID)
	}

	errs := make([]error, 0)
	for _, con := range args {
		errs = append(errs, containerCli.RestartContainer(ctx, con))
	}
	return errors.Join(errs...)
}
