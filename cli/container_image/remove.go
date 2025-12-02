/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_image

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/image"
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
		Short: "Remove a container image",
		Example: `
Example 1: Remove a container image

	$ tedge-container container remove alpine --module-version 3.21
				`,
		Args:    cobra.ExactArgs(1),
		PreRunE: IsEnabled(cliContext),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			ctx := context.Background()
			imageName := args[0]

			var imageRef string
			if command.ModuleVersion != "" {
				imageRef = fmt.Sprintf("%s:%s", imageName, command.ModuleVersion)
			} else {
				imageRef = imageName
			}

			cli, err := container.NewContainerClient(ctx, cliContext.GetContainerClientOptions()...)
			if err != nil {
				return err
			}

			_, imageErr := cli.Client.ImageRemove(context.Background(), imageRef, image.RemoveOptions{})
			if errdefs.IsNotFound(imageErr) {
				slog.Info("Image reference not found, so nothing to remove", "imageRef", imageRef)
				return nil
			}
			if imageErr != nil {
				return imageErr
			}
			slog.Info("Successfully removed image", "imageRef", imageRef)
			return nil
		},
	}
	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to remove")
	return cmd
}
