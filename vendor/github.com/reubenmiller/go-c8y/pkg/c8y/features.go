package c8y

import (
	"context"
	"net/http"
)

const (
	FeaturePhaseInDevelopment      = "IN_DEVELOPMENT"
	FeaturePhasePrivatePreview     = "PRIVATE_PREVIEW"
	FeaturePhasePublicPreview      = "PUBLIC_PREVIEW"
	FeaturePhaseGenerallyAvailable = "GENERALLY_AVAILABLE"
)

const (
	FeatureStrategyDefault = "DEFAULT"
	FeatureStrategyTenant  = "TENANT"
)

// FeatureToggle enables/disables specific functionality in Cumulocity
type FeatureToggle struct {
	// A unique key of the feature toggle
	Key string `json:"key,omitempty"`

	// Current phase of feature toggle rollout.
	Phase string `json:"phase,omitempty"`

	// Current value of the feature toggle marking whether the feature is active or not.
	Active bool `json:"active"`

	// The source of the feature toggle value - either it's feature toggle definition provided default, or per tenant provided override.
	Strategy string `json:"strategy,omitempty"`

	// Tenant id where the feature is active (only set when using the by-tenant api)
	TenantId string `json:"tenantId,omitempty"`
}

// Check if the feature toggle is set to the default for the tenant
func (ft *FeatureToggle) IsDefault() bool {
	return ft.Strategy == FeatureStrategyDefault
}

// Create a new feature toggle
func NewFeatureToggle(active bool) *FeatureToggle {
	return &FeatureToggle{
		Active: active,
	}
}

// FeaturesService manages the feature toggles in Cumulocity
type FeaturesService service

// Retrieve a list of all defined feature toggles with values calculated for a tenant of authenticated user
func (s *FeaturesService) GetFeatures(ctx context.Context) ([]FeatureToggle, *Response, error) {
	data := make([]FeatureToggle, 0)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         "features",
		ResponseData: &data,
	})
	return data, resp, err
}

// GetFeature Retrieve a specific feature toggles defined under given key, with value calculated for a tenant of authenticated user
func (s *FeaturesService) GetFeature(ctx context.Context, key string) (*FeatureToggle, *Response, error) {
	data := new(FeatureToggle)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         "features/" + key,
		ResponseData: data,
	})
	return data, resp, err
}

// Enable a feature in the current tenant
func (s *FeaturesService) Enable(ctx context.Context, key string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   "features/" + key + "/by-tenant",
		Body: FeatureToggle{
			Active: true,
		},
	})
	return resp, err
}

// Disable a feature in the current tenant
func (s *FeaturesService) Disable(ctx context.Context, key string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   "features/" + key + "/by-tenant",
		Body: FeatureToggle{
			Active: false,
		},
	})
	return resp, err
}

// Update a feature toggle in the current tenant
func (s *FeaturesService) Update(ctx context.Context, key string, toggle FeatureToggle) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   "features/" + key + "/by-tenant",
		Body:   toggle,
	})
	return resp, err
}

// Delete a feature toggle in the current tenant
func (s *FeaturesService) Delete(ctx context.Context, key string, body *Tenant) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodDelete,
		Path:   "features/" + key + "/by-tenant",
	})
	return resp, err
}

//
// Feature toggle management of other tenants
//

// GetFeature Retrieve a specific feature toggles defined under given key, with value calculated for a tenant of authenticated user
// Should be called from the management tenant
func (s *FeaturesService) GetFeatureByTenant(ctx context.Context, key string) (*FeatureToggle, *Response, error) {
	data := new(FeatureToggle)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodGet,
		Path:         "features/" + key + "/by-tenant",
		ResponseData: &data,
	})
	return data, resp, err
}

// Enable a feature for a given tenant
// Should be called from the management tenant
func (s *FeaturesService) EnableByTenant(ctx context.Context, key string, tenantID string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   "features/" + key + "/by-tenant/" + tenantID,
		Body: FeatureToggle{
			Active: true,
		},
	})
	return resp, err
}

// Disable a feature for a given tenant
// Should be called from the management tenant
func (s *FeaturesService) DisableByTenant(ctx context.Context, key string, tenantID string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   "features/" + key + "/by-tenant/" + tenantID,
		Body: FeatureToggle{
			Active: false,
		},
	})
	return resp, err
}

// Update a feature toggle for a given tenant
// Should be called from the management tenant
func (s *FeaturesService) UpdateByTenant(ctx context.Context, key string, toggle FeatureToggle, tenantID string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   "features/" + key + "/by-tenant/" + tenantID,
		Body:   toggle,
	})
	return resp, err
}

// Delete a feature toggle for a given tenant
// Should be called from the management tenant
func (s *FeaturesService) DeleteByTenant(ctx context.Context, key string, tenantID string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: http.MethodDelete,
		Path:   "features/" + key + "/by-tenant/" + tenantID,
	})
	return resp, err
}
