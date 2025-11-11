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
	updateResults  chan error
	wg             sync.WaitGroup
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

	HTTPHost string
	HTTPPort uint16

	MQTTHost string
	MQTTPort uint16

	CumulocityHost string
	CumulocityPort uint16
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
		updateRequests:  make(chan ActionRequest),
		updateResults:   make(chan error),
		shutdown:        make(chan struct{}),
		wg:              sync.WaitGroup{},
	}

	// Start background task to process requests
	application.wg.Add(1)
	go application.worker()

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
			go func(name string) {
				opts := container.FilterOptions{}
				// If the name matches the current service name, then
				// update all containers
				if name != a.config.ServiceName {
					opts.Names = []string{
						fmt.Sprintf("^%s$", name),
					}
				}
				a.updateRequests <- NewUpdateAllAction(opts)
			}(parts[4])
		}
	})

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

func (a *App) worker() {
	defer a.wg.Done()
	for {
		select {
		case opts := <-a.updateRequests:

			switch opts.Action {
			case ActionUpdateAll:
				slog.Info("Processing update request")
				err := a.doUpdate(opts.Options.(container.FilterOptions))
				// Don't block when publishing results
				go func() {
					a.updateResults <- err
				}()
			case ActionUpdateMetrics:
				items, err := a.ContainerClient.List(context.Background(), opts.Options.(container.FilterOptions))
				if err != nil {
					slog.Warn("Could not get container list.", "err", err)
				} else {
					if updateErr := a.updateMetrics(items); updateErr != nil {
						slog.Warn("Error updating metrics.", "err", updateErr)
					}
				}
			}

		case <-a.shutdown:
			slog.Info("Stopping background task")
			return
		}
	}
}

func (a *App) Update(filterOptions container.FilterOptions) error {
	a.updateRequests <- NewUpdateAllAction(filterOptions)
	err := <-a.updateResults
	return err
}

func (a *App) UpdateMetrics(filterOptions container.FilterOptions) error {
	a.updateRequests <- NewUpdateMetricsAction(filterOptions)
	err := <-a.updateResults
	return err
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
					go func(evt events.Message) {
						// Delay before trigger update to allow the service status to be updated
						time.Sleep(500 * time.Millisecond)
						if err := a.Update(container.FilterOptions{
							IDs: []string{evt.Actor.ID},

							// Preserve default filter options
							Names:            filterOptions.Names,
							Labels:           filterOptions.Labels,
							Types:            filterOptions.Types,
							ExcludeNames:     filterOptions.ExcludeNames,
							ExcludeWithLabel: filterOptions.ExcludeWithLabel,
						}); err != nil {
							slog.Warn("Error updating container state.", "err", err)
						}
					}(evt)
				case events.ActionDestroy, events.ActionRemove, events.ActionDie:
					slog.Info("Container removed/destroyed", "container", evt.Actor.ID, "attributes", evt.Actor.Attributes)
					// TODO: Trigger a removal instead of checking the whole state
					// Lookup container name by container id (from the entity store) as lookup by name won't work for container-groups
					go func(evt events.Message) {
						// Delay before trigger update to allow the service status to be updated
						time.Sleep(500 * time.Millisecond)
						if err := a.Update(filterOptions); err != nil {
							slog.Warn("Error updating container state.", "err", err)
						}
					}(evt)
				}

				if a.config.EnableEngineEvents {
					if len(payload) > 0 {
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

	// Publish health messages
	for _, item := range items {
		target := a.Device.Service(item.Name)

		payload := map[string]any{
			"status": item.Status,
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

	// Delete removed values, via MQTT and c8y API
	markedForDeletion := make([]tedge.Target, 0)
	if removeStaleServices {
		slog.Info("Checking for any stale services")
		for staleTopicID := range existingServices {
			slog.Info("Removing stale service.", "topic-id", staleTopicID)
			target := tedge.NewTarget(a.Device.RootPrefix, staleTopicID)
			if err := tedgeClient.DeregisterEntity(*target); err != nil {
				slog.Warn("Failed to deregister entity.", "err", err)
			}

			// Update entities map
			delete(entities, staleTopicID)

			// mark targets for deletion from the cloud, but don't delete them yet to give time
			// for thin-edge.io to process the status updates
			markedForDeletion = append(markedForDeletion, *target)
		}

		// Delete cloud
		if len(markedForDeletion) > 0 {
			// Delay before deleting messages
			time.Sleep(500 * time.Millisecond)
			for _, target := range markedForDeletion {
				slog.Info("Removing service from the cloud", "topic", target.Topic())

				// FIXME: How to handle if the device is deregistered locally, but still exists in the cloud?
				// Should it try to reconcile with the cloud to delete orphaned services?
				// Delete service directly from Cumulocity using the local Cumulocity Proxy
				target.CloudIdentity = tedgeClient.Target.CloudIdentity
				if target.CloudIdentity != "" {
					// Delay deleting the value
					if _, err := tedgeClient.DeleteCumulocityManagedObject(target); err != nil {
						slog.Warn("Failed to delete managed object.", "err", err)
					}
				}
			}
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
