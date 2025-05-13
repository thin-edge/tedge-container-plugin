package notification2

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/pkg/logger"
	"github.com/tidwall/gjson"
	tomb "gopkg.in/tomb.v2"
)

var Logger logger.Logger

func init() {
	Logger = logger.NewLogger("notifications2")
}

const (
	// MaximumRetryInterval is the maximum interval (in seconds) between reconnection attempts
	MaximumRetryInterval int64 = 300

	// MinimumRetryInterval is the minimum interval (in seconds) between reconnection attempts
	MinimumRetryInterval int64 = 5

	// RetryBackoffFactor is the backoff factor applied to the retry interval for every unsuccessful reconnection attempt.
	// i.e. the next retry interval is calculated as follows
	// interval = MinimumRetryInterval
	// interval = Min(MaximumRetryInterval, interval * RetryBackoffFactor)
	RetryBackoffFactor float64 = 2
)

func SetLogger(log logger.Logger) {
	if log == nil {
		Logger = logger.NewDummyLogger("notification2")
	} else {
		Logger = log
	}
}

type ConnectionOptions struct {
	// Send pings to client with this interval. Must be less than pongWait.
	PingInterval time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration
	Insecure     bool
}

func (o *ConnectionOptions) GetWriteDuration() time.Duration {
	if o.WriteWait == 0 {
		return 60 * time.Second
	}
	return o.WriteWait
}

func (o *ConnectionOptions) GetPongDuration() time.Duration {
	if o.PongWait == 0 {
		return 120 * time.Second
	}
	return o.PongWait
}

func (o *ConnectionOptions) GetPingDuration() time.Duration {
	if o.PingInterval == 0 {
		return 60 * time.Second
	}
	return o.PingInterval
}

// Notification2Client is a client used for the notification2 interface
type Notification2Client struct {
	mtx               sync.RWMutex
	host              string
	url               *url.URL
	tomb              *tomb.Tomb
	messages          chan *Message
	connected         bool
	dialer            *websocket.Dialer
	ws                *websocket.Conn
	Subscription      Subscription
	ConnectionOptions ConnectionOptions

	hub  *Hub
	send chan []byte
}

type Subscription struct {
	Consumer string `json:"consumer,omitempty"`
	Token    string `json:"token,omitempty"`

	TokenRenewal func(string) (string, error)
}

type ClientSubscription struct {
	Pattern  string
	Action   string
	Out      chan<- Message
	Disabled bool
}

// Message is the type delivered to subscribers.
type Message struct {
	Identifier  []byte `json:"identifier"`
	Description []byte `json:"description"`
	Action      []byte `json:"action"`
	Payload     []byte `json:"data,omitempty"`
}

type ActionType string

var ActionTypeCreate ActionType = "CREATE"
var ActionTypeUpdate ActionType = "UPDATE"
var ActionTypeDelete ActionType = "DELETE"

func (m *Message) JSON() gjson.Result {
	return gjson.ParseBytes(m.Payload)
}

func getEndpoint(host string, subscription Subscription) *url.URL {
	fullHost := "wss://" + host
	if index := strings.Index(host, "://"); index > -1 {
		fullHost = "wss" + host[index:]
	}
	tempUrl, err := url.Parse(fullHost)
	if err != nil {
		Logger.Fatalf("Invalid url. %s", err)
	}
	c8yHost := tempUrl.ResolveReference(&url.URL{Path: "notification2/consumer/"})
	c8yHost.RawQuery = "token=" + subscription.Token

	if subscription.Consumer != "" {
		c8yHost.RawQuery += "&consumer=" + subscription.Consumer
	}
	return c8yHost
}

// NewNotification2Client initializes a new notification2 client used to subscribe to realtime notifications from Cumulocity
func NewNotification2Client(host string, wsDialer *websocket.Dialer, subscription Subscription, options ConnectionOptions) *Notification2Client {
	if wsDialer == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  45 * time.Second,
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: options.Insecure,
			},
		}
	}

	client := &Notification2Client{
		host:              host,
		url:               getEndpoint(host, subscription),
		dialer:            wsDialer,
		messages:          make(chan *Message, 100),
		Subscription:      subscription,
		ConnectionOptions: options,

		send: make(chan []byte),

		hub: NewHub(),
	}

	go client.hub.Run()
	go client.writeHandler()
	return client
}

