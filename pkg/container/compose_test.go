package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/thin-edge/tedge-container-plugin/pkg/cmdbuilder"
)

func Test_Podman(t *testing.T) {
	output := `
podman-compose build
['podman', '--version', '']
using podman version: 4.3.1
podman build -t app1_app1 -f ./Dockerfile .
STEP 1/2: FROM nginx
Error: creating build container: short-name "nginx" did not resolve to an alias and no unqualified-search registries are defined in "/etc/containers/registries.conf"
exit code: 125
`
	if CheckPodmanComposeError(output) == nil {
		t.Fail()
	}
}

func Test_PodmanMultipleExitCodes(t *testing.T) {
	output := `
['podman', '--version', '']
using podman version: 4.3.1
** excluding:  set()
podman volume inspect nodered_node_red_data || podman volume create nodered_node_red_data
['podman', 'volume', 'inspect', 'nodered_node_red_data']
['podman', 'network', 'exists', 'tedge']
podman run --name=nodered_nodered_1 -d --label io.podman.compose.config-hash=123 --label io.podman.compose.project=nodered --label io.podman.compose.version=0.0.1 --label com.docker.compose.project=nodered --label com.docker.compose.project.working_dir=/var/tedge-container-plugin/compose/nodered --label com.docker.compose.project.config_files=docker-compose.yaml --label com.docker.compose.container-number=1 --label com.docker.compose.service=nodered -e NODE_RED_ENABLE_PROJECTS=false -e TEDGE_MQTT_HOST=host.containers.internal -e TEDGE_MQTT_PORT=1884 -v nodered_node_red_data:/data --net tedge --network-alias nodered -p 1880:1880 docker.io/nodered/node-red:4.0.3-22-minimal
Error: creating container storage: the container name "nodered_nodered_1" is already in use by 557756c5b134746b00c128f36f01627635d2f01b491127ea4a206e8381af0310. You have to remove that container to be able to reuse that name: that name is already in use
exit code: 125
podman start nodered_nodered_1
exit code: 0
`
	if CheckPodmanComposeError(output) != nil {
		t.Errorf("did not expect an error")
	}
}

func TestReadImages(t *testing.T) {
	contents := `
services:
  app1:
    image: hello-world

  app2:
    build: "."
  app3:
    image: another-image:latest
`
	workingDir, err := os.MkdirTemp("", "compose")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workingDir)

	composeFile := filepath.Join(workingDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(contents), 0o644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	images, err := ReadImages(context.Background(), []string{composeFile}, workingDir)
	assert.NoError(t, err)
	assert.Len(t, images, 2)
	assert.Contains(t, images, "hello-world")
	assert.Contains(t, images, "another-image:latest")

	t.Run("no image field", func(t *testing.T) {
		noImageContents := `
services:
  app1:
    build: "."
`
		noImageComposeFile := filepath.Join(workingDir, "docker-compose-no-image.yml")
		if err := os.WriteFile(noImageComposeFile, []byte(noImageContents), 0o644); err != nil {
			t.Fatalf("Failed to write compose file: %v", err)
		}
		images, err := ReadImages(context.Background(), []string{noImageComposeFile}, workingDir)
		assert.NoError(t, err)
		assert.Empty(t, images)
	})

	t.Run("invalid compose file path", func(t *testing.T) {
		_, err := ReadImages(context.Background(), []string{"/non/existent/path/docker-compose.yml"}, workingDir)
		assert.Error(t, err)
	})

	t.Run("invalid compose file content", func(t *testing.T) {
		invalidContents := `services: app1: image: hello-world` // malformed yaml
		invalidComposeFile := filepath.Join(workingDir, "docker-compose-invalid.yml")
		if err := os.WriteFile(invalidComposeFile, []byte(invalidContents), 0o644); err != nil {
			t.Fatalf("Failed to write compose file: %v", err)
		}
		_, err := ReadImages(context.Background(), []string{invalidComposeFile}, workingDir)
		assert.Error(t, err)
	})
}

