package serviceplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thin-edge/tedge-container-plugin/pkg/utils"
)

// ServiceControlSetting is the tedge config setting introduced along with the
// service control contract (a single cmd/control command per service, with the
// actions declared in its metadata). Earlier thin-edge.io versions with service
// plugin support used one cmd/<action> command per action instead.
const ServiceControlSetting = "c8y.enable.service_control"

// DetectSupport checks if thin-edge.io runs the given service plugin, i.e. it
// supports the service control contract (the c8y.enable.service_control setting
// exists), the service.plugin_paths setting exists and one of its directories
// holds the plugin. The reason explains the outcome.
func DetectSupport(ctx context.Context, pluginName string) (supported bool, reason string) {
	if err := exec.CommandContext(ctx, "tedge", "config", "get", ServiceControlSetting).Run(); err != nil {
		return false, fmt.Sprintf("thin-edge.io lacks service control support ('tedge config get %s' failed: %s)", ServiceControlSetting, err)
	}

	out, err := exec.CommandContext(ctx, "tedge", "config", "get", "service.plugin_paths").Output()
	if err != nil {
		return false, fmt.Sprintf("thin-edge.io lacks service plugin support ('tedge config get service.plugin_paths' failed: %s)", err)
	}

	paths := ParsePluginPaths(string(out))
	for _, dir := range paths {
		pluginPath := filepath.Join(dir, pluginName)
		if utils.PathExists(pluginPath) {
			return true, fmt.Sprintf("found service plugin %s", pluginPath)
		}
	}
	return false, fmt.Sprintf("the %s service plugin is not installed in any of the service.plugin_paths %v", pluginName, paths)
}

// ParsePluginPaths parses the value of a plugin_paths setting as printed by
// 'tedge config get', e.g. ["/usr/share/tedge/service-plugins"], also accepting
// a plain comma separated list
func ParsePluginPaths(value string) []string {
	value = strings.TrimSpace(value)
	paths := make([]string, 0)
	if err := json.Unmarshal([]byte(value), &paths); err != nil {
		paths = strings.Split(value, ",")
	}

	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
