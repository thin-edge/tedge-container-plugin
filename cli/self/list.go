/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package self

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

func newNotSupportedErr(err error) error {
	return cli.ExitCodeError{
		Code: 2,
		Err:  err,
	}
}

// listCmd represents the list command
func NewListCommand(cliContext cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List containers",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Executing", "cmd", cmd.CalledAs(), "args", args)

			if CalledAsSMPlugin() {
				slog.Info("Called as sm-plugin, ignoring legacy plugin. 'container' is used for updates")
				return nil
			}

			ctx := context.Background()
			containerCli, err := container.NewContainerClient(context.TODO(), cliContext.GetContainerClientOptions()...)
			if err != nil {
				return newNotSupportedErr(err)
			}

			currentContainer, err := containerCli.Self(ctx)
			if err != nil {
				return newNotSupportedErr(err)
			}

			name := strings.TrimPrefix(currentContainer.Name, "/")
			version := currentContainer.Config.Image
			_, wErr := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", name, version)
			return wErr
		},
	}
}
