/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package container_image

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// updateListCmd represents the updateList command
func NewUpdateListCommand(ctx cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "update-list",
		Short: "Not implemented",
		Long:  `Not implemented`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("update-list is not supported")
			os.Exit(1)
		},
	}
}
