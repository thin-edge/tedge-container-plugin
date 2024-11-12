package c8y

import (
	"context"
	"errors"
	"time"

	"github.com/tidwall/gjson"
)

// DeviceCredentialsService provides api to get/set/delete alarms in Cumulocity
type DeviceCredentialsService service

// Cumulocity New Device Request statuses
const (
	NewDeviceRequestWaitingForConnection = "WAITING_FOR_CONNECTION"
	NewDeviceRequestPendingAcceptance    = "PENDING_ACCEPTANCE"
	NewDeviceRequestAccepted             = "ACCEPTED"
)

// NewDeviceRequestOptions options which can be used when requesting the New Device Requests
type NewDeviceRequestOptions struct {
	// Status alarm status filter
	// Status string `url:"status,omitempty"`

	PaginationOptions
}

// NewDeviceRequest representation
type NewDeviceRequest struct {
	ID           string `json:"id,omitempty"`
	Status       string `json:"status,omitempty"`
	Self         string `json:"self,omitempty"`
	Owner        string `json:"owner,omitempty"`
	CreationTime string `json:"creationTime,omitempty"`
	TenantID     string `json:"tenantId,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// NewDeviceRequestCollection todo
type NewDeviceRequestCollection struct {
	*BaseResponse

	NewDeviceRequests []NewDeviceRequest `json:"newDeviceRequests"`

	Items []gjson.Result `json:"-"`
}

// GetNewDeviceRequest returns a New Device Request by its id
func (s *DeviceCredentialsService) GetNewDeviceRequest(ctx context.Context, ID string) (*NewDeviceRequest, *Response, error) {
	data := new(NewDeviceRequest)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "devicecontrol/newDeviceRequests/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetNewDeviceRequests returns a collection of New Device requests
func (s *DeviceCredentialsService) GetNewDeviceRequests(ctx context.Context, opt *NewDeviceRequestOptions) (*NewDeviceRequestCollection, *Response, error) {
	data := new(NewDeviceRequestCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "devicecontrol/newDeviceRequests",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create creates a new Device Request
func (s *DeviceCredentialsService) Create(ctx context.Context, ID string) (*NewDeviceRequest, *Response, error) {
	data := new(NewDeviceRequest)
	body := map[string]string{
		"id": ID,
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "devicecontrol/newDeviceRequests",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates an existing New Device Requests status
func (s *DeviceCredentialsService) Update(ctx context.Context, ID string, status string) (*NewDeviceRequest, *Response, error) {
	data := new(NewDeviceRequest)
	body := &NewDeviceRequest{
		Status: status,
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "devicecontrol/newDeviceRequests/" + ID,
		ResponseData: data,
		Body:         body,
	})
	return data, resp, err
}

// Delete removes an existing New Device Request
func (s *DeviceCredentialsService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "devicecontrol/newDeviceRequests/" + ID,
	})
}

// DeviceCredentials is the representation of credentials to be used by a device
type DeviceCredentials struct {
	ID       string `json:"id,omitempty"`
	TenantID string `json:"tenantId,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Self     string `json:"self,omitempty"`
}

// CreateDeviceCredentials creates new device credentials
func (s *DeviceCredentialsService) CreateDeviceCredentials(ctx context.Context, ID string) (*DeviceCredentials, *Response, error) {
	data := new(DeviceCredentials)
	body := &DeviceCredentials{
		ID: ID,
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "devicecontrol/deviceCredentials",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// PollNewDeviceRequest continuously polls a device request for a specified id at defined intervales. The func will wait until the device request has been set to ACCEPTED.
// If the device request does not reach the ACCEPTED state in the defined timeout period, then an error will be returned.
func (s *DeviceCredentialsService) PollNewDeviceRequest(ctx context.Context, deviceID string, interval time.Duration, timeout time.Duration) (<-chan struct{}, <-chan error) {
	ticker := time.NewTicker(interval)
	timeoutTimer := time.NewTimer(timeout)

	done := make(chan struct{})
	err := make(chan error)

	go func() {
		defer func() {
			ticker.Stop()
			timeoutTimer.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				err <- ctx.Err()
				return

			case <-ticker.C:
				Logger.Infof("Polling for device request")
				deviceRequest, _, err := s.GetNewDeviceRequest(ctx, deviceID)
				if err != nil {
					continue
				}
				if deviceRequest.Status == NewDeviceRequestAccepted {
					done <- struct{}{}
				}

			case <-timeoutTimer.C:
				err <- errors.New("timeout waiting for device request to reach ACCEPTED state")
				return
			}
		}
	}()

	return done, err
}
