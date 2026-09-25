package container

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// ComposeSettingsKey is the top-level compose extension field used to
// customize how tedge-container-plugin manages a container-group.
// The keys under it use the same names as the [container_group]
// settings in the plugin's configuration file, and take precedence over them.
//
//	x-tedge:
//	  remove_volumes: false
//	  remove_timeout: 30s
const ComposeSettingsKey = "x-tedge"

// Compose file names in the order that docker compose looks for them
var composeFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

// ComposeSettings are the per-project settings read from the compose file.
// A nil value means that the setting was not defined.
type ComposeSettings struct {
	// Remove the project's volumes when the project is removed
	RemoveVolumes *bool `yaml:"remove_volumes"`

	// Time to wait for containers to stop before they are killed
	// when the project is removed
	RemoveTimeout *ComposeDuration `yaml:"remove_timeout"`
}

// ComposeDuration is a duration which accepts either a duration
// string (e.g. "1m30s") or a number of seconds (e.g. 90).
type ComposeDuration time.Duration

func (d *ComposeDuration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("invalid duration on line %d. expected a duration string (e.g. 30s) or number of seconds", value.Line)
	}
	v, err := parseComposeDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration on line %d. %w", value.Line, err)
	}
	*d = ComposeDuration(v)
	return nil
}

func parseComposeDuration(v string) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if seconds, err := strconv.ParseFloat(v, 64); err == nil {
		v = fmt.Sprintf("%gs", seconds)
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, fmt.Errorf("duration must not be negative. value=%s", v)
	}
	return d, nil
}

// FindComposeFile returns the path to the compose file in the given
// directory, or an empty string if no compose file exists
func FindComposeFile(dir string) string {
	for _, name := range composeFileNames {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// ReadComposeSettings reads the x-tedge settings from a compose file.
// The raw yaml is read (rather than loading the compose project) so that
// the settings can still be read if the project's variables can't be resolved.
func ReadComposeSettings(path string) (ComposeSettings, error) {
	settings := ComposeSettings{}
	data, err := os.ReadFile(path)
	if err != nil {
		return settings, err
	}
	var doc struct {
		Settings yaml.Node `yaml:"x-tedge"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return settings, err
	}
	if doc.Settings.IsZero() || doc.Settings.Tag == "!!null" {
		return settings, nil
	}
	if err := doc.Settings.Decode(&settings); err != nil {
		return settings, fmt.Errorf("invalid %s settings. %w", ComposeSettingsKey, err)
	}
	return settings, nil
}

// ComposeDownOptions control how a compose project is removed
type ComposeDownOptions struct {
	// Remove the project's volumes
	RemoveVolumes bool

	// Time to wait for containers to stop before they are killed.
	// Zero uses the compose default
	RemoveTimeout time.Duration
}

// WithSettings returns the options with any settings defined in
// the compose file taking precedence
func (o ComposeDownOptions) WithSettings(settings ComposeSettings) ComposeDownOptions {
	if settings.RemoveVolumes != nil {
		o.RemoveVolumes = *settings.RemoveVolumes
	}
	if settings.RemoveTimeout != nil {
		o.RemoveTimeout = time.Duration(*settings.RemoveTimeout)
	}
	return o
}

func (o ComposeDownOptions) timeoutArgs() []string {
	if o.RemoveTimeout <= 0 {
		return nil
	}
	// compose only accepts whole seconds, so round up to never wait less than requested
	seconds := int64(math.Ceil(o.RemoveTimeout.Seconds()))
	return []string{"--timeout", strconv.FormatInt(seconds, 10)}
}

// StopArgs returns the compose arguments used to stop the project
func (o ComposeDownOptions) StopArgs() []string {
	return append([]string{"stop"}, o.timeoutArgs()...)
}

// DownArgs returns the compose arguments used to remove the project
func (o ComposeDownOptions) DownArgs() []string {
	args := []string{"down", "--remove-orphans"}
	if o.RemoveVolumes {
		args = append(args, "--volumes")
	}
	return append(args, o.timeoutArgs()...)
}
