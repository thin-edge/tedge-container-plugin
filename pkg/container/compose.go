package container

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
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
