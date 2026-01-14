package container_image

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// IsEnabled check if the container-image software management plugin is enabled or not
func IsEnabled(cmdCli cli.Cli) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		enabled := cmdCli.GetBool("container_image.enabled")
		if !enabled {
			slog.Info("The container-image sm-plugin is not enabled. Enabled it using the 'container_image.enabled' setting or 'CONTAINER_CONTAINER_IMAGE_ENABLED' env variable")
			return cli.ExitCodeError{
				Code:   1,
				Err:    fmt.Errorf("container-image is not enabled"),
				Silent: true,
			}
		}
		return nil
	}
}

// NewCommand returns a cobra command for `container` subcommands
func NewCommand(cmdCli cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container-image",
		Short: "container-image software management plugin",
	}
	viper.SetDefault("container_image.enabled", false)
	cmd.AddCommand(
		NewPrepareCommand(cmdCli),
		NewInstallCommand(cmdCli),
		NewRemoveCommand(cmdCli),
		NewUpdateListCommand(cmdCli),
		NewListCommand(cmdCli),
		NewFinalizeCommand(cmdCli),
	)
	return cmd
}
