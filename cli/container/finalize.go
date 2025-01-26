/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container

import (
	"context"
	"log/slog"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

func NewFinalizeCommand(ctx cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
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
			resp, err := cli.ImagesPruneUnused(ctx)
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
	viper.SetDefault("container.pruneImages", true)
	return cmd
}