// Connect performs a handshake with the server and will repeatedly initiate a
// websocket connection until `Close` is called on the client.
func (c *Notification2Client) Connect() error {
	if !c.IsConnected() {
		err := c.connect()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Notification2Client) Endpoint() string {
	// TODO: Support hiding of sensitive information (same as the client)
	return c.url.String()
}

func (c *Notification2Client) URL() string {
	return getEndpoint(c.url.Host, c.Subscription).String()
}

// IsConnected returns true if the websocket is connected
func (c *Notification2Client) IsConnected() bool {
	c.mtx.RLock()
	isConnected := c.connected
	c.mtx.RUnlock()
	return isConnected
}

// Close the connection
func (c *Notification2Client) Close() error {
	if err := c.disconnect(); err != nil {
		Logger.Warnf("Failed to disconnect. %s", err)
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.tomb != nil {
		Logger.Debugf("Stopping worker")
		c.tomb.Killf("Close")
		c.tomb = nil
	}
	return nil
}

func (c *Notification2Client) disconnect() error {
	// Change to disconnected state, as the server will not send a reply upon receiving the /meta/disconnect command
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.connected = false
	if c.ws != nil {
		return c.ws.Close()
	}

	// TODO: Add option to unsubscribe on disconnect (e.g. no offline messages required?)
	// Note: If you unsubscribe, then notifications will be ignored when the client is offline
	// if c.ws != nil {
	// 	if err := c.Unsubscribe(); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (c *Notification2Client) createWebsocket() (*websocket.Conn, error) {

	if c.Subscription.TokenRenewal != nil {
		token, err := c.Subscription.TokenRenewal(c.Subscription.Token)
		if err != nil {
			return nil, err
		}
		c.Subscription.Token = token
	}

	Logger.Debugf("Establishing connection to %s", c.Endpoint())
	ws, _, err := c.dialer.Dial(c.URL(), nil)

	if err != nil {
		Logger.Warnf("Failed to establish connection. %s", err)
		return ws, err
	}
	Logger.Debug("Established websocket connection")
	return ws, nil
}

func (c *Notification2Client) reconnect() error {
	c.Close()

	connected := false
	interval := MinimumRetryInterval

	for !connected {
		Logger.Warnf("Retrying in %ds", interval)
		<-time.After(time.Duration(interval) * time.Second)
		err := c.connect()

		if err != nil {
			Logger.Warnf("Failed to connect. %s", err)
			interval = int64(math.Min(float64(MaximumRetryInterval), RetryBackoffFactor*float64(interval)))
			continue
		}

		connected = true
	}

	Logger.Warn("Reestablished connection")
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *Notification2Client) connect() error {
	if c.dialer == nil {
		panic("Missing dialer for realtime client")
	}

	// This may be overkill, but is in place to prevent unexpected errors
	if c.ws != nil {
		if err := c.ws.Close(); err != nil {
			Logger.Infof("Error whilst closing connection before connecting. %s", err)
		}
	}
	ws, err := c.createWebsocket()

	if err != nil {
		return err
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.ws = ws
	if c.tomb == nil {
		c.tomb = &tomb.Tomb{}
		c.tomb.Go(c.worker)
	}
	c.connected = true

	return nil
}

func parseMessage(raw []byte) *Message {
	inHeader := true
	message := &Message{}

	scanner := bufio.NewScanner(bytes.NewReader(raw))

	i := 0
	for scanner.Scan() {
		// Note: .Bytes() does not allocate memory, so you need to allocate the data
		// and copy the data to another variable if you want it to persist
		line := scanner.Bytes()
		if len(line) == 0 {
			inHeader = false
			// empty line is the border between the header and body
			continue
		}
		if inHeader {
			if i == 0 {
				message.Identifier = make([]byte, len(line))
				copy(message.Identifier, line)
			} else if i == 1 {
				message.Description = make([]byte, len(line))
				copy(message.Description, line)
			} else if i == 2 {
				message.Action = make([]byte, len(line))
				copy(message.Action, line)
			}
			// Ignore unknown header indexes
		} else {
			// Copy payload
			message.Payload = make([]byte, len(line))
			copy(message.Payload, line)
			// TODO: Check if a single websocket message can continue multiple messages
			// Stop processing further messages
			break
		}
		i++
	}
	return message
}

func (c *Notification2Client) writeHandler() {
	ticker := time.NewTicker(c.ConnectionOptions.GetPingDuration())

	defer func() {
		ticker.Stop()
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

			if c.ws != nil {
				c.ws.SetWriteDeadline(time.Now().Add(c.ConnectionOptions.GetWriteDuration()))
				if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
					Logger.Warnf("Failed to send message. %s", err)
				}
			}

		case <-ticker.C:
			// Regularly check if the Websocket is alive by sending a PingMessage to the server
			if c.ws != nil {
				// A websocket ping should initiate a websocket pong response from the server
				// If the pong is not received in the minimum time, then the connection will be reset
				c.ws.SetWriteDeadline(time.Now().Add(c.ConnectionOptions.GetWriteDuration()))
				if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					Logger.Warnf("Failed to send ping message to server. %s", err)
					// go c.reconnect()
					continue
				}
				Logger.Debug("Sent ping successfully")
			}
		}
	}
}

