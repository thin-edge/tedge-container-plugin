package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/google/go-querystring/query"
)

type ResponsePruneImage struct {
	Id   string `json:"Id,omitempty"`
	Size uint64 `json:"Size,omitempty"`
}

type SocketClient struct {
	BaseURL string
	Client  *http.Client
}

func NewDefaultLibPodHTTPClient() *SocketClient {
	return NewLibPodHTTPClient(findContainerEngineSocket())
}

func NewLibPodHTTPClient(sock string) *SocketClient {
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", strings.TrimPrefix(sock, "unix://"))
			},
		},
	}

	return &SocketClient{
		Client:  &httpc,
		BaseURL: "http://d/v5.0.0/libpod",
	}
}

func (c *SocketClient) resolveURL(path string) string {
	return strings.Join([]string{c.BaseURL, strings.TrimPrefix(path, "/")}, "/")
}

var ErrPodmanAPIError = errors.New("podman api not available")

func (c *SocketClient) Test(ctx context.Context) error {
	r, err := c.Client.Get(c.resolveURL("info"))
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return ErrPodmanAPIError
	}
	return nil
}

// GetEventsBackend queries GET /libpod/info and returns the configured events
// backend (e.g. "journald", "file", "none").  An empty string is returned if
// the field cannot be read; callers should treat that as unknown and proceed.
func (c *SocketClient) GetEventsBackend(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolveURL("info"), nil)
	if err != nil {
		return ""
	}
	r, err := c.Client.Do(req)
	if err != nil || r.StatusCode != http.StatusOK {
		return ""
	}
	defer func() { _ = r.Body.Close() }()
	var info LibPodInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		return ""
	}
	return info.Host.EventLogger
}

// Prune all images and return object in same format as the docker prune response
func (c *SocketClient) PruneImages(body io.Reader) (report image.PruneReport, err error) {
	r, err := c.Client.Post(c.resolveURL("images/prune?all=true"), "application/json", body)
	if err != nil {
		return
	}

	defer func() { _ = r.Body.Close() }()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	prunedImages := make([]ResponsePruneImage, 0)
	err = json.Unmarshal(b, &prunedImages)
	if err != nil {
		return
	}

	var spaceReclaimed uint64
	for _, item := range prunedImages {
		report.ImagesDeleted = append(report.ImagesDeleted, image.DeleteResponse{
			Deleted: item.Id,
		})
		spaceReclaimed += item.Size
	}
	report.SpaceReclaimed = spaceReclaimed
	return
}

type PodmanAPIPullOptions struct {
	AllTags   *bool  `url:"allTags,omitempty"`
	Quiet     *bool  `url:"quiet,omitempty"`
	Policy    string `url:"policy,omitempty"`
	Reference string `url:"reference"`
}

func (po *PodmanAPIPullOptions) WithPolicy(v string) *PodmanAPIPullOptions {
	po.Policy = v
	return po
}

func (po *PodmanAPIPullOptions) WithAllTags(v bool) *PodmanAPIPullOptions {
	po.AllTags = &v
	return po
}

func (po *PodmanAPIPullOptions) WithQuiet(v bool) *PodmanAPIPullOptions {
	po.Quiet = &v
	return po
}

type PodmanPullOptions struct {
	image.PullOptions

	Quiet bool
}

// ContainerInspect fetches the libpod-native inspect data for the named
// container.  The returned struct contains fields that the Docker-compat API
// normalises away (e.g. UsernsMode "keep-id" becomes "private" via compat).
func (c *SocketClient) ContainerInspect(ctx context.Context, nameOrID string) (*LibPodInspectResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolveURL(fmt.Sprintf("containers/%s/json", nameOrID)), nil)
	if err != nil {
		return nil, err
	}
	r, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Body.Close() }()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("libpod inspect: unexpected status %s", r.Status)
	}
	var resp LibPodInspectResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("libpod inspect: decode: %w", err)
	}
	return &resp, nil
}

func (c *SocketClient) PullImages(ctx context.Context, imageRef string, alwaysPull bool, pullOptions PodmanPullOptions) error {
	options := PodmanAPIPullOptions{
		Reference: imageRef,
	}
	options.WithQuiet(pullOptions.Quiet)
	if alwaysPull {
		options.WithPolicy("always")
	}

	queryParams, err := query.Values(options)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.resolveURL(fmt.Sprintf("images/pull?%s", queryParams.Encode())), nil)
	if err != nil {
		return err
	}

	if pullOptions.RegistryAuth != "" {
		req.Header.Set("X-Registry-Auth", pullOptions.RegistryAuth)
	}

	r, err := c.Client.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = r.Body.Close() }()
	if _, ioErr := io.Copy(os.Stderr, r.Body); ioErr != nil {
		slog.Warn("Could not write to stderr.", "err", ioErr)
	}

	slog.Info("Podman API response was successful.", "status", r.Status)
	statusOK := r.StatusCode >= 200 && r.StatusCode < 400
	if !statusOK {
		return fmt.Errorf("podman api failed. code=%s", r.Status)
	}
	return nil
}

// Events subscribes to the libpod native events stream (GET /libpod/events)
// and returns channels with the same signature as the Docker-compat client
// Events API so that MonitorEvents can use it transparently.
//
// The libpod events endpoint sends newline-delimited JSON objects whose field
// names match the Go docker/docker events.Message type directly (Type, Action,
// Actor.ID, Actor.Attributes, time, timeNano), so we decode straight into
// events.Message without a separate intermediate type.
func (c *SocketClient) Events(ctx context.Context) (<-chan events.Message, <-chan error) {
	msgCh := make(chan events.Message)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)

		sendErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
			close(errCh)
		}

		url := c.resolveURL("events?stream=true")
		slog.Info("Connecting to libpod events stream.", "url", url)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			sendErr(err)
			return
		}

		resp, err := c.Client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				close(errCh)
				return
			}
			slog.Warn("libpod events: request failed.", "err", err)
			sendErr(err)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		slog.Info("libpod events stream connected.", "status", resp.StatusCode)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			sendErr(fmt.Errorf("libpod events: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body))))
			return
		}

		dec := json.NewDecoder(resp.Body)
		for {
			var evt events.Message
			if err := dec.Decode(&evt); err != nil {
				if ctx.Err() != nil {
					close(errCh)
					return
				}
				if err == io.EOF {
					slog.Warn("libpod events stream closed unexpectedly (EOF).")
					sendErr(fmt.Errorf("libpod events stream closed: %w", io.EOF))
				} else {
					slog.Warn("libpod events decode error.", "err", err)
					sendErr(err)
				}
				return
			}
			// Normalize libpod-native action names to their Docker-compat
			// equivalents so that the rest of the codebase can use the
			// standard events.ActionXxx constants regardless of engine.
			// Podman's native API emits "died" and "exec_died"; the Docker
			// compat API (and our switch statements) expect "die" / "exec_die".
			// Reference: https://docs.podman.io/en/stable/markdown/podman-events.1.html
			switch evt.Action {
			case "died":
				evt.Action = events.ActionDie
			case "exec_died":
				evt.Action = events.ActionExecDie
			}
			slog.Debug("libpod event received.", "type", evt.Type, "action", evt.Action, "id", evt.Actor.ID)
			select {
			case <-ctx.Done():
				close(errCh)
				return
			case msgCh <- evt:
			}
		}
	}()

	return msgCh, errCh
}
