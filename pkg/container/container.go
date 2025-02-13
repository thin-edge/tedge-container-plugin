package container

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/thin-edge/tedge-container-plugin/pkg/utils"
)

var ErrNoImage = errors.New("no container image found")

var ContainerType string = "container"
var ContainerGroupType string = "container-group"

func NewJSONTime(t time.Time) JSONTime {
	return JSONTime{
		Time: t,
	}
}

type JSONTime struct {
	time.Time
	AsRFC3339 bool
}

func (t JSONTime) MarshalJSON() ([]byte, error) {
	if t.AsRFC3339 {
		v := fmt.Sprintf("\"%s\"", time.Time(t.Time).Format(time.RFC3339))
		return []byte(v), nil
	}
	v := fmt.Sprintf("%d", t.Time.Unix())
	return []byte(v), nil
}

func (t *JSONTime) UnmarshalJSON(data []byte) error {
	var tmpValue any
	if err := json.Unmarshal(data, tmpValue); err != nil {
		return err
	}

	switch value := tmpValue.(type) {
	case int32:
		t.Time = time.Unix(int64(value), 0)
	case int64:
		t.Time = time.Unix(value, 0)
	case float64:
		sec, dec := math.Modf(value)
		t.Time = time.Unix(int64(sec), int64(dec*(1e9)))
	case string:
		v, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return err
		}
		t.Time = v
	default:
		return fmt.Errorf("invalid format. only Unix timestamp or RFC3339 formats are supported")
	}

	return nil
}

type TedgeContainer struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	ServiceType string    `json:"serviceType"`
	Container   Container `json:"container"`
	Time        JSONTime  `json:"time"`
}

