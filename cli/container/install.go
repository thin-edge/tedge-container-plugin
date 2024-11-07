/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"

	containerSDK "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
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
		Args:  cobra.ExactArgs(1),
		RunE:  command.RunE,
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

			if strings.HasPrefix(imageDetails.Stream, "Loaded image: ") {
				imageRef = strings.TrimPrefix(imageDetails.Stream, "Loaded image: ")
				slog.Info("Using imageRef from loaded image.", "name", imageRef)
			}
			slog.Info("Loaded image.", "stream", imageDetails.Stream)
		}
	}

	// Create shared network
	if err := cli.CreateSharedNetwork(ctx, commonNetwork); err != nil {
		return err
	}

	//
	// Check and pull image if it is not present
	images, err := cli.Client.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageRef)),
	})
	if err != nil {
		return err
	}

	if len(images) == 0 || c.CommandContext.GetBool("container.alwaysPull") {
		slog.Info("Pulling image.", "ref", imageRef)
		out, err := cli.Client.ImagePull(ctx, imageRef, image.PullOptions{})
		if err != nil {
			return err
		}
		defer out.Close()
		if _, ioErr := io.Copy(os.Stderr, out); ioErr != nil {
			slog.Warn("Could not write to stderr.", "err", ioErr)
		}
	} else {
		slog.Info("Image already exists.", "ref", imageRef, "id", images[0].ID, "tags", images[0].RepoTags)
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

	resp, err := cli.Client.ContainerCreate(
		ctx,
		containerConfig,
		&containerSDK.HostConfig{
			PublishAllPorts: true,
			RestartPolicy: containerSDK.RestartPolicy{
				Name: containerSDK.RestartPolicyAlways,
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				commonNetwork: {
					NetworkID: commonNetwork,
				},
			},
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
