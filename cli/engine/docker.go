/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package engine

import (
	"context"
	"log/slog"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type DockerCommand struct {
	*cobra.Command
}

// NewRunCommand create a new run command
func NewRunCommand(ctx cli.Cli) *cobra.Command {
	command := &DockerCommand{}
	cmd := &cobra.Command{
		Use:                "docker",
		Short:              "Run a command using the detected container engine",
		Long:               `The command allows you to run the underlying container engine cli commands directly but ensuring the same DOCKER_HOST is being used`,
		RunE:               command.RunE,
		DisableFlagParsing: true,
		SilenceUsage:       true,
	}
	command.Command = cmd
	return cmd
}

func (c *DockerCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Debug("Executing", "cmd", cmd.CalledAs(), "args", args)
	containerCli, err := container.NewContainerClient(context.TODO())
	if err != nil {
		return err
	}

	bin, binArgs, err := containerCli.DockerCommand(args...)
	if err != nil {
		return err
	}

	command := exec.Command(bin, binArgs...)
	command.Stderr = cmd.ErrOrStderr()
	command.Stdout = cmd.OutOrStdout()
	command.Stdin = cmd.InOrStdin()

	runErr := command.Run()
	if runErr != nil {
		return cli.ExitCodeError{
			Err:    runErr,
			Silent: true,
		}
	}
	return nil
}
