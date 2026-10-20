package app

import (
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/thin-edge/tedge-container-plugin/pkg/container"
	"github.com/thin-edge/tedge-container-plugin/pkg/tedge"
)

type fakeToken struct {
	err error
}

func (t *fakeToken) Wait() bool                       { return true }
func (t *fakeToken) WaitTimeout(_ time.Duration) bool { return true }
func (t *fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (t *fakeToken) Error() error { return t.err }

// fakeMQTTClient records the published messages
type fakeMQTTClient struct {
	mqtt.Client
	published map[string]any
	err       error
}

func (c *fakeMQTTClient) Publish(topic string, qos byte, retained bool, payload any) mqtt.Token {
	if c.err == nil {
		c.published[topic] = payload
	}
	return &fakeToken{err: c.err}
}

func newServiceActionsApp(config Config) (*App, *fakeMQTTClient) {
	mqttClient := &fakeMQTTClient{published: make(map[string]any)}
	return &App{
		client:                 &tedge.Client{Client: mqttClient},
		config:                 config,
		serviceActionsDeclared: make(map[string]struct{}),
	}, mqttClient
}

func newServiceItem(name string, id string) (container.TedgeContainer, *tedge.Target) {
	device := tedge.NewTarget("te", "device/main//")
	return container.TedgeContainer{
		Name:      name,
		Container: container.Container{Id: id},
	}, device.Service(name)
}

func Test_DeclareServiceActions(t *testing.T) {
	a, mqttClient := newServiceActionsApp(Config{
		ManageServiceActions: true,
		ServiceActions:       []string{"start", "stop", "restart"},
	})
	item, target := newServiceItem("app1", "abc")

	a.declareServiceActions(item, target)
	assert.Equal(t, map[string]any{
		"te/device/main/service/app1/cmd/start":   "{}",
		"te/device/main/service/app1/cmd/stop":    "{}",
		"te/device/main/service/app1/cmd/restart": "{}",
	}, mqttClient.published)

	// Only declared once per process
	mqttClient.published = make(map[string]any)
	a.declareServiceActions(item, target)
	assert.Empty(t, mqttClient.published)

	// Declared again once the service has been deregistered
	delete(a.serviceActionsDeclared, target.TopicID)
	a.declareServiceActions(item, target)
	assert.Len(t, mqttClient.published, 3)
}

func Test_DeclareServiceActionsRetriesAfterFailedPublish(t *testing.T) {
	a, mqttClient := newServiceActionsApp(Config{
		ManageServiceActions: true,
		ServiceActions:       []string{"restart"},
	})
	item, target := newServiceItem("app1", "abc")

	mqttClient.err = errors.New("timed out")
	a.declareServiceActions(item, target)
	assert.NotContains(t, a.serviceActionsDeclared, target.TopicID)

	mqttClient.err = nil
	a.declareServiceActions(item, target)
	assert.Equal(t, "{}", mqttClient.published["te/device/main/service/app1/cmd/restart"])
	assert.Contains(t, a.serviceActionsDeclared, target.TopicID)
}

func Test_DeclareServiceActionsSkipsSelf(t *testing.T) {
	a, mqttClient := newServiceActionsApp(Config{
		ManageServiceActions: true,
		ServiceActions:       []string{"start", "stop", "restart"},
	})
	a.selfContainerID = "self-id"
	item, target := newServiceItem("tedge", "self-id")

	a.declareServiceActions(item, target)
	assert.Len(t, mqttClient.published, 3)
	for topic, payload := range mqttClient.published {
		assert.Equal(t, "", payload, "topic=%s", topic)
	}
}

func Test_DeclareServiceActionsDisabled(t *testing.T) {
	// Disabled: all supported actions are cleared
	a, mqttClient := newServiceActionsApp(Config{ManageServiceActions: true})
	item, target := newServiceItem("app1", "abc")
	a.declareServiceActions(item, target)
	assert.Len(t, mqttClient.published, 3)
	for topic, payload := range mqttClient.published {
		assert.Equal(t, "", payload, "topic=%s", topic)
	}

	// Not managed (thin-edge.io lacks support): nothing is published
	a, mqttClient = newServiceActionsApp(Config{ManageServiceActions: false, ServiceActions: []string{"start"}})
	a.declareServiceActions(item, target)
	assert.Empty(t, mqttClient.published)
}
