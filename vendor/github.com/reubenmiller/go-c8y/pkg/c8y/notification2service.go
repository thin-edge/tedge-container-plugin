package c8y

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/reubenmiller/go-c8y/pkg/c8y/notification2"
	"github.com/tidwall/gjson"
)

var (
	MinTokenMinutes int64 = 1
)

// Notification2Service manages tokens and subscriptions for the notification2 interface
type Notification2Service service

// EventCollectionOptions todo
type Notification2TokenOptions struct {
	// The token expiration duration
	ExpiresInMinutes int64 `json:"expiresInMinutes,omitempty"`

	// The subscriber name which the client wishes to be identified with
	Subscriber string `json:"subscriber,omitempty"`

	// Default subscriber to use if a token is not provided by the user or an explicit subscriber value
	DefaultSubscriber string `json:"-"`

	// The subscription name. This value must match the same that was used when the subscription was created
	Subscription string `json:"subscription,omitempty"`

	// Subscription is shared by multiple consumers
	Shared bool `json:"shared,omitempty"`
}

func (nt *Notification2TokenOptions) GetDefaultSubscriber() string {
	if nt.DefaultSubscriber != "" {
		return nt.DefaultSubscriber
	}
	return "goc8y"
}

// Notification2Subscription notification subscription object
type Notification2Subscription struct {
	ID                 string                          `json:"id,omitempty"`
	Self               string                          `json:"self,omitempty"`
	Context            string                          `json:"context,omitempty"`
	FragmentsToCopy    []string                        `json:"fragmentsToCopy,omitempty"`
	Source             *Source                         `json:"source,omitempty"`
	Subscription       string                          `json:"subscription,omitempty"`
	SubscriptionFilter Notification2SubscriptionFilter `json:"subscriptionFilter,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// Notification2SubscriptionCollectionOptions collection options
type Notification2SubscriptionCollectionOptions struct {
	Context string `url:"context,omitempty"`
	Source  string `url:"source,omitempty"`

	PaginationOptions
}

// Notification2SubscriptionDeleteOptions options when deleting a subscription by source
type Notification2SubscriptionDeleteOptions struct {
	Context string `url:"context,omitempty"`
	Source  string `url:"source,omitempty"`
}

type Notification2SubscriptionFilter struct {
	Apis       []string `json:"apis,omitempty"`
	TypeFilter string   `json:"typeFilter,omitempty"`
}

// Notification2Token notification2 token which can be used by client to subscribe to notifications
type Notification2Token struct {
	*BaseResponse

	Token string `json:"token"`

	// Allow access to custom fields
	Items []gjson.Result `json:"-"`
}

type Notification2SubscriptionCollection struct {
	*BaseResponse

	Subscriptions []Notification2Subscription `json:"subscriptions"`

	// Allow access to custom fields
	Items []gjson.Result `json:"-"`
}

// UnsubscribeResponse response after unsubscribing a subscriber
type UnsubscribeResponse struct {
	Result string `json:"result,omitempty"`
}

// Get subscription by id
func (s *Notification2Service) GetSubscription(ctx context.Context, ID string) (*Notification2Subscription, *Response, error) {
	data := new(Notification2Subscription)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         "notification2/subscriptions/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// Get collection of subscriptions
func (s *Notification2Service) GetSubscriptions(ctx context.Context, opt *Notification2SubscriptionCollectionOptions) (*Notification2SubscriptionCollection, *Response, error) {
	data := new(Notification2SubscriptionCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         "notification2/subscriptions",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create token
func (s *Notification2Service) CreateToken(ctx context.Context, options Notification2TokenOptions) (*Notification2Token, *Response, error) {
	data := new(Notification2Token)

	// Set a default subscriber if necessary
	if options.Subscriber == "" {
		options.Subscriber = options.GetDefaultSubscriber()
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "notification2/token",
		Body:         options,
		ResponseData: data,
	})
	return data, resp, err
}

// Unsubscribe a notification subscriber using the notification token
func (s *Notification2Service) UnsubscribeSubscriber(ctx context.Context, token string) (*UnsubscribeResponse, *Response, error) {
	data := new(UnsubscribeResponse)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "notification2/unsubscribe?token=" + token,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates properties on an existing event
func (s *Notification2Service) CreateSubscription(ctx context.Context, ID string, subscription Notification2Subscription) (*Event, *Response, error) {
	data := new(Event)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodPost,
		Path:         "notification2/subscriptions",
		Body:         subscription,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete subscription by id
func (s *Notification2Service) DeleteSubscription(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "notification2/subscriptions/" + ID,
	})
}

// DeleteSubscription removes a subscription by source
func (s *Notification2Service) DeleteSubscriptionBySource(ctx context.Context, opt Notification2SubscriptionDeleteOptions) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "notification2/subscriptions",
		Query:  opt,
	})
}

type Notification2ClientOptions struct {
	Token             string
	Consumer          string
	Options           Notification2TokenOptions
	ConnectionOptions notification2.ConnectionOptions
}

type Notification2TokenClaim struct {
	Subscriber string `json:"sub,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Shared     string `json:"shared,omitempty"`
	jwt.RegisteredClaims
}

func (c *Notification2TokenClaim) IsShared() bool {
	return strings.EqualFold(c.Shared, "true")
}