type Container struct {
	Name        string   `json:"-"`
	Id          string   `json:"containerId,omitempty"`
	State       string   `json:"state,omitempty"`
	Status      string   `json:"containerStatus,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	Image       string   `json:"image,omitempty"`
	Ports       string   `json:"ports,omitempty"`
	NetworkIDs  []string `json:"-"`
	Networks    string   `json:"networks,omitempty"`
	RunningFor  string   `json:"runningFor,omitempty"`
	Filesystem  string   `json:"filesystem,omitempty"`
	Command     string   `json:"command,omitempty"`
	NetworkMode string   `json:"networkMode,omitempty"`

	// Only used for container groups
	ServiceName string `json:"serviceName,omitempty"`
	ProjectName string `json:"projectName,omitempty"`

	// Private values
	Labels map[string]string `json:"-"`
}

func NewContainerFromDockerContainer(item *types.Container) TedgeContainer {
	container := Container{
		Id:          item.ID,
		Name:        ConvertName(item.Names),
		State:       item.State,
		Status:      item.Status,
		Image:       item.Image,
		Command:     item.Command,
		CreatedAt:   time.Unix(item.Created, 0).Format(time.RFC3339),
		Ports:       FormatPorts(item.Ports),
		NetworkMode: item.HostConfig.NetworkMode,
		Labels:      item.Labels,
	}

	// Mimic filesystem
	srw := units.HumanSizeWithPrecision(float64(item.SizeRw), 3)
	sv := units.HumanSizeWithPrecision(float64(item.SizeRootFs), 3)
	container.Filesystem = srw
	if item.SizeRootFs > 0 {
		container.Filesystem = fmt.Sprintf("%s (virtual %s)", srw, sv)
	}

	if v, ok := item.Labels["com.docker.compose.project"]; ok {
		container.ProjectName = v
	}

	if v, ok := item.Labels["com.docker.compose.service"]; ok {
		container.ServiceName = v
	}

	container.NetworkIDs = make([]string, 0)
	if item.NetworkSettings != nil && len(item.NetworkSettings.Networks) > 0 {
		for _, v := range item.NetworkSettings.Networks {
			container.NetworkIDs = append(container.NetworkIDs, v.NetworkID)
		}
	}

	containerType := ContainerType
	// Set service type. A docker compose project is a "container-group"
	if _, ok := item.Labels["com.docker.compose.project"]; ok {
		containerType = ContainerGroupType
	}

	return TedgeContainer{
		Name: container.GetName(),
		Time: JSONTime{
			Time: time.Now(),
		},
		Status:      ConvertToTedgeStatus(item.State),
		ServiceType: containerType,
		Container:   container,
	}
}

func (c *Container) GetName() string {
	if c.ProjectName == "" {
		return c.Name
	}
	return fmt.Sprintf("%s@%s", c.ProjectName, c.ServiceName)
}

func ConvertToTedgeStatus(v string) string {
	switch v {
	case "up", "running":
		return "up"
	default:
		return "down"
	}
}

func FormatPorts(values []types.Port) string {
	formatted := make([]string, 0, len(values))
	for _, port := range values {
		if port.PublicPort == 0 {
			formatted = append(formatted, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
		} else {
			if port.IP == "" {
				formatted = append(formatted, fmt.Sprintf("%d:%d/%s", port.PublicPort, port.PrivatePort, port.Type))
			} else {
				formatted = append(formatted, fmt.Sprintf("%s:%d:%d/%s", port.IP, port.PublicPort, port.PrivatePort, port.Type))
			}
		}
	}
	return strings.Join(formatted, ", ")
}

func ConvertName(v []string) string {
	return strings.TrimPrefix(v[0], "/")
}

type ContainerClient struct {
	Client *client.Client
}

func socketExists(p string) bool {
	_, err := os.Stat(strings.TrimPrefix(p, "unix://"))
	return err == nil
}

func findContainerEngineSocket() (socketAddr string) {
	// Check env variables to normalize differences
	// between docker and podman
	containerSockets := make([]string, 0)

	envVariables := []string{
		// docker
		"DOCKER_HOST",
		// podman
		"CONTAINER_HOST",
	}
	for _, name := range envVariables {
		if v := os.Getenv(name); v != "" {
			containerSockets = append(containerSockets, v)
		}
	}

	// docker
	containerSockets = append(
		containerSockets,
		"unix:///var/run/docker.sock",
	)
	// podman
	containerSockets = append(
		containerSockets,
		"unix:///run/podman/podman.sock",
		"unix:///run/user/0/podman/podman.sock",
	)

	for _, addr := range containerSockets {
		if strings.HasPrefix(addr, "unix://") {
			// Check if socket exists
			if socketExists(addr) {
				socketAddr = addr
				break
			}
		} else {
			// Assume the user has configured a valid non-socket based endpoint
			socketAddr = addr
			break
		}
	}
	return socketAddr
}

func NewContainerClient() (*ContainerClient, error) {
	// Find container socket
	addr := findContainerEngineSocket()
	if addr != "" {
		if err := os.Setenv("DOCKER_HOST", addr); err != nil {
			return nil, err
		}
		// Used by podman and podman-remote
		if err := os.Setenv("CONTAINER_HOST", addr); err != nil {
			return nil, err
		}
		slog.Info("Using container engine socket.", "value", addr)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &ContainerClient{
		Client: cli,
	}, nil
}

type ContainerTelemetryMessage struct {
	Container ContainerStats `json:"container"`
}

// Custom float representation which controls how many decimal places
// are used when marshalling the value to JSON
type LowPrecisionFloat struct {
	// Value
	Value float64

	// Number of digital to display
	Digits int
}

func (l LowPrecisionFloat) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("%.*f", l.Digits, l.Value)
	return []byte(s), nil
}

func NewLowerPrecisionFloat64(value float64, precision int) LowPrecisionFloat {
	return LowPrecisionFloat{
		Value:  value,
		Digits: precision,
	}
}

type ContainerStats struct {
	Cpu    LowPrecisionFloat `json:"cpu"`
	Memory LowPrecisionFloat `json:"memory"`
	NetIO  LowPrecisionFloat `json:"netio"`
}

func (c *ContainerClient) GetStats(ctx context.Context, containerID string) (*ContainerTelemetryMessage, error) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	containerStats := &Stats{
		StatsEntry: StatsEntry{
			Container: containerID,
		},
	}

	// Start collecting statistics
	collect(ctx, containerStats, c.Client, false, &wg)
	wg.Wait()

	s := containerStats.GetStatistics()
	stats := &ContainerTelemetryMessage{
		Container: ContainerStats{
			Cpu:    NewLowerPrecisionFloat64(s.CPUPercentage, 2),
			Memory: NewLowerPrecisionFloat64(s.MemoryPercentage, 2),
			NetIO:  NewLowerPrecisionFloat64(s.NetworkTx, 0),
		},
	}
	return stats, nil
}

type FilterOptions struct {
	Names  []string
	Labels []string
	IDs    []string

	// Client side filters
	Types            []string
	ExcludeNames     []string
	ExcludeWithLabel []string
}

func (fo FilterOptions) IsEmpty() bool {
	return len(fo.Names) == 0 && len(fo.Labels) == 0 && len(fo.IDs) == 0
}

func (c *ContainerClient) GetContainer(ctx context.Context, containerID string) (*TedgeContainer, error) {
	containers, err := c.List(ctx, FilterOptions{
		IDs: []string{containerID},
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, fmt.Errorf("container not found")
	}
	return &containers[0], nil
}

// Stop and remove a container
// Don't fail if the container does not exist
func (c *ContainerClient) StopRemoveContainer(ctx context.Context, containerID string) error {
	slog.Info("Stopping container.", "id", containerID)
	err := c.Client.ContainerStop(ctx, containerID, container.StopOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			slog.Info("Container does not exist, so nothing to stop")
			return nil
		}
		return err
	}
	slog.Info("Removing container.", "id", containerID)
	err = c.Client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
	})
	if err != nil {
		if errdefs.IsNotFound(err) {
			slog.Info("Container does not exist, so nothing to stop")
			return nil
		}
	}
	return err
}

// RestartContainer a container
func (c *ContainerClient) RestartContainer(ctx context.Context, containerID string) error {
	slog.Info("Restarting container.", "id", containerID)
	return c.Client.ContainerRestart(ctx, containerID, container.StopOptions{})
}

func (c *ContainerClient) List(ctx context.Context, options FilterOptions) ([]TedgeContainer, error) {
	// Filter for docker compose projects
	listOptions := container.ListOptions{
		Size: true,
		All:  true,
	}

	filterValues := make([]filters.KeyValuePair, 0)

	// Match by container name
	for _, name := range options.Names {
		filterValues = append(filterValues, filters.KeyValuePair{
			Key:   "name",
			Value: name,
		})
	}

	// Match by container id
	for _, value := range options.IDs {
		filterValues = append(filterValues, filters.KeyValuePair{
			Key:   "id",
			Value: value,
		})
	}

	// Match by label
	for _, label := range options.Labels {
		filterValues = append(filterValues, filters.KeyValuePair{
			Key:   "label",
			Value: label,
		})
	}

	if len(filterValues) > 0 {
		listOptions.Filters = filters.NewArgs(filterValues...)
	}

	containers, err := c.Client.ContainerList(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	// Pre-compile regular expressions
	excludeNamesRegex := make([]regexp.Regexp, 0, len(options.ExcludeNames))
	for _, pattern := range options.ExcludeNames {
		if p, err := regexp.Compile(pattern); err != nil {
			slog.Warn("Invalid excludeNames regex pattern.", "pattern", pattern, "err", err)
		} else {
			excludeNamesRegex = append(excludeNamesRegex, *p)
		}
	}

	items := make([]TedgeContainer, 0, len(containers))
	for _, i := range containers {
		item := NewContainerFromDockerContainer(&i)

		// Apply client side filters
		if len(options.Types) > 0 {
			if !slices.Contains(options.Types, item.ServiceType) {
				continue
			}
		}

		if len(excludeNamesRegex) > 0 {
			ignoreContainer := false
			for _, pattern := range excludeNamesRegex {
				if pattern.MatchString(item.Container.Name) || pattern.MatchString(item.Name) {
					ignoreContainer = true
					break
				}
			}
			if ignoreContainer {
				continue
			}
		}

		if len(options.ExcludeWithLabel) > 0 {
			hasLabel := false
			for _, label := range options.ExcludeWithLabel {
				if _, hasLabel = item.Container.Labels[label]; hasLabel {
					break
				}
			}
			if hasLabel {
				continue
			}
		}
		items = append(items, item)
	}

	return items, nil
}

func (c *ContainerClient) MonitorEvents(ctx context.Context) (<-chan events.Message, <-chan error) {
	return c.Client.Events(context.Background(), events.ListOptions{})
}

type ImagePullOptions struct {
	AuthFunc    func(context.Context, int) (string, error)
	MaxAttempts int
	Wait        time.Duration
}

// Check if the given docker.io image has fully qualified (e.g. docker.io/library/<image>)
// if not, then expand it to its fully qualified name.
func ResolveDockerIOImage(imageRef string) (string, bool) {
	if !strings.HasPrefix(imageRef, "docker.io/") {
		return imageRef, false
	}

	slashCount := strings.Count(imageRef, "/")

	if strings.HasPrefix(imageRef, "docker.io/library/") || slashCount >= 2 {
		// Is already normalized
		return imageRef, false
	}
	if !strings.HasPrefix(imageRef, "docker.io/") {
		// Not a docker.io image
		return imageRef, false
	}

	return "docker.io/library/" + strings.TrimPrefix(imageRef, "docker.io/"), true
}

func NormalizeImageRef(imageRef string) string {
	fullRef, _ := ResolveDockerIOImage(imageRef)
	return fullRef
}

// Pull a container image. The image will be verified if it exists afterwards
//
// Use credentials function to generate initial credentials
// and call again if the credentials fail which gives the credentials
// helper to invalid its own cache
func (c *ContainerClient) ImagePullWithRetries(ctx context.Context, imageRef string, alwaysPull bool, opts ImagePullOptions) (*types.ImageInspect, error) {
	// Check if image exists
	// Use ImageInspectWithRaw over ImageList as inspect is able to look up images either with or without
	// the repository details making it more compatible between docker and podman
	if imageInspect, _, err := c.Client.ImageInspectWithRaw(ctx, imageRef); err != nil {
		// Don't fail, just log it and continue
		slog.Info("Image does not already exist, trying to pull image.", "response", err)
	} else if !alwaysPull {
		slog.Info("Image already exists.", "ref", imageRef, "id", imageInspect.ID, "tags", imageInspect.RepoTags)
		return &imageInspect, nil
	}

	result, err := utils.Retry(opts.MaxAttempts, opts.Wait, func(attempt int) (any, error) {
		slog.Info("Pulling image.", "attempt", attempt)
		pullOptions := image.PullOptions{}

		// Get authentication header
		if opts.AuthFunc != nil {
			if auth, err := opts.AuthFunc(ctx, attempt); auth != "" && err == nil {
				pullOptions.RegistryAuth = auth
			}
		}

		// Note: ImagePull does not seem to return an error if the private registries authentication fails
		// so after pulling the image, check if it is loaded to confirm everything worked as expected
		useDockerPull := true

		// try podman api first and but fallback to docker pull API fails
		// Note: Podman 4.4 was observed to have an issue pulling images via the docker API where the only reported error is:
		// "write /dev/stderr: input/output error"
		podmanLib := NewDefaultLibPodHTTPClient()
		if podmanLib.Test(ctx) == nil {
			slog.Info("Trying to pull image using podman API")
			libpodErr := podmanLib.PullImages(ctx, imageRef, alwaysPull, PodmanPullOptions{
				PullOptions: pullOptions,
				Quiet:       false,
			})

			// Don't fail as it is unclear how stable the libpod API is
			// and a check for the image is done afterwards anyway
			if libpodErr != nil {
				slog.Warn("podman (libpod) pull images failed but error will be ignored.", "err", libpodErr)

			} else {
				useDockerPull = false
			}
		}

		if useDockerPull {
			slog.Info("Trying to pull image using docker API")
			out, err := c.Client.ImagePull(ctx, imageRef, pullOptions)
			if err != nil {
				return nil, err
			}
			defer out.Close()
			if _, ioErr := io.Copy(os.Stderr, out); ioErr != nil {
				slog.Warn("Could not write to stderr.", "err", ioErr)
			}
		}

		//
		// Check if image is not present
		imageInspect, _, imageErr := c.Client.ImageInspectWithRaw(ctx, imageRef)
		if imageErr != nil {
			slog.Error("No image found after pulling.", "err", imageErr)
			return nil, imageErr
		}
		slog.Info("Image found after pull.", "id", imageInspect.ID, "name", imageInspect.RepoTags)
		return &imageInspect, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*types.ImageInspect), err
}

//nolint:all
func (c *ContainerClient) runComposeInContainer(ctx context.Context, projectName string, workingDir string) (output []byte, err error) {
	imageRef := "docker.io/library/docker:27.3.1-cli"

	//
	// Check and pull image if it is not present
	images, err := c.Client.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageRef)),
	})
	if err != nil {
		return nil, err
	}

	if len(images) == 0 {
		slog.Info("Pulling image.", "ref", imageRef)
		out, err := c.Client.ImagePull(ctx, imageRef, image.PullOptions{})
		if err != nil {
			return nil, err
		}
		defer out.Close()
		if _, ioErr := io.Copy(os.Stderr, out); ioErr != nil {
			slog.Warn("Could not write to stderr.", "err", ioErr)
		}
	} else {
		slog.Info("Image already exists.", "ref", imageRef, "id", images[0].ID, "tags", images[0].RepoTags)
	}

	// docker run --privileged --name some-docker -v /my/own/var-lib-docker:/var/lib/docker -t docker.io/library/docker:27.3.1-cli
	slog.Info("Pulling image.", "ref", imageRef)
	out, err := c.Client.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return nil, err
	}
	defer out.Close()
	if _, ioErr := io.Copy(os.Stderr, out); ioErr != nil {
		slog.Warn("Could not write to stderr.", "err", ioErr)
	}

	containerCreate, err := c.Client.ContainerCreate(ctx, &container.Config{
		Image: imageRef,
	}, &container.HostConfig{
		AutoRemove: true,
		Binds: []string{
			fmt.Sprintf("%s:/var/run/docker.sock", c.Client.DaemonHost()),

			// Mirror host structure so that
			fmt.Sprintf("%s:%s", workingDir, workingDir),
		},
	}, &network.NetworkingConfig{}, nil, "")

	if err != nil {
		return nil, err
	}
	slog.Info("Created container.", "id", containerCreate.ID)

	err = c.Client.ContainerStart(ctx, containerCreate.ID, container.StartOptions{})
	if err != nil {
		return
	}

	conLogs, err := c.Client.ContainerLogs(ctx, containerCreate.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, err
	}
	defer conLogs.Close()

	if _, ioErr := io.Copy(os.Stderr, conLogs); ioErr != nil {
		slog.Warn("Could not write to stderr.", "err", ioErr)
	}
	return nil, nil

}

// Create shared network
func (c *ContainerClient) CreateSharedNetwork(ctx context.Context, name string) error {
	netw, err := c.Client.NetworkInspect(ctx, name, network.InspectOptions{})
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return err
		}
		// Create network
		netwResp, err := c.Client.NetworkCreate(ctx, name, network.CreateOptions{})
		if err != nil {
			return err
		}
		slog.Info("Created network.", "name", name, "id", netwResp.ID)
	} else {
		// Network already exists
		slog.Info("Network already exists.", "name", netw.Name, "id", netw.ID)
	}
	return nil
}

func (c *ContainerClient) DockerCommand(args ...string) (string, []string, error) {
	return prepareDockerCommand(args...)
}

func (c *ContainerClient) ComposeUp(ctx context.Context, w io.Writer, projectName string, workingDir string, extraArgs ...string) error {
	slog.Info("Starting compose project.", "name", projectName, "dir", workingDir)
	command, args, err := prepareComposeCommand("up", "--detach", "--remove-orphans")
	if err != nil {
		return err
	}
	args = append(args, extraArgs...)
	prog := exec.Command(command, args...)
	prog.Dir = workingDir
	out, err := prog.CombinedOutput()
	fmt.Fprintf(w, "%s", out)

	if err != nil {
		return err
	}

	// Check if podman returned an error
	if strings.EqualFold(command, "podman-compose") {
		return CheckPodmanComposeError(string(out))
	}

	return nil
}

func CheckPodmanComposeError(b string) error {
	// Due to a podman bug, the exit code is not propagated back to the user
	// which means it is hard to determine if the command was successful or not
	// Link: https://github.com/thin-edge/tedge-container-plugin/issues/70
	// Only use the last message
	var lastErr error
	for _, line := range strings.Split(b, "\n") {
		_, value, ok := strings.Cut(line, "exit code: ")
		if ok {
			if i, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				if i == 0 {
					lastErr = nil
				} else {
					lastErr = fmt.Errorf("command failed. exit_code=%v", i)
				}
			}
		}
	}
	return lastErr
}

func (c *ContainerClient) ComposeDown(ctx context.Context, w io.Writer, projectName string, defaultWorkingDir string) error {
	// TODO: Read setting from configuration
	manualCleanup := false
	errs := make([]error, 0)

	projectFilter := filters.NewArgs(
		filters.Arg("label", "com.docker.compose.project="+projectName),
	)
	slog.Info("Searching for project containers.", "project", projectName)
	projectContainers, err := c.Client.ContainerList(ctx, container.ListOptions{
		Filters: projectFilter,
	})
	if err != nil {
		errs = append(errs, err)
	}
	slog.Info("Found project containers.", "count", len(projectContainers))

	workingDir := ""

	// Prefer using the working_dir on the container rather than
	// the default project working dir as the project could
	for _, item := range projectContainers {
		if v, ok := item.Labels["com.docker.compose.project.working_dir"]; ok {
			if utils.PathExists(v) {
				slog.Info("Using project working dir found on container label.", "project", projectName, "working_dir", workingDir, "container_id", item.ID, "container_name", item.Names)
				workingDir = v
				break
			}
		}
	}

	// Fallback to the default project dir
	if workingDir == "" {
		slog.Info("Using default project working dir.", "path", defaultWorkingDir)
		workingDir = defaultWorkingDir
	}

	// Find
	if utils.PathExists(workingDir) {
		command, args, err := prepareComposeCommand("down", "--remove-orphans", "--volumes")
		if err != nil {
			return err
		}
		slog.Info("Stopping compose project.", "name", projectName, "dir", workingDir, "command", command, "args", strings.Join(args, " "))
		prog := exec.Command(command, args...)
		prog.Dir = workingDir
		out, err := prog.CombinedOutput()
		fmt.Fprintf(w, "%s", out)

		if err == nil {
			slog.Info("Removing project directory.", "dir", workingDir)
			if removeErr := os.RemoveAll(workingDir); removeErr != nil {
				// non critical error
				slog.Warn("Failed to remove project directory.", "err", removeErr)
			}
		}

		errs = append(errs, err)
		slog.Warn("compose failed.", "err", err)
	} else {
		errs = append(errs, fmt.Errorf("compose project working directory does not exist. dir=%s", workingDir))
	}

	if !manualCleanup {
		return errors.Join(errs...)
	}

	// Manually remove in case if docker compose fail
	// Get containers

	// Stop containers
	for _, item := range projectContainers {
		slog.Info("Manually stopping container.", "id", item.ID, "names", item.Names)
		if err := c.Client.ContainerStop(ctx, item.ID, container.StopOptions{}); err != nil {
			slog.Warn("Failed to stop container.", "err", err)
			errs = append(errs, err)
		}
	}

	// Remove containers
	for _, item := range projectContainers {
		slog.Info("Manually removing container.", "id", item.ID, "names", item.Names)
		if err := c.Client.ContainerRemove(ctx, item.ID, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   true,
			Force:         true,
		}); err != nil {
			slog.Warn("Failed to stop container.", "err", err)
			errs = append(errs, err)
		}
	}

	// Get networks
	projectNetworks, err := c.Client.NetworkList(ctx, network.ListOptions{
		Filters: projectFilter,
	})
	if err != nil {
		errs = append(errs, err)
	}

	// Remove networks
	for _, item := range projectNetworks {
		slog.Info("Manually removing network.", "name", item.ID)
		if err := c.Client.NetworkRemove(ctx, item.ID); err != nil {
			slog.Warn("Failed to remove network.", "err", err)
			errs = append(errs, err)
		}
	}

	// Remove volumes
	projectVolumes, err := c.Client.VolumeList(ctx, volume.ListOptions{
		Filters: projectFilter,
	})
	if err != nil {
		errs = append(errs, err)
	}

	for _, item := range projectVolumes.Volumes {
		slog.Info("Manually removing volume.", "name", item.Name)
		if err := c.Client.VolumeRemove(ctx, item.Name, true); err != nil {
			slog.Warn("Failed to remove volume.", "err", err)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

var ContainerStatusHealthy = "healthy"

func (c *ContainerClient) WaitForHealthy(ctx context.Context, containerID string) error {

	runningCount := 0
	for {
		// Check if the
		con, err := c.Client.ContainerInspect(ctx, containerID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// Just log error
			slog.Info("Could not get container status.", "err", err)

			time.Sleep(5 * time.Second)
			continue
		}

		// Check if container has a health check command
		if con.Config.Healthcheck == nil {
			slog.Info("Container does not have a health-check script, using state.", "state", con.State.Status, "ok_count", runningCount)
			if con.State.Running && runningCount > 1 {
				return nil
			}
			runningCount += 1
			time.Sleep(5 * time.Second)
			continue
		}

		// container has a health check
		if con.State != nil && con.State.Health != nil {
			if strings.HasPrefix(strings.ToLower(con.State.Health.Status), ContainerStatusHealthy) {
				slog.Info("Container is healthy.", "status", con.State.Health.Status, "failing_streak", con.State.Health.FailingStreak)
				return nil
			}
			slog.Info("Container is not healthy yet.", "status", con.State.Health.Status, "failing_streak", con.State.Health.FailingStreak)
		} else {
			slog.Info("Container is not healthy yet.", "id", containerID, "state", con.State)
		}

		time.Sleep(5 * time.Second)
	}
}

// Wait for a container to be stopped by polling it's status
// Avoid using ContainerWait is it is not compatible with older docker versions
// and probably less compatible with podman
func (c *ContainerClient) WaitForStop(ctx context.Context, containerID string) error {
	for {
		con, err := c.Client.ContainerInspect(ctx, containerID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// Just log error
			slog.Info("Could not get container status.", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if !con.State.Running {
			slog.Info("Container is not running.", "id", containerID, "state", con.State)
			return nil
		}
		slog.Info("Container is still running.", "id", containerID, "state", con.State)
		time.Sleep(5 * time.Second)
	}
}

func (c *ContainerClient) UpdateRequired(ctx context.Context, containerID string, newImage string) (bool, types.ContainerJSON, error) {
	prevContainer, err := c.Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, prevContainer, err
	}

	if newImage == "" {
		newImage = prevContainer.Config.Image
	}

	images, err := c.Client.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", newImage)),
	})
	if err != nil {
		return false, prevContainer, err
	}

	if len(images) == 0 {
		slog.Info("Image does not exist locally, assuming update is required.", "reference", newImage)
		return true, prevContainer, err
	}

	nextImage := images[0]

	// Check if new image matches existing image
	// nextImage, _, err := c.Client.ImageInspectWithRaw(ctx, newImage)
	// if err != nil {
	// 	return false, prevContainer, err
	// }

	slog.Info("Current container image.", "name", prevContainer.Config.Image, "id", prevContainer.Image)
	slog.Info("Next container image.", "name", strings.Join(nextImage.RepoTags, ","), "id", nextImage.ID)
	if prevContainer.Image == nextImage.ID {
		slog.Info("Container image is already up to date.", "image", prevContainer.Config.Image, "image_hash", prevContainer.Image)
		return false, prevContainer, nil
	}
	return true, prevContainer, nil
}

// Log options type alias
type LogsOptions container.LogsOptions

func (c *ContainerClient) ContainerLogs(ctx context.Context, w io.Writer, containerID string, opts LogsOptions) error {
	reader, logErr := c.Client.ContainerLogs(ctx, containerID, container.LogsOptions(opts))
	if logErr != nil {
		return logErr
	}
	defer reader.Close()
	_, err := StdCopy(w, w, reader)
	if err != nil {
		return err
	}
	return nil
}

type CloneOptions struct {
	Name         string
	Image        string
	HealthyAfter time.Duration
	StopAfter    time.Duration
	StopTimeout  time.Duration
	WaitForExit  bool
	AutoRemove   bool
	Env          []string
	ExtraHosts   []string
	Cmd          strslice.StrSlice
	Entrypoint   strslice.StrSlice
	IgnorePorts  bool
	Labels       map[string]string

	SkipNetwork   bool
	IgnoreEnvVars []string
}

func FormatContainerName(v string) string {
	return strings.TrimPrefix(v, "/")
}

// Clone an existing container by spawning a new container with the same configuration
// but using a new image
func (c *ContainerClient) CloneContainer(ctx context.Context, containerID string, opts CloneOptions) error {
	prevContainer, err := c.Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}

	if opts.Image == "" {
		opts.Image = prevContainer.Config.Image
	}

	prevImage := prevContainer.Image
	backupContainerName := FormatContainerName(fmt.Sprintf("%s-%s-%d", prevContainer.Name, "bak", time.Now().Unix()))
	containerName := FormatContainerName(prevContainer.Name)

	slog.Info("Removing previous backup container if it exists")
	if err := c.StopRemoveContainer(ctx, backupContainerName); err != nil {
		return err
	}

	if opts.WaitForExit {
		slog.Info("Disabling restart policy of previous container.", "id", prevContainer.ID)
		updateResp, updateErr := c.Client.ContainerUpdate(ctx, prevContainer.ID, container.UpdateConfig{
			RestartPolicy: container.RestartPolicy{
				Name: container.RestartPolicyDisabled,
			},
		})
		if updateErr != nil {
			if !errdefs.IsNotFound(updateErr) {
				return updateErr
			}
			slog.Warn("Failed to change restart policy.", "id", prevContainer.ID, "response", updateResp)
		} else {
			slog.Info("Changed restart policy.", "id", prevContainer.ID, "response", updateResp)
		}
		slog.Info("Waiting for previous container to stop.", "id", prevContainer.ID, "name", prevContainer.Name)
		timeoutCtx, cancel := context.WithTimeout(ctx, opts.StopTimeout)
		defer cancel()
		if err := c.WaitForStop(timeoutCtx, prevContainer.ID); err != nil {
			return err
		}
		slog.Info("Container stopped.", "id", prevContainer.ID)
	} else {
		// Pause before stopping the container
		if opts.StopAfter > 0 {
			slog.Info("Waiting before stopping container.", "id", prevContainer.ID, "duration", opts.StopAfter)
			time.Sleep(opts.StopAfter)
		}

		slog.Info("Stopping previous container.", "id", prevContainer.ID, "name", prevContainer.Name)
		stopErr := c.Client.ContainerStop(ctx, prevContainer.ID, container.StopOptions{})
		if stopErr != nil {
			return stopErr
		}
	}

	slog.Info("Renaming container.", "id", prevContainer.ID, "old", prevContainer.Name, "new", backupContainerName)
	if err := c.Client.ContainerRename(ctx, containerID, backupContainerName); err != nil {
		return err
	}

	slog.Info("Copying configuration from an existing container.", "name", prevContainer.Name, "newImage", opts.Image, "prevImage", prevImage, "config", prevContainer.Config, "host_config", prevContainer.HostConfig)

	// Container config
	clonedConfig := CloneContainerConfig(prevContainer.Config, opts)

	// Copy host config
	hostConfig := CloneHostConfig(prevContainer.HostConfig, opts)

	// Copy network config
	var networkConfig *network.NetworkingConfig
	if !opts.SkipNetwork {
		networkConfig = CloneNetworkConfig(prevContainer.NetworkSettings)
	}

	slog.Info("Creating new container.", "name", prevContainer.Name, "newImage", opts.Image, "prevImage", prevImage, "config", prevContainer.Config, "host_config", prevContainer.HostConfig)
	nextContainer, createErr := c.Client.ContainerCreate(ctx, clonedConfig, hostConfig, networkConfig, nil, containerName)

	if createErr != nil {
		return createErr
	}
	slog.Info("Created new container.", "id", nextContainer.ID, "name", containerName)

	// start container
	if err := c.Client.ContainerStart(ctx, nextContainer.ID, container.StartOptions{}); err != nil {
		slog.Warn("Container failed to start.", "id", nextContainer.ID, "err", err)
	}

	// Check if the container is healthy
	slog.Info("Waiting for container to be healthy.", "id", nextContainer.ID, "name", containerName)
	healthCtx, cancel := context.WithTimeout(ctx, opts.HealthyAfter)
	defer cancel()
	if err := c.WaitForHealthy(healthCtx, nextContainer.ID); err != nil {
		slog.Info("New container is not healthy, reverting to the previous container.", "prevContainerID", prevContainer.ID, "name", containerName)

		// Collect logs from it
		reader, logErr := c.Client.ContainerLogs(ctx, nextContainer.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: false,
			Details:    false,
			Tail:       "100",
		})
		if logErr != nil {
			slog.Warn("Could not get logs from new container.", "id", nextContainer.ID, "err", logErr)
		} else {
			defer reader.Close()

			_, err := StdCopy(os.Stderr, os.Stderr, reader)
			if err != nil {
				return err
			}
		}

		// Revert
		stopErr := c.Client.ContainerStop(ctx, prevContainer.ID, container.StopOptions{})
		if stopErr != nil {
			slog.Warn("Could not stop new container but restoring old container anyway.", "id", prevContainer.ID)
		}

		// Delete the new container
		if err := c.StopRemoveContainer(ctx, nextContainer.ID); err != nil {
			// Just log, don't fail as the previous container needs to be restored
			slog.Warn("Could not stop and remove newly spawned container.", "err", err)
		}

		slog.Info("Restoring previous container instance.", "id", prevContainer.ID, "old", backupContainerName, "new", containerName)
		if err := c.Client.ContainerRename(ctx, prevContainer.ID, containerName); err != nil {
			return err
		}

		if err := c.Client.ContainerStart(ctx, prevContainer.ID, container.StartOptions{}); err != nil {
			return err
		}
		slog.Info("Restored previous container instance.", "id", prevContainer.ID, "name", containerName)
		return nil
	}

	slog.Info("Removing previous container")
	if err := c.StopRemoveContainer(ctx, prevContainer.ID); err != nil {
		slog.Warn("Failed to remove previous container.", "err", err)
	}

	// TODO: Should the previous container now be destroyed?
	slog.Info("Successfully created new container.", "id", nextContainer.ID, "name", containerName, "image", opts.Image)
	return nil
}

// Get the container id which is running the current process
//
// Finding the container that the process is running in is fairly complicated
// due to the differences between the container engines and versions (e.g. podman, docker etc.)
//
//  1. Check if hostname matches the container id/name
//  2. Look through each con
//     2.1 Check container ID file (if the file exists)
//     2.2 Check HOSTNAME env variable (e.g. HOSTNAME={hostname})
//     2.3 Check HostConfig.Hostname value
func (c *ContainerClient) Self(ctx context.Context) (types.ContainerJSON, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	envHostname := fmt.Sprintf("HOSTNAME=%s", hostname)

	// Lookup container by hostname
	// This can work on docker, and some versions of podman
	// and it is still worth trying as it saves some api requests
	if con, err := c.Client.ContainerInspect(ctx, hostname); err == nil {
		slog.Info("Found container id/name by referencing hostname.", "hostname", hostname, "id", con.ID, "name", con.Name)
		return con, nil
	}

	// Fallback to searching each container for the matching hostname
	// Search for a container with the same hostname
	// Note: the container list does not contain the hostname config
	// so we have to inspect each container until we find a match
	conList, err := c.Client.ContainerList(ctx, container.ListOptions{
		// only include running containers
		All: false,
	})
	if err != nil {
		return types.ContainerJSON{}, err
	}
	for _, conItem := range conList {
		if con, err := c.Client.ContainerInspect(ctx, conItem.ID); err == nil {
			// Check container id file (if present)
			if con.HostConfig != nil && con.HostConfig.ContainerIDFile != "" {
				if contID, err := os.ReadFile(con.HostConfig.ContainerIDFile); err == nil {
					if hostname == string(bytes.TrimSpace(contID)) {
						slog.Info("Found container id/name by container id file.", "hostname", hostname, "cidFile", con.HostConfig.ContainerIDFile, "id", con.ID, "name", con.Name)
						return con, nil
					}
				}
			}

			// Ignore forked containers as these are only temporary containers
			if _, found := con.Config.Labels["io.tedge.fork"]; found {
				continue
			}
			if con.Config.Hostname == hostname {
				slog.Info("Found container id/name by config.hostname.", "hostname", hostname, "id", con.ID, "name", con.Name)
				return con, nil
			}
			for _, v := range con.Config.Env {
				if v == envHostname {
					slog.Info("Found container id/name by HOSTNAME env.", "hostname", hostname, "id", con.ID, "name", con.Name)
					return con, nil
				}
			}
		}
	}
	return types.ContainerJSON{}, errdefs.NotFound(fmt.Errorf("could not find container by hostname"))
}

// Prune both unused and dangling images
func (c *ContainerClient) ImagesPruneUnused(ctx context.Context) (image.PruneReport, error) {
	pruneFilters := filters.NewArgs()
	// Note: dangling=false is the equivalent to docker image prune -a
	pruneFilters.Add("dangling", strconv.FormatBool(false))
	report, apiErr := c.Client.ImagesPrune(ctx, pruneFilters)
	if apiErr == nil {
		return report, apiErr
	}

	// Note: Due to a bug in podman <= 4.8, the above call will fail, so the direct libpod is used instead
	// Reference: https://github.com/containers/podman/issues/20469
	// NewDefaultLibPodHTTPClient

	slog.Info("Prune images failed. This is expected when using podman < 4.8.", "err", apiErr)

	slog.Info("Using podman api to prune unused images")
	report, libpodErr := NewDefaultLibPodHTTPClient().PruneImages(nil)

	// Don't fail as it is unclear how stable the libpod api is
	if libpodErr != nil {
		slog.Warn("podman (libpod) prune images failed but error will be ignored.", "err", libpodErr)
	}
	return report, nil
}

func (c *ContainerClient) Fork(ctx context.Context, currentContainer types.ContainerJSON, cloneOptions CloneOptions) error {
	if cloneOptions.Image == "" {
		cloneOptions.Image = currentContainer.Config.Image
	}

	cloneOptions.Labels["io.tedge.fork"] = "1"
	cloneOptions.Labels["io.tedge.forked.name"] = currentContainer.Name

	containerConfig := CloneContainerConfig(currentContainer.Config, cloneOptions)

	hostConfig := CloneHostConfig(currentContainer.HostConfig, cloneOptions)

	// TODO: Protect against the updater from shutting down due to a machine restart
	// or is this not required if the restart policy of the container being updated
	// is left at RestartAlways (as it would restart after a device reboot)
	hostConfig.RestartPolicy = container.RestartPolicy{
		Name: container.RestartPolicyDisabled,
	}

	var networkConfig *network.NetworkingConfig
	if !cloneOptions.SkipNetwork {
		slog.Info("Cloning network configuration")
		networkConfig = CloneNetworkConfig(currentContainer.NetworkSettings)
	} else {
		slog.Info("Ignoring network configuration in forked container")
	}
	slog.Info("Forking container.", "new_image", containerConfig.Image, "from_id", currentContainer.ID)

	if cloneOptions.Name != "" {
		if err := c.StopRemoveContainer(ctx, cloneOptions.Name); err != nil {
			return err
		}
	}

	resp, respErr := c.Client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, cloneOptions.Name)
	if respErr != nil {
		return respErr
	}

	startErr := c.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if startErr != nil {
		slog.Error("Failed to start container.", "id", resp.ID, "err", startErr)
		return startErr
	}
	slog.Info("Successfully created forked container.", "id", resp.ID)

	// Wait for the new container to be stable?
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.WaitForHealthy(timeoutCtx, resp.ID)
}

func IsInsideContainer() bool {
	paths := []string{
		"/.dockerenv",
		"/run/.containerenv",
	}
	for _, p := range paths {
		if utils.PathExists(p) {
			return true
		}
	}
	return false
}

func CloneContainerConfig(ref *container.Config, opts CloneOptions) *container.Config {
	clonedConfig := &container.Config{
		User:            ref.User,
		Cmd:             ref.Cmd,
		Entrypoint:      ref.Entrypoint,
		Env:             FilterEnvVariables(ref.Env, opts.IgnoreEnvVars),
		NetworkDisabled: false,
		StopSignal:      ref.StopSignal,
		Image:           ref.Image,
		Volumes:         ref.Volumes,
		Tty:             ref.Tty,
		ExposedPorts:    ref.ExposedPorts,
		Domainname:      ref.Domainname,
		// Don't copy oci labels as they are included in the image itself
		Labels: FilterLabels(ref.Labels, []string{"org.opencontainers."}),
	}
	if len(opts.Cmd) > 0 {
		clonedConfig.Cmd = opts.Cmd
	}

	if len(opts.Entrypoint) > 0 {
		clonedConfig.Entrypoint = opts.Entrypoint
	}
	if opts.Image != "" {
		clonedConfig.Image = opts.Image
	}

	for label, value := range opts.Labels {
		clonedConfig.Labels[label] = value
	}

	clonedConfig.Env = append(clonedConfig.Env, opts.Env...)
	return clonedConfig
}

func CloneHostConfig(ref *container.HostConfig, opts CloneOptions) *container.HostConfig {
	clone := &container.HostConfig{
		Binds:       ref.Binds,
		AutoRemove:  opts.AutoRemove,
		Annotations: ref.Annotations,
		CapAdd:      ref.CapAdd,
		CapDrop:     ref.CapDrop,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyAlways,
		},
		DNS:             ref.DNS,
		DNSOptions:      ref.DNSOptions,
		Links:           ref.Links,
		Privileged:      ref.Privileged,
		Mounts:          ref.Mounts,
		Tmpfs:           ref.Tmpfs,
		PortBindings:    ref.PortBindings,
		PublishAllPorts: ref.PublishAllPorts,
		ExtraHosts:      append(ref.ExtraHosts, opts.ExtraHosts...),
		OomScoreAdj:     ref.OomScoreAdj,
		ReadonlyRootfs:  ref.ReadonlyRootfs,
		VolumeDriver:    ref.VolumeDriver,
		VolumesFrom:     ref.VolumesFrom,
		Init:            ref.Init,
		LogConfig:       ref.LogConfig,
		DNSSearch:       ref.DNSSearch,
		StorageOpt:      ref.StorageOpt,
		ReadonlyPaths:   ref.ReadonlyPaths,
		SecurityOpt:     ref.SecurityOpt,
		GroupAdd:        ref.GroupAdd,
		Runtime:         ref.Runtime,
		ContainerIDFile: ref.ContainerIDFile,
	}

	if opts.SkipNetwork {
		clone.NetworkMode = network.NetworkNone
	} else {
		clone.NetworkMode = ref.NetworkMode
	}

	if opts.IgnorePorts {
		clone.PortBindings = nat.PortMap{}
		clone.PublishAllPorts = false
	}

	return clone
}

// Clone network settings, but only clone the network ids that the container is part of
// don't clone everything as it leads to incompatibilities between engine versions
func CloneNetworkConfig(ref *types.NetworkSettings) *network.NetworkingConfig {
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	for networkName, value := range ref.Networks {
		if value.NetworkID != "" {
			networkConfig.EndpointsConfig[networkName] = &network.EndpointSettings{
				NetworkID: value.NetworkID,
			}
		}
	}
	return networkConfig
}

func FormatLabels(labels []string) map[string]string {
	labelMap := make(map[string]string)
	for _, label := range labels {
		if name, value, found := strings.Cut(label, "="); found {
			labelMap[name] = value
		}
	}
	return labelMap
}

func FilterLabels(l map[string]string, exclude []string) map[string]string {
	filtered := make(map[string]string)

	for k, v := range l {
		if !strings.HasPrefix(k, "org.opencontainers.") {
			filtered[k] = v
		}
	}
	return filtered
}

func FilterEnvVariables(l []string, exclude []string) []string {
	filtered := make([]string, 0, len(l))

	for _, envItem := range l {
		ignore := false
		for _, excludePattern := range exclude {
			if strings.HasPrefix(envItem, excludePattern) {
				ignore = false
				break
			}
		}
		if !ignore {
			filtered = append(filtered, envItem)
		}
	}
	return filtered
}
