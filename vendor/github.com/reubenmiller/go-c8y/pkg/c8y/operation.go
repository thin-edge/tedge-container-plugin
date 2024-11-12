package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

// OperationService todo
type OperationService service

// OperationCollectionOptions todo
type OperationCollectionOptions struct {
	// Source device to filter measurements by
	Status string `url:"status,omitempty"`

	AgentID string `url:"agentId,omitempty"`

	DeviceID string `url:"deviceId,omitempty"`

	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	BulkOperationId string `url:"bulkOperationId,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Revert bool `url:"revert,omitempty"`

	// Pagination options
	PaginationOptions
}

// OperationCollection todo
type OperationCollection struct {
	*BaseResponse

	Operations []Operation `json:"operations"`

	Items []gjson.Result `json:"-"`
}

// Cumulocity Operation Status states
const (
	OperationStatusPending    = "PENDING"
	OperationStatusExecuting  = "EXECUTING"
	OperationStatusSuccessful = "SUCCESSFUL"
	OperationStatusFailed     = "FAILED"
)

// OperationUpdateOptions todo
type OperationUpdateOptions struct {
	// Status Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING
	Status string `json:"status,omitempty"`

	// FailureReason is the Reason for the failure
	FailureReason string `json:"failureReason,omitempty"`
}

// GetOperation returns a collection of Cumulocity operations
func (s *OperationService) GetOperation(ctx context.Context, ID string) (*Operation, *Response, error) {
	data := new(Operation)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "devicecontrol/operations/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetOperations returns a collection of Cumulocity operations
func (s *OperationService) GetOperations(ctx context.Context, opt *OperationCollectionOptions) (*OperationCollection, *Response, error) {
	data := new(OperationCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "devicecontrol/operations",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteOperations deletes a collection of Cumulocity operations
func (s *OperationService) DeleteOperations(ctx context.Context, opt *OperationCollectionOptions) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "devicecontrol/operations",
		Query:  opt,
	})
	return resp, err
}

// Create creates a new operation for a device
func (s *OperationService) Create(ctx context.Context, body interface{}) (*Operation, *Response, error) {
	data := new(Operation)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "devicecontrol/operations",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates a Cumulocity operation
func (s *OperationService) Update(ctx context.Context, ID string, body *OperationUpdateOptions) (*Operation, *Response, error) {
	data := new(Operation)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "devicecontrol/operations/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}
