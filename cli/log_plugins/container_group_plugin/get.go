/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group_plugin

import (
	"context"
	"fmt"
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
		Short:        "Get container-group logs",
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

	// lookup
	projectName, serviceName, found := container.ParseContainerGroup(args[0])
	if !found {
		return fmt.Errorf("invalid service name. expected name in format of '<project>@<service>'")
	}
	services, err := containerCli.LookupProject(context.Background(), projectName, serviceName)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		slog.Info("No matching services found")
		return nil
	}

	// containerCli.List(context.Background(), container.FilterOptions{})

	// Write container logs (stdout and stderr) to stdout
	out := cmd.OutOrStdout()
	slog.Info("Fetching logs", "id", services[0].ID, "name", services[0].Names)
	err = containerCli.ContainerLogs(ctx, out, services[0].ID, container.LogsOptions{
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
