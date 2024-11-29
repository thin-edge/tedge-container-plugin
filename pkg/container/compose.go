package container

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	composeCli "github.com/compose-spec/compose-go/v2/cli"
)

func detectCompose() (command string, args []string, err error) {
	composeBackends := [][]string{
		{"docker", "compose"},
		{"docker-compose"},
		{"podman-compose"},
	}

	for _, backend := range composeBackends {
		command := append(backend, "version")
		output, err := exec.Command(command[0], command[1:]...).CombinedOutput()
		if err == nil {
			if len(command) == 1 {
				return command[0], []string{}, nil
			}
			return command[0], command[1 : len(command)-1], nil
		}
		slog.Info("command failed.", "command", strings.Join(backend, " "), "output", output)
	}

	return "", nil, fmt.Errorf("compose cli not found")
}

func prepareComposeCommand(commandArgs ...string) (command string, args []string, err error) {
	command, args, err = detectCompose()
	if err != nil {
		return
	}

	// Normalized differences between the commands
	if command == "podman-compose" && len(commandArgs) > 0 && commandArgs[0] == "down" {
		// Note: podman-compose down does not support "--remove-orphans" argument, so strip it out
		commandArgs = filter(commandArgs, func(s string) bool {
			return s != "--remove-orphans"
		})
	}

	args = append(args, commandArgs...)
	return command, args, nil
}

func filter(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
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
