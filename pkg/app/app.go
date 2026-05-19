package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/random"
	"github.com/thin-edge/tedge-container-plugin/pkg/tedge"
)

type Action int

const (
	ActionUpdateAll Action = iota
	ActionUpdateMetrics
)

type ActionRequest struct {
	Action  Action
	Options any
	// result, when non-nil, receives the outcome of the request so that the
	// caller can block waiting for completion. Leave nil for fire-and-forget.
	result chan<- error
}

func NewUpdateAllAction(filter container.FilterOptions) ActionRequest {
	return ActionRequest{
		Action:  ActionUpdateAll,
		Options: filter,
	}
}

func NewUpdateMetricsAction(filter container.FilterOptions) ActionRequest {
	return ActionRequest{
		Action:  ActionUpdateMetrics,
		Options: filter,
	}
}

type App struct {
	client          *tedge.Client
	ContainerClient *container.ContainerClient

	Device *tedge.Target

	config         Config
	shutdown       chan struct{}
	updateRequests chan ActionRequest
	debouncer      *UpdateDebouncer
	// restartBaseline maps service name → the Docker RestartCount observed the
	// last time the container was considered healthy (or first seen). On each
	// ActionStart event we inspect the container; the delta between its current
	// RestartCount and this baseline is the number of daemon-initiated restarts
	// since the last known-good state.
	restartBaseline   map[string]int
	restartBaselineMu sync.Mutex
	crashLoopAlarms   map[string]struct{}
	crashLoopAlarmsMu sync.Mutex
	// eventLimiter rate-limits per-(container, event-type) MQTT publishes so
	// that a crash-looping container cannot flood the broker.
	eventLimiter *EventRateLimiter
	wg           sync.WaitGroup
}

type Config struct {
	ContainerHost string

	ServiceName string

	// TLS
	KeyFile  string
	CertFile string
	CAFile   string

	// Feature flags
	EnableMetrics      bool
	EnableEngineEvents bool
	DeleteFromCloud    bool
	DeleteOrphans      bool
	RunOnce            bool

	HTTPHost string
	HTTPPort uint16

	MQTTHost string
	MQTTPort uint16

	CumulocityHost string
	CumulocityPort uint16

	// CrashLoopThreshold is the number of daemon-initiated restarts since the
	// last healthy state required to declare a crash loop.
	CrashLoopThreshold int

	// UseModuleNameForService controls whether the thin-edge service name for
	// container-group services is derived from the stored module name (true,
	// default) or from the compose project name taken from Docker labels
	// (false). Set to false when you want the runtime service identity to be
	// decoupled from the software module name, e.g. the module is "myapp-dev"
	// but the compose project name (and therefore the service name) is "myapp".
	UseModuleNameForService bool
}

