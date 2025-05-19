package tedge

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
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
	TedgeAPI         *TedgeAPIClient

	Entities map[string]any
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
	HTTPHost string
	HTTPPort uint16

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

		// Configure subscriptions
		subscriptions := make(map[string]byte)
		subscriptions[target.RootPrefix+"/+/+/+/+"] = 1
		subscriptions[GetTopic(*target.Service("+"), "cmd", "health", "check")] = 1
		slog.Info("Subscribing to topics.", "topics", subscriptions)
		tok := c.SubscribeMultiple(subscriptions, nil)
		tok.Wait()

		payload, err := PayloadHealthStatus(map[string]any{}, StatusUp)
		if err != nil {
			return
		}
		topic := GetHealthTopic(target)
		slog.Info("Updating health topic.", "topic", topic)
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
		TedgeAPI:         NewTedgeAPIClient(useCerts, config),
		Entities:         make(map[string]any),
	}

	return c
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
	slog.Info("Publishing MQTT Message.", "topic", topic, "payload", payload, "qos", qos, "retained", retained)
	tok := c.Client.Publish(topic, 1, retained, payload)
	if !tok.WaitTimeout(100 * time.Millisecond) {
		return fmt.Errorf("timed out")
	}
	return tok.Error()
}

// Deregister a thin-edge.io entity
// Clear the status health topic as well as the registration topic
func (c *Client) DeregisterEntity(target Target, retainedTopicPartials ...string) error {
	_, err := c.TedgeAPI.DeleteEntity(context.Background(), target)
	return err
}

// Get the thin-edge.io entities that have already been registered (as retained messages)
func (c *Client) GetEntities() (map[string]Entity, error) {
	resp, err := c.TedgeAPI.GetEntities(context.Background())
	if err != nil {
		return nil, err
	}

	values := make([]Entity, 0)
	if err := resp.Decode(&values); err != nil {
		return nil, err
	}

	data := make(map[string]Entity)
	for _, v := range values {
		resp, err := c.TedgeAPI.GetEntityTwin(context.Background(), Target{TopicID: v.TedgeTopicID})
		if err == nil && resp.StatusCode() == 200 {
			ev := &Entity{}
			if err := resp.Decode(&ev); err == nil {
				v.Type = ev.Type
			}
			data[v.TedgeTopicID] = v
		}
	}

	return data, nil
}

type TedgeAPIClient struct {
	Client  *http.Client
	BaseURL string
}

func NewTedgeAPIClient(useCerts bool, config *ClientConfig) *TedgeAPIClient {
	tr := &http.Transport{
		TLSClientConfig: nil,
	}

	if useCerts {
		tr = &http.Transport{
			TLSClientConfig: NewTLSConfig(config.KeyFile, config.CertFile, config.CAFile),
		}
	}
	client := &http.Client{
		Transport: tr,
	}

	scheme := "http"
	if useCerts {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d", scheme, config.HTTPHost, config.HTTPPort)

	return &TedgeAPIClient{
		Client:  client,
		BaseURL: baseURL,
	}
}

type Entity struct {
	TedgeID       string `json:"@id,omitempty"`
	TedgeType     string `json:"@type,omitempty"`
	TedgeTopicID  string `json:"@topic-id,omitempty"`
	TedgeParentID string `json:"@parent,omitempty"`
	Name          string `json:"name,omitempty"`
	Type          string `json:"type,omitempty"`
}

type Responder func(*Response, error) (*Response, error)

func OkResponder(allowedCodes ...int) Responder {
	return func(r *Response, err error) (*Response, error) {
		if r.IsError() {
			if !slices.Contains(allowedCodes, r.StatusCode()) {
				return r, nil
			}
			return r, fmt.Errorf("invalid api response. status_code=%d", r.StatusCode())
		}
		return r, nil
	}
}

func (c *TedgeAPIClient) Do(req *http.Request, responders ...Responder) (*Response, error) {
	resp, err := c.Client.Do(req)
	wrappedResponse := NewResponse(resp)

	if err == nil {
		for _, responder := range responders {
			wrappedResponse, err = responder(wrappedResponse, err)
			if err != nil {
				break
			}
		}
	}

	return wrappedResponse, err
}

func (c *TedgeAPIClient) GetURL(partials ...string) string {
	parts := make([]string, 0, 1+len(partials))
	parts = append(parts, c.BaseURL)
	parts = append(parts, partials...)
	return strings.Join(parts, "/")
}

func (c *TedgeAPIClient) CreateEntity(ctx context.Context, entity Entity) (*Response, error) {
	b, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	slog.Info("Registering device by http api.", "body", b)
	reqURL := c.GetURL("te/v1/entities")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req, OkResponder(201, 409))
}

func (c *TedgeAPIClient) PatchEntity(ctx context.Context, entity Entity, data any) (*Response, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	slog.Info("Patching entity by http api.", "body", b)
	reqURL := c.GetURL("te/v1/entities", entity.TedgeTopicID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, reqURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req, OkResponder(200))
}

func (c *TedgeAPIClient) UpdateTwin(ctx context.Context, entity Entity, name string, data any) (*Response, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	slog.Info("Updating twin by http api.", "body", b)
	reqURL := c.GetURL("te/v1/entities", entity.TedgeTopicID, "twin", name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req, OkResponder(200))
}

func (c *TedgeAPIClient) DeleteEntity(ctx context.Context, target Target) (*Response, error) {
	reqURL := c.GetURL("te/v1/entities", target.TopicID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req, OkResponder(200, 204))
}

func (c *TedgeAPIClient) GetEntities(ctx context.Context) (*Response, error) {
	reqURL := c.GetURL("te/v1/entities")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req, OkResponder(200))
}

func (c *TedgeAPIClient) GetEntity(ctx context.Context, target Target) (*Response, error) {
	reqURL := c.GetURL("te/v1/entities", target.TopicID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req, OkResponder(200))
}

func (c *TedgeAPIClient) GetEntityTwin(ctx context.Context, target Target, name ...string) (*Response, error) {
	reqURL := c.GetURL("te/v1/entities", target.TopicID, "twin")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req, OkResponder(200))
}

type Response struct {
	RawResponse *http.Response
}

func NewResponse(r *http.Response) *Response {
	return &Response{
		RawResponse: r,
	}
}

func (r *Response) Decode(v any) error {
	defer r.RawResponse.Body.Close()
	return json.NewDecoder(r.RawResponse.Body).Decode(v)
}

// IsSuccess method returns true if HTTP status `code >= 200 and <= 299` otherwise false.
func (r *Response) IsSuccess() bool {
	return r.StatusCode() > 199 && r.StatusCode() < 300
}

// IsError method returns true if HTTP status `code >= 400` otherwise false.
func (r *Response) IsError() bool {
	return r.StatusCode() > 399
}

func (r *Response) StatusCode() int {
	if r.RawResponse == nil {
		return 0
	}
	return r.RawResponse.StatusCode
}
