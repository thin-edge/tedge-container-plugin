package c8y

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/obeattie/ohmyglob"
	"github.com/tidwall/gjson"
	"golang.org/x/net/publicsuffix"
	tomb "gopkg.in/tomb.v2"
)

const (
	// VERSION preferred Bayeux version
	VERSION = "1.0"

	// MINIMUM_VERSION supported Bayeux version
	MINIMUM_VERSION = "1.0"

	// MinimumRetryDelay is the minimum retry delay in milliseconds to wait before sending another /meta/connect message
	MinimumRetryDelay int64 = 500
)

const (
	// MaximumRetryInterval is the maximum interval (in seconds) between reconnection attempts
	MaximumRetryInterval int64 = 30

	// MinimumRetryInterval is the minimum interval (in seconds) between reconnection attempts
	MinimumRetryInterval int64 = 5

	// RetryBackoffFactor is the backoff factor applied to the retry interval for every unsuccessful reconnection attempt.
	// i.e. the next retry interval is calculated as follows
	// interval = MinimumRetryInterval
	// interval = Min(MaximumRetryInterval, interval * RetryBackoffFactor)
	RetryBackoffFactor float64 = 1.5
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10
)

// RealtimeClient allows connecting to a Bayeux server and subscribing to channels.
type RealtimeClient struct {
	mtx           sync.RWMutex
	url           *url.URL
	c8yURL        *url.URL
	clientID      string
	tomb          *tomb.Tomb
	messages      chan *Message
	connected     bool
	dialer        *websocket.Dialer
	ws            *websocket.Conn
	extension     interface{}
	tenant        string
	username      string
	password      string
	requestID     uint64
	requestHeader http.Header

	send chan *request

	hub *Hub

	pendingRequests sync.Map
}

// Message is the type delivered to subscribers.
type Message struct {
	Channel      string       `json:"channel"`
	Payload      RealtimeData `json:"data,omitempty"`
	ID           string       `json:"id,omitempty"`
	ClientID     string       `json:"clientId,omitempty"`
	Extension    interface{}  `json:"ext,omitempty"`
	Advice       *advice      `json:"advice,omitempty"`
	Successful   bool         `json:"successful,omitempty"`
	Subscription string       `json:"subscription,omitempty"`
}

// RealtimeData contains the websocket frame data
type RealtimeData struct {
	RealtimeAction string          `json:"realtimeAction,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`

	Item gjson.Result `json:"-"`
}

type subscription struct {
	glob       ohmyglob.Glob
	out        chan<- *Message
	isWildcard bool
	disabled   bool
}

type request struct {
	Channel                  string          `json:"channel"`
	Data                     json.RawMessage `json:"data,omitempty"`
	ID                       string          `json:"id,omitempty"`
	ClientID                 string          `json:"clientId,omitempty"`
	Extension                interface{}     `json:"ext,omitempty"`
	Version                  string          `json:"version,omitempty"`
	MinimumVersion           string          `json:"minimumVersion,omitempty"`
	SupportedConnectionTypes []string        `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string          `json:"connectionType,omitempty"`
	Subscription             string          `json:"subscription,omitempty"`
	Advice                   *advice         `json:"advice,omitempty"`
}

type advice struct {
	Reconnect string `json:"reconnect,omitempty"`
	Timeout   int64  `json:"timeout"` // don't use omitempty, otherwise timeout: 0 will be removed
	Interval  int64  `json:"interval,omitempty"`
}

// MetaMessage Bayeux message
type MetaMessage struct {
	Message
	Version                  string   `json:"version,omitempty"`
	MinimumVersion           string   `json:"minimumVersion,omitempty"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string   `json:"connectionType,omitempty"`
	Timestamp                string   `json:"timestamp,omitempty"`
	Successful               bool     `json:"successful"`
	Subscription             string   `json:"subscription,omitempty"`
	Error                    string   `json:"error,omitempty"`
	Advice                   *advice  `json:"advice,omitempty"`
}

type c8yExtensionMessage struct {
	ComCumulocityAuthn comCumulocityAuthn `json:"com.cumulocity.authn"`
}

type comCumulocityAuthn struct {
	Token     string `json:"token,omitempty"`
	XSRFToken string `json:"xsrfToken,omitempty"`
}

