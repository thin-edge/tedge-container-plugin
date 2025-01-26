package tedge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

var StatusUp = "up"
var StatusDown = "down"
var StatusUnknown = "unknown"

func PayloadHealthStatusDown() string {
	return fmt.Sprintf(`{"status":"%s"}`, StatusDown)
}

func PayloadHealthStatus(payload map[string]any, status string) ([]byte, error) {
	payload["status"] = status
	payload["time"] = time.Now().Unix()
	b, err := json.Marshal(payload)
	return b, err
}

func PayloadRegistration(payload map[string]any, name string, entityType string, parent string) ([]byte, error) {
	payload["@type"] = entityType
	payload["name"] = name
	if parent != "" {
		payload["@parent"] = parent
	}
	b, err := json.Marshal(payload)
	return b, err
}

type Client struct {
	Parent           Target
	ServiceName      string
	Client           mqtt.Client
	Target           Target
	CumulocityClient *c8y.Client

	Entities map[string]any
	mutex    sync.RWMutex
}

func fileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	return !errors.Is(error, os.ErrNotExist)
}

func NewTLSConfig(keyFile string, certFile string, caFile string) *tls.Config {
	// Import trusted certificates from CAfile.pem.
	// Alternatively, manually add CA certificates to
	// default openssl CA bundle.
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	pemCerts, err := os.ReadFile(caFile)
	if err == nil {
		rootCAs.AppendCertsFromPEM(pemCerts)
	}

	// Import client certificate/key pair
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(err)
	}

	// Just to print out the client certificate..
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		panic(err)
	}

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: rootCAs,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: false,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}
}

type ClientConfig struct {
	MqttHost string
	MqttPort uint16

	CertFile string
	KeyFile  string
	CAFile   string

	C8yHost string
	C8yPort uint16
}

func CumulocityClientFromConfig(useCerts bool, config *ClientConfig) *c8y.Client {
	var httpClient *http.Client
	c8yURL := fmt.Sprintf("http://%s:%d/c8y", config.C8yHost, config.C8yPort)
	if useCerts {
		tlsconfig := NewTLSConfig(config.KeyFile, config.CertFile, config.CAFile)
		transport := &http.Transport{TLSClientConfig: tlsconfig}
		httpClient = &http.Client{Transport: transport}
		c8yURL = fmt.Sprintf("https://%s:%d/c8y", config.C8yHost, config.C8yPort)
	}
	return c8y.NewClient(httpClient, c8yURL, "", "", "", true)
}

func NewClient(parent Target, target Target, serviceName string, config *ClientConfig) *Client {
	opts := mqtt.NewClientOptions()
	useCerts := fileExists(config.KeyFile) && fileExists(config.CertFile)
	if useCerts && config.MqttPort != 1883 {
		slog.Info("Using client certificates to connect to thin-edge.io services")
		opts.AddBroker(fmt.Sprintf("ssl://%s:%d", config.MqttHost, config.MqttPort))
		tlsconfig := NewTLSConfig(config.KeyFile, config.CertFile, config.CAFile)
		opts.SetTLSConfig(tlsconfig)
	} else {
		opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MqttHost, config.MqttPort))
	}

	opts.SetClientID(serviceName)
	opts.SetClientID(fmt.Sprintf("%s#%s", serviceName, target.Topic()))
	opts.SetCleanSession(true)
	// opts.SetOrderMatters(true)
	opts.SetWill(GetHealthTopic(target), PayloadHealthStatusDown(), 1, true)
	opts.SetAutoReconnect(true)
	opts.SetAutoAckDisabled(false)
	opts.SetResumeSubs(false)
	opts.SetKeepAlive(60 * time.Second)

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		slog.Info("MQTT Client is disconnected.", "err", err)
	})

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		slog.Info("MQTT Client is connected")

		payload, err := PayloadRegistration(map[string]any{}, serviceName, "service", parent.TopicID)
		if err != nil {
			slog.Error("Could not convert payload.", "err", err)
			return
		}
		tok := c.Publish(GetTopicRegistration(target), 1, true, payload)
		<-tok.Done()
		if err := tok.Error(); err != nil {
			slog.Error("Failed to publish registration topic.", "err", err)
			return
		}
		slog.Info("Registered service", "topic", GetTopicRegistration(target))

		// Configure subscriptions
		subscriptions := make(map[string]byte)
		subscriptions[target.RootPrefix+"/+/+/+/+"] = 1
		subscriptions[GetTopic(*target.Service("+"), "cmd", "health", "check")] = 1
		slog.Info("Subscribing to topics.", "topics", subscriptions)
		tok = c.SubscribeMultiple(subscriptions, nil)
		tok.Wait()

		// Delay before publishing health status
		// FIXME: This can be removed once thin-edge.io supports a registration API
		time.Sleep(1000 * time.Millisecond)
		payload, err = PayloadHealthStatus(map[string]any{}, StatusUp)
		if err != nil {
			return
		}
		topic := GetHealthTopic(target)
		tok = c.Publish(topic, 1, true, payload)
		<-tok.Done()
		if err := tok.Error(); err != nil {
			slog.Warn("Failed to publish health message.", "err", err)
			return
		}
		slog.Info("Published health message.", "topic", topic, "payload", payload)
	})

	client := mqtt.NewClient(opts)

	c8yclient := CumulocityClientFromConfig(useCerts, config)
	slog.Info("MQTT Client options.", "clientID", opts.ClientID)

	c := &Client{
		ServiceName:      serviceName,
		Client:           client,
		Parent:           parent,
		Target:           target,
		CumulocityClient: c8yclient,
		Entities:         make(map[string]any),
	}

	registrationTopics := GetTopic(*target.Service("+"))
	slog.Info("Subscribing to registration topics.", "topic", registrationTopics)
	c.Client.AddRoute(GetTopic(*target.Service("+")), func(mqttc mqtt.Client, m mqtt.Message) {
		go c.handleRegistrationMessage(mqttc, m)
	})
	return c
}