func (c *Notification2TokenClaim) Tenant() string {
	index := strings.Index(c.Topic, "/")
	if index == -1 {
		return ""
	}
	return c.Topic[0:index]
}

func (c *Notification2TokenClaim) Subscription() string {
	index := strings.LastIndex(c.Topic, "/")
	if index == -1 {
		return ""
	}
	return c.Topic[index+1:]
}

func (c *Notification2TokenClaim) HasExpired() bool {
	var v = jwt.NewValidator(jwt.WithLeeway(5 * time.Second))
	err := v.Validate(c)
	return err != nil
}

func (s *Notification2Service) ParseToken(tokenString string) (*Notification2TokenClaim, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token. expected 3 fields")
	}
	raw, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	claim := &Notification2TokenClaim{}
	err = json.Unmarshal(raw, claim)
	return claim, err
}

func (s *Notification2Service) RenewToken(ctx context.Context, opt Notification2ClientOptions) (string, error) {
	isValid := true
	claimMatch := true // default to true in case if the user does not provide any expected information

	subscription := opt.Options.Subscription
	subscriber := opt.Options.Subscriber
	expiresInMinutes := opt.Options.ExpiresInMinutes
	shared := opt.Options.Shared

	if opt.Token != "" {

		claims := Notification2TokenClaim{}
		parser := jwt.NewParser()
		token, _, err := parser.ParseUnverified(opt.Token, &claims)

		if err != nil {
			Logger.Infof("Token is invalid. %s", err)
			isValid = false
		} else if err := jwt.NewValidator(jwt.WithLeeway(5 * time.Second)).Validate(token.Claims); err != nil {
			Logger.Infof("Token is invalid. %s", err)
			isValid = false
		}

		Logger.Infof("Existing token: alg=%s, valid=%v, expired=%v, issuedAt: %v, expiresAt: %v, subscription=%s, subscriber=%s, shared=%v, tenant=%s", token.Method.Alg(), isValid, claims.HasExpired(), claims.IssuedAt, claims.ExpiresAt, claims.Subscription(), claims.Subscriber, claims.IsShared(), claims.Tenant())

		if opt.Options.Subscription != "" {
			if claims.Subscription() != opt.Options.Subscription {
				claimMatch = false
			}
		} else {
			subscription = claims.Subscription()
		}

		if opt.Options.Subscriber != "" {
			if claims.Subscriber != opt.Options.Subscriber {
				claimMatch = false
			}
		} else {
			subscriber = claims.Subscriber
		}

		shared = claims.IsShared()

		if claimMatch && expiresInMinutes == 0 {
			// Reuse the expiration time given in the token
			if claims.ExpiresAt != nil && claims.IssuedAt != nil {
				expiresInMinutes = claims.ExpiresAt.Unix() - claims.IssuedAt.Unix()
			}
		}

		if isValid && claimMatch {
			Logger.Infof("Using existing valid token")
			return opt.Token, nil
		}
		Logger.Infof("Token does not match claim. Invalid information will be ignored in the token")
	}

	if expiresInMinutes < MinTokenMinutes {
		expiresInMinutes = MinTokenMinutes
	}

	Logger.Infof("Creating new token")
	updatedToken, _, err := s.CreateToken(ctx, Notification2TokenOptions{
		ExpiresInMinutes:  expiresInMinutes,
		Subscription:      subscription,
		Subscriber:        subscriber,
		DefaultSubscriber: opt.Options.DefaultSubscriber,
		Shared:            shared,
	})
	if err != nil {
		return "", err
	}
	return updatedToken.Token, nil
}

// Create a notification2 client to subscribe to new options
//
// # Example
//
// ```
//
//	notificationsClient, err := client.Notification2.CreateClient(context.Background(), c8y.Notification2ClientOptions{
//	    Token:    os.Getenv("NOTIFICATION2_TOKEN"),
//	    Consumer: *consumer,
//	    Options: &c8y.Notification2TokenOptions{
//	    	   ExpiresInMinutes: 2,
//	    	   Subscription:     *subscription,
//	    	   Subscriber:       *subscriber,
//	    },
//	})
//
//	if err != nil {
//	    panic(err)
//	}
//
// messagesCh := make(chan notifications2.Message)
// notificationsClient.Register("*", messagesCh)
// signalCh := make(chan os.Signal, 1)
// signal.Notify(signalCh, os.Interrupt)
//
//	for {
//	  select {
//	  case msg := <-messagesCh:
//		      log.Printf("Received message. %s", msg.Payload)
//	       notificationsClient.SendMessageAck(msg.Identifier)
//
//	  case <-signalCh:
//	  	// Enable ctrl-c to stop
//	  	notificationClient.Close()
//	  	return
//	  }
//	}
//
// ```
func (s *Notification2Service) CreateClient(ctx context.Context, opt Notification2ClientOptions) (*notification2.Notification2Client, error) {
	// Validate token against expected subscriptions
	token, err := s.RenewToken(ctx, opt)
	if err != nil {
		return nil, err
	}

	client := notification2.NewNotification2Client(s.client.BaseURL.String(), nil, notification2.Subscription{
		TokenRenewal: func(v string) (string, error) {
			return s.RenewToken(ctx, Notification2ClientOptions{
				Token: v,
			})
		},
		Consumer: opt.Consumer,
		Token:    token,
	}, opt.ConnectionOptions)
	return client, nil
}
