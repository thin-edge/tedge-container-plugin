/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group

import (
	"context"
	"log/slog"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

func NewFinalizeCommand(ctx cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "finalize",
		Short: "Finalize container install/remove operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)

			pruneImages := ctx.GetBool("container.pruneImages")
			if !pruneImages {
				return nil
			}
			cli, err := container.NewContainerClient()
			if err != nil {
				return err
			}
			slog.Info("Pruning images")
			ctx := context.Background()
			resp, err := cli.Client.ImagesPrune(ctx, filters.Args{})
			if err != nil {
				return err
			}
			for _, image := range resp.ImagesDeleted {
				slog.Info("Deleted image.", "deleted", image.Deleted, "untagged", image.Untagged)
			}
			slog.Info("Reclaimed space.", "size", units.HumanSizeWithPrecision(float64(resp.SpaceReclaimed), 3))
			return nil
		},
	}
}
