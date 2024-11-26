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

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/utils"
)

type ContainerCloneCommand struct {
	*cobra.Command

	CommandContext cli.Cli

	// Options
	ForceUpdate    bool
	Fork           bool
	WaitForExit    bool
	CheckForUpdate bool
	ContainerID    string
	Image          string
	Duration       time.Duration
	StopTimeout    time.Duration
	AutoRemove     bool
	AddHost        []string
	Env            []string
}

// NewContainerCloneCommand creates a new container clone command
func NewContainerCloneCommand(ctx cli.Cli) *cobra.Command {
	command := &ContainerCloneCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:          "container-clone",
		Short:        "Clone an existing container and replace the container image",
		RunE:         command.RunE,
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&command.ContainerID, "container", "", "Container to clone. Either container id or name")
	cmd.Flags().StringVar(&command.Image, "image", "", "Container image")
	cmd.Flags().DurationVar(&command.Duration, "duration", 15*time.Second, "How long to wait for the clone container to be healthy")
	cmd.Flags().DurationVar(&command.StopTimeout, "stop-timeout", 60*time.Second, "Timeout used whilst waiting for container to stop. Only used with --wait-for-exit")
	cmd.Flags().BoolVar(&command.AutoRemove, "rm", false, "Auto remove the closed container on exit")
	cmd.Flags().StringSliceVar(&command.AddHost, "add-host", []string{}, "Add extra hosts to the container")
	cmd.Flags().StringSliceVarP(&command.Env, "env", "e", []string{}, "Environment variables to add to the container")
	cmd.Flags().BoolVar(&command.ForceUpdate, "force", false, "Force an update, disable the image comparison check")
	cmd.Flags().BoolVar(&command.Fork, "fork", false, "Spawn a new container to do the update")
	cmd.Flags().BoolVar(&command.WaitForExit, "wait-for-exit", false, "Wait for the container to stop/exit before updating")
	cmd.Flags().BoolVar(&command.CheckForUpdate, "check", false, "Only check if an update is necessary, don't perform the update")

	command.Command = cmd
	return cmd
}

func (c *ContainerCloneCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)
	containerCli, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if c.ContainerID == "" {
		// Default to the container
		slog.Info("No container provided, inferring the update is intended for the current container")
		if selfContainer, err := containerCli.Self(ctx); err == nil {
			slog.Info("Found current container.", "id", selfContainer.ID, "name", selfContainer.Name, "image", selfContainer.Config.Image)
			c.ContainerID = selfContainer.ID
			if c.Image == "" {
				c.Image = selfContainer.Config.Image
			}
		}
	}

	// Check if the container exists
	currentContainer, err := containerCli.Client.ContainerInspect(ctx, c.ContainerID)
	if err != nil {
		return err
	}

	if c.Image == "" {
		// Default to the image name of the current container
		c.Image = currentContainer.Config.Image
		slog.Info("Using image of current container.", "image", c.Image)
	}

	// Pull potentially new image
	if _, err := containerCli.ImagePullWithRetries(ctx, c.Image, c.CommandContext.ImageAlwaysPull(), container.ImagePullOptions{
		AuthFunc:    c.CommandContext.GetContainerRepositoryCredentialsFunc(c.Image),
		MaxAttempts: 2,
		Wait:        5 * time.Second,
	}); err != nil {
		return err
	}

	if c.CheckForUpdate {
		if c.ForceUpdate {
			slog.Info("Forcing an update")
			return nil
		}
		needsUpdate, _, err := containerCli.UpdateRequired(ctx, c.ContainerID, c.Image)
		if err != nil {
			return err
		}

		if needsUpdate {
			slog.Info("Image needs updating")
			return nil
		}
		return cli.ExitCodeError{
			Code:   2,
			Err:    fmt.Errorf("image does not need updating"),
			Silent: true,
		}
	}

	if c.Fork {
		if !container.IsInsideContainer() {
			return fmt.Errorf("can't fork from outside of a container")
		}

		entrypoint := make([]string, 0)

		if utils.CommandExists("sudo") {
			entrypoint = append(entrypoint, "sudo")
		}
		entrypoint = append(
			entrypoint,
			"tedge-container",
			"tools",
			"container-clone",
			"--container",
			c.ContainerID,
			"--image",
			c.Image,
		)

		// TODO: Pull in new image, and fail early if it does not work (before forking etc.)

		// TODO: Should the container be run as root instead?

		if c.WaitForExit {
			// Wait for exit does not work if the restart policy can't be changed
			// For example podman <5.1 does not support changing of the restart policy
			// of a container after creation
			// "--wait-for-exit",
			entrypoint = append(entrypoint, "--wait-for-exit")
			entrypoint = append(entrypoint, "stop-timeout", c.StopTimeout.String())
		}

		if c.AutoRemove {
			entrypoint = append(entrypoint, "--rm")
		}

		for _, v := range c.AddHost {
			entrypoint = append(entrypoint, "--add-host", v)
		}

		for _, v := range c.Env {
			entrypoint = append(entrypoint, "--env", v)
		}

		slog.Info("Forking container.", "command", strings.Join(entrypoint, " "))
		return containerCli.Fork(context.Background(), entrypoint, []string{})
	}

	return containerCli.CloneContainer(context.Background(), c.ContainerID, container.CloneOptions{
		Image:        c.Image,
		HealthyAfter: c.Duration,
		WaitForExit:  c.WaitForExit,
		StopTimeout:  c.StopTimeout,
		AutoRemove:   c.AutoRemove,
		Env:          c.Env,
		ExtraHosts:   c.AddHost,
	})
}
