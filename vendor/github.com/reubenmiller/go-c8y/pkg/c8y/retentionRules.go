package c8y

import (
	"context"
)

const (
	// RetentionRuleAPI base endpoint
	RetentionRuleAPI = "retention/retentions"
)

// RetentionRuleService does something
type RetentionRuleService service

// RetentionRule todo
type RetentionRule struct {
	// RetentionRule id
	ID string `json:"id,omitempty"`

	// RetentionRule will be applied to documents with source
	Source string `json:"source,omitempty"`

	// RetentionRule will be applied to documents with type
	Type string `json:"type,omitempty"`

	// RetentionRule will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *]
	DataType string `json:"dataType,omitempty"`

	// RetentionRule will be applied to documents with fragmentType
	FragmentType string `json:"fragmentType,omitempty"`

	// Link to this resource
	Self string `json:"self,omitempty"`

	// Maximum age of document in days
	MaximumAge int64 `json:"maximumAge,omitempty"`

	// Whether the rule is editable. Can be updated only by management tenant
	Editable bool `json:"editable,omitempty"`
}

// RetentionRuleCollection todo
type RetentionRuleCollection struct {
	*BaseResponse

	RetentionRules []RetentionRule `json:"retentionRules"`
}

// GetRetentionRule returns the retention rule related to the id
func (s *RetentionRuleService) GetRetentionRule(ctx context.Context, ID string) (*RetentionRule, *Response, error) {
	data := new(RetentionRule)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         RetentionRuleAPI + "/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetRetentionRules returns a list of events based on given filters
func (s *RetentionRuleService) GetRetentionRules(ctx context.Context, opt *PaginationOptions) (*RetentionRuleCollection, *Response, error) {
	data := new(RetentionRuleCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         RetentionRuleAPI,
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Create creates a new event object
func (s *RetentionRuleService) Create(ctx context.Context, body RetentionRule) (*RetentionRule, *Response, error) {
	data := new(RetentionRule)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         RetentionRuleAPI,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates properties on an existing retention rule
func (s *RetentionRuleService) Update(ctx context.Context, ID string, body RetentionRule) (*RetentionRule, *Response, error) {
	data := new(RetentionRule)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         RetentionRuleAPI + "/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete retention rule by its ID
func (s *RetentionRuleService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   RetentionRuleAPI + "/" + ID,
	})
}