func NewApp(device tedge.Target, config Config) (*App, error) {
	serviceTarget := device.Service(config.ServiceName)
	tedgeOpts := &tedge.ClientConfig{
		HTTPHost: config.HTTPHost,
		HTTPPort: config.HTTPPort,
		MqttHost: config.MQTTHost,
		MqttPort: config.MQTTPort,
		C8yHost:  config.CumulocityHost,
		C8yPort:  config.CumulocityPort,
		CertFile: config.CertFile,
		KeyFile:  config.KeyFile,
		CAFile:   config.CAFile,
	}
	if config.RunOnce {
		// use a randomized client id in run-once mode so it doesn't affect the main
		// service or any other instances also running in run-once mode
		tedgeOpts.MQTTClientID = fmt.Sprintf("%s-%s#%s", config.ServiceName, random.String(8), serviceTarget.Topic())
	}
	tedgeClient := tedge.NewClient(device, *serviceTarget, config.ServiceName, tedgeOpts)

	ctx, ctxCancel := context.WithTimeout(context.TODO(), 300*time.Second)
	defer ctxCancel()

	clientOptions := make([]container.Opt, 0)

	// Use a time-based timeout instead of limiting number of retries
	clientOptions = append(clientOptions, container.WithInfiniteRetries())

	if config.ContainerHost != "" {
		clientOptions = append(clientOptions, container.WithHost(config.ContainerHost))
	}

	containerClient, err := container.NewContainerClient(
		ctx,
		clientOptions...,
	)
	if err != nil {
		return nil, err
	}

	// Register via http interface
	_, registrationErr := tedgeClient.TedgeAPI.CreateEntity(context.Background(), tedge.Entity{
		TedgeType:    tedge.EntityTypeService,
		Name:         serviceTarget.Name,
		TedgeTopicID: serviceTarget.TopicID,
	})
	if registrationErr == nil {
		slog.Info("Registered service", "topic", serviceTarget.Topic())
	} else {
		slog.Error("Could not register tedge entity.", "err", registrationErr)
	}

	if err := tedgeClient.Connect(); err != nil {
		return nil, err
	}

	if tedgeClient.Target.CloudIdentity == "" {
		for {
			slog.Info("Looking up thin-edge.io Cumulocity ExternalID")
			if currentUser, _, err := tedgeClient.CumulocityClient.User.GetCurrentUser(context.Background()); err == nil {
				externalID := strings.TrimPrefix(currentUser.Username, "device_")
				tedgeClient.Target.CloudIdentity = externalID
				device.CloudIdentity = externalID
				slog.Info("Found Cumulocity ExternalID", "value", tedgeClient.Target.CloudIdentity)
				break
			} else {
				slog.Warn("Failed to lookup Cumulocity ExternalID.", "err", err)
				// retry until it is successful
				time.Sleep(10 * time.Second)
			}
		}
	}

	application := &App{
		client:          tedgeClient,
		ContainerClient: containerClient,
		Device:          &device,
		config:          config,
		// Buffered so that the debouncer's dispatch and synchronous Update()
		// calls never block when the worker is briefly busy.
		updateRequests: make(chan ActionRequest, 8),
		shutdown:       make(chan struct{}),
		wg:             sync.WaitGroup{},
	}

	// The debouncer coalesces rapid-fire event-driven update requests into a
	// single doUpdate call. It is used by Monitor() and Subscribe() callbacks.
	application.debouncer = NewUpdateDebouncer(2*time.Second, func(req ActionRequest) {
		select {
		case application.updateRequests <- req:
		default:
			slog.Warn("Dropped debounced update request: worker queue is full.")
		}
	})

	if config.CrashLoopThreshold <= 0 {
		config.CrashLoopThreshold = 5
	}
	// Track container restart baselines to detect crash loops using the Docker
	// daemon's own RestartCount.
	application.restartBaseline = make(map[string]int)
	application.crashLoopAlarms = make(map[string]struct{})

	// Rate-limit engine event publishes: at most 1 event per
	// (container, event-type) per 5 seconds. Combined with crash-loop
	// suppression this eliminates broker flooding from restart storms.
	application.eventLimiter = NewEventRateLimiter(5 * time.Second)

	// Start background task to process requests
	application.wg.Add(1)
	go application.worker()

	// Register MQTT route callbacks. This must be done after the struct is
	// constructed (so the callbacks can reference it) but is safe to call
	// here because AddRoute only registers in-process handlers — no broker
	// interaction occurs.
	if err := application.Subscribe(); err != nil {
		return nil, err
	}

	return application, nil
}

func (a *App) DeleteLegacyService(deleteFromCloud bool) {
	target := a.client.Target.Service("tedge-container-monitor")
	slog.Info("Removing legacy service from the cloud", "topic", target.Topic())

	if _, err := a.client.TedgeAPI.DeleteEntity(context.Background(), *target); err != nil {
		slog.Warn("Failed to clear registration message.", "topic-id", target.TopicID)
	}

	time.Sleep(500 * time.Millisecond)

	if target.CloudIdentity != "" && deleteFromCloud {
		if _, err := a.client.DeleteCumulocityManagedObject(*target); err != nil {
			slog.Warn("Failed to delete managed object.", "err", err)
		}
	}
}