func TestParsePodmanComposeVersion(t *testing.T) {
	tests := []struct {
		name        string
		commandName string
		input       string
		expected    string
	}{
		{"podman-compose valid", "podman-compose", "podman-compose version 1.0.3\nother stuff", "1.0.3"},
		{"podman-compose with extra details", "podman-compose", "podman-compose version 1.0.6\n['podman', '--version', '']\nusing podman version: 4.3.1", "1.0.6"},
		{"podman-compose no version line", "podman-compose", "some other output", ""},
		{"podman-compose invalid version", "podman-compose", "podman-compose version not-a-version", "not-a-version"},
		{"other command", "docker", "Docker version 20.10.7", ""},
		{"empty input", "podman-compose", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := parsePodmanComposeVersion(tt.input)
			assert.Equal(t, tt.expected, v)
		})
	}
}

func TestParseDockerComposeVersion(t *testing.T) {
	tests := []struct {
		name        string
		commandName string
		input       string
		expected    string
	}{
		// docker compose
		{
			"docker compose valid",
			"docker compose",
			"Docker Compose version 2.29.7",
			"2.29.7",
		},
		{
			"docker-compose v2 alias valid",
			"docker-compose",
			"Docker Compose version 2.29.7",
			"2.29.7",
		},

		// docker-compose
		{
			"docker-compose valid",
			"docker-compose",
			`docker-compose version 1.29.2, build unknown
docker-py version: 5.0.3
CPython version: 3.11.2
OpenSSL version: OpenSSL 3.0.16 11 Feb 2025`,
			"1.29.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := parseDockerComposeVersion(tt.input)
			assert.Equal(t, tt.expected, v)
		})
	}
}

// TestHelperProcess isn't a real test. It's used as a helper process
// to simulate external command execution.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	if len(args) < 2 { // command + subcommand (e.g. docker compose or podman-compose version)
		fmt.Fprintf(os.Stderr, "Helper process: Insufficient arguments: %v\n", args)
		os.Exit(1)
	}

	cmd, subArgs := args[0], args[1:]
	fullCmd := strings.Join(args, " ")

	// Simulate "no compose found" if this env var is set
	if os.Getenv("GO_TEST_SIMULATE_NO_COMPOSE_FOUND") == "1" {
		fmt.Fprintf(os.Stderr, "Helper process: Simulating command not found for %s\n", fullCmd)
		os.Exit(1)
	}

	if strings.Contains(cmd, "docker") && subArgs[0] == "compose" && subArgs[1] == "version" {
		fmt.Fprintln(os.Stdout, "Docker Compose version v2.10.0")
		os.Exit(0)
	}
	if strings.Contains(cmd, "docker-compose") && subArgs[0] == "version" {
		fmt.Fprintln(os.Stdout, "docker-compose version 1.29.2, build abcdef")
		os.Exit(0)
	}
	if strings.Contains(cmd, "podman-compose") && subArgs[0] == "version" {
		// Use the version provided by GO_TEST_PODMAN_COMPOSE_VERSION if set
		versionOutput := "podman-compose version 1.0.6\n['podman', '--version', '']\nusing podman version: 4.3.1"
		if v := os.Getenv("GO_TEST_PODMAN_COMPOSE_VERSION"); v != "" {
			versionOutput = fmt.Sprintf("podman-compose version %s\n['podman', '--version', '']\nusing podman version: 4.x.x", v)
		}
		fmt.Fprintln(os.Stdout, versionOutput)
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "Helper process: Unknown command: %s\n", fullCmd)
	os.Exit(1)
}

func TestDetectCompose(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	// Mock exec.Command
	execCommand = func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_TEST_WANT_HELPER_PROCESS=1")
		return cmd
	}

	t.Run("docker compose found", func(t *testing.T) {
		// TestHelperProcess is set up to make "docker compose version" succeed by default

		cmd, err := detectCompose()
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, "docker", cmd.Base.Name())
		assert.Equal(t, []string{"compose"}, cmd.Base.Args()) // version is stripped
		assert.Equal(t, "2.10.0", cmd.Version.String())
	})

	// To test other backends, we'd need to make TestHelperProcess more dynamic
	// or run sub-tests that set environment variables to make only one backend "succeed".
	// For simplicity, the current TestHelperProcess prioritizes docker compose.

	t.Run("no compose found", func(t *testing.T) {
		os.Setenv("GO_TEST_SIMULATE_NO_COMPOSE_FOUND", "1")
		defer os.Unsetenv("GO_TEST_SIMULATE_NO_COMPOSE_FOUND")

		cmd, err := detectCompose()
		assert.Error(t, err)
		assert.Nil(t, cmd)
		assert.Contains(t, err.Error(), "compose cli not found")
	})
}