func (c *Client) handleRegistrationMessage(_ mqtt.Client, m mqtt.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(m.Payload()) > 0 {
		payload := make(map[string]any)
		if err := json.Unmarshal(m.Payload(), &payload); err != nil {
			slog.Warn("Could not unmarshal registration message", "err", err)
		} else {
			c.Entities[m.Topic()] = payload
		}
	} else {
		slog.Info("Removing entity from store.", "topic", m.Topic())
		delete(c.Entities, m.Topic())
	}
}

// Connect the MQTT client to the thin-edge.io broker
func (c *Client) Connect() error {
	tok := c.Client.Connect()
	if !tok.WaitTimeout(30 * time.Second) {
		panic("Failed to connect to broker")
	}
	<-tok.Done()
	return tok.Error()
}

// Delete a Cumulocity Managed object by External ID
func (c *Client) DeleteCumulocityManagedObject(target Target) (bool, error) {
	slog.Info("Deleting service by external ID.", "name", target.ExternalID())
	extID, resp, err := c.CumulocityClient.Identity.GetExternalID(context.Background(), "c8y_Serial", target.ExternalID())

	if err != nil {
		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	if _, err := c.CumulocityClient.Inventory.Delete(context.Background(), extID.ManagedObject.ID); err != nil {
		slog.Warn("Failed to delete service", "id", extID.ManagedObject.ID, "err", err)
		return false, err
	}
	return true, nil
}

// Publish an MQTT message
func (c *Client) Publish(topic string, qos byte, retained bool, payload any) error {
	tok := c.Client.Publish(topic, 1, retained, payload)
	if !tok.WaitTimeout(100 * time.Millisecond) {
		return fmt.Errorf("timed out")
	}
	return tok.Error()
}

// Deregister a thin-edge.io entity
// Clear the status health topic as well as the registration topic
func (c *Client) DeregisterEntity(target Target, retainedTopicPartials ...string) error {
	delay := 500 * time.Millisecond
	// Clear any additional topics with retained messages before deregistering
	for _, topicPartial := range retainedTopicPartials {
		if err := c.Publish(GetTopic(target, topicPartial), 1, true, ""); err != nil {
			return err
		}
		time.Sleep(delay)
	}

	if err := c.Publish(GetTopic(target, "status", "health"), 1, true, ""); err != nil {
		return err
	}
	time.Sleep(delay)

	if err := c.Publish(GetTopic(target), 1, true, ""); err != nil {
		return err
	}

	return nil
}

// Get the thin-edge.io entities that have already been registered (as retained messages)
func (c *Client) GetEntities() (map[string]any, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Entities, nil
}
