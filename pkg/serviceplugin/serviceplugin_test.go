package serviceplugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParsePluginPaths(t *testing.T) {
	testcases := []struct {
		Value  string
		Expect []string
	}{
		{Value: `["/usr/share/tedge/service-plugins"]` + "\n", Expect: []string{"/usr/share/tedge/service-plugins"}},
		{Value: `["/usr/local/share/tedge/service-plugins", "/usr/share/tedge/service-plugins"]`, Expect: []string{"/usr/local/share/tedge/service-plugins", "/usr/share/tedge/service-plugins"}},
		{Value: `/a, /b`, Expect: []string{"/a", "/b"}},
		{Value: `[]`, Expect: []string{}},
		{Value: ``, Expect: []string{}},
	}
	for _, testcase := range testcases {
		assert.Equal(t, testcase.Expect, ParsePluginPaths(testcase.Value), "value=%q", testcase.Value)
	}
}

func Test_FilterActions(t *testing.T) {
	valid, unknown := FilterActions([]string{"start", "stop", "reboot", "start", " restart ", "pause"})
	assert.Equal(t, []string{"start", "stop", "restart"}, valid)
	assert.Equal(t, []string{"reboot", "pause"}, unknown)
}
