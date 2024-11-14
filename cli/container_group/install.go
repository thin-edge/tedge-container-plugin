/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/codeclysm/extract/v4"
	"github.com/spf13/cobra"
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
		Short: "Install/run a container-group",
		Args:  cobra.ExactArgs(1),
		RunE:  command.RunE,
	}

	cmd.Flags().StringVar(&command.ModuleVersion, "module-version", "", "Software version to install")
	cmd.Flags().StringVar(&command.File, "file", "", "File")
	command.Command = cmd
	return cmd
}

func (c *InstallCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
	projectName := args[0]
	stderr := cmd.ErrOrStderr()

	cli, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Run docker compose down before up
	// TODO: Move to settings file
	downFirst := false
	workingDir := filepath.Join(c.CommandContext.PersistentDir(true), "compose", projectName)

	// Stop project
	if downFirst && utils.PathExists(workingDir) {
		if err := cli.ComposeDown(ctx, stderr, projectName); err != nil {
			slog.Warn("Compose down failed, but continuing anyway.", "err", err)
		}
	}

	// Check artifact type
	file, err := os.Open(c.File)
	if err != nil {
		return err
	}

	slog.Info("Creating project directory.", "path", workingDir)
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return err
	}

	composeUpExtraArgs := []string{"--build"}
	if err := extract.Archive(ctx, file, workingDir, nil); err != nil {
		// Fallback to treating it as a text file
		dst := filepath.Join(workingDir, "docker-compose.yaml")
		slog.Info("Copying file.", "src", c.File, "dst", dst)
		if err := utils.CopyFile(c.File, dst); err != nil {
			return err
		}
		composeUpExtraArgs = []string{}
	}

	// Create shared network
	if err := cli.CreateSharedNetwork(ctx, c.CommandContext.GetSharedContainerNetwork()); err != nil {
		return err
	}

	if err := cli.ComposeUp(ctx, stderr, projectName, workingDir, composeUpExtraArgs...); err != nil {
		slog.Error("Failed to start compose project.", "err", err)
		return err
	}

	versionFile := filepath.Join(workingDir, "version")
	slog.Info("Writing version to file.", "path", versionFile, "version", c.ModuleVersion)
	if err := os.WriteFile(versionFile, []byte(c.ModuleVersion), 0644); err != nil {
		return err
	}

	return nil
}
