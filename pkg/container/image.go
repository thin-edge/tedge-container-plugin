package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/docker/docker/client"
)

type imageLoadResponse struct {
	Stream string `json:"stream"`
}

// LoadImageFromFile loads a container image archive, e.g. created via 'docker save', and
// returns the reference of the requested image as reported by the container engine.
//
// It is an error if the archive does not contain the requested image. Tagging another
// image with the requested reference is intentionally not done, as the reference would
// then shadow the registry for any subsequent install.
func (c *ContainerClient) LoadImageFromFile(ctx context.Context, path string, imageRef string) (string, error) {
	slog.Info("Loading image from file.", "file", path)
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	resp, err := c.Client.ImageLoad(ctx, file, client.ImageLoadWithQuiet(true))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if !resp.JSON {
		return imageRef, nil
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	details := &imageLoadResponse{}
	if err := json.Unmarshal(b, details); err != nil {
		return "", err
	}
	slog.Info("Loaded image.", "stream", details.Stream)

	images := make([]string, 0)
	for _, line := range strings.Split(details.Stream, "\n") {
		if loadedRef, ok := strings.CutPrefix(line, "Loaded image: "); ok {
			slog.Info("Found image reference in file.", "file", path, "image", loadedRef)
			images = append(images, loadedRef)
		}
	}

	for _, loadedRef := range images {
		if ImageRefsEqual(loadedRef, imageRef) {
			// Use the reference reported by the engine as it is guaranteed to
			// resolve regardless of how the engine normalises tag strings
			return loadedRef, nil
		}
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no image detected in file. image=%s, file=%s", imageRef, path)
	}
	return "", fmt.Errorf("file does not contain the requested image. image=%s, images=%s, file=%s", imageRef, strings.Join(images, ","), path)
}
