package container

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// detectEngineCapabilities
// ---------------------------------------------------------------------------

func Test_detectEngineCapabilities_podman(t *testing.T) {
	caps := detectEngineCapabilities("Podman Engine")
	assert.Equal(t, EnginePodman, caps.Type)
	assert.True(t, caps.HasLibPodAPI)
}

func Test_detectEngineCapabilities_podmanLowerCase(t *testing.T) {
	caps := detectEngineCapabilities("podman")
	assert.Equal(t, EnginePodman, caps.Type)
	assert.True(t, caps.HasLibPodAPI)
}

func Test_detectEngineCapabilities_docker(t *testing.T) {
	caps := detectEngineCapabilities("Docker Engine - Community")
	assert.Equal(t, EngineDocker, caps.Type)
	assert.False(t, caps.HasLibPodAPI)
}

func Test_detectEngineCapabilities_unknown(t *testing.T) {
	caps := detectEngineCapabilities("some-other-runtime")
	assert.Equal(t, EngineUnknown, caps.Type)
	assert.False(t, caps.HasLibPodAPI)
}

func Test_detectEngineCapabilities_empty(t *testing.T) {
	caps := detectEngineCapabilities("")
	assert.Equal(t, EngineUnknown, caps.Type)
	assert.False(t, caps.HasLibPodAPI)
}

// ---------------------------------------------------------------------------
// SocketClient.ContainerInspect
// ---------------------------------------------------------------------------

// newTestLibPodServer starts a minimal httptest.Server that responds to
// /v5.0.0/libpod/containers/{name}/json with the provided LibPodInspectResponse.
// It returns both the server and a *SocketClient wired to it.
func newTestLibPodServer(t *testing.T, resp LibPodInspectResponse) (*httptest.Server, *SocketClient) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/containers/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	sc := &SocketClient{
		BaseURL: srv.URL + "/v5.0.0/libpod",
		Client:  srv.Client(),
	}
	return srv, sc
}

func Test_SocketClient_ContainerInspect_keepID(t *testing.T) {
	want := LibPodInspectResponse{
		HostConfig: LibPodHostConfig{UsernsMode: "keep-id"},
	}
	srv, sc := newTestLibPodServer(t, want)
	defer srv.Close()

	got, err := sc.ContainerInspect(context.Background(), "mycontainer")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "keep-id", got.HostConfig.UsernsMode)
}

func Test_SocketClient_ContainerInspect_private(t *testing.T) {
	want := LibPodInspectResponse{
		HostConfig: LibPodHostConfig{UsernsMode: "private"},
	}
	srv, sc := newTestLibPodServer(t, want)
	defer srv.Close()

	got, err := sc.ContainerInspect(context.Background(), "mycontainer")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "private", got.HostConfig.UsernsMode)
}

func Test_SocketClient_ContainerInspect_error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	sc := &SocketClient{
		BaseURL: srv.URL + "/v5.0.0/libpod",
		Client:  srv.Client(),
	}

	_, err := sc.ContainerInspect(context.Background(), "noexist")
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "404")
}

// ---------------------------------------------------------------------------
// enrichHostConfigForPodman
// ---------------------------------------------------------------------------

func Test_enrichHostConfigForPodman_correctsKeepID(t *testing.T) {
	libpodResp := LibPodInspectResponse{
		HostConfig: LibPodHostConfig{UsernsMode: "keep-id"},
	}
	srv, sc := newTestLibPodServer(t, libpodResp)
	defer srv.Close()

	c := &ContainerClient{
		Engine: EngineCapabilities{Type: EnginePodman, HasLibPodAPI: true},
		LibPod: sc,
	}

	hc := &dockercontainer.HostConfig{
		UsernsMode: dockercontainer.UsernsMode("private"), // normalised value from compat API
	}

	c.enrichHostConfigForPodman(context.Background(), "mycontainer", hc)

	assert.Equal(t, dockercontainer.UsernsMode("keep-id"), hc.UsernsMode)
}

func Test_enrichHostConfigForPodman_noChangeWhenAlreadyCorrect(t *testing.T) {
	libpodResp := LibPodInspectResponse{
		HostConfig: LibPodHostConfig{UsernsMode: ""},
	}
	srv, sc := newTestLibPodServer(t, libpodResp)
	defer srv.Close()

	c := &ContainerClient{
		Engine: EngineCapabilities{Type: EnginePodman, HasLibPodAPI: true},
		LibPod: sc,
	}

	hc := &dockercontainer.HostConfig{
		UsernsMode: dockercontainer.UsernsMode(""),
	}

	c.enrichHostConfigForPodman(context.Background(), "mycontainer", hc)

	assert.Equal(t, dockercontainer.UsernsMode(""), hc.UsernsMode)
}

func Test_enrichHostConfigForEngine_skipsDockerEngine(t *testing.T) {
	// Docker uses no libpod enrichment — LibPod pointer stays nil and the
	// method must be a no-op.
	c := &ContainerClient{
		Engine: EngineCapabilities{Type: EngineDocker, HasLibPodAPI: false},
		LibPod: nil,
	}

	hc := &dockercontainer.HostConfig{
		UsernsMode: dockercontainer.UsernsMode("host"),
	}

	// Must not panic.
	c.enrichHostConfigForEngine(context.Background(), "anycontainer", hc)

	assert.Equal(t, dockercontainer.UsernsMode("host"), hc.UsernsMode)
}
