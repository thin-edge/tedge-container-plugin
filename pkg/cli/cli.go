package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/tedge"
	"github.com/thin-edge/tedge-container-plugin/pkg/utils"
)

var LinuxConfigFilePath = "/etc/tedge-container-plugin/config.toml"

type SilentError error

type Cli struct {
	ConfigFile string
}

func (c *Cli) OnInit() {

	// Set shared config
	viper.SetDefault("container.network", "tedge")
	viper.SetDefault("delete_legacy", true)
	viper.SetDefault("data_dir", []string{"/var/tedge-container-plugin", "/data/tedge-container-plugin"})

	if c.ConfigFile != "" && utils.PathExists(c.ConfigFile) {
		// Use config file from the flag.
		viper.SetConfigFile(c.ConfigFile)
	} else {
		if home, err := os.UserHomeDir(); err == nil {
			// Add home directory.
			viper.AddConfigPath(home)
		}

		if utils.PathExists(LinuxConfigFilePath) {
			viper.SetConfigFile(LinuxConfigFilePath)
		} else {
			// Search config in home directory with name ".cobra" (without extension).
			viper.SetConfigType("json")
			viper.SetConfigType("toml")
			viper.SetConfigType("yaml")
			viper.SetConfigName(".tedge-container-plugin")
		}
	}

	viper.SetEnvPrefix("CONTAINER")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "path", viper.ConfigFileUsed())
	}
}

func (c *Cli) GetString(key string) string {
	return viper.GetString(key)
}

func (c *Cli) GetBool(key string) bool {
	return viper.GetBool(key)
}

func (c *Cli) PrintConfig() {
	keys := viper.AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		slog.Info("setting", "item", fmt.Sprintf("%s=%v", key, viper.Get(key)))
	}
}

func (c *Cli) GetServiceName() string {
	return viper.GetString("service_name")
}

func (c *Cli) GetKeyFile() string {
	return viper.GetString("client.key")
}

func (c *Cli) GetCertificateFile() string {
	return viper.GetString("client.cert_file")
}

func (c *Cli) GetCAFile() string {
	return viper.GetString("client.ca_file")
}

func (c *Cli) GetTopicRoot() string {
	return viper.GetString("topic_root")
}

func (c *Cli) GetTopicID() string {
	return viper.GetString("topic_id")
}

func (c *Cli) GetDeviceID() string {
	return viper.GetString("device_id")
}

func (c *Cli) MetricsEnabled() bool {
	return viper.GetBool("metrics.enabled")
}

func (c *Cli) EngineEventsEnabled() bool {
	return viper.GetBool("events.enabled")
}

func (c *Cli) DeleteFromCloud() bool {
	return viper.GetBool("delete_from_cloud.enabled")
}

func (c *Cli) GetMQTTHost() string {
	return viper.GetString("client.mqtt.host")
}

func (c *Cli) GetSharedContainerNetwork() string {
	return viper.GetString("container.network")
}

func (c *Cli) DeleteLegacyService() bool {
	return viper.GetBool("delete_legacy")
}

func (c *Cli) GetMetricsInterval() time.Duration {
	interval := viper.GetDuration("metrics.interval")
	if interval < 60*time.Second {
		slog.Warn("metrics.interval is lower than allowed limit.", "old", interval, "new", 60*time.Second)
		interval = 60 * time.Second
	}
	return interval
}

func (c *Cli) GetMQTTPort() uint16 {
	v := viper.GetUint16("client.mqtt.port")
	if v == 0 {
		if utils.PathExists(c.GetCertificateFile()) && utils.PathExists(c.GetKeyFile()) {
			return 8883
		}
		return 1883
	}
	return v
}

func (c *Cli) GetCumulocityHost() string {
	return viper.GetString("client.c8y.host")
}

func (c *Cli) GetCumulocityPort() uint16 {
	v := viper.GetUint16("monitor.c8y.proxy.client..port")
	if v == 0 {
		return 8001
	}
	return v
}

func (c *Cli) GetDeviceTarget() tedge.Target {
	return tedge.Target{
		RootPrefix:    c.GetTopicRoot(),
		TopicID:       c.GetTopicID(),
		CloudIdentity: c.GetDeviceID(),
	}
}

func (c *Cli) PersistentDir(check_writable bool) string {
	paths := viper.GetStringSlice("data_dir")
	defaultDir := filepath.Join(os.TempDir(), c.GetServiceName())

	if !check_writable {
		if len(paths) > 0 {
			return paths[0]
		}
		return defaultDir
	}

	for _, p := range paths {
		if ok, _ := utils.IsDirWritable(p, 0755); ok {
			return p
		}
	}
	return defaultDir
}

func getExpandedStringSlice(key string) []string {
	v := viper.GetStringSlice(key)
	out := make([]string, 0, len(v))
	for _, item := range v {
		out = append(out, strings.Split(item, ",")...)
	}
	return out
}

func (c *Cli) GetFilterOptions() container.FilterOptions {
	options := container.FilterOptions{
		Names:            getExpandedStringSlice("filter.include.names"),
		IDs:              getExpandedStringSlice("filter.include.ids"),
		Labels:           getExpandedStringSlice("filter.include.labels"),
		Types:            getExpandedStringSlice("filter.include.types"),
		ExcludeNames:     getExpandedStringSlice("filter.exclude.names"),
		ExcludeWithLabel: getExpandedStringSlice("filter.exclude.labels"),
	}
	return options
}
