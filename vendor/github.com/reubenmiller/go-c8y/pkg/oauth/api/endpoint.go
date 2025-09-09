package api

import (
	"net/url"
	"strings"
)

// AuthorizationRequest OAuth2 authorization request data
type AuthorizationRequest struct {
	// The token value, typically a 40-character random string.
	ClientID string

	// Audience
	Audience string

	// Scopes
	Scopes []string

	// The refresh token value, associated with the access token.
	URL *url.URL
}

// AuthEndpoints OAuth2 endpoints used to get retrieve the device code and access token
type AuthEndpoints struct {
	// Device Authorization URL e.g. /oauth/device/code
	DeviceAuthorizationURL string

	// Token Authorization URL, e.g. /oauth/token
	TokenURL string

	OpenIDConfigurationURL string

	// User defined scopes to add to request
	Scopes []string

	// Custom auth request options
	AuthRequestOptions []AuthRequestEditorFn
}

// GetEndpointUrl get the full url related to a given oauth endpoint
func GetEndpointUrl(endpoint *url.URL, u string) string {
	if endpoint == nil {
		return u
	}

	// check if already a fully formed url
	if strings.Contains(u, "://") {
		return u
	}

	out, err := endpoint.Parse(u)
	if err != nil {
		return u
	}
	return out.String()
}