func (c *Notification2Client) Register(pattern string, out chan<- Message) {
	Logger.Debugf("Subscribing to %s", pattern)

	c.hub.register <- &ClientSubscription{
		Pattern:  pattern,
		Out:      out,
		Disabled: false,
	}
}

func (c *Notification2Client) SendMessageAck(messageIdentifier []byte) error {
	Logger.Debugf("Sending message ack: %s", messageIdentifier)
	c.send <- messageIdentifier
	return nil
}

func (c *Notification2Client) worker() error {
	done := make(chan struct{})

	c.ws.SetReadDeadline(time.Now().Add(c.ConnectionOptions.GetPongDuration()))
	c.ws.SetPongHandler(func(string) error {
		Logger.Debug("Received pong message")
		c.ws.SetReadDeadline(time.Now().Add(c.ConnectionOptions.GetPongDuration()))
		return nil
	})

	go func() {
		defer close(done)
		for {
			messageType, rawMessage, err := c.ws.ReadMessage()

			if err == nil {
				Logger.Debugf("Received websocket message: type=%d, len=%d", messageType, len(rawMessage))
			}

			if err != nil {
				// Taken from https://github.com/gorilla/websocket/blob/main/examples/chat/client.go
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					Logger.Warnf("unexpected websocket error. %v", err)
				} else {
					Logger.Infof("websocket error. %v", err)
				}
				go c.reconnect()
				break
			}

			switch messageType {
			case websocket.TextMessage:
				Logger.Debugf("Raw notification2 message (len=%d):\n%s", len(rawMessage), rawMessage)
				message := parseMessage(rawMessage)
				Logger.Debugf("message id: %s", message.Identifier)
				Logger.Debugf("message description: %s", message.Description)
				Logger.Debugf("message action: %s", message.Action)
				Logger.Debugf("message payload: %s", message.Payload)
				c.hub.broadcast <- *message

			case websocket.CloseMessage:
				Logger.Warnf("Received close message. %v", rawMessage)
			case websocket.PingMessage:
				Logger.Debugf("Received ping message. %v", rawMessage)

			case websocket.PongMessage:
				Logger.Debugf("Received pong message. %v", rawMessage)
			}
		}
	}()

	defer c.ws.Close()
	<-c.tomb.Dying()
	Logger.Info("Worker is shutting down")
	return nil
}

// Unsubscribe unsubscribe to a given pattern
func (c *Notification2Client) Unsubscribe() error {
	Logger.Info("unsubscribing")
	c.send <- []byte("unsubscribe_subscriber")
	return nil
}
