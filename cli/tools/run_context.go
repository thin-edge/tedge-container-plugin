/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	containerSDK "github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/random"
)

type ContainerRunInContextCommand struct {
	*cobra.Command

	CommandContext cli.Cli

	// Options
	ContainerID string
	Image       string
	Env         []string
	AutoRemove  bool
	Entrypoint  string
	Labels      []string
	NamePrefix  string
}

// NewRunRemoteAccessCommand creates a new c8y remote access command
func NewContainerRunInContextCommand(ctx cli.Cli) *cobra.Command {
	command := &ContainerRunInContextCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:          "run-in-context",
		Short:        "Run a command in a new container and copy context of an existing container",
		RunE:         command.RunE,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&command.ContainerID, "container", "", "Container to clone. Either container id or name. By default the container id of the current container will be used")
	cmd.Flags().StringVar(&command.Image, "image", "", "Container image")
	cmd.Flags().StringVar(&command.Entrypoint, "entrypoint", "", "Overwrite the default ENTRYPOINT of the image")
	cmd.Flags().StringVar(&command.NamePrefix, "name-prefix", "", "Prefix to be added when generating the name. If left blank then the default will be used")
	cmd.Flags().StringSliceVarP(&command.Env, "env", "e", []string{}, "Set environment variables")
	cmd.Flags().BoolVar(&command.AutoRemove, "rm", false, "Auto remove the cloned container on exit")
	cmd.Flags().StringSliceVarP(&command.Labels, "label", "l", []string{}, "Set meta data on a container")

	command.Command = cmd
	return cmd
}

func (c *ContainerRunInContextCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)

	containerCli, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	currentContainer := containerSDK.InspectResponse{
		Config:            &containerSDK.Config{},
		ContainerJSONBase: &containerSDK.ContainerJSONBase{},
		NetworkSettings:   &containerSDK.NetworkSettings{},
	}

	if c.ContainerID == "" {
		slog.Info("No container provided, inferring the update is intended for the current container")
		con, err := containerCli.Self(ctx)
		if err != nil {
			return err
		}
		currentContainer = con
	} else {
		con, err := containerCli.Client.ContainerInspect(ctx, c.ContainerID)
		if err != nil {
			return err
		}
		currentContainer = con
	}

	slog.Info("Found current container.", "id", currentContainer.ID, "name", currentContainer.Name, "image", currentContainer.Config.Image)

	// Check if the container exists
	if c.Image == "" && currentContainer.Config != nil {
		c.Image = currentContainer.Config.Image
	}

	if c.Image == "" {
		return fmt.Errorf("container image is empty")
	}

	// Pull potentially new image
	if _, err := containerCli.ImagePullWithRetries(ctx, c.Image, c.CommandContext.ImageAlwaysPull(), container.ImagePullOptions{
		AuthFunc:    c.CommandContext.GetContainerRepositoryCredentialsFunc(c.Image),
		MaxAttempts: 2,
		Wait:        5 * time.Second,
	}); err != nil {
		slog.Warn("Failed to pull image.", "err", err)
		return err
	}

	entrypoint := make([]string, 0)
	if c.Entrypoint != "" {
		entrypoint = append(entrypoint, c.Entrypoint)
	}

	// Pass all arguments passed "--" as the entrypoint
	containerCmd := make([]string, 0)
	if i := cmd.ArgsLenAtDash(); len(args)-1 > i {
		containerCmd = append(containerCmd, args[i:]...)
		slog.Info("Custom args.", "args", containerCmd)
	}

	// Set environment variables so the new container knows how to reach the current container
	// where the services are running (e.g. MQTT Broker)
	containerHostName := strings.TrimPrefix(currentContainer.Name, "/")
	if c.Env, err = WithTedgeEnv(c.Env,
		WithTedgeConfigValue("c8y.url"),
		WithValue("mqtt.client.host", containerHostName),
		WithTedgeConfigValue("mqtt.client.port"),
		WithValue("http.client.host", containerHostName),
		WithTedgeConfigValue("http.client.port"),
	); err != nil {
		slog.Error("Failed to set thin-edge.io environment variables.", "err", err)
		return err
	}

	slog.Info("Starting container.", "command", strings.Join(entrypoint, " "), "env", c.Env)

	opts := container.CloneOptions{
		Image: c.Image,
		// Disable auto remove to help with debugging
		AutoRemove:  c.AutoRemove,
		Env:         c.Env,
		Entrypoint:  entrypoint,
		Cmd:         containerCmd,
		IgnorePorts: true,
		Labels:      container.FormatLabels(c.Labels),
	}

	// Container config
	clonedConfig := container.CloneContainerConfig(currentContainer.Config, opts)

	// Copy host config
	hostConfig := container.CloneHostConfig(currentContainer.HostConfig, opts)
	hostConfig.RestartPolicy.Name = containerSDK.RestartPolicyDisabled

	// Copy network config
	networkConfig := container.CloneNetworkConfig(currentContainer.NetworkSettings)

	// container name
	containerName := "" // default. let docker create a random name
	if c.NamePrefix != "" {
		containerName = c.NamePrefix + "_" + random.String(8)
	}

	nextContainer, createErr := containerCli.Client.ContainerCreate(ctx, clonedConfig, hostConfig, networkConfig, nil, containerName)

	if createErr != nil {
		slog.Warn("Failed to create container.", "err", createErr)
		return createErr
	}
	slog.Info("Created new container.", "id", nextContainer.ID)

	// start container
	if err := containerCli.Client.ContainerStart(ctx, nextContainer.ID, containerSDK.StartOptions{}); err != nil {
		slog.Warn("Container failed to start container.", "id", nextContainer.ID, "err", err)
		return err
	}

	// TODO: Should the container be verified if it worked successfully or not?
	return nil
}

type TedgeOption func() (name string, value string, err error)

func WithTedgeEnv(env []string, options ...TedgeOption) ([]string, error) {
	for _, opt := range options {
		name, value, err := opt()
		if err != nil {
			return env, err
		}
		envName := "TEDGE_" + strings.ReplaceAll(strings.ToUpper(name), ".", "_")
		env = append(env, fmt.Sprintf("%s=%s", envName, value))
	}
	return env, nil
}

func WithValue(prop string, value string) TedgeOption {
	return func() (string, string, error) {
		return prop, value, nil
	}
}

func WithTedgeConfigValue(prop string) TedgeOption {
	return func() (string, string, error) {
		value, err := cli.GetTedgeConfig(prop)
		if err != nil {
			slog.Warn("Could not get tedge config value.", "property", prop)
			return "", "", err
		}
		return prop, value, nil
	}
}
