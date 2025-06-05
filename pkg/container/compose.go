package container

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/thin-edge/tedge-container-plugin/pkg/cmdbuilder"

	composeCli "github.com/compose-spec/compose-go/v2/cli"
)

// execCommand is a variable to enable mocking of exec.Command
var execCommand = exec.Command

// detectComposeFunc is a variable to enable mocking of detectCompose
var detectComposeFunc = detectCompose

func parseValueWithPrefix(in string, prefix string) string {
	for _, line := range strings.Split(in, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.Split(strings.TrimPrefix(line, prefix), ",")[0]
		}
	}
	return ""
}

func parsePodmanComposeVersion(in string) string {
	return parseValueWithPrefix(in, "podman-compose version ")
}

func parseDockerComposeVersion(in string) string {
	if v := parseValueWithPrefix(in, "Docker Compose version "); v != "" {
		return v
	}
	return parseValueWithPrefix(in, "docker-compose version ")
}

func detectCompose() (*cmdbuilder.Command, error) {
	composeBackends := []struct {
		Command     cmdbuilder.BaseCommand
		VersionFunc func(string) string
	}{
		{
			Command:     cmdbuilder.NewBaseCommand("docker", "compose"),
			VersionFunc: parseDockerComposeVersion,
		},
		{
			Command:     cmdbuilder.NewBaseCommand("docker-compose"),
			VersionFunc: parseDockerComposeVersion,
		},
		{
			Command:     cmdbuilder.NewBaseCommand("podman-compose"),
			VersionFunc: parsePodmanComposeVersion,
		},
	}

	for _, backend := range composeBackends {
		output, err := execCommand(backend.Command.Name(), backend.Command.Args("version")...).CombinedOutput()
		if err == nil {
			ver, verErr := version.NewVersion(backend.VersionFunc(string(output)))
			if verErr != nil {
				ver, _ = version.NewVersion("0.0.0")
			}
			composeCommand := &cmdbuilder.Command{
				Base:    backend.Command,
				Args:    []string{},
				Version: ver,
			}
			return composeCommand, nil
		}
		slog.Info("command failed.", "command", backend.Command.Name(), "output", output)
	}

	return nil, fmt.Errorf("compose cli not found")
}

func prepareComposeCommand(args ...string) (string, []string, error) {
	command, err := detectComposeFunc()
	if err != nil {
		return "", []string{}, err
	}

	command.Args = append(command.Args, args...)

	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	// Normalized differences between the commands
	err = cmdbuilder.WithConditionalFlags(
		command,
		subcommand,
		// podman-compose down does not support "--remove-orphans" argument, so strip it out
		cmdbuilder.RemoveFlag("podman-compose", "down", "--remove-orphans", nil),

		// Due to a bug in podman-compose where it swallows the exit code, the output is parsed
		// to check of any errors, however in newer podman versions, e.g. podman 5.2
		// https://github.com/thin-edge/tedge-container-plugin/issues/70
		cmdbuilder.PrependFlag("podman-compose", "up", "--verbose", cmdbuilder.MustVersionConstraint(">=1.1.0")),
	)

	return command.Base.Name(), command.Base.Args(command.Args...), err
}

func ReadImages(ctx context.Context, paths []string, workingDir string) ([]string, error) {
	images := make([]string, 0)

	project, err := composeCli.NewProjectOptions(
		paths,
		composeCli.WithDotEnv,
	)
	if err != nil {
		return images, err
	}

	projectT, err := project.LoadProject(ctx)
	if err != nil {
		return images, err
	}
	for name, service := range projectT.Services {
		if service.Image != "" {
			slog.Info("Found image for service.", "service", name, "image", service.Image)
			images = append(images, service.Image)
		}
	}

	return images, nil
}
