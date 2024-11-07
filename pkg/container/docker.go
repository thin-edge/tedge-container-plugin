package container

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

func detectDocker() (command string, args []string, err error) {
	composeBackends := [][]string{
		{"docker"},
		{"podman"},
	}

	for _, backend := range composeBackends {
		command := append(backend, "ps")
		slog.Debug("Checking command.", "command", command)
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

func prepareDockerCommand(commandArgs ...string) (command string, args []string, err error) {
	command, args, err = detectDocker()
	if err != nil {
		return
	}
	args = append(args, commandArgs...)
	return command, args, nil
}
