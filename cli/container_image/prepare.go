/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_image

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// prepareCmd represents the prepare command
func NewPrepareCommand(ctx cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "prepare",
		Short: "Prepare for container image install/removal",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
		},
	}
}
