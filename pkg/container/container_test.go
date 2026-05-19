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

func Test_NormalizeImageRef(t *testing.T) {
	testcases := []struct {
		Input  string
		Expect string
	}{
		// Already fully-qualified — unchanged
		{Input: "docker.io/library/httpd:2.4", Expect: "docker.io/library/httpd:2.4"},
		{Input: "docker.io/library/app3:latest", Expect: "docker.io/library/app3:latest"},
		{Input: "docker.io/myuser/app:1.0", Expect: "docker.io/myuser/app:1.0"},
		{Input: "ghcr.io/owner/image:tag", Expect: "ghcr.io/owner/image:tag"},
		// docker.io shorthand expanded to fully-qualified library ref
		{Input: "docker.io/httpd:2.4", Expect: "docker.io/library/httpd:2.4"},
		{Input: "docker.io/app3:latest", Expect: "docker.io/library/app3:latest"},
		// Bare image names reported by older Docker versions are NOT expanded
		// by NormalizeImageRef; the LabelModuleVersion label is used instead.
		{Input: "app3:latest", Expect: "app3:latest"},
		{Input: "httpd:2.4", Expect: "httpd:2.4"},
	}

	for _, tc := range testcases {
		got := NormalizeImageRef(tc.Input)
		if got != tc.Expect {
			t.Errorf("NormalizeImageRef(%q) = %q, want %q", tc.Input, got, tc.Expect)
		}
	}
}

func Test_ImageRefsEqual(t *testing.T) {
	equal := []struct{ a, b string }{
		// Identical
		{"docker.io/library/app3:latest", "docker.io/library/app3:latest"},
		// docker.io shorthand vs fully-qualified
		{"docker.io/app3:latest", "docker.io/library/app3:latest"},
		// Bare name (Docker v20) vs fully-qualified (what the SM layer sends)
		{"app3:latest", "docker.io/library/app3:latest"},
		{"httpd:2.4", "docker.io/library/httpd:2.4"},
		// User/image form
		{"myuser/app:1.0", "docker.io/myuser/app:1.0"},
	}
	for _, tc := range equal {
		if !ImageRefsEqual(tc.a, tc.b) {
			t.Errorf("ImageRefsEqual(%q, %q) should be true", tc.a, tc.b)
		}
	}

	notEqual := []struct{ a, b string }{
		{"ghcr.io/owner/image:tag", "docker.io/library/image:tag"},
		{"app3:latest", "app3:1.0"},
		{"docker.io/library/httpd:2.4", "docker.io/library/nginx:2.4"},
	}
	for _, tc := range notEqual {
		if ImageRefsEqual(tc.a, tc.b) {
			t.Errorf("ImageRefsEqual(%q, %q) should be false", tc.a, tc.b)
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

func Test_PruneImages(t *testing.T) {
	client, err := NewContainerClient(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.ImagesPruneUnused(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}