// Delete any unclaimed/orphaned cloud services which haven't been registered with
func (a *App) DeleteOrphanedCloudServices(tedgeEntities map[string]tedge.Entity) error {
	extID, _, err := a.client.CumulocityClient.Identity.GetExternalID(context.Background(), "c8y_Serial", a.Device.ExternalID())
	if err != nil {
		slog.Warn("Could not lookup device's managed object by its external id.", "err", err, "externalId", a.Device.ExternalID())
	}

	mos, _, err := a.client.CumulocityClient.Inventory.GetChildAdditions(context.Background(), extID.ManagedObject.ID, &c8y.ManagedObjectOptions{
		Query:             fmt.Sprintf("type eq 'c8y_Service' and (serviceType eq '%s' or serviceType eq '%s')", container.ContainerType, container.ContainerGroupType),
		PaginationOptions: *c8y.NewPaginationOptions(100),
	})
	if err != nil {
		return err
	}

	slog.Info("Found cloud services.", "count", len(mos.References))

	for _, ref := range mos.References {
		target := a.client.Target.Service(ref.ManagedObject.Name)
		slog.Info("Check if service is registered locally.", "topic-id", target.TopicID)

		if _, found := tedgeEntities[target.TopicID]; !found {
			slog.Info("Found orphaned cloud service.", "service", ref.ManagedObject.Name, "type", ref.ManagedObject.Type, "moID", ref.ManagedObject.ID)

			if _, respErr := a.client.CumulocityClient.Inventory.Delete(context.Background(), ref.ManagedObject.ID); respErr != nil {
				slog.Warn("Could not delete orphaned cloud service.", "err", respErr)
			} else {
				slog.Info("Successfully deleted orphaned cloud service.", "moID", ref.ManagedObject.ID)
			}
		} else {
			slog.Info("Service is registered locally.", "topic-id", target.TopicID)
		}
	}
	return nil
}

func (a *App) Subscribe() error {
	topic := tedge.GetTopic(*a.Device.Service("+"), "cmd", "health", "check")
	slog.Info("Listening to commands on topic.", "topic", topic)

	a.client.Client.AddRoute(topic, func(c mqtt.Client, m mqtt.Message) {
		parts := strings.Split(m.Topic(), "/")
		if len(parts) > 5 {
			slog.Info("Received request to update service data.", "service", parts[4], "topic", topic)
			name := parts[4]
			opts := container.FilterOptions{}
			// If the name matches the current service name, then update all containers
			if name != a.config.ServiceName {
				opts.Names = []string{fmt.Sprintf("^%s$", name)}
			}
			a.debouncer.Enqueue(NewUpdateAllAction(opts))
		}
	})

	// Subscribe to cloud bridge health topics so we can retry any failed cloud
	// deletions and trigger a full resync when connectivity is restored.
	// Both the built-in bridge (tedge-mapper-c8y) and the mosquitto bridge
	// (c8y-mapper) variants are covered.
	for _, bridgeService := range []string{"tedge-mapper-c8y", "tedge-mapper-bridge-c8y", "mosquitto-c8y-bridge"} {
		bridgeTopic := tedge.GetHealthTopic(*a.Device.Service(bridgeService))
		slog.Info("Subscribing to bridge health topic.", "topic", bridgeTopic)
		a.client.Client.AddRoute(bridgeTopic, func(c mqtt.Client, m mqtt.Message) {
			if len(m.Payload()) == 0 {
				return
			}
			if isBridgeOnline(m.Payload()) {
				slog.Info("Cloud bridge is online, triggering service resync to process any pending cloud deletions.", "topic", m.Topic())
				a.debouncer.Enqueue(NewUpdateAllAction(container.FilterOptions{}))
			}
		})
	}

	return nil
}

func (a *App) Stop(clean bool) {
	if a.client != nil {
		if clean {
			slog.Info("Disconnecting MQTT client cleanly")
			a.client.Client.Disconnect(250)
		}
	}
	a.shutdown <- struct{}{}

	// Wait for shutdown confirmation
	a.wg.Wait()
}

// sendResult delivers err to the request's result channel when one was
// provided. It is non-blocking: if the channel is full the send is dropped.
func sendResult(req ActionRequest, err error) {
	if req.result != nil {
		select {
		case req.result <- err:
		default:
		}
	}
}

func (a *App) worker() {
	defer a.wg.Done()
	for {
		select {
		case req := <-a.updateRequests:

			switch req.Action {
			case ActionUpdateAll:
				slog.Info("Processing update request")
				err := a.doUpdate(req.Options.(container.FilterOptions))
				sendResult(req, err)
			case ActionUpdateMetrics:
				items, err := a.ContainerClient.List(context.Background(), req.Options.(container.FilterOptions))
				if err != nil {
					slog.Warn("Could not get container list.", "err", err)
				} else {
					items = a.applyServiceNamePolicy(items)
					err = a.updateMetrics(items)
					if err != nil {
						slog.Warn("Error updating metrics.", "err", err)
					}
				}
				sendResult(req, err)
			}

		case <-a.shutdown:
			slog.Info("Stopping background task")
			return
		}
	}
}

