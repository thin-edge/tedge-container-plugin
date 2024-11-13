package c8y

import (
	"context"
	"fmt"
	"net/http"
)

const (
	RemoteAccessProtocolPassthrough = "PASSTHROUGH"
	RemoteAccessProtocolSSH         = "SSH"
	RemoteAccessProtocolVNC         = "VNC"
	RemoteAccessProtocolTelnet      = "TELNET"
)

// RemoteAccessService
type RemoteAccessService service

// RemoteAccessCollectionOptions remote access collection filter options
type RemoteAccessCollectionOptions struct {
	// Pagination options
	PaginationOptions
}

// RemoteAccessCollection collection of remote access configurations
type RemoteAccessConfiguration struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Hostname    string                  `json:"hostname"`
	Port        int                     `json:"port"`
	Protocol    string                  `json:"protocol"`
	Credentials RemoteAccessCredentials `json:"credentials"`
}

// RemoteAccessCredentials
type RemoteAccessCredentials struct {
	Type       string `json:"type"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	HostKey    string `json:"hostKey"`
}

func (s *RemoteAccessService) path(mo_id string) string {
	return fmt.Sprintf("/service/remoteaccess/devices/%s/configurations", mo_id)
}
func (s *RemoteAccessService) config_path(mo_id string, config_id string) string {
	return fmt.Sprintf("/service/remoteaccess/devices/%s/configurations/%s", mo_id, config_id)
}

// GetConfiguration return a specific remote access configuration for a given device
func (s *RemoteAccessService) GetConfiguration(ctx context.Context, mo_id, config_id string) (*RemoteAccessConfiguration, *Response, error) {
	data := new(RemoteAccessConfiguration)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         s.config_path(mo_id, config_id),
		ResponseData: data,
	})
	return data, resp, err
}

// GetConfigurations returns a collection of Cumulocity remote access configurations
// for a given managed object
func (s *RemoteAccessService) GetConfigurations(ctx context.Context, mo_id string, opt *RemoteAccessCollectionOptions) ([]RemoteAccessConfiguration, *Response, error) {
	data := make([]RemoteAccessConfiguration, 0)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         s.path(mo_id),
		Query:        opt,
		ResponseData: &data,
	})
	return data, resp, err
}

// DeleteConfiguration delete remote access configuration
func (s *RemoteAccessService) DeleteConfiguration(ctx context.Context, mo_id string, config_id string, opt *RemoteAccessCollectionOptions) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodDelete,
		Path:   s.config_path(mo_id, config_id),
		Query:  opt,
	})
	return resp, err
}

// Create creates a new operation for a device
func (s *RemoteAccessService) Create(ctx context.Context, mo_id string, config_id string, body interface{}) (*RemoteAccessConfiguration, *Response, error) {
	data := new(RemoteAccessConfiguration)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodPost,
		Path:         s.config_path(mo_id, config_id),
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates a Cumulocity operation
func (s *RemoteAccessService) Update(ctx context.Context, mo_id string, config_id string, body *OperationUpdateOptions) (*Operation, *Response, error) {
	data := new(Operation)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodPut,
		Path:         s.config_path(mo_id, config_id),
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}
