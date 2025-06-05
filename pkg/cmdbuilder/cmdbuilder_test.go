package cmdbuilder

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestNewBaseCommand(t *testing.T) {
	cmd := NewBaseCommand("test", "foo", "bar")
	assert.Equal(t, "test", cmd.Name())
	assert.Equal(t, "test foo bar", cmd.String())
	assert.Equal(t, []string{"foo", "bar"}, cmd.Args())
	assert.Equal(t, []string{"foo", "bar", "1", "2", "3"}, cmd.Args("1", "2", "3"))
}

func TestMustVersionConstraint(t *testing.T) {
	t.Run("valid constraint", func(t *testing.T) {
		constraintStr := ">= 1.0.0"
		constraints := MustVersionConstraint(constraintStr)
		assert.NotNil(t, constraints)
		v, _ := version.NewVersion("1.0.0")
		assert.True(t, (*constraints).Check(v))
	})

	t.Run("invalid constraint panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustVersionConstraint("not a valid constraint string ((((")
		})
	})
}

func TestCommand_WithConditionalFlags(t *testing.T) {
	v1, _ := version.NewVersion("1.0.0")
	constraint1 := func(c *Command, subcommand string) error {
		c.Args = append(c.Args, "--flag1")
		return nil
	}
	constraint2 := func(c *Command, subcommand string) error {
		if c.Base.Equal("test-cmd") {
			c.Args = append(c.Args, "--flag2")
		}
		return nil
	}
	constraintErr := func(c *Command, subcommand string) error {
		return fmt.Errorf("constraint error")
	}

	t.Run("apply multiple constraints", func(t *testing.T) {
		localCmd := &Command{Base: NewBaseCommand("test-cmd"), Args: []string{"sub"}, Version: v1}
		err := WithConditionalFlags(localCmd, "sub", constraint1, constraint2)
		assert.NoError(t, err)
		assert.Equal(t, []string{"sub", "--flag1", "--flag2"}, localCmd.Args)
	})

	t.Run("apply constraint with error", func(t *testing.T) {
		localCmd := &Command{Base: NewBaseCommand("test-cmd"), Args: []string{"sub"}, Version: v1}
		err := WithConditionalFlags(localCmd, "sub", constraint1, constraintErr, constraint2)
		assert.Error(t, err)
		assert.Equal(t, "constraint error", err.Error())
		assert.Equal(t, []string{"sub", "--flag1"}, localCmd.Args) // Args up to the error
	})
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

func TestCommandFlagOptions(t *testing.T) {
	cmd := &Command{
		Base:    NewBaseCommand("docker", "compose"),
		Args:    []string{"up", "--really-new"},
		Version: func() *version.Version { v, _ := version.NewVersion("1.0.0"); return v }(),
	}

	err := WithConditionalFlags(
		cmd,
		"up",
		AppendFlag("docker compose", "up", "--example1", MustVersionConstraint(">=1.1")),
		AppendFlag("docker compose", "up", "--example2", MustVersionConstraint("<1.1")),
		PrependFlag("docker compose", "up", "--dry-run", MustVersionConstraint(">=1.0, <2.0")),
		RemoveFlag("docker compose", "up", "--really-new", nil),
	)
	assert.NoError(t, err)
	args := cmd.Base.Args(cmd.Args...)
	assert.Equal(t, []string{"compose", "--dry-run", "up", "--example2"}, args)
}