func getC8yExtension(tenant, username, password string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			// Always use the tenant name as prefix in the c8y username!!! This ensures you connect to the correct tenant!
			Token: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s/%s:%s", tenant, username, password))),
		},
	}
}

func getC8yExtensionFromToken(token string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			Token: token,
		},
	}
}

func getC8yExtensionFromXSRFToken(token string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			// Always use the tenant name as prefix in the c8y username!!! This ensures you connect to the correct tenant!
			XSRFToken: token,
		},
	}
}

func getRealtimeURL(host string) *url.URL {
	c8yHost, _ := url.Parse(host)

	if c8yHost.Scheme == "http" {
		c8yHost.Scheme = "ws"
	} else {
		c8yHost.Scheme = "wss"
	}

	return c8yHost.ResolveReference(&url.URL{Path: "cep/realtime"})
}

// NewRealtimeClient initializes a new Bayeux client. By default `http.DefaultClient`
// is used for HTTP connections.
func NewRealtimeClient(host string, wsDialer *websocket.Dialer, tenant, username, password string) *RealtimeClient {
	if wsDialer == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  10 * time.Second,
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Convert url to a websocket
	websocketURL := getRealtimeURL(host)
	c8yURL, _ := url.Parse(host)

	client := &RealtimeClient{
		url:       websocketURL,
		dialer:    wsDialer,
		messages:  make(chan *Message, 100),
		extension: getC8yExtension(tenant, username, password),

		c8yURL:   c8yURL,
		tenant:   tenant,
		username: username,
		password: password,

		send: make(chan *request),

		hub: NewHub(),
	}

	go client.hub.Run()
	go client.writeHandler()
	return client
}

// SetRequestHeader sets the header to use when establishing the realtime connection.
func (c *RealtimeClient) SetRequestHeader(header http.Header) {
	c.requestHeader = header
}

// SetCookies sets the cookies used for outgoing requests
func (c *RealtimeClient) SetCookies(cookies []*http.Cookie) error {
	if c.dialer == nil {
		return fmt.Errorf("dialer is nil")
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}
	jar.SetCookies(c.c8yURL, cookies)
	c.dialer.Jar = jar
	return nil
}

// SetXSRFToken set the token required for authentication via OAUTH
func (c *RealtimeClient) SetXSRFToken(token string) {
	c.extension = getC8yExtensionFromXSRFToken(token)
}

// SetBearerToken set the token required for authentication via OAUTH
func (c *RealtimeClient) SetBearerToken(token string) {
	c.extension = getC8yExtensionFromToken(token)
}

// TenantName returns the tenant name used in the client
func (c *RealtimeClient) TenantName() string {
	return c.tenant
}

// Connect performs a handshake with the server and will repeatedly initiate a
// websocket connection until `Close` is called on the client.
func (c *RealtimeClient) Connect() error {
	if !c.IsConnected() {

		err := <-c.connect()
		if err != nil {
			return err
		}

		err = <-c.getAdvice()
		if err != nil {
			return err
		}
	}
	return nil
}

// IsConnected returns true if the websocket is connected
func (c *RealtimeClient) IsConnected() bool {
	c.mtx.RLock()
	isConnected := c.connected
	c.mtx.RUnlock()
	return isConnected
}

// Close notifies the Bayeux server of the intent to disconnect and terminates
// the background polling loop.
func (c *RealtimeClient) Close() error {
	if err := c.disconnect(); err != nil {
		Logger.Infof("Failed to disconnect. %s", err)
	}
	Logger.Infof("Killing go routine")
	c.tomb.Killf("Close")
	return nil
}

// Disconnect sends a disconnect signal to the server and closes the websocket
func (c *RealtimeClient) Disconnect() error {
	return c.disconnect()
}

func (c *RealtimeClient) disconnect() error {
	message := &request{
		ID:       c.nextMessageID(),
		Channel:  "/meta/disconnect",
		ClientID: c.clientID,
	}

	// Change to disconnected state, as the server will not send a reply upon receiving the /meta/disconnect command
	c.mtx.Lock()
	c.connected = false
	c.mtx.Unlock()
	c.send <- message

	return nil
}

