/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_group

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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
	persistentDir, err := c.CommandContext.PersistentDir(true)
	if err != nil {
		return err
	}
	workingDir := filepath.Join(persistentDir, "compose", projectName)

	// Stop project
	if downFirst {
		if err := cli.ComposeDown(ctx, stderr, projectName, workingDir); err != nil {
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

	composeFile := filepath.Join(workingDir, "docker-compose.yaml")

	composeUpExtraArgs := []string{"--build"}
	if err := extract.Archive(ctx, file, workingDir, nil); err != nil {
		// Fallback to treating it as a text file
		slog.Info("Copying file.", "src", c.File, "dst", composeFile)
		if err := utils.CopyFile(c.File, composeFile); err != nil {
			return err
		}
		composeUpExtraArgs = []string{}
	}

	// Pull images which allows uses to avoid having to set any private credentials
	// as tedge-container-plugin supports user set credentials
	if !utils.PathExists(composeFile) {
		if p := filepath.Join(workingDir, "docker-compose.yml"); utils.PathExists(p) {
			composeFile = p
		}
	}
	images, err := container.ReadImages(ctx, []string{composeFile}, workingDir)
	if err != nil {
		return err
	}
	for _, imageRef := range images {
		if _, err := cli.ImagePullWithRetries(ctx, imageRef, c.CommandContext.ImageAlwaysPull(), container.ImagePullOptions{
			AuthFunc:    c.CommandContext.GetContainerRepositoryCredentialsFunc(imageRef),
			MaxAttempts: 2,
			Wait:        5 * time.Second,
		}); err != nil {
			// Proceed anyway so docker-compose can potentially pull in the images
			slog.Warn("Error whilst pulling images. Trying to proceed anyway.", "err", err)
		}
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
