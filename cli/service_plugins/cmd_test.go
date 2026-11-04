package service_plugins

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thin-edge/tedge-container-plugin/pkg/cli"
	"github.com/thin-edge/tedge-container-plugin/pkg/serviceplugin"
)

func execute(args ...string) error {
	cmd := NewCommand(cli.Cli{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func Test_UnsupportedActionExitsWithCode2(t *testing.T) {
	for _, serviceType := range []string{"container", "container-group"} {
		for _, action := range []string{"enable", "disable", "--help", "-h", "RESTART"} {
			err := execute(serviceType, action, "app1")

			var exitErr cli.ExitCodeError
			if assert.True(t, errors.As(err, &exitErr), "type=%s action=%s", serviceType, action) {
				assert.Equal(t, serviceplugin.ExitCodeUnsupported, exitErr.ExitCode())
				assert.ErrorIs(t, err, serviceplugin.ErrUnsupportedAction)
			}
		}
	}
}

func Test_WrongNumberOfArguments(t *testing.T) {
	for _, args := range [][]string{
		{"container"},
		{"container", "restart"},
		{"container", "restart", "app1", "extra"},
	} {
		err := execute(args...)
		assert.ErrorContains(t, err, "expected 2 arguments", "args=%v", args)

		var exitErr cli.ExitCodeError
		assert.False(t, errors.As(err, &exitErr), "args=%v", args)
	}
}

func Test_SupportedActions(t *testing.T) {
	assert.Equal(t, []string{"pause", "restart", "start", "stop", "unpause"}, serviceplugin.Actions())
	assert.False(t, serviceplugin.IsSupported("enable"))
}
