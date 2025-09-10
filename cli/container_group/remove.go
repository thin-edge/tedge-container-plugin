/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type RemoveCommand struct {
	*cobra.Command

	CommandContext cli.Cli
	ModuleVersion  string
}

// removeCmd represents the remove command
func NewRemoveCommand(ctx cli.Cli) *cobra.Command {
	command := &RemoveCommand{
		CommandContext: ctx,
	}
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a container",
		Args:  cobra.ExactArgs(1),
		RunE:  command.RunE,
	}
	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to remove")
	return cmd
}

func (c *RemoveCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
	ctx := context.Background()
	projectName := args[0]

	cli, err := container.NewContainerClient(context.TODO(), c.CommandContext.GetContainerClientOptions()...)
	if err != nil {
		return err
	}

	persistentDir, err := c.CommandContext.PersistentDir(false)
	if err != nil {
		return err
	}
	workingDir := filepath.Join(persistentDir, "compose", projectName)
	return cli.ComposeDown(ctx, cmd.ErrOrStderr(), projectName, workingDir)
}
