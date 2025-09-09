package c8y

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/password"
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

// PollNewDeviceRequest continuously polls a device request for a specified id at defined intervals. The func will wait until the device request has been set to ACCEPTED.
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

//
// Bulk Registration
//

// BulkNewDeviceRequest response which details the results of the bulk registration
type BulkNewDeviceRequest struct {
	NumberOfAll        int64 `json:"numberOfAll,omitempty"`
	NumberOfCreated    int64 `json:"numberOfCreated,omitempty"`
	NumberOfFailed     int64 `json:"numberOfFailed,omitempty"`
	NumberOfSuccessful int64 `json:"numberOfSuccessful,omitempty"`

	CredentialUpdatedList []BulkNewDeviceRequestDetails `json:"credentialUpdatedList,omitempty"`
	FailedCreationList    []BulkNewDeviceRequestDetails `json:"failedCreationList,omitempty"`

	Item gjson.Result `json:"-"`
}

type BulkNewDeviceRequestDetails struct {
	BulkNewDeviceStatus string `json:"bulkNewDeviceStatus,omitempty"`
	DeviceID            string `json:"deviceId,omitempty"`

	FailureReason string `json:"failureReason,omitempty"`
	Line          string `json:"line,omitempty"`
}

// CreateBulk allows multiple devices to be registered in one request
func (s *DeviceCredentialsService) CreateBulk(ctx context.Context, csvContents io.Reader) (*BulkNewDeviceRequest, *Response, error) {
	formData := make(map[string]io.Reader)
	formData["file"] = csvContents

	data := new(BulkNewDeviceRequest)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "devicecontrol/bulkNewDeviceRequests",
		FormData:     formData,
		ResponseData: data,
	})

	if err != nil {
		return nil, resp, err
	}
	return data, resp, err
}

// GeneratePassword generates a device password with the recommended password length by default
// and uses symbols which are compatible with the Bulk Registration API.
func (s *DeviceCredentialsService) GeneratePassword(opts ...password.PasswordOption) (string, error) {
	defaults := []password.PasswordOption{
		// enforce min/max that the api supports
		password.WithLengthConstraints(8, 32),

		// use max length
		password.WithLength(32),

		// use all available symbols to increase complexity
		password.WithSymbols(2),
	}
	defaults = append(defaults, opts...)
	return password.NewRandomPassword(defaults...)
}

const (
	// BulkRegistrationAuthTypeBasic Basic Authorization
	BulkRegistrationAuthTypeBasic string = "BASIC"

	// BulkRegistrationAuthTypeCertificates Certificate Authorization
	BulkRegistrationAuthTypeCertificates string = "CERTIFICATES"
)

type BulkRegistrationRecord struct {
	// External ID
	ID string `json:"externalId,omitempty"`

	// External Id Type
	IDType string `json:"externalType,omitempty"`

	// Authorization Type, BASIC, CERTIFICATES
	AuthType string `json:"authType,omitempty"`

	// Basic Auth credentials
	Credentials string `json:"password,omitempty"`

	// Enrollment one-time password
	EnrollmentOTP string `json:"enrollmentOTP,omitempty"`

	// Name
	Name string `json:"name,omitempty"`

	// Type
	Type string `json:"type,omitempty"`

	// ICCID
	ICCID string `json:"iccid,omitempty"`

	// Tenant
	Tenant string `json:"tenant,omitempty"`

	// Path / Group hierarchy
	Path string `json:"group,omitempty"`

	// Is the device an agent
	IsAgent bool `json:"isAgent,omitempty"`
}

// SetBasicAuth sets the record for basic authentication
func (r *BulkRegistrationRecord) SetBasicAuth(v string) {
	r.AuthType = BulkRegistrationAuthTypeBasic
	r.Credentials = v
	r.IsAgent = true
}

// SetCertificateAuth set certificate authentication (for externally issued certificates)
func (r *BulkRegistrationRecord) SetCertificateAuth() {
	r.AuthType = BulkRegistrationAuthTypeCertificates
	r.Credentials = ""
	r.IsAgent = true
}

// SetEnrollmentPassword set certificate authentication with a one-time password for the Cumulocity certificate authority
func (r *BulkRegistrationRecord) SetEnrollmentPassword(v string) {
	r.AuthType = BulkRegistrationAuthTypeCertificates
	r.EnrollmentOTP = v
	r.Credentials = ""
	r.IsAgent = true
}

// BulkRegistrationColumns bulk registration CSV columns
var BulkRegistrationColumns = []string{
	"ID",             // External ID
	"AUTH_TYPE",      // Authorization type
	"CREDENTIALS",    // Basic Auth
	"ENROLLMENT_OTP", // One-time password for EST enrollment
	"NAME",           // Device name
	"TYPE",           // Device type
	"IDTYPE",         // External ID Type
	"ICCID",          // ICCID
	"TENANT",         // Tenant
	"PATH",           // Path / group
	"com_cumulocity_model_Agent.active",
}

// WriteCSV the records to the give writer
func BulkRegistrationRecordWriter(w io.Writer, records ...BulkRegistrationRecord) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = '\t'
	if err := csvWriter.Write(BulkRegistrationColumns); err != nil {
		return err
	}
	for _, item := range records {
		if err := csvWriter.Write([]string{
			item.ID,
			item.AuthType,
			item.Credentials,
			item.EnrollmentOTP,
			item.Name,
			item.Type,
			item.IDType,
			item.ICCID,
			item.Tenant,
			item.Path,
			fmt.Sprintf("%v", item.IsAgent),
		}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}
