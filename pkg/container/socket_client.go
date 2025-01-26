package container

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/image"
)

type ResponsePruneImage struct {
	Id   string `json:"Id,omitempty"`
	Size uint64 `json:"Size,omitempty"`
}

type SocketClient struct {
	BaseURL string
	Client  *http.Client
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
