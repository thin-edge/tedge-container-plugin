/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package self

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

type SoftwareModule struct {
	Type    string         `json:"type,omitempty"`
	Modules []SoftwareItem `json:"modules"`
}

type SoftwareItem struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Url     string `json:"url,omitempty"`
	Action  string `json:"action,omitempty"`
}

type UpdateInfo struct {
	ContainerName string           `json:"containerName"`
	Image         string           `json:"image"`
	UpdateList    []SoftwareModule `json:"updateList"`
}

var ExitYes = 0
var ExitNo = 1
var ExitError = 2

func newUnexpectedError(err error) error {
	return cli.ExitCodeError{
		Err:  err,
		Code: ExitError,
	}
}

var SoftwareManagementTypeSelf = "self"
var SoftwareManagementTypeContainer = "container"
var SoftwareManagementActionInstall = "install"
var SoftwareManagementActionRemove = "remove"

type CheckCommand struct {
	*cobra.Command

	ContainerName string
}

// NewCheckCommand represents the check command
func NewCheckCommand(ctx cli.Cli) *cobra.Command {
	command := &CheckCommand{}
	cmd := &cobra.Command{
		Use:   "check",
		Args:  cobra.ExactArgs(1),
		Short: "Check if an update is required (not part of sm-plugin interface!)",
		RunE:  command.RunE,
	}
	cmd.Flags().StringVar(&command.ContainerName, "container", "", "Set current container name")
	command.Command = cmd
	return command.Command
}

func (c *CheckCommand) RunE(cmd *cobra.Command, args []string) error {
	slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)

	updateList := make([]SoftwareModule, 0)
	if err := json.Unmarshal([]byte(args[0]), &updateList); err != nil {
		return newUnexpectedError(err)
	}

	// Check container self
	containerCLI, err := container.NewContainerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	selfContainerName := c.ContainerName
	selfContainerID := ""
	if con, err := containerCLI.Self(ctx); err == nil {
		selfContainerName = strings.TrimPrefix(con.Name, "/")
		selfContainerID = con.ID
	} else if selfContainerName == "" {
		slog.Info("self container.", "err", err)
	}

	includesSelfUpdate := false
	outputUpdateModules := make([]SoftwareModule, 0)

	match := UpdateInfo{}
	for _, module := range updateList {
		if module.Type == SoftwareManagementTypeSelf {
			// self update (to be removed in the future)
			for _, item := range module.Modules {
				if item.Action == SoftwareManagementActionInstall {
					match.ContainerName = item.Name
					match.Image = item.Version
					includesSelfUpdate = true
					break
				}
			}
			break
		} else if module.Type == SoftwareManagementTypeContainer {
			// container update
			// Filter non self-updated modules
			filteredModules := make([]SoftwareItem, 0)

			for _, item := range module.Modules {
				if selfContainerName == item.Name {
					if item.Action == SoftwareManagementActionRemove {
						// Protect against the user deleting the tedge container itself
						return fmt.Errorf("tedge's own container cannot be removed. name=%s, version=%s, containerId=%s", item.Name, item.Version, selfContainerID)
					} else if item.Action == SoftwareManagementActionInstall {
						// install
						match.ContainerName = item.Name
						match.Image = item.Version
						includesSelfUpdate = true
					}
				} else {
					filteredModules = append(filteredModules, item)
				}
			}
			if len(filteredModules) > 0 {
				outputUpdateModules = append(outputUpdateModules, SoftwareModule{
					Type:    module.Type,
					Modules: filteredModules,
				})
			}
		} else {
			outputUpdateModules = append(outputUpdateModules, module)
		}
	}

	match.UpdateList = outputUpdateModules

	// Check if the container is itself
	if !includesSelfUpdate {
		return cli.ExitCodeError{
			Code:   ExitNo,
			Err:    fmt.Errorf("no self-update detected"),
			Silent: true,
		}
	}

	slog.Info("Update included a self update")
	payload, err := json.Marshal(match)
	if err != nil {
		return newUnexpectedError(err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), ":::begin-tedge:::\n%s\n:::end-tedge:::\n", payload)

	// includes self update
	return nil
}
