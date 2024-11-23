package container

import (
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
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-units"
	"github.com/thin-edge/tedge-container-plugin/pkg/utils"
)

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

	// filterValues = append(filterValues, filters.Arg("label", "com.docker.compose.project"))

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

func (c *ContainerClient) ComposeDown(ctx context.Context, w io.Writer, projectName string) error {
	// TODO: Read setting from configuration
	manualCleanup := false
	errs := make([]error, 0)

	workingDir := ""

	projectFilter := filters.NewArgs(
		filters.Arg("label", "com.docker.compose.project="+projectName),
	)
	projectContainers, err := c.Client.ContainerList(ctx, container.ListOptions{
		Filters: projectFilter,
	})
	if err != nil {
		errs = append(errs, err)
	}
	for _, item := range projectContainers {
		if v, ok := item.Labels["com.docker.compose.project.working_dir"]; ok {
			workingDir = v
			break
		}
	}

	// Find
	if workingDir != "" && utils.PathExists(workingDir) {
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
			if err := os.RemoveAll(workingDir); err != nil {
				// non critical error
				slog.Warn("Failed to remove project directory.", "err", err)
			}
			return nil
		}

		errs = append(errs, err)
		slog.Warn("compose failed.", "err", err)
	}

	if !manualCleanup {
		return errors.Join(errs...)
	}

	// Manually remove in case if docker compose fail
	// Get containers

	// Stop containers
	for _, item := range projectContainers {
		if err := c.Client.ContainerStop(ctx, item.ID, container.StopOptions{}); err != nil {
			slog.Warn("Failed to stop container.", "err", err)
			errs = append(errs, err)
		}
	}

	// Remove containers
	for _, item := range projectContainers {
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
		if err := c.Client.VolumeRemove(ctx, item.Name, true); err != nil {
			slog.Warn("Failed to remove volume.", "err", err)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