func (a *App) Update(filterOptions container.FilterOptions) error {
	result := make(chan error, 1)
	a.updateRequests <- ActionRequest{
		Action:  ActionUpdateAll,
		Options: filterOptions,
		result:  result,
	}
	return <-result
}

func (a *App) UpdateMetrics(filterOptions container.FilterOptions) error {
	result := make(chan error, 1)
	a.updateRequests <- ActionRequest{
		Action:  ActionUpdateMetrics,
		Options: filterOptions,
		result:  result,
	}
	return <-result
}

var ContainerEventText = map[events.Action]string{
	events.ActionCreate:                "created",
	events.ActionStart:                 "started",
	events.ActionStop:                  "stopped",
	events.ActionDestroy:               "destroyed",
	events.ActionRemove:                "removed",
	events.ActionDie:                   "died",
	events.ActionPause:                 "paused",
	events.ActionUnPause:               "unpaused",
	events.ActionExecDie:               "process died",
	events.ActionHealthStatusHealthy:   "healthy",
	events.ActionHealthStatusUnhealthy: "unhealthy",
}

func mustMarshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// isBridgeOnline returns true when a bridge health payload indicates the bridge
// is online. It handles two formats:
//   - The mosquitto bridge format: a plain "1" (online) or "0" (offline).
//   - The thin-edge built-in bridge format: JSON {"status":"up"}.
func isBridgeOnline(payload []byte) bool {
	// Mosquitto bridge publishes "1" when connected and "0" when disconnected.
	p := strings.TrimSpace(string(payload))
	if p == "1" {
		return true
	}
	if p == "0" {
		return false
	}
	var s struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(payload, &s); err != nil {
		return false
	}
	return s.Status == tedge.StatusUp
}

func getEventAttributes(attr map[string]string, props ...string) []string {
	out := make([]string, 0)
	for _, prop := range props {
		value := ""
		if v, ok := attr[prop]; ok {
			value = v
		}
		out = append(out, value)
	}
	return out
}

// serviceNameFromEventAttrs derives the thin-edge service name from Docker
// event Actor.Attributes. For plain containers this is the container name;
// for compose services it follows the same "project@service" convention used
// by Container.GetName() so the event handler and doUpdate() agree on the
// identity of a service.
func (a *App) serviceNameFromEventAttrs(attr map[string]string) string {
	project := attr["com.docker.compose.project"]
	service := attr["com.docker.compose.service"]
	if project != "" && service != "" {
		if a.config.UseModuleNameForService {
			if workingDir := attr["com.docker.compose.project.working_dir"]; workingDir != "" {
				if moduleName := container.ReadModuleName(workingDir); moduleName != "" {
					project = moduleName
				}
			}
		}
		return project + "@" + service
	}
	return attr["name"]
}

