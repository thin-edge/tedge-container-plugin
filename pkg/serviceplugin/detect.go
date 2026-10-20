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

// DetectSupport checks if thin-edge.io runs the given service plugin, i.e. the
// service.plugin_paths setting exists and one of its directories holds the plugin.
// The reason explains the outcome.
func DetectSupport(ctx context.Context, pluginName string) (supported bool, reason string) {
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
