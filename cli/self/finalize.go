/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package self

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

func NewFinalizeCommand(ctx cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finalize",
		Short: "Finalize container install/remove operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)
			return nil
		},
	}
	viper.SetDefault("container.pruneImages", true)
	return cmd
}
