/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package tools

import (
	"context"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type ImageInstallCommand struct {
	*cobra.Command

	CommandContext cli.Cli

	// Options
	Image string
	File  string
}

// NewImageInstallCommand creates a new image-install command
func NewImageInstallCommand(ctx cli.Cli) *cobra.Command {
	command := &ImageInstallCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:   "image-install",
		Short: "Make a container image available in the local container engine",
		Long: `The image is loaded from an archive if one is given, otherwise it is pulled from a
container registry. Pulling is skipped if the image is already present locally.`,
		Example: `
Example 1: Pull an image from a container registry

  $ tedge-container tools image-install --image ghcr.io/thin-edge/tedge-container-bundle:1.6.0

Example 2: Load an image from an archive created via 'docker save'

  $ tedge-container tools image-install --image ghcr.io/thin-edge/tedge-container-bundle:1.6.0 --file ./bundle.tar.gz
		`,
		RunE:         command.RunE,
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&command.Image, "image", "", "Container image reference")
	cmd.Flags().StringVar(&command.File, "file", "", "Image archive to load the image from. The image is pulled from a registry if not given")
	_ = cmd.MarkFlagRequired("image")

	command.Command = cmd
	return cmd
}

func (c *ImageInstallCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)

	containerCli, err := container.NewContainerClient(context.TODO(), c.CommandContext.GetContainerClientOptions()...)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if c.File != "" {
		imageRef, err := containerCli.LoadImageFromFile(ctx, c.File, c.Image)
		if err != nil {
			return err
		}
		slog.Info("Installed image from file.", "image", imageRef, "file", c.File)
		return nil
	}

	if _, err := containerCli.ImagePullWithRetries(ctx, c.Image, c.CommandContext.ImageAlwaysPull(), container.ImagePullOptions{
		AuthFunc:    c.CommandContext.GetContainerRepositoryCredentialsFunc(c.Image),
		MaxAttempts: 2,
		Wait:        5 * time.Second,
	}); err != nil {
		return err
	}
	slog.Info("Installed image.", "image", c.Image)
	return nil
}
