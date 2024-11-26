/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package self

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

type SoftwareModule struct {
	Type    string         `json:"type,omitempty"`
	Modules []SoftwareItem `json:"modules"`
}

type SoftwareItem struct {
	Action  string `json:"action,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type UpdateInfo struct {
	ContainerName string `json:"containerName"`
	Image         string `json:"image"`
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

var SoftwareManagementType = "self"
var SoftwareManagementActionInstall = "install"

// NewCheckCommand represents the check command
func NewCheckCommand(ctx cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Args:  cobra.ExactArgs(1),
		Short: "Check if an update is required (not part of sm-plugin interface!)",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)

			updateList := make([]SoftwareModule, 0)
			if err := json.Unmarshal([]byte(args[0]), &updateList); err != nil {
				return newUnexpectedError(err)
			}

			includesSelfUpdate := false

			match := UpdateInfo{}
			for _, module := range updateList {
				if module.Type == SoftwareManagementType {
					for _, item := range module.Modules {
						if item.Action == SoftwareManagementActionInstall {
							match.ContainerName = item.Name
							match.Image = item.Version
							includesSelfUpdate = true
							break
						}
					}
					break
				}
			}

			if !includesSelfUpdate {
				return cli.ExitCodeError{
					Code:   ExitNo,
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
		},
	}
}
