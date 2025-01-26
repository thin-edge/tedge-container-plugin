package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

// AlarmService provides api to get/set/delete alarms in Cumulocity
type AlarmService service

// Cumulocity alarm Severity types
const (
	AlarmSeverityCritical = "CRITICAL"
	AlarmSeverityMajor    = "MAJOR"
	AlarmSeverityMinor    = "MINOR"
	AlarmSeverityWarning  = "WARNING"
)

// Cumulocity alarm status states
const (
	AlarmStatusActive       = "ACTIVE"
	AlarmStatusAcknowledged = "ACKNOWLEDGED"
	AlarmStatusCleared      = "CLEARED"
)

// AlarmCollectionOptions to use when search for alarms
type AlarmCollectionOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Status string `url:"status,omitempty"`

	Severity string `url:"severity,omitempty"`

	Resolved bool `url:"resolved,omitempty"`

	WithAssets bool `url:"withAssets,omitempty"`

	WithDevices bool `url:"withDevices,omitempty"`

	PaginationOptions
}

// AlarmUpdateOptions option which can be used to filter when updating alarms
type AlarmUpdateOptions struct {
	// Status alarm status filter
	Status string `url:"status,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	Resolved bool `url:"resolved,omitempty"`

	Severity string `url:"severity,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`
}

// Alarm representation
type Alarm struct {
	ID                  string     `json:"id,omitempty"`
	Source              *Source    `json:"source,omitempty"`
	Type                string     `json:"type,omitempty"`
	Time                *Timestamp `json:"time,omitempty"`
	CreationTime        *Timestamp `json:"creationTime,omitempty"`
	FirstOccurrenceTime *Timestamp `json:"firstOccurrenceTime,omitempty"`
	Text                string     `json:"text,omitempty"`
	Status              string     `json:"status,omitempty"`
	Severity            string     `json:"severity,omitempty"`
	Count               uint64     `json:"count,omitempty"`
	Self                string     `json:"self,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// AlarmCollection todo
type AlarmCollection struct {
	*BaseResponse

	Alarms []Alarm `json:"alarms"`

	Items []gjson.Result `json:"-"`
}

// GetAlarm returns an alarm object by its ID
func (s *AlarmService) GetAlarm(ctx context.Context, ID string) (*Alarm, *Response, error) {
	data := new(Alarm)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "alarm/alarms/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetAlarms returns a collection of alarms using the specified search options
func (s *AlarmService) GetAlarms(ctx context.Context, opt *AlarmCollectionOptions) (*AlarmCollection, *Response, error) {
	data := new(AlarmCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "alarm/alarms",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create creates a new alarm object
func (s *AlarmService) Create(ctx context.Context, body interface{}) (*Alarm, *Response, error) {
	data := new(Alarm)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "alarm/alarms",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// BulkUpdateAlarms bulk update of alarm collection
// The PUT method allows for updating alarms collections. Currently only the status of alarms can be changed.
// Response status:
// 200 - if the process has completed, all alarms have been updated
// 202 - if process continues in background
//
// Since this operations can take a lot of time, request returns after maximum 0.5 sec of processing, and updating is continued as a background process in the platform.
func (s *AlarmService) BulkUpdateAlarms(ctx context.Context, status string, opts AlarmUpdateOptions) (*Response, error) {
	body := map[string]string{
		"status": status,
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "PUT",
		Path:   "alarm/alarms",
		Query:  opts,
		Body:   body,
	})
	return resp, err
}

// AlarmUpdateProperties properties which can be updated on an existing alarm
type AlarmUpdateProperties struct {
	Text     string `json:"text,omitempty"`
	Status   string `json:"status,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// Update updates specific properties for an existing alarm
// Changes to alarms will generate a new audit record. The audit record will include the username and application that triggered the update, if applicable. To get the list of audits for alarm, use the following request: GET /audit/auditRecords?source=
//
// Please notice that if update actually doesn't change anything (i.e. request body contains data that is identical to already present in database), there will be no audit record added and no notifications will be sent.
//
// Only text, status, severity and custom properties can be modified. Non-modifiable fields will be ignored when provided in request.
// Required : ROLE_ALARM_ADMIN or owner of source object
func (s *AlarmService) Update(ctx context.Context, ID string, body AlarmUpdateProperties) (*Alarm, *Response, error) {
	data := new(Alarm)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "alarm/alarms/" + ID,
		ResponseData: data,
		Body:         body,
	})
	return data, resp, err
}

// DeleteAlarms removes a list of alarms using the specified search options
func (s *AlarmService) DeleteAlarms(ctx context.Context, opt *AlarmCollectionOptions) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "alarm/alarms",
		Query:  opt,
	})
	return resp, err
}
