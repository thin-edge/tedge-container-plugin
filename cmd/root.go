/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/cli/container"
	"github.com/thin-edge/tedge-container-plugin/cli/container_group"
	"github.com/thin-edge/tedge-container-plugin/cli/engine"
	"github.com/thin-edge/tedge-container-plugin/cli/initcmd"
	"github.com/thin-edge/tedge-container-plugin/cli/run"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
)

// Build data
var buildVersion string
var buildBranch string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "tedge-container",
	Short:   "thin-edge.io container engine plugin to manage and monitor containers on a device",
	Version: fmt.Sprintf("%s (branch=%s)", buildVersion, buildBranch),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return SetLogLevel()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	args := os.Args
	name := filepath.Base(args[0])
	switch name {
	case "container", "container-group":
		slog.Debug("Calling as a software management plugin.", "name", name, "args", args)
		rootCmd.SetArgs(append([]string{name}, args[1:]...))
	default:
		slog.Debug("Using subcommands.", "args", args)
	}

	err := rootCmd.Execute()
	if err != nil {
		exitCode := 1
		switch vErr := err.(type) {
		case cli.ExitCodeError:
			exitCode = vErr.ExitCode()
			if !vErr.Silent {
				slog.Error("Command error", "err", err)
			}
		default:
			slog.Error("Command error", "err", err)
		}
		os.Exit(exitCode)
	}
}

func SetLogLevel() error {
	value := strings.ToLower(viper.GetString("log_level"))
	slog.Debug("Setting log level.", "new", value)
	switch value {
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	}
	return nil
}

func init() {
	cliConfig := cli.Cli{}
	cobra.OnInitialize(cliConfig.OnInit)
	rootCmd.AddCommand(
		container.NewContainerCommand(cliConfig),
		container_group.NewContainerGroupCommand(cliConfig),
		run.NewRunCommand(cliConfig),
		engine.NewCliCommand(cliConfig),
		initcmd.NewInitCommand(cliConfig),
	)

	rootCmd.PersistentFlags().String("log-level", "info", "Log level")
	rootCmd.PersistentFlags().StringVarP(&cliConfig.ConfigFile, "config", "c", "", "Configuration file")
	rootCmd.SilenceUsage = true

	// viper.Bind
	_ = viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
}
