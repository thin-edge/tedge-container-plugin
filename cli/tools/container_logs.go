/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package tools

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type ContainerLogsCommand struct {
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

// NewContainerLogsCommand creates a new container logs command
func NewContainerLogsCommand(ctx cli.Cli) *cobra.Command {
	command := &ContainerLogsCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:          "container-logs [OPTIONS] CONTAINER",
		Short:        "Fetch the logs of a container",
		RunE:         command.RunE,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&command.Since, "since", "", "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	cmd.Flags().StringVar(&command.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	cmd.Flags().StringVarP(&command.Tail, "tail", "n", "all", "Number of lines to show from the end of the logs")
	cmd.Flags().BoolVarP(&command.Follow, "follow", "f", false, "Follow log output")
	cmd.Flags().BoolVarP(&command.Timestamps, "timestamps", "t", false, "Show timestamps")
	cmd.Flags().BoolVar(&command.Details, "details", false, "Show extra details provided to logs")
	command.Command = cmd
	return cmd
}

func (c *ContainerLogsCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)

	containerCli, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if len(args) == 0 {
		slog.Info("Fetching logs for current container (if running inside a container)")
		con, err := containerCli.Self(ctx)
		if err != nil {
			return err
		}
		args = append(args, con.ID)
	}

	// Write container logs (stdout and stderr) to stdout
	out := cmd.OutOrStdout()
	errs := make([]error, 0)
	for _, con := range args {
		idOrName := strings.TrimPrefix(con, "/")
		slog.Info("Fetching logs.", "id", idOrName)
		if err := containerCli.ContainerLogs(ctx, out, idOrName, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Since:      c.Since,
			Until:      c.Until,
			Timestamps: c.Timestamps,
			Follow:     c.Follow,
			Tail:       c.Tail,
			Details:    c.Details,
		}); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
