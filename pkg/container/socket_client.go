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

// Prune all images and return object in same format as the docker prune response
func (c *SocketClient) PruneImages(body io.Reader) (report image.PruneReport, err error) {
	r, err := c.Client.Post(c.resolveURL("images/prune?all=true"), "application/json", body)
	if err != nil {
		return
	}

	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
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

	defer r.Body.Close()
	if _, ioErr := io.Copy(os.Stderr, r.Body); ioErr != nil {
		slog.Warn("Could not write to stderr.", "err", ioErr)
	}

	slog.Info("Podman API response was successful.", "status", r.Status)
	return nil
}
