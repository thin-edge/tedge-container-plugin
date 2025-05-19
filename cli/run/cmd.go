/*
Copyright © 2024 thin-edge.io <info@thin-edge.io>
*/
package run

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/app"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

var (
	DefaultServiceName = "tedge-container-plugin"
	DefaultTopicRoot   = "te"
	DefaultTopicPrefix = "device/main//"
)

type RunCommand struct {
	*cobra.Command

	RunOnce bool
}

func NewRunCommand(cliContext cli.Cli) *cobra.Command {
	// runCmd represents the run command
	command := &RunCommand{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the container monitor",
		Long: `Start the container monitor which will periodically publish container information
	to the thin-edge.io interface.
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliContext.PrintConfig()

			device := cliContext.GetDeviceTarget()
			application, err := app.NewApp(device, app.Config{
				ServiceName:        cliContext.GetServiceName(),
				EnableMetrics:      cliContext.MetricsEnabled(),
				DeleteFromCloud:    cliContext.DeleteFromCloud(),
				DeleteOrphans:      cliContext.DeleteOrphans(),
				EnableEngineEvents: cliContext.EngineEventsEnabled(),

				HTTPHost:       cliContext.GetHTTPHost(),
				HTTPPort:       cliContext.GetHTTPPort(),
				MQTTHost:       cliContext.GetMQTTHost(),
				MQTTPort:       cliContext.GetMQTTPort(),
				CumulocityHost: cliContext.GetCumulocityHost(),
				CumulocityPort: cliContext.GetCumulocityPort(),

				KeyFile:  cliContext.GetKeyFile(),
				CertFile: cliContext.GetCertificateFile(),
				CAFile:   cliContext.GetCAFile(),
			})
			if err != nil {
				return err
			}

			// FIXME: Wait until the entity store has been filled
			time.Sleep(200 * time.Millisecond)

			if command.RunOnce {
				// Cleanly stop the application in run-once mode
				// so that the service still appears to be "up" as the Last Will and Testament
				// message should not be sent (as the exit is expected)
				// This logic is similar to SystemD's RemainAfterExit=yes setting
				defer application.Stop(true)
				return application.Update(cliContext.GetFilterOptions())
			}

			// Remove the legacy service
			if cliContext.DeleteLegacyService() {
				go application.DeleteLegacyService(cliContext.DeleteFromCloud())
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			// Start background monitor
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				for {
					slog.Info("Monitor container engine events")
					err := application.Monitor(ctx, cliContext.GetFilterOptions())
					if errors.Is(err, context.Canceled) {
						return
					}
					if err != nil {
						slog.Warn("Monitor stopped. Restarting after 2 seconds.", "err", err)
						time.Sleep(2 * time.Second)
					}
				}
			}()

			if cliContext.MetricsEnabled() {
				go func() {
					_ = backgroundMetric(ctx, cliContext, application, cliContext.GetMetricsInterval())
				}()
			}

			<-stop
			cancel()
			application.Stop(false)
			slog.Info("Shutting down...")
			return nil
		},
	}

	cmd.Flags().String("service-name", DefaultServiceName, "Service name")
	cmd.Flags().StringSlice("name", []string{}, "Only include given container names")
	cmd.Flags().StringSlice("label", []string{}, "Only include containers with the given labels")
	cmd.Flags().StringSlice("id", []string{}, "Only include containers with the given ids")
	cmd.Flags().StringSlice("type", []string{container.ContainerType, container.ContainerGroupType}, "Filter by container type")
	cmd.Flags().String("topic-root", DefaultTopicRoot, "MQTT root prefix")
	cmd.Flags().String("topic-id", DefaultTopicPrefix, "The device MQTT topic identifier")
	cmd.Flags().BoolVar(&command.RunOnce, "once", false, "Only run the monitor once")
	cmd.Flags().String("device-id", "", "thin-edge.io device id")
	cmd.Flags().Duration("interval", 300*time.Second, "Metrics update interval")

	//
	// viper bindings

	// Service
	viper.SetDefault("service_name", DefaultServiceName)
	_ = viper.BindPFlag("service_name", cmd.Flags().Lookup("service-name"))

	// MQTT topics
	viper.SetDefault("topic_root", DefaultTopicRoot)
	_ = viper.BindPFlag("topic_root", cmd.Flags().Lookup("topic-root"))
	viper.SetDefault("topic_id", DefaultTopicPrefix)
	_ = viper.BindPFlag("topic_id", cmd.Flags().Lookup("topic-id"))
	_ = viper.BindPFlag("device_id", cmd.Flags().Lookup("device-id"))

	// Include filters
	_ = viper.BindPFlag("filter.include.names", cmd.Flags().Lookup("name"))
	_ = viper.BindPFlag("filter.include.labels", cmd.Flags().Lookup("label"))
	_ = viper.BindPFlag("filter.include.ids", cmd.Flags().Lookup("id"))
	_ = viper.BindPFlag("filter.include.types", cmd.Flags().Lookup("type"))

	// Exclude filters
	viper.SetDefault("filter.exclude.names", "")
	viper.SetDefault("filter.exclude.labels", []string{"tedge.ignore"})

	// Metrics
	_ = viper.BindPFlag("metrics.interval", cmd.Flags().Lookup("interval"))
	viper.SetDefault("metrics.interval", "300s")
	viper.SetDefault("metrics.enabled", true)

	// Feature flags
	viper.SetDefault("events.enabled", true)
	viper.SetDefault("delete_from_cloud.enabled", true)
	viper.SetDefault("delete_from_cloud.orphans", true)

	// thin-edge.io services
	viper.SetDefault("client.http.host", "127.0.0.1")
	viper.SetDefault("client.http.port", 8000)
	viper.SetDefault("client.mqtt.host", "127.0.0.1")
	// client.mqtt.port: 0 = auto-detection, where 8883 is used when the cert files exist, or 1883 otherwise
	viper.SetDefault("client.mqtt.port", 0)
	viper.SetDefault("client.c8y.host", "127.0.0.1")
	viper.SetDefault("client.c8y.port", 8001)

	// TLS
	viper.SetDefault("client.key", "")
	viper.SetDefault("client.cert_file", "")
	viper.SetDefault("client.ca_file", "")

	command.Command = cmd
	return cmd
}

func backgroundMetric(ctx context.Context, cliContext cli.Cli, application *app.App, interval time.Duration) error {
	timerCh := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping metrics task")
			return ctx.Err()

		case <-timerCh.C:
			go func() {
				slog.Info("Refreshing metrics")
				if err := application.UpdateMetrics(cliContext.GetFilterOptions()); err != nil {
					slog.Warn("Error updating metrics.", "err", err)
				}
			}()

		}
	}
}
