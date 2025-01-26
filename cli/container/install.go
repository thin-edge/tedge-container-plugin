/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	containerSDK "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
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
func NewInstallCommand(ctx cli.Cli) *cobra.Command {
	command := &InstallCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:   "install <MODULE_NAME>",
		Short: "Install/run a container",
		Example: `
Example 1: Install a container and pull in the image from any available registries

  $ tedge-container container install myapp1 --module-version docker.io/nginx:latest


Example 2: Save an image (using an explicit platform) and install/create a container with the saved image file

  $ docker pull --platform linux/arm64 docker.io/nginx:latest
  $ docker save docker.io/nginx:latest > nginx:latest.tar
  $ gzip nginx:latest.tar
  $ tedge-container container install myapp1 --module-version nginx:latest --file ./nginx:latest.tar.gz
		`,
		Args: cobra.ExactArgs(1),
		RunE: command.RunE,
	}

	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to install")
	cmd.Flags().StringVar(&command.File, "file", "", "File")
	viper.SetDefault("container.alwaysPull", false)
	command.Command = cmd
	return cmd
}

func (c *InstallCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
	commonNetwork := c.CommandContext.GetSharedContainerNetwork()
	containerName := args[0]
	imageRef := c.ModuleVersion

	// Only enable pulling if the user is providing a file
	disablePull := c.File != ""

	cli, err := container.NewContainerClient()
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
		defer file.Close()

		imageResp, err := cli.Client.ImageLoad(ctx, file, true)
		if err != nil {
			return err
		}
		defer imageResp.Body.Close()
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
			moduleVersionFound := false
			for _, line := range strings.Split(imageDetails.Stream, "\n") {
				if strings.HasPrefix(line, "Loaded image: ") {
					imageName := strings.TrimPrefix(line, "Loaded image: ")
					slog.Info("Found image reference in file.", "file", c.File, "image", imageName)
					images = append(images, imageName)
					if imageName == c.ModuleVersion {
						moduleVersionFound = true
					}
				}
			}

			// Check if the user has given correct image to use from the
			if !moduleVersionFound {
				switch count := len(images); count {
				case 0:
					slog.Warn("No images detected in stream output. Aborting to prevent accidentally load image from network.", "file", c.File, "images", images)
					// Fail hard to prevent potentially trying to pull in the image (as the user has opted into file based images)
					return fmt.Errorf("no image detected in file. name=%s, version=%s, file=%s", containerName, c.ModuleVersion, c.File)
				default:
					if count > 1 {
						slog.Warn("More than 1 image detected in file. Only using the first image.", "file", c.File, "images", images, "image_count", count)
					}

					imageRef = images[0]
					slog.Info("Detected image reference does not match the module-version. Using first imageRef from loaded image.", "imageRef", imageRef, "version", c.ModuleVersion)
				}
			}
		}
	}

	// Create shared network
	if commonNetwork != "" {
		if err := cli.CreateSharedNetwork(ctx, commonNetwork); err != nil {
			return err
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

	//
	// Stop/remove any existing images with the same name
	if err := cli.StopRemoveContainer(ctx, containerName); err != nil {
		slog.Warn("Could not stop and remove the existing container.", "err", err)
		return err
	}

	//
	// Create new container
	containerConfig := &containerSDK.Config{
		Image:  imageRef,
		Labels: map[string]string{},
	}

	networkConfig := make(map[string]*network.EndpointSettings)
	if commonNetwork != "" {
		slog.Info("Connecting container to common network.", "network", commonNetwork)
		networkConfig[commonNetwork] = &network.EndpointSettings{
			NetworkID: commonNetwork,
		}
	}

	resp, err := cli.Client.ContainerCreate(
		ctx,
		containerConfig,
		&containerSDK.HostConfig{
			PublishAllPorts: true,
			NetworkMode:     network.NetworkBridge,
			RestartPolicy: containerSDK.RestartPolicy{
				Name: containerSDK.RestartPolicyAlways,
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: networkConfig,
		},
		nil,
		containerName,
	)
	if err != nil {
		return err
	}

	if err := cli.Client.ContainerStart(ctx, resp.ID, containerSDK.StartOptions{}); err != nil {
		return err
	}

	slog.Info("created container.", "id", resp.ID, "name", containerName)
	return nil
}
