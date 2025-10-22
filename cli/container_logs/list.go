/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_logs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

// listCmd represents the list command
func NewListCommand(cliContext cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List containers",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			ctx := context.Background()
			cli, err := container.NewContainerClient(ctx, cliContext.GetContainerClientOptions()...)
			if err != nil {
				return err
			}
			containers, err := cli.List(ctx, cliContext.GetFilterOptions())
			if err != nil {
				return err
			}
			stdout := cmd.OutOrStdout()
			for _, item := range containers {
				switch item.ServiceType {
				case container.ContainerType:
					fmt.Fprintf(stdout, "%s\n", item.Name)
				case container.ContainerGroupType:
					// TODO: Should container groups actually reference use <project>@<service>
					// so it is easier for users to see the mapping? Or just create a new container-group-logs
					// subcommand to handle this independently
					fmt.Fprintf(stdout, "%s\n", item.Container.Name)
				}
			}
			return nil
		},
	}
}
