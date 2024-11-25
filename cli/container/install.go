/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	containerSDK "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/utils"
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
	if err := cli.CreateSharedNetwork(ctx, commonNetwork); err != nil {
		return err
	}

	//
	// Check and pull image if it is not present
	if !disablePull {
		images, err := cli.Client.ImageList(ctx, image.ListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", imageRef)),
		})
		if err != nil {
			return err
		}

		if len(images) == 0 || c.CommandContext.GetBool("container.alwaysPull") {
			credentialsFunc := func(ctx context.Context, attempt int) (string, error) {
				// Check credentials config
				creds := c.CommandContext.GetRegistryCredentials(imageRef)

				// Check credentials helper script
				credentialsScript := "registry-credentials"
				if utils.CommandExists(credentialsScript) {
					scriptCtx, cancelScript := context.WithTimeout(ctx, 60*time.Second)
					defer cancelScript()
					scriptArgs := make([]string, 0)
					scriptArgs = append(scriptArgs, "get", imageRef)
					if attempt > 1 {
						// signal to the credential helper that it should refresh the credentials
						// in case it is cache them
						scriptArgs = append(scriptArgs, "--refresh")
					}
					if customCreds, err := c.CommandContext.GetCredentialsFromScript(scriptCtx, credentialsScript, scriptArgs...); err != nil {
						slog.Warn("Failed to get registry credentials.", "script", credentialsScript, "err", err)
						return "", err
					} else {
						if customCreds.Username != "" && customCreds.Password != "" {
							slog.Info("Using registry credentials returned by a helper.", "script", credentialsScript, "username", customCreds.Username)
							creds.Username = customCreds.Username
							creds.Password = customCreds.Password
						}
					}
				}

				if creds.IsSet() {
					slog.Info("Pulling image from private registry.", "ref", imageRef, "username", creds.Username)
					return GetRegistryAuth(creds.Username, creds.Password), nil
				} else {
					slog.Info("Pulling image.", "ref", imageRef)
				}
				// no explicit auth, but don't fail
				return "", nil
			}

			pullErr := cli.ImagePullWithRetries(ctx, imageRef, container.ImagePullOptions{
				AuthFunc:    credentialsFunc,
				MaxAttempts: 2,
				Wait:        5 * time.Second,
			})
			if pullErr != nil {
				return err
			}
		} else {
			slog.Info("Image already exists.", "ref", imageRef, "id", images[0].ID, "tags", images[0].RepoTags)
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

func GetRegistryAuth(username, password string) string {
	authConfig := registry.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	return authStr
}
