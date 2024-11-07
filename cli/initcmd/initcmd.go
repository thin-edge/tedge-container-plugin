/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package initcmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

type InitCommand struct {
	*cobra.Command

	Format string
}

func NewInitCommand(cliContext cli.Cli) *cobra.Command {
	command := &InitCommand{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Creates the configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.MkdirTemp("", "tedge-container-plugin")
			if err != nil {
				return nil
			}
			defer os.RemoveAll(dir)

			tmpfile := filepath.Join(dir, "config."+command.Format)
			slog.Debug("Writing to temp file.", "path", tmpfile)
			if err := viper.WriteConfigAs(tmpfile); err != nil {
				return err
			}
			b, err := os.ReadFile(tmpfile)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b)
			return err
		},
	}

	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]string{
		"json",
		"toml",
		"yaml",
	}, cobra.ShellCompDirectiveDefault))

	cmd.Flags().StringVar(&command.Format, "format", "toml", "Configuration file format")
	command.Command = cmd
	return cmd
}
