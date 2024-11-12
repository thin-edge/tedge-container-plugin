package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

// AuditService provides api to get/set/delete audit entries in Cumulocity
type AuditService service

// AuditRecordCollectionOptions to use when search for audit entries
type AuditRecordCollectionOptions struct {
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	Application string `url:"application,omitempty"`

	User string `url:"user,omitempty"`

	Revert bool `url:"revert,omitempty"`

	PaginationOptions
}

// AuditRecord representation
type AuditRecord struct {
	ID           string     `json:"id,omitempty"`
	Self         string     `json:"self,omitempty"`
	CreationTime *Timestamp `json:"creationTime,omitempty"`
	Type         string     `json:"type,omitempty"`
	Time         *Timestamp `json:"time,omitempty"`
	Text         string     `json:"text,omitempty"`
	Source       *Source    `json:"source,omitempty"`
	User         string     `json:"user,omitempty"`
	Application  string     `json:"application,omitempty"`
	Activity     string     `json:"activity,omitempty"`
	Severity     string     `json:"severity,omitempty"`
	// Changes     []ChangeDescription     `json:"changes,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// AuditRecordCollection todo
type AuditRecordCollection struct {
	*BaseResponse

	AuditRecords []AuditRecord `json:"auditRecords"`

	Items []gjson.Result `json:"-"`
}

// GetAuditRecord returns a specific audit record
func (s *AuditService) GetAuditRecord(ctx context.Context, ID string) (*AuditRecord, *Response, error) {
	data := new(AuditRecord)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "audit/auditRecords/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetAuditRecords returns a collection of audit records using search options
func (s *AuditService) GetAuditRecords(ctx context.Context, opt *AuditRecordCollectionOptions) (*AuditRecordCollection, *Response, error) {
	data := new(AuditRecordCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "audit/auditRecords",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create creates a new alarm object
func (s *AuditService) Create(ctx context.Context, body interface{}) (*AuditRecord, *Response, error) {
	data := new(AuditRecord)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "audit/auditRecords",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteAuditRecords removes a collection of audit records based on search options
func (s *AuditService) DeleteAuditRecords(ctx context.Context, opt *AuditRecordCollectionOptions) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "audit/auditRecords",
		Query:  opt,
	})
	return resp, err
}