func (c *RealtimeClient) createWebsocket() error {
	Logger.Infof("Establishing connection to %s", c.url.String())
	ws, _, err := c.dialer.Dial(c.url.String(), c.requestHeader)

	if err != nil {
		return err
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.ws = ws
	return nil
}

func (c *RealtimeClient) reconnect() error {
	connected := false

	c.mtx.Lock()
	if c.tomb != nil {
		c.tomb.Kill(errors.New("websocket died"))
	}
	c.tomb = nil
	c.connected = false
	c.mtx.Unlock()

	// Remove all pending requests
	c.pendingRequests.Range(func(key, value interface{}) bool {
		c.pendingRequests.Delete(key)
		return true
	})

	interval := MinimumRetryInterval

	for !connected {
		Logger.Infof("Retrying in %ds", interval)
		<-time.After(time.Duration(interval) * time.Second)
		c.ws.Close()
		err := c.createWebsocket()

		if err != nil {
			interval = int64(math.Min(float64(MaximumRetryInterval), RetryBackoffFactor*float64(interval)))
			continue
		}

		if err := c.Connect(); err != nil {
			Logger.Infof("Failed to get advice from server. %s", err)
		} else {
			connected = true
		}
	}

	Logger.Info("Established connection, any subscriptions will be also be resubmitted")

	c.reactivateSubscriptions()
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *RealtimeClient) connect() chan error {
	if c.dialer == nil {
		panic("Missing dialer for realtime client")
	}
	Logger.Infof("Establishing connection to %s", c.url.String())
	ws, _, err := c.dialer.Dial(c.url.String(), c.requestHeader)

	if err != nil {
		panic(err)
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.ws = ws

	if c.tomb == nil {
		c.tomb = &tomb.Tomb{}
		c.tomb.Go(c.worker)
	}

	return c.handshake()
}

func (c *RealtimeClient) worker() error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})

	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go func() {
		defer close(done)
		for {
			messages := []Message{}

			err := c.ws.ReadJSON(&messages)

			if err != nil {
				Logger.Infof("wc ReadJSON: error=%s, message=%v", err, messages)

				if !c.IsConnected() {
					Logger.Info("Connection has been closed by the client")
					return
				}
				Logger.Info("Handling connection error. You need to reconnect")

				go c.reconnect()
				return
			}

			for _, message := range messages {
				if strings.HasPrefix(message.Channel, "/meta") {
					if messageText, err := json.Marshal(message); err == nil {
						Logger.Infof("ws (recv): %s : %s", message.Channel, messageText)
					}
				}

				switch channelType := message.Channel; channelType {
				case "/meta/handshake":
					if message.Successful {
						c.mtx.Lock()
						c.clientID = message.ClientID
						c.connected = true
						c.mtx.Unlock()
					} else {
						Logger.Fatalf("No clientID present in handshake. Check that the tenant, username and password is correct. Raw Message: %v", message)
					}

				case "/meta/subscribe":
					if message.Successful {
						Logger.Infof("Successfully subscribed to channel %s", message.Subscription)
					} else {
						Logger.Infof("Failed to subscribe to channel %s", message.Subscription)
					}

				case "/meta/unsubscribe":
					if message.Successful {
						Logger.Infof("Successfully unsubscribed to channel %s", message.Subscription)
					}

				case "/meta/connect":
					// https://docs.cometd.org/current/reference/
					wasConnected := c.IsConnected()
					connected := message.Successful

					if message.Advice != nil {
						retryDelay := message.Advice.Interval
						if retryDelay <= 0 {
							// Minimum retry delay
							retryDelay = MinimumRetryDelay
						}
						switch message.Advice.Reconnect {
						case "handshake":
							Logger.Infof("Scheduling sending of new handshake to server with %d ms delay", retryDelay)
							time.AfterFunc(time.Duration(retryDelay)*time.Millisecond, func() {
								c.handshake()
							})
						case "retry":
							Logger.Infof("Resending /meta/connect heartbeat with %d ms delay", retryDelay)
							time.AfterFunc(time.Duration(retryDelay)*time.Millisecond, func() {
								c.sendMeta()
							})
						case "none":
							// Do not attempt to retry or send a handshake as it must respect the servers response
							panic("Server indicated that no retry or handshake should be done")
						}
						// Server indicated that a handshake should be sent again
						break
					}

					if !wasConnected && connected {
						// Reconnected
					} else if wasConnected && !connected {
						// Disconnected
						c.disconnect()
					} else if connected {
						// New connection
						c.mtx.Lock()
						c.connected = true
						c.mtx.Unlock()

						go c.sendMeta()
					}

				case "/meta/disconnect":
					if message.Successful {
						Logger.Infof("Successfully disconnected with server")
					}

				default:
					// Data package received
					message.Payload.Item = gjson.ParseBytes(message.Payload.Data)
					c.hub.broadcast <- &message
				}

				// remove the message from the queue
				if message.ID != "" {
					Logger.Infof("Removing message from pending requests: %s", message.ID)
					c.pendingRequests.Delete(message.ID)
					c.logRemainingResponses()
				}
			}
		}
	}()

	for {
		defer c.ws.Close()
		select {
		case <-c.tomb.Dying():
			return nil

		case <-interrupt:
			Logger.Info("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			if err := c.Disconnect(); err != nil {
				Logger.Info("Failed to send disconnect to server:", err)
				return err
			}

			return fmt.Errorf("stopping websocket")
		}
	}
}

