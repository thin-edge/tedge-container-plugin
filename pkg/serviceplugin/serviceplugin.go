// Package serviceplugin implements the actions of the thin-edge.io service plugins
// for the container and container-group service types.
// See https://thin-edge.github.io/thin-edge.io/next/references/service-plugin-api/
package serviceplugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

// ErrUnsupportedAction is returned for an action the plugin does not implement.
// It maps to the exit code 2 reserved by the service plugin API.
var ErrUnsupportedAction = errors.New("unsupported action")

// ExitCodeUnsupported is the exit code reserved by the service plugin API
// for an action which is not supported
const ExitCodeUnsupported = 2

type actionFunc func(c *container.ContainerClient, ctx context.Context, containerID string) error

var actions = map[string]actionFunc{
	"start":   (*container.ContainerClient).StartContainer,
	"stop":    (*container.ContainerClient).StopContainer,
	"restart": (*container.ContainerClient).RestartContainer,
	"pause":   (*container.ContainerClient).PauseContainer,
	"unpause": (*container.ContainerClient).UnpauseContainer,
}

// declarableActions are the actions which can be declared on a service, as
// tedge-agent ships an operation workflow for them. Other actions, like pause
// and unpause, need a custom workflow before they can be used from the cloud.
var declarableActions = []string{"start", "stop", "restart"}

// DeclarableActions returns the actions which can be declared on a service
func DeclarableActions() []string {
	return slices.Clone(declarableActions)
}

// Actions returns the sorted names of all supported actions
func Actions() []string {
	names := make([]string, 0, len(actions))
	for name := range actions {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// IsSupported checks if an action is implemented by the plugin
func IsSupported(action string) bool {
	_, ok := actions[action]
	return ok
}

// FilterActions splits a list of action names into the declarable ones
// (deduplicated, in the given order) and the others
func FilterActions(names []string) (valid []string, unknown []string) {
	valid = make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if !slices.Contains(declarableActions, name) {
			unknown = append(unknown, name)
			continue
		}
		if !slices.Contains(valid, name) {
			valid = append(valid, name)
		}
	}
	return valid, unknown
}

// Run executes an action on a container
func Run(ctx context.Context, cli *container.ContainerClient, action string, containerID string) error {
	fn, ok := actions[action]
	if !ok {
		return fmt.Errorf("%w '%s'", ErrUnsupportedAction, action)
	}
	return fn(cli, ctx, containerID)
}
