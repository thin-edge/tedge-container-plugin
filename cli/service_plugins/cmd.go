/*
Copyright © 2026 thin-edge.io <info@thin-edge.io>
*/
package service_plugins

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/serviceplugin"
)

// resolveFunc resolves a service name to a container id
type resolveFunc func(ctx context.Context, cli *container.ContainerClient, name string) (string, error)

// NewCommand returns a cobra command for `service-plugins` subcommands
func NewCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-plugins",
		Short: "thin-edge.io service plugins",
	}
	cmd.AddCommand(
		newPluginCommand(cmdCli, container.ContainerType, resolveContainer),
		newPluginCommand(cmdCli, container.ContainerGroupType, resolveContainerGroupService),
	)
	return cmd
}

type ServicePluginCommand struct {
	*cobra.Command

	CommandContext cli.Cli
	ServiceType    string
	Resolve        resolveFunc
}

func newPluginCommand(ctx cli.Cli, serviceType string, resolve resolveFunc) *cobra.Command {
	command := &ServicePluginCommand{
		CommandContext: ctx,
		ServiceType:    serviceType,
		Resolve:        resolve,
	}
	cmd := &cobra.Command{
		Use:   serviceType + " <ACTION> <SERVICE_NAME>",
		Short: fmt.Sprintf("%s service plugin", serviceType),
		Long: fmt.Sprintf(`Run an action on a %s service, as called by 'tedge service'.

Supported actions: %s

Exit codes:
  0  the action succeeded
  2  the action is not supported
  1  the action failed`, serviceType, strings.Join(serviceplugin.Actions(), ", ")),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("expected 2 arguments <ACTION> <SERVICE_NAME>, got %d", len(args))
			}
			return nil
		},
		// The arguments come from the cloud, so never interpret them as flags
		DisableFlagParsing: true,
		RunE:               command.RunE,
		SilenceUsage:       true,
	}
	command.Command = cmd
	return cmd
}

func (c *ServicePluginCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)
	action, name := args[0], args[1]

	// Reject unsupported actions before contacting the container engine
	if !serviceplugin.IsSupported(action) {
		return cli.ExitCodeError{
			Code: serviceplugin.ExitCodeUnsupported,
			Err:  fmt.Errorf("%s plugin: %w '%s'", c.ServiceType, serviceplugin.ErrUnsupportedAction, action),
		}
	}

	ctx := context.Background()
	containerCli, err := container.NewContainerClient(ctx, c.CommandContext.GetContainerClientOptions()...)
	if err != nil {
		return err
	}

	containerID, err := c.Resolve(ctx, containerCli, name)
	if err != nil {
		return err
	}

	if err := refuseSelf(ctx, containerCli, containerID, action); err != nil {
		return err
	}

	slog.Info("Running service action.", "action", action, "service", name, "type", c.ServiceType, "id", containerID)
	return serviceplugin.Run(ctx, containerCli, action, containerID)
}

func resolveContainer(ctx context.Context, cli *container.ContainerClient, name string) (string, error) {
	con, err := cli.Client.ContainerInspect(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "", fmt.Errorf("container not found: %s", name)
		}
		return "", err
	}
	return con.ID, nil
}

func resolveContainerGroupService(ctx context.Context, cli *container.ContainerClient, name string) (string, error) {
	items, err := cli.LookupGroupService(ctx, name)
	if err != nil {
		return "", err
	}
	switch len(items) {
	case 0:
		return "", fmt.Errorf("container-group service not found: %s", name)
	case 1:
		return items[0].ID, nil
	default:
		names := make([]string, 0, len(items))
		for _, item := range items {
			names = append(names, container.ConvertName(item.Names))
		}
		return "", fmt.Errorf("container-group service %s is ambiguous, it matches %d containers: %s", name, len(items), strings.Join(names, ", "))
	}
}

// refuseSelf rejects actions on the container hosting the plugin (and thin-edge.io),
// as stopping it would cut off the device and restarting it would kill tedge-agent
// in the middle of the operation workflow
func refuseSelf(ctx context.Context, cli *container.ContainerClient, containerID string, action string) error {
	if !container.IsInsideContainer() {
		return nil
	}
	self, err := cli.Self(ctx)
	if err != nil {
		if !errors.Is(err, errdefs.ErrNotFound) {
			slog.Warn("Could not detect the container hosting thin-edge.io.", "err", err)
		}
		return nil
	}
	if self.ID == containerID {
		return fmt.Errorf("refusing to %s the container hosting thin-edge.io", action)
	}
	return nil
}