func (c *RealtimeClient) handshake() chan error {
	message := &request{
		ID:                       c.nextMessageID(),
		Channel:                  "/meta/handshake",
		Version:                  VERSION,
		MinimumVersion:           MINIMUM_VERSION,
		SupportedConnectionTypes: []string{"websocket", "long-polling"},
		Extension:                c.extension,
		Advice: &advice{
			Interval:  0,
			Timeout:   60000,
			Reconnect: "retry",
		},
	}

	c.send <- message
	return c.WaitForMessage(message.ID)
}

func (c *RealtimeClient) sendMeta() error {
	if c.ws == nil {
		return fmt.Errorf("websocket is nil")
	}
	message := &request{
		ID:             c.nextMessageID(),
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	c.send <- message
	return nil
}

func (c *RealtimeClient) getAdvice() chan error {
	clientID := c.clientID
	message := &request{
		ID:             c.nextMessageID(),
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       clientID,
		Advice: &advice{
			Timeout: 0,
		},
	}

	c.send <- message
	return c.WaitForMessage(message.ID)
}

func getRealtimeID(id ...string) string {
	if len(id) > 0 {
		return id[0]
	}
	return "*"
}

// RealtimeAlarms subscribes to events on alarms objects from the CEP realtime engine
func RealtimeAlarms(id ...string) string {
	return "/alarms/" + getRealtimeID(id...)
}

// RealtimeAlarmsWithChildren subscribes to events on alarms (including children) objects from the CEP realtime engine
func RealtimeAlarmsWithChildren(id ...string) string {
	return "/alarmsWithChildren/" + getRealtimeID(id...)
}

// RealtimeEvents subscribes to events on event objects from the CEP realtime engine
func RealtimeEvents(id ...string) string {
	return "/events/" + getRealtimeID(id...)
}

// RealtimeManagedObjects subscribes to events on managed objects from the CEP realtime engine
func RealtimeManagedObjects(id ...string) string {
	return "/managedobjects/" + getRealtimeID(id...)
}

// RealtimeMeasurements subscribes to events on measurement objects from the CEP realtime engine
func RealtimeMeasurements(id ...string) string {
	return "/measurements/" + getRealtimeID(id...)
}

// RealtimeOperations subscribes to events on operations objects from the CEP realtime engine
func RealtimeOperations(id ...string) string {
	return "/operations/" + getRealtimeID(id...)
}

// Subscribe setup a subscription to the given element
func (c *RealtimeClient) Subscribe(pattern string, out chan<- *Message) chan error {
	Logger.Infof("Subscribing to %s", pattern)

	glob, err := ohmyglob.Compile(pattern, nil)
	if err != nil {
		errCh := make(chan error)
		defer func() {
			fmt.Println("Closing channel")
			close(out)
		}()
		errCh <- fmt.Errorf("invalid pattern: %s", err)
		return errCh
	}

	message := &request{
		ID:             c.nextMessageID(),
		Channel:        "/meta/subscribe",
		Subscription:   pattern,
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	c.hub.register <- &subscription{
		glob:       glob,
		out:        out,
		isWildcard: strings.HasSuffix(glob.String(), "*"),
		disabled:   false,
	}

	c.send <- message

	return c.WaitForMessage(message.ID)
}

// reactivateSubscriptions sends subscription messages to the Bayeux server for each active subscription currently set in the client
func (c *RealtimeClient) reactivateSubscriptions() {
	ids := []string{}

	for _, subscription := range c.hub.GetActiveChannels() {
		message := &request{
			ID:             c.nextMessageID(),
			Channel:        "/meta/subscribe",
			Subscription:   subscription,
			ConnectionType: "websocket",
			ClientID:       c.clientID,
		}

		ids = append(ids, message.ID)
		c.send <- message
	}

	c.WaitForMessages(ids...)
}

// UnsubscribeAll unsubscribes to all of the subscribed channels.
// The channel related to the subscription is left open, and will be
// reused if another call with the same pattern is made to Subscribe()
func (c *RealtimeClient) UnsubscribeAll() chan error {
	ids := []string{}

	subs := c.hub.GetActiveChannels()
	for _, pattern := range subs {
		message := &request{
			ID:           c.nextMessageID(),
			Channel:      "/meta/unsubscribe",
			Subscription: pattern,
			ClientID:     c.clientID,
		}

		ids = append(ids, message.ID)
		c.send <- message
	}

	// Wait for the server to response to the unsubscribe messages
	// only when all of them have been received (or a timeout has occurred) then return
	return c.WaitForMessages(ids...)
}

// Unsubscribe unsubscribe to a given pattern
func (c *RealtimeClient) Unsubscribe(pattern string) chan error {
	Logger.Infof("unsubscribing to %s", pattern)

	message := &request{
		ID:           c.nextMessageID(),
		Channel:      "/meta/unsubscribe",
		Subscription: pattern,
		ClientID:     c.clientID,
	}

	c.hub.unregister <- pattern
	c.send <- message
	return c.WaitForMessage(message.ID)
}

func (c *RealtimeClient) nextMessageID() string {
	return strconv.FormatUint(atomic.AddUint64(&c.requestID, 1), 10)
}

func (c *RealtimeClient) logMessage(r *request) {
	if text, err := json.Marshal(r); err == nil {
		Logger.Infof("ws (send): %s : %s", r.Channel, text)
	} else {
		Logger.Infof("Could not marshal message for sending. %s", err)
	}
}

func (c *RealtimeClient) logRemainingResponses() {
	ids := []string{}
	c.pendingRequests.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	Logger.Infof("Pending messages ids: %s", strings.Join(ids, ","))
}

// WaitForMessages waits for a server response related to the list of message ids
func (c *RealtimeClient) WaitForMessages(ids ...string) chan error {
	wg := new(sync.WaitGroup)
	wg.Add(len(ids))

	errorChannel := make(chan error)
	defer close(errorChannel)

	for _, id := range ids {
		go func(id string) {
			err := <-c.WaitForMessage(id)
			if err != nil {
				errorChannel <- err
			}
			wg.Done()
		}(id)
	}
	wg.Wait()
	return errorChannel
}

// WaitForMessage waits for a message with the corresponding id to be sent by the server
func (c *RealtimeClient) WaitForMessage(ID string) chan error {
	out := make(chan error)

	waitInterval := 10 * time.Second

	Logger.Infof("Waiting for message: id=%s", ID)

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer func() {
			ticker.Stop()
			close(out)
		}()
		timeout := time.After(waitInterval)

		for {
			select {
			case <-ticker.C:
				// Logger.Infof("Checking if ID has been removed")
				if _, exists := c.pendingRequests.Load(ID); !exists {
					Logger.Infof("Received message %s", ID)
					out <- nil
					return
				}
			case <-timeout:
				out <- errors.New("Timeout")
				return
			}
		}
	}()

	return out
}

func (c *RealtimeClient) writeHandler() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()

		// Close
		close(c.send)
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The send channel has been closed
				Logger.Info("Channel has been closed")
				return
			}

			if message.ID == "" {
				message.ID = c.nextMessageID()
			}

			// Store message id
			c.pendingRequests.Store(message.ID, message)

			c.logMessage(message)
			c.logRemainingResponses()

			if c.ws != nil {
				if err := c.ws.WriteJSON([]request{*message}); err != nil {
					Logger.Infof("Failed to send JSON message. %s", err)
				}
			}

		case <-ticker.C:
			// Regularly check if the Websocket is alive by sending a PingMessage to the server
			if c.ws != nil {
				// A websocket ping should initiate a websocket pong response from the server
				// If the pong is not received in the minimum time, then the connection will be reset
				c.ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					Logger.Info("Failed to send ping message to server")
					go c.reconnect()
					break
				}
				Logger.Info("Sent ping successfully")
			}
		}
	}
}
