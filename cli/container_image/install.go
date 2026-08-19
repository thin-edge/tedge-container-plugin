/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_image

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/client"
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

type ImageResponse struct {
	Stream string `json:"stream"`
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
		slog.Info("Loading image from file.", "file", c.File)
		file, err := os.Open(c.File)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()

		imageResp, err := cli.Client.ImageLoad(ctx, file, client.ImageLoadWithQuiet(true))
		if err != nil {
			return err
		}
		defer func() { _ = imageResp.Body.Close() }()
		if imageResp.JSON {
			b, err := io.ReadAll(imageResp.Body)
			if err != nil {
				return nil
			}
			imageDetails := &ImageResponse{}
			if err := json.Unmarshal(b, &imageDetails); err != nil {
				return err
			}

			slog.Info("Loaded image.", "stream", imageDetails.Stream)
			images := make([]string, 0)
			imageRefFound := false
			for _, line := range strings.Split(imageDetails.Stream, "\n") {
				if strings.HasPrefix(line, "Loaded image: ") {
					loadedRef := strings.TrimPrefix(line, "Loaded image: ")
					slog.Info("Found image reference in file.", "file", c.File, "image", loadedRef)
					images = append(images, loadedRef)
					if !imageRefFound && container.ImageRefsEqual(loadedRef, imageRef) {
						// Use the reference reported by the engine as it is guaranteed to
						// resolve regardless of how the engine normalises tag strings
						imageRef = loadedRef
						imageRefFound = true
					}
				}
			}

			// The user has opted into file based images, so fail hard rather than
			// falling back to pulling the image from a registry
			if !imageRefFound {
				switch count := len(images); count {
				case 0:
					return fmt.Errorf("no image detected in file. name=%s, version=%s, file=%s", imageName, c.ModuleVersion, c.File)
				case 1:
					slog.Info("Tagging loaded image with the requested reference.", "source", images[0], "target", imageRef)
					if err := cli.Client.ImageTag(ctx, images[0], imageRef); err != nil {
						return err
					}
				default:
					return fmt.Errorf("more than 1 image detected in file and none match the requested image. image=%s, images=%s, file=%s", imageRef, strings.Join(images, ","), c.File)
				}
			}
		}
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