func (a *App) Monitor(ctx context.Context, filterOptions container.FilterOptions) error {
	evtCh, errCh := a.ContainerClient.MonitorEvents(ctx)

	// Update after subscribing to the events but before reacting to them
	if err := a.Update(filterOptions); err != nil {
		slog.Warn("Error updating container state.", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping engine monitor")
			return ctx.Err()
		case evt := <-evtCh:
			slog.Info("Received event.", "value", evt)
			switch evt.Type {
			case events.ContainerEventType:
				// Note: health checks will run command inside the containers periodically, and this results
				// in periodic exec_die events, however these events are interesting (as we're already listening to the health status transitions)
				// Check if the exec dying was related to the container's main exec or not.
				// Log event on debug level so it does not spam the logs
				if _, isRelatedToExec := evt.Actor.Attributes["execID"]; isRelatedToExec {
					// Just check for the prefix, as some events will have additional context appended to the status
					if strings.HasPrefix(string(evt.Action), "exec_") {
						slog.Debug("Ignoring event.", "value", evt)
						continue
					}
				}

				payload := make(map[string]any)
				if action, ok := ContainerEventText[evt.Action]; ok {
					props := getEventAttributes(evt.Actor.Attributes, "name", "image", "com.docker.compose.project")
					name := props[0]
					image := props[1]
					project := props[2]
					if name != "" && image != "" {
						if project != "" {
							payload["text"] = fmt.Sprintf("%s %s. project=%s, name=%s, image=%s", "container", action, project, name, image)
						} else {
							payload["text"] = fmt.Sprintf("%s %s. name=%s, image=%s", "container", action, name, image)
						}
					} else {
						payload["text"] = fmt.Sprintf("%s %s", "container", action)
					}
					payload["containerID"] = evt.Actor.ID
					payload["attributes"] = evt.Actor.Attributes
				}

				switch evt.Action {
				case events.ActionExecDie, events.ActionCreate, events.ActionStart, events.ActionStop, events.ActionPause, events.ActionUnPause, events.ActionHealthStatusHealthy, events.ActionHealthStatusUnhealthy:
					// When the container becomes healthy, reset the restart baseline
					// to the current count so that future crash loops are measured
					// from this healthy state.
					if evt.Action == events.ActionHealthStatusHealthy {
						serviceName := a.serviceNameFromEventAttrs(evt.Actor.Attributes)
						if serviceName != "" {
							if rc, err := a.ContainerClient.GetRestartCount(context.Background(), evt.Actor.ID); err == nil {
								a.restartBaselineMu.Lock()
								a.restartBaseline[serviceName] = rc
								a.restartBaselineMu.Unlock()
							}
							a.clearCrashLoopAlarm(serviceName)
							a.eventLimiter.Remove(serviceName)
						}
					}
					// On each daemon-initiated restart the Docker daemon increments
					// RestartCount before firing the start event, so by the time we
					// inspect here the count is already current.
					if evt.Action == events.ActionStart {
						serviceName := a.serviceNameFromEventAttrs(evt.Actor.Attributes)
						if serviceName != "" {
							if rc, err := a.ContainerClient.GetRestartCount(context.Background(), evt.Actor.ID); err == nil && rc > 0 {
								a.restartBaselineMu.Lock()
								baseline, exists := a.restartBaseline[serviceName]
								if !exists {
									// First time seeing this container: treat the current
									// count as the baseline so pre-existing restarts
									// (before the plugin started) are not counted.
									a.restartBaseline[serviceName] = rc
									a.restartBaselineMu.Unlock()
								} else {
									delta := rc - baseline
									a.restartBaselineMu.Unlock()
									if delta >= a.config.CrashLoopThreshold {
										slog.Warn("Crash loop detected.", "container", serviceName, "restartCount", rc, "baseline", baseline, "delta", delta)
										a.publishCrashLoopAlarm(serviceName, rc)
									}
								}
							}
						}
					}
					// Enqueue a debounced scoped update for this container. The 2-second
					// quiet period replaces the old 500ms sleep and also coalesces any
					// burst of events (e.g. rapid start/stop cycles) into a single call.
					a.debouncer.Enqueue(NewUpdateAllAction(container.FilterOptions{
						IDs: []string{evt.Actor.ID},
						// Preserve global filter options
						Names:            filterOptions.Names,
						Labels:           filterOptions.Labels,
						Types:            filterOptions.Types,
						ExcludeNames:     filterOptions.ExcludeNames,
						ExcludeWithLabel: filterOptions.ExcludeWithLabel,
					}))
				case events.ActionDestroy, events.ActionRemove, events.ActionDie:
					slog.Info("Container removed/destroyed", "container", evt.Actor.ID, "attributes", evt.Actor.Attributes)
					serviceName := a.serviceNameFromEventAttrs(evt.Actor.Attributes)
					if serviceName != "" {
						switch evt.Action {
						case events.ActionDestroy, events.ActionRemove:
							// Container was permanently removed — clear baseline, alarm, and rate-limit state.
							a.restartBaselineMu.Lock()
							delete(a.restartBaseline, serviceName)
							a.restartBaselineMu.Unlock()
							a.clearCrashLoopAlarm(serviceName)
							a.eventLimiter.Remove(serviceName)
						}
					}
					// TODO: Trigger a removal instead of checking the whole state
					// Lookup container name by container id (from the entity store) as lookup by name won't work for container-groups
					a.debouncer.Enqueue(NewUpdateAllAction(filterOptions))
				}

				if a.config.EnableEngineEvents && len(payload) > 0 {
					serviceName := a.serviceNameFromEventAttrs(evt.Actor.Attributes)
					key := serviceName + "/" + string(evt.Action)

					// Determine whether a crash loop is active for this container.
					a.crashLoopAlarmsMu.Lock()
					_, inCrashLoop := a.crashLoopAlarms[serviceName]
					a.crashLoopAlarmsMu.Unlock()

					switch {
					case inCrashLoop:
						// Suppress all events for a crash-looping container to
						// avoid saturating the broker. The alarm already signals
						// the operator.
						slog.Debug("Suppressing engine event for crash-looping container.",
							"container", serviceName, "action", evt.Action)
					case !a.eventLimiter.Allow(key):
						// Rate-limit: too many events for this (container,
						// action) pair within the window.
						slog.Debug("Rate-limiting engine event.",
							"container", serviceName, "action", evt.Action)
					default:
						if err := a.client.Publish(tedge.GetTopic(a.client.Target, "e", string(evt.Action)), 1, false, mustMarshalJSON(payload)); err != nil {
							slog.Warn("Failed to publish container event.", "err", err)
						}
					}
				}
			}
		case err := <-errCh:
			if errors.Is(err, io.EOF) {
				slog.Info("No more events")
			} else {
				slog.Warn("Received error.", "value", err)
			}
			return err
		}
	}
}

