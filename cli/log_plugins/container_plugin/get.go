/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_plugin

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type ContainerLogsCommand struct {
	*cobra.Command

	CommandContext cli.Cli

	// Options
	Since string
	Until string
}

func NewGetCommand(ctx cli.Cli) *cobra.Command {
	command := &ContainerLogsCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:          "get",
		Short:        "Get container logs",
		RunE:         command.RunE,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&command.Since, "since", "", "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	cmd.Flags().StringVar(&command.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	command.Command = cmd
	return cmd
}

func (c *ContainerLogsCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)

	containerCli, err := container.NewContainerClient(context.TODO(), c.CommandContext.GetContainerClientOptions()...)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Write container logs (stdout and stderr) to stdout
	out := cmd.OutOrStdout()
	slog.Info("Fetching logs.", "id", args[0])
	err = containerCli.ContainerLogs(ctx, out, args[0], container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      c.Since,
		Until:      c.Until,
		Timestamps: false,
		Follow:     false,
		Tail:       "100000",
		Details:    false,
	})
	if err != nil {
		return err
	}
	return nil
}
