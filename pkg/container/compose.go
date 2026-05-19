package container

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/thin-edge/tedge-container-plugin/pkg/cmdbuilder"
	"go.yaml.in/yaml/v3"

	composeCli "github.com/compose-spec/compose-go/v2/cli"
)

// execCommand is a variable to enable mocking of exec.Command
var execCommand = exec.Command

// detectComposeFunc is a variable to enable mocking of detectCompose
var detectComposeFunc = detectCompose

func parseValueWithPrefix(in string, prefix string) string {
	for _, line := range strings.Split(in, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.Split(strings.TrimPrefix(line, prefix), ",")[0]
		}
	}
	return ""
}

func parsePodmanComposeVersion(in string) string {
	return parseValueWithPrefix(in, "podman-compose version ")
}

func parseDockerComposeVersion(in string) string {
	if v := parseValueWithPrefix(in, "Docker Compose version "); v != "" {
		return v
	}
	return parseValueWithPrefix(in, "docker-compose version ")
}

func detectCompose() (*cmdbuilder.Command, error) {
	composeBackends := []struct {
		Command     cmdbuilder.BaseCommand
		VersionFunc func(string) string
	}{
		{
			Command:     cmdbuilder.NewBaseCommand("docker", "compose"),
			VersionFunc: parseDockerComposeVersion,
		},
		{
			Command:     cmdbuilder.NewBaseCommand("docker-compose"),
			VersionFunc: parseDockerComposeVersion,
		},
		{
			Command:     cmdbuilder.NewBaseCommand("podman-compose"),
			VersionFunc: parsePodmanComposeVersion,
		},
	}

	for _, backend := range composeBackends {
		output, err := execCommand(backend.Command.Name(), backend.Command.Args("version")...).CombinedOutput()
		if err == nil {
			ver, verErr := version.NewVersion(backend.VersionFunc(string(output)))
			if verErr != nil {
				ver, _ = version.NewVersion("0.0.0")
			}
			composeCommand := &cmdbuilder.Command{
				Base:    backend.Command,
				Args:    []string{},
				Version: ver,
			}
			return composeCommand, nil
		}
		slog.Info("command failed.", "command", backend.Command.Name(), "output", output)
	}

	return nil, fmt.Errorf("compose cli not found")
}

func prepareComposeCommand(args ...string) (string, []string, error) {
	command, err := detectComposeFunc()
	if err != nil {
		return "", []string{}, err
	}

	command.Args = append(command.Args, args...)

	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	// Normalized differences between the commands
	err = cmdbuilder.WithConditionalFlags(
		command,
		subcommand,
		// podman-compose down does not support "--remove-orphans" argument, so strip it out
		cmdbuilder.RemoveFlag("podman-compose", "down", "--remove-orphans", nil),

		// Due to a bug in podman-compose where it swallows the exit code, the output is parsed
		// to check of any errors, however in newer podman versions, e.g. podman 5.2
		// https://github.com/thin-edge/tedge-container-plugin/issues/70
		cmdbuilder.PrependFlag("podman-compose", "up", "--verbose", cmdbuilder.MustVersionConstraint(">=1.1.0")),
	)

	return command.Base.Name(), command.Base.Args(command.Args...), err
}

func ReadImages(ctx context.Context, paths []string, workingDir string) ([]string, error) {
	images := make([]string, 0)

	project, err := composeCli.NewProjectOptions(
		paths,
		composeCli.WithDotEnv,
	)
	if err != nil {
		return images, err
	}

	projectT, err := project.LoadProject(ctx)
	if err != nil {
		return images, err
	}
	for name, service := range projectT.Services {
		if service.Image != "" {
			slog.Info("Found image for service.", "service", name, "image", service.Image)
			images = append(images, service.Image)
		}
	}

	return images, nil
}

// EnsureExtraHost ensures every service in composePaths[0] that does not
// already define hostname in extra_hosts has "hostname=ipValue" added.
// Services that already define the hostname (under either the "=" or ":"
// separator convention) are left unchanged. The file is not written when
// no patch is required. The yaml node tree is modified in-place so
// existing formatting and comments are preserved.
func EnsureExtraHost(ctx context.Context, composePaths []string, _ string, hostname, ipValue string) error {
	if len(composePaths) == 0 {
		return nil
	}
	composePath := composePaths[0]

	project, err := composeCli.NewProjectOptions(composePaths, composeCli.WithDotEnv)
	if err != nil {
		return err
	}
	projectT, err := project.LoadProject(ctx)
	if err != nil {
		return err
	}

	needsPatch := make(map[string]bool)
	for name, service := range projectT.Services {
		if _, exists := service.ExtraHosts[hostname]; !exists {
			needsPatch[name] = true
		}
	}
	if len(needsPatch) == 0 {
		return nil
	}

	data, err := os.ReadFile(composePath)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected yaml document structure in %s", composePath)
	}
	root := doc.Content[0]

	servicesNode := findMappingValue(root, "services")
	if servicesNode == nil {
		return fmt.Errorf("no 'services' key found in %s", composePath)
	}
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		if name := servicesNode.Content[i].Value; needsPatch[name] {
			addExtraHostEntry(servicesNode.Content[i+1], hostname, ipValue)
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	services := make([]string, 0, len(needsPatch))
	for name := range needsPatch {
		services = append(services, name)
	}
	slog.Info("Added extra_hosts entry to compose file.", "file", composePath, "hostname", hostname, "ip", ipValue, "services", services)
	return os.WriteFile(composePath, buf.Bytes(), 0644)
}

// findMappingValue returns the value node for key within a YAML MappingNode,
// or nil if the key is not present.
func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// addExtraHostEntry appends "hostname=ipValue" to the extra_hosts of a
// service MappingNode. Both sequence-style and mapping-style extra_hosts are
// handled. The hostname is matched against both "=" and ":" separator styles
// to avoid creating duplicates.
func addExtraHostEntry(serviceNode *yaml.Node, hostname, ipValue string) {
	// Use ":" separator for maximum compatibility: docker-compose v1 (Python,
	// <=1.29) only accepts ":" and passes entries verbatim to "docker run
	// --add-host". The "=" format is compose-spec v2 only. compose-go
	// accepts both separators when reading, so this works with all versions.
	entry := hostname + ":" + ipValue

	extraHostsVal := findMappingValue(serviceNode, "extra_hosts")
	if extraHostsVal == nil {
		serviceNode.Content = append(serviceNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "extra_hosts"},
			&yaml.Node{
				Kind: yaml.SequenceNode,
				Tag:  "!!seq",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: entry},
				},
			},
		)
		return
	}

	// A null/empty scalar extra_hosts (e.g. "extra_hosts:" with no value) is
	// promoted to an empty sequence so we can append to it.
	if extraHostsVal.Kind == yaml.ScalarNode {
		extraHostsVal.Kind = yaml.SequenceNode
		extraHostsVal.Tag = "!!seq"
		extraHostsVal.Value = ""
		extraHostsVal.Content = nil
	}

	switch extraHostsVal.Kind {
	case yaml.SequenceNode:
		for _, item := range extraHostsVal.Content {
			for _, sep := range []string{"=", ":"} {
				if h, _, ok := strings.Cut(item.Value, sep); ok && h == hostname {
					return
				}
			}
		}
		extraHostsVal.Content = append(extraHostsVal.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: entry},
		)
	case yaml.MappingNode:
		for i := 0; i+1 < len(extraHostsVal.Content); i += 2 {
			if extraHostsVal.Content[i].Value == hostname {
				return
			}
		}
		extraHostsVal.Content = append(extraHostsVal.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: hostname},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ipValue},
		)
	}
}