// publishCrashLoopAlarm raises a CRITICAL alarm on the container's service
// topic when a crash loop is detected. Duplicate alarms for the same
// container are suppressed until the alarm is cleared.
//
// Publishing is done asynchronously so that Monitor's event loop is never
// blocked.  Before sending the alarm the container's thin-edge entity is
// registered (idempotent), which ensures the mapper can route the alarm to
// Cumulocity even when the container dies faster than the debounced
// doUpdate() manages to register it.
func (a *App) publishCrashLoopAlarm(name string, count int) {
	a.crashLoopAlarmsMu.Lock()
	_, alreadyRaised := a.crashLoopAlarms[name]
	if !alreadyRaised {
		a.crashLoopAlarms[name] = struct{}{}
	}
	a.crashLoopAlarmsMu.Unlock()

	if alreadyRaised {
		return
	}

	go func() {
		target := a.Device.Service(name)
		topic := tedge.GetTopic(*target, "a", "ContainerCrashLoop")
		payload := mustMarshalJSON(map[string]any{
			"severity": "CRITICAL",
			"text":     fmt.Sprintf("Container is in a crash loop (%d daemon-initiated restarts since last healthy).", count),
			"time":     time.Now().UTC().Format(time.RFC3339),
		})

		// Ensure the entity is registered before publishing the alarm.
		// A crash-looping container may die before the 2-second debounced
		// doUpdate() fires, leaving the entity unknown to the mapper.
		// Infer the service type from the name: "project@service" means
		// container-group, anything else is a plain container.
		entityType := container.ContainerType
		if strings.Contains(name, "@") {
			entityType = container.ContainerGroupType
		}
		if _, err := a.client.TedgeAPI.CreateEntity(context.Background(), tedge.Entity{
			TedgeType:     tedge.EntityTypeService,
			TedgeTopicID:  target.TopicID,
			Name:          name,
			Type:          entityType,
			TedgeParentID: a.client.Parent.TopicID,
		}); err != nil {
			slog.Warn("Could not pre-register entity for crash-loop alarm.", "container", name, "err", err)
		}

		// Mark the service status as "down" immediately so the operator can
		// see the crash loop in the service list without waiting for the
		// debounced doUpdate() to fire. The next doUpdate() will overwrite
		// this with the real container status (e.g. "up" once fixed).
		healthTopic := tedge.GetHealthTopic(*target)
		healthPayload := mustMarshalJSON(map[string]any{
			"status": tedge.StatusDown,
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
		if err := a.client.Publish(healthTopic, 1, true, healthPayload); err != nil {
			slog.Warn("Could not set crash-loop container status to down.", "container", name, "err", err)
		}

		slog.Warn("Publishing crash-loop alarm.", "container", name, "topic", topic, "restarts", count)
		for attempt := 1; attempt <= 5; attempt++ {
			// Stop retrying if the alarm was cleared (container recovered/removed).
			a.crashLoopAlarmsMu.Lock()
			_, stillActive := a.crashLoopAlarms[name]
			a.crashLoopAlarmsMu.Unlock()
			if !stillActive {
				return
			}

			if err := a.client.Publish(topic, 1, true, payload); err != nil {
				slog.Warn("Failed to publish crash-loop alarm, retrying.",
					"err", err, "attempt", attempt, "container", name)
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			slog.Warn("Crash-loop alarm published.", "container", name, "restarts", count)
			return
		}

		// All retries exhausted — un-mark so the alarm can be re-raised on
		// the next threshold crossing once the broker recovers.
		slog.Warn("Gave up publishing crash-loop alarm after retries.", "container", name)
		a.crashLoopAlarmsMu.Lock()
		delete(a.crashLoopAlarms, name)
		a.crashLoopAlarmsMu.Unlock()
	}()
}

// clearCrashLoopAlarm clears any active crash-loop alarm for the named
// container. It is a no-op when no alarm has been raised.
func (a *App) clearCrashLoopAlarm(name string) {
	a.crashLoopAlarmsMu.Lock()
	_, had := a.crashLoopAlarms[name]
	delete(a.crashLoopAlarms, name)
	a.crashLoopAlarmsMu.Unlock()

	if !had {
		return
	}

	target := a.Device.Service(name)
	topic := tedge.GetTopic(*target, "a", "ContainerCrashLoop")
	slog.Info("Clearing crash-loop alarm.", "container", name, "topic", topic)
	if err := a.client.Publish(topic, 1, true, ""); err != nil {
		slog.Warn("Failed to clear crash-loop alarm.", "err", err)
	}
}

func (a *App) updateMetrics(items []container.TedgeContainer) error {
	totalWorkers := 5
	numJobs := len(items)
	jobs := make(chan container.TedgeContainer, numJobs)
	results := make(chan error, numJobs)

	doWork := func(jobs <-chan container.TedgeContainer, results chan<- error) {
		for j := range jobs {
			var jobErr error
			// TODO: Check if container exists, if not then do nothing
			target := a.Device.Service(j.Name)
			if _, entityErr := a.client.TedgeAPI.GetEntity(context.Background(), *target); entityErr != nil {
				slog.Info("Entity has not be registered yet, skipping metric for it.", "topic-id", target.TopicID)
				results <- jobErr
				continue
			}

			stats, jobErr := a.ContainerClient.GetStats(context.Background(), j.Container.Id)
			if jobErr == nil {
				topic := tedge.GetTopic(*target, "m", "resource_usage")
				payload, err := json.Marshal(stats)
				if err == nil {
					slog.Info("Publish container stats.", "topic", topic, "payload", payload)
					jobErr = a.client.Publish(topic, 1, false, payload)
				}
			}
			results <- jobErr
		}
	}

	for w := 1; w <= totalWorkers; w++ {
		go doWork(jobs, results)
	}

	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	jobErrors := make([]error, 0)
	for a := 1; a <= numJobs; a++ {
		err := <-results
		jobErrors = append(jobErrors, err)
		if err != nil {
			slog.Warn("Failed to update metrics.", "err", err)
		}
	}
	return errors.Join(jobErrors...)
}

// applyServiceNamePolicy enforces the UseModuleNameForService setting on a
// list of containers. When UseModuleNameForService is false the ModuleName
// stored in the version file is ignored so that the compose project name
// (from the Docker label) is used as the service name instead.
func (a *App) applyServiceNamePolicy(items []container.TedgeContainer) []container.TedgeContainer {
	if a.config.UseModuleNameForService {
		return items
	}
	for i := range items {
		if items[i].Container.ModuleName != "" {
			items[i].Container.ModuleName = ""
			items[i].Name = items[i].Container.GetName()
		}
	}
	return items
}

func (a *App) doUpdate(filterOptions container.FilterOptions) error {
	tedgeClient := a.client
	entities, err := tedgeClient.GetEntities()
	if err != nil {
		return err
	}

	// Don't remove stale services when doing client side filtering
	// as there is no clean way to tell
	removeStaleServices := filterOptions.IsEmpty()

	// Record all registered services
	// The entities store lookup is keyed by the topic_id (not topic!)
	existingServices := make(map[string]struct{})
	for k, v := range entities {
		if v.Type == container.ContainerType || v.Type == container.ContainerGroupType {
			existingServices[k] = struct{}{}
		}
	}
	slog.Info("Found entities.", "total", len(entities))
	for key, value := range entities {
		slog.Debug("Entity store.", "key", key, "value", value)
	}

	slog.Info("Reading containers")
	items, err := a.ContainerClient.List(context.Background(), filterOptions)
	if err != nil {
		return err
	}
	items = a.applyServiceNamePolicy(items)

	// Register devices
	slog.Info("Registering containers")
	for _, item := range items {
		target := a.Device.Service(item.Name)
		delete(existingServices, target.TopicID)

		// Register using HTTP API
		entity := tedge.Entity{
			TedgeType:     tedge.EntityTypeService,
			TedgeTopicID:  target.TopicID,
			Name:          item.Name,
			Type:          item.ServiceType,
			TedgeParentID: tedgeClient.Parent.TopicID,
		}
		resp, err := tedgeClient.TedgeAPI.CreateEntity(context.Background(), entity)

		if err == nil {
			slog.Info("Registered container.", "topic", target.Topic(), "url", resp.RawResponse.Request.URL.String(), "status_code", resp.RawResponse.Status)
		} else {
			slog.Error("Failed to register container.", "topic", target.Topic(), "err", err)
		}

		// Manually add to entities store for re-use later without having to fetch a new list of entities
		entities[entity.TedgeTopicID] = entity
	}

	// update digital twin information
	slog.Info("Updating digital twin information")
	for _, item := range items {
		target := a.Device.Service(item.Name)

		// Create status
		_, err := tedgeClient.TedgeAPI.UpdateTwin(
			context.Background(),
			tedge.Entity{
				TedgeTopicID: target.TopicID,
			},
			"container",
			item.Container,
		)
		if err != nil {
			slog.Error("Could not publish container status", "err", err)
		}
	}

	// Publish health messages last so the health status always wins over any
	// implicit status that the C8Y mapper may emit when processing the entity
	// registration or twin update messages (which could otherwise race and
	// override an explicit "down" status with "up").
	for _, item := range items {
		target := a.Device.Service(item.Name)

		// If this container is in a crash loop, report it as down regardless of
		// what Docker currently reports — during a crash loop the container is
		// transiently running between restarts when the scan fires.
		status := item.Status
		a.crashLoopAlarmsMu.Lock()
		_, inCrashLoop := a.crashLoopAlarms[item.Name]
		a.crashLoopAlarmsMu.Unlock()
		if inCrashLoop {
			status = tedge.StatusDown
		}

		payload := map[string]any{
			"status": status,
			"time":   item.Time,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			slog.Warn("Could not marshal registration message", "err", err)
			continue
		}
		topic := tedge.GetHealthTopic(*target)
		slog.Info("Publishing container health status", "topic", topic, "payload", b)
		if err := tedgeClient.Publish(topic, 1, true, b); err != nil {
			slog.Error("Failed to update health status", "target", topic, "err", err)
		}
	}

	// Delete stale services — cloud first, then thin-edge.
	//
	// Cloud deletion is attempted before deregistering from thin-edge. If it
	// fails (e.g. the local proxy is down), deregistration is skipped so the
	// entity remains in the thin-edge entity store. The next doUpdate() will
	// detect it as stale again and retry — including after a process restart,
	// which is why an in-memory pending-deletions queue is insufficient.
	if removeStaleServices {
		slog.Info("Checking for any stale services")
		for staleTopicID := range existingServices {
			slog.Info("Removing stale service.", "topic-id", staleTopicID)
			target := tedge.NewTarget(a.Device.RootPrefix, staleTopicID)

			// Attempt cloud deletion first. On failure, leave the entity
			// registered in thin-edge so it is re-detected on the next run.
			if a.config.DeleteFromCloud {
				target.CloudIdentity = tedgeClient.Target.CloudIdentity
				if target.CloudIdentity != "" {
					slog.Info("Removing service from the cloud", "topic", target.Topic())
					if _, err := tedgeClient.DeleteCumulocityManagedObject(*target); err != nil {
						slog.Warn("Failed to delete managed object, will retry on next update.", "err", err, "topic", target.Topic())
						continue
					}
				}
			}

			if err := tedgeClient.DeregisterEntity(*target); err != nil {
				slog.Warn("Failed to deregister entity.", "err", err)
			}
			delete(entities, staleTopicID)
		}
	}

	// Delete orphaned cloud services
	if a.config.DeleteFromCloud && a.config.DeleteOrphans {
		if err := a.DeleteOrphanedCloudServices(entities); err != nil {
			slog.Warn("Could not delete orphaned cloud services.", "err", err)
		}
	}

	// Update tedge-agent log types
	if err := a.client.SyncLogTypes(tedgeClient.Target); err != nil {
		slog.Warn("Failed to send tedge-agent sync request to update the log types", "err", err)
	}

	return nil
}
