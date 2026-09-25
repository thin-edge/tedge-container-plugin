package cli

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetComposeDownOptions(t *testing.T) {
	t.Cleanup(viper.Reset)

	cases := []struct {
		name    string
		value   any
		timeout time.Duration
	}{
		{name: "default", value: nil, timeout: 0},
		// A number must be seconds (not nanoseconds) so that it matches the x-tedge compose setting
		{name: "integer", value: 30, timeout: 30 * time.Second},
		{name: "numeric string", value: "30", timeout: 30 * time.Second},
		{name: "float", value: 1.5, timeout: 1500 * time.Millisecond},
		{name: "duration", value: "1m30s", timeout: 90 * time.Second},
		{name: "invalid", value: "abc", timeout: 0},
		{name: "negative", value: "-10s", timeout: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			viper.SetDefault("container_group.remove_volumes", true)
			if tc.value != nil {
				viper.Set("container_group.remove_timeout", tc.value)
			}
			opts := (&Cli{}).GetComposeDownOptions()
			assert.True(t, opts.RemoveVolumes)
			assert.Equal(t, tc.timeout, opts.RemoveTimeout)
		})
	}
}
