package container

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	composeCli "github.com/compose-spec/compose-go/v2/cli"
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

// ComposeSettings are the per-project settings read from the compose file.
// A nil value means that the setting was not defined.
type ComposeSettings struct {
	// Remove the project's volumes when the project is removed
	RemoveVolumes *bool `yaml:"remove_volumes"`

	// Time to wait for containers to stop before they are killed
	// when the project is removed
	RemoveTimeout *ComposeDuration `yaml:"remove_timeout"`

	// Keys which are not supported (e.g. due to a typo, or a setting
	// added in a newer version). They are ignored but reported to the user
	UnknownKeys []string `yaml:"-"`
}

// Merge returns the settings with any settings defined in other taking precedence
func (s ComposeSettings) Merge(other ComposeSettings) ComposeSettings {
	if other.RemoveVolumes != nil {
		s.RemoveVolumes = other.RemoveVolumes
	}
	if other.RemoveTimeout != nil {
		s.RemoveTimeout = other.RemoveTimeout
	}
	s.UnknownKeys = append(slices.Clone(s.UnknownKeys), other.UnknownKeys...)
	return s
}

// composeSettingsKeys returns the keys supported under x-tedge
func composeSettingsKeys() []string {
	keys := []string{}
	t := reflect.TypeFor[ComposeSettings]()
	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ","); name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	return keys
}

// ComposeDuration is a duration which accepts either a duration
// string (e.g. "1m30s") or a number of seconds (e.g. 90).
type ComposeDuration time.Duration

func (d *ComposeDuration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("invalid duration on line %d. expected a duration string (e.g. 30s) or number of seconds", value.Line)
	}
	v, err := ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration on line %d. %w", value.Line, err)
	}
	*d = ComposeDuration(v)
	return nil
}

// ParseDuration parses either a duration string (e.g. "1m30s") or a
// number of seconds (e.g. "90"). An empty value is a zero duration.
// This is used for both the compose settings and the plugin configuration
// so that a value has the same meaning in both places.
func ParseDuration(v string) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
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

// FindComposeFiles returns the compose files which docker compose uses by default
// in the given directory: the first main compose file, followed by the
// first override file (if one exists). The same file names and order
// as docker compose are used.
func FindComposeFiles(dir string) []string {
	files := make([]string, 0, 2)
	for _, names := range [][]string{composeCli.DefaultFileNames, composeCli.DefaultOverrideFileNames} {
		for _, name := range names {
			p := filepath.Join(dir, name)
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				files = append(files, p)
				break
			}
		}
		if len(files) == 0 {
			// an override file is only used together with a main file
			break
		}
	}
	return files
}

// ReadComposeSettings reads the x-tedge settings from the given compose files,
// where the settings in later files take precedence.
func ReadComposeSettings(paths ...string) (ComposeSettings, error) {
	settings := ComposeSettings{}
	for _, path := range paths {
		fileSettings, err := readComposeSettingsFile(path)
		if err != nil {
			return settings, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		settings = settings.Merge(fileSettings)
	}
	return settings, nil
}

// readComposeSettingsFile reads the x-tedge settings from a compose file.
// The raw yaml is read (rather than loading the compose project) so that
// the settings can still be read if the project's variables can't be resolved.
func readComposeSettingsFile(path string) (ComposeSettings, error) {
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
	node := &doc.Settings
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	if node.IsZero() || node.Tag == "!!null" {
		return settings, nil
	}
	if err := node.Decode(&settings); err != nil {
		return settings, fmt.Errorf("invalid %s settings. %w", ComposeSettingsKey, err)
	}

	knownKeys := composeSettingsKeys()
	for i := 0; i+1 < len(node.Content); i += 2 {
		if key := node.Content[i].Value; !slices.Contains(knownKeys, key) {
			settings.UnknownKeys = append(settings.UnknownKeys, key)
		}
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
