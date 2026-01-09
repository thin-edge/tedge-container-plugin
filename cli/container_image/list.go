/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_image

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

// listCmd represents the list command
func NewListCommand(cliContext cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List container images",
		Args:    cobra.ExactArgs(0),
		PreRunE: IsEnabled(cliContext),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			ctx := context.Background()
			cli, err := container.NewContainerClient(ctx, cliContext.GetContainerClientOptions()...)
			if err != nil {
				return err
			}
			images, err := cli.Client.ImageList(context.Background(), image.ListOptions{})
			if err != nil {
				return err
			}
			stdout := cmd.OutOrStdout()
			for _, item := range images {
				for _, tag := range item.RepoTags {
					if name, version, ok := strings.Cut(tag, ":"); ok {
						fmt.Fprintf(stdout, "%s\t%s\n", name, version)
					}
				}
			}
			return nil
		},
	}
}
