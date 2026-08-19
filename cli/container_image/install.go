/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_image

import (
	"context"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type InstallCommand struct {
	*cobra.Command

	CommandContext cli.Cli
	ModuleVersion  string
	File           string
}

// installCmd represents the install command
func NewInstallCommand(cliContext cli.Cli) *cobra.Command {
	command := &InstallCommand{
		CommandContext: cliContext,
	}
	cmd := &cobra.Command{
		Use:   "install <MODULE_NAME>",
		Short: "Install a container image",
		Example: `
Example 1: Install a container and pull in the image from any available registries

  $ tedge-container container-image install docker.io/nginx --module-version latest

Example 2: Install an image from an archive created via 'docker save'

  $ tedge-container container-image install docker.io/nginx:latest --file ./nginx.tar.gz

		`,
		Args:    cobra.ExactArgs(1),
		PreRunE: IsEnabled(cliContext),
		RunE:    command.RunE,
	}

	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "latest", "Software version to install")
	cmd.Flags().StringVar(&command.File, "file", "", "File")
	viper.SetDefault("container.alwaysPull", false)
	command.Command = cmd
	return cmd
}

func (c *InstallCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
	imageName := args[0]
	imageRef := container.BuildImageRef(imageName, c.ModuleVersion)

	// Only enable pulling if the user is providing a file
	disablePull := c.File != ""

	cli, err := container.NewContainerClient(context.TODO(), c.CommandContext.GetContainerClientOptions()...)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if c.File != "" {
		loadedRef, err := cli.LoadImageFromFile(ctx, c.File, imageRef)
		if err != nil {
			return err
		}
		imageRef = loadedRef
	}

	//
	// Check and pull image if it is not present
	if !disablePull {
		if _, err := cli.ImagePullWithRetries(ctx, imageRef, c.CommandContext.ImageAlwaysPull(), container.ImagePullOptions{
			AuthFunc:    c.CommandContext.GetContainerRepositoryCredentialsFunc(imageRef),
			MaxAttempts: 2,
			Wait:        5 * time.Second,
		}); err != nil {
			return err
		}
	}

	slog.Info("Installed image.", "name", imageName, "version", c.ModuleVersion, "imageRef", imageRef)
	return nil
}