func TestPrepareComposeCommand(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	execCommand = func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_TEST_WANT_HELPER_PROCESS=1")
		return cmd
	}

	t.Run("docker compose up", func(t *testing.T) {
		// TestHelperProcess will make "docker compose version" succeed
		cmdName, args, err := prepareComposeCommand("up", "--detach")
		assert.NoError(t, err)
		assert.Equal(t, "docker", cmdName)
		assert.Equal(t, []string{"compose", "up", "--detach"}, args)
	})

	t.Run("podman-compose down - remove-orphans removed", func(t *testing.T) {
		// Make podman-compose the "detected" one by simulating others fail
		// This is tricky with the current helper, so we'll assume podman-compose was detected
		// and test the argument manipulation directly.
		// For a more robust test, TestHelperProcess would need more complex logic
		// or we'd need to simulate the detection order.

		// Simulate podman-compose was detected
		os.Setenv("GO_TEST_PODMAN_COMPOSE_VERSION", "1.0.6") // Version < 1.1.0
		defer os.Unsetenv("GO_TEST_PODMAN_COMPOSE_VERSION")

		// Temporarily override detectCompose for this specific sub-test
		// This is a common pattern if direct exec mocking is too complex for all scenarios
		oldDetect := detectComposeFunc
		detectComposeFunc = func() (*cmdbuilder.Command, error) {
			v, _ := version.NewVersion("1.0.6")
			return &cmdbuilder.Command{Base: cmdbuilder.NewBaseCommand("podman-compose"), Args: []string{}, Version: v}, nil
		}
		defer func() { detectComposeFunc = oldDetect }()

		cmdName, args, err := prepareComposeCommand("down", "--remove-orphans", "--volumes")
		assert.NoError(t, err)
		assert.Equal(t, "podman-compose", cmdName)
		assert.Equal(t, []string{"down", "--volumes"}, args) // --remove-orphans should be gone
	})

	t.Run("podman-compose up - verbose added for version >= 1.1.0", func(t *testing.T) {
		os.Setenv("GO_TEST_PODMAN_COMPOSE_VERSION", "1.1.0")
		defer os.Unsetenv("GO_TEST_PODMAN_COMPOSE_VERSION")

		oldDetect := detectComposeFunc
		detectComposeFunc = func() (*cmdbuilder.Command, error) {
			v, _ := version.NewVersion("1.1.0")
			return &cmdbuilder.Command{Base: cmdbuilder.NewBaseCommand("podman-compose"), Args: []string{}, Version: v}, nil
		}
		defer func() { detectComposeFunc = oldDetect }()

		cmdName, args, err := prepareComposeCommand("up", "--detach")
		assert.NoError(t, err)
		assert.Equal(t, "podman-compose", cmdName)
		assert.Equal(t, []string{"--verbose", "up", "--detach"}, args)
	})

	t.Run("podman-compose up - verbose NOT added for version < 1.1.0", func(t *testing.T) {
		os.Setenv("GO_TEST_PODMAN_COMPOSE_VERSION", "1.0.0")
		defer os.Unsetenv("GO_TEST_PODMAN_COMPOSE_VERSION")

		oldDetect := detectComposeFunc
		detectComposeFunc = func() (*cmdbuilder.Command, error) {
			v, _ := version.NewVersion("1.0.0")

			return &cmdbuilder.Command{Base: cmdbuilder.NewBaseCommand("podman-compose"), Args: []string{}, Version: v}, nil
		}
		defer func() { detectComposeFunc = oldDetect }()

		cmdName, args, err := prepareComposeCommand("up", "--detach")
		assert.NoError(t, err)
		assert.Equal(t, "podman-compose", cmdName)
		assert.Equal(t, []string{"up", "--detach"}, args)
	})
}
