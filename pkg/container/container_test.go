package container

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ResolveDockerIOImage(t *testing.T) {
	testcases := []struct {
		Image  string
		Expect string
	}{
		{
			Image:  "docker.io/library/app",
			Expect: "docker.io/library/app",
		},
		{
			Image:  "docker.io/other/app2",
			Expect: "docker.io/other/app2",
		},
		{
			Image:  "foo.io/other/app2",
			Expect: "foo.io/other/app2",
		},
	}

	for _, testcase := range testcases {
		got, _ := ResolveDockerIOImage(testcase.Image)

		if got != testcase.Expect {
			t.Errorf("Resolved image does not match expected. got=%s, wanted=%s", got, testcase.Expect)
		}
	}

}

func Test_FilterLabels(t *testing.T) {
	in := map[string]string{
		"foo":                              "bar",
		"org.opencontainers.image.authors": "thin-edge.io",
		"org.opencontainers.image.version": "1.2.3",
	}
	out := FilterLabels(in, []string{"org.opencontainers."})

	assert.Len(t, out, 1)
	assert.Equal(t, out["foo"], "bar")
}

func Test_FilterEnvVariables(t *testing.T) {
	in := []string{
		"FOO=bar",
		"BAR=2",
		"HOSTNAME=bar",
	}
	out := FilterEnvVariables(in, []string{"HOSTNAME"})

	assert.Len(t, out, 2)
	assert.Equal(t, out[0], "FOO=bar")
	assert.Equal(t, out[1], "BAR=2")
}

func Test_PruneIMages(t *testing.T) {
	client, err := NewContainerClient()
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.ImagesPruneUnused(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}
