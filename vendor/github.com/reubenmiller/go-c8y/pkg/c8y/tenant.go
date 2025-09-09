package c8y

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/oauth/api"
	"github.com/reubenmiller/go-c8y/pkg/oauth/device"
)

var ErrSSOInvalidConfiguration = errors.New("invalid sso configuration")

// TenantService does something
type TenantService service

// TenantSummaryOptions todo
type TenantSummaryOptions struct {
	DateFrom string `url:"dateFrom,omitempty"`
	DateTo   string `url:"dateTill,omitempty"`
}

type TenantStatisticsOptions struct {
	DateFrom string `url:"dateFrom,omitempty"`
	DateTo   string `url:"dateTill,omitempty"`

	PaginationOptions
}

// TenantSummary todo
type TenantSummary struct {
	Self                    string   `json:"self"`
	Day                     string   `json:"day"`
	DeviceCount             int64    `json:"deviceCount"`
	DeviceWithChildrenCount int64    `json:"deviceWithChildrenCount"`
	DeviceEndpointCount     int64    `json:"deviceEndpointCount"`
	DeviceRequestCount      int64    `json:"deviceRequestCount"`
	RequestCount            int64    `json:"requestCount"`
	StorageSize             int64    `json:"storageSize"`
	SubscribedApplications  []string `json:"subscribedApplications"`
}

type TenantUsageStatisticsCollection struct {
	*BaseResponse
	UsageStatistics []TenantSummary `json:"usageStatistics,omitempty"`
}

// CurrentTenant todo
type CurrentTenant struct {
	Name               string      `json:"name"`
	DomainName         string      `json:"domainName"`
	AllowCreateTenants bool        `json:"allowCreateTenants"`
	CustomProperties   interface{} `json:"customProperties"`
}

type TenantUsageStatisticsSummary struct {
	DeviceCount             int64    `json:"deviceCount"`
	DeviceWithChildrenCount int64    `json:"deviceWithChildrenCount"`
	DeviceEndpointCount     int64    `json:"deviceEndpointCount"`
	DeviceRequestCount      int64    `json:"deviceRequestCount"`
	RequestCount            int64    `json:"requestCount"`
	StorageSize             int64    `json:"storageSize"`
	SubscribedApplications  []string `json:"subscribedApplications"`
}

type TenantUsageStatisticsSummaryExtended struct {
	DeviceCount             int64    `json:"deviceCount,omitempty"`
	DeviceWithChildrenCount int64    `json:"deviceWithChildrenCount,omitempty"`
	DeviceEndpointCount     int64    `json:"deviceEndpointCount,omitempty"`
	DeviceRequestCount      int64    `json:"deviceRequestCount,omitempty"`
	RequestCount            int64    `json:"requestCount,omitempty"`
	StorageSize             int64    `json:"storageSize,omitempty"`
	SubscribedApplications  []string `json:"subscribedApplications,omitempty"`

	// All
	TenantID                          string    `json:"tenantId,omitempty"`
	ParentTenantID                    string    `json:"parentTenantId,omitempty"`
	TenantDomain                      string    `json:"tenantDomain,omitempty"`
	InventoriesUpdateCount            int64     `json:"inventoriesUpdateCount,omitempty"`
	CreationTime                      Timestamp `json:"creationTime,omitempty"`
	EventsCreatedCount                int64     `json:"eventsCreatedCount,omitempty"`
	TotalResourceCreateAndUpdateCount int64     `json:"totalResourceCreateAndUpdateCount,omitempty"`
	PeakDeviceCount                   int64     `json:"peakDeviceCount,omitempty"`
	TenantCompany                     int64     `json:"tenantCompany,omitempty"`
	InventoriesCreatedCount           int64     `json:"inventoriesCreatedCount,omitempty"`
	MeasurementsCreatedCount          int64     `json:"measurementsCreatedCount,omitempty"`
	PeakDeviceWithChildrenCount       int64     `json:"peakDeviceWithChildrenCount,omitempty"`
	PeakStorageSize                   int64     `json:"peakStorageSize,omitempty"`
	AlarmsUpdatedCount                int64     `json:"alarmsUpdatedCount,omitempty"`
	EventsUpdatedCount                int64     `json:"eventsUpdatedCount,omitempty"`
	AlarmsCreatedCount                int64     `json:"alarmsCreatedCount,omitempty"`
}

// TenantLoginOptions tenant login options
type TenantLoginOptions struct {
	Self         string              `json:"self"`
	LoginOptions []TenantLoginOption `json:"loginOptions"`
}

// TenantLoginOption tenant login option
type TenantLoginOption struct {
	ID                   string `json:"id"`
	Self                 string `json:"self"`
	Type                 string `json:"type"`
	UserManagementSource string `json:"userManagementSource,omitempty"`
	TFAStrategy          string `json:"tfaStrategy,omitempty"`
	InitRequest          string `json:"initRequest,omitempty"`
	GrantType            string `json:"grantType,omitempty"`
	VisibleOnLoginPage   bool   `json:"visibleOnLoginPage"`
}

// GetTenantStatisticsSummary returns summary of requests and database usage from the start of this month until now.
func (s *TenantService) GetTenantStatisticsSummary(ctx context.Context, opt *TenantSummaryOptions) (*TenantSummary, *Response, error) {
	data := new(TenantSummary)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/statistics/summary",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetLoginOptions returns the login options available for the tenant
func (s *TenantService) GetLoginOptions(ctx context.Context) (*TenantLoginOptions, *Response, error) {
	data := new(TenantLoginOptions)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/loginOptions",
		AuthFunc:     WithNoAuthorization(),
		ResponseData: data,
	})
	return data, resp, err
}

// GetTenantStatistics returns statics for the current tenant between the specified days
func (s *TenantService) GetTenantStatistics(ctx context.Context, opt *TenantStatisticsOptions) (*TenantUsageStatisticsCollection, *Response, error) {
	data := new(TenantUsageStatisticsCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/statistics",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetAllTenantsStatisticsSummary returns the usage statistics from all of the subtenants
// Note: It will only returns results if the current tenant has subtenants or it is called from the managed tenant
func (s *TenantService) GetAllTenantsStatisticsSummary(ctx context.Context, opt *TenantStatisticsOptions) ([]TenantUsageStatisticsSummaryExtended, *Response, error) {
	data := make([]TenantUsageStatisticsSummaryExtended, 0)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/statistics/allTenantsSummary",
		Query:        opt,
		ResponseData: &data,
	})
	return data, resp, err
}

// GetCurrentTenant returns tenant for the currently logged in service user's tenant
func (s *TenantService) GetCurrentTenant(ctx context.Context) (*CurrentTenant, *Response, error) {
	data := new(CurrentTenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/currentTenant",
		ResponseData: data,
	})
	return data, resp, err
}

type ApplicationReferenceCollection struct {
	*BaseResponse
	References []ApplicationReference `json:"references,omitempty"`
	Self       string                 `json:"self,omitempty"`
}

// Tenant [application/vnd.com.nsn.cumulocity.tenant+json]
type Tenant struct {
	ID                     string                          `json:"id,omitempty"`
	Self                   string                          `json:"self,omitempty"`
	Status                 string                          `json:"status,omitempty"`
	AdminName              string                          `json:"adminName,omitempty"`
	AdminEmail             string                          `json:"adminEmail,omitempty"`
	AdminPassword          string                          `json:"adminPassword,omitempty"`
	Domain                 string                          `json:"domain,omitempty"`
	Company                string                          `json:"company,omitempty"`
	ContactName            string                          `json:"contactName,omitempty"`
	ContactPhone           string                          `json:"contactPhone,omitempty"`
	CustomProperties       interface{}                     `json:"customProperties,omitempty"`
	Parent                 string                          `json:"parent,omitempty"`
	StorageLimitPerDevice  int64                           `json:"storageLimitPerDevice,omitempty"`
	Applications           *ApplicationReferenceCollection `json:"applications,omitempty"`
	OwnedApplications      *ApplicationReferenceCollection `json:"ownedApplications,omitempty"`
	AllowCreateTenants     bool                            `json:"allowCreateTenants,omitempty"`
	SendPasswordResetEmail bool                            `json:"sendPasswordResetEmail,omitempty"`
}

// NewTenant returns a tenant object with the required fields
func NewTenant(company, domain string) *Tenant {
	return &Tenant{
		Company: company,
		Domain:  domain,
	}
}

// TenantCollection todo
type TenantCollection struct {
	*BaseResponse

	Tenants []Tenant `json:"tenants"`
}

type ApplicationReference struct {
	Self        string       `json:"self,omitempty"`
	Application *Application `json:"application"`
}

// GetTenants returns collection of tenants
func (s *TenantService) GetTenants(ctx context.Context, opt *PaginationOptions) (*TenantCollection, *Response, error) {
	data := new(TenantCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetTenant returns a tenant using its ID
func (s *TenantService) GetTenant(ctx context.Context, ID string) (*Tenant, *Response, error) {
	data := new(Tenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// Create adds a new tenant
func (s *TenantService) Create(ctx context.Context, body *Tenant) (*Tenant, *Response, error) {
	data := new(Tenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "tenant/tenants",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update adds an existing tenant
func (s *TenantService) Update(ctx context.Context, ID string, body *Tenant) (*Tenant, *Response, error) {
	data := new(Tenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "tenant/tenants/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes a tenant and all of its data
func (s *TenantService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "tenant/tenants/" + ID,
	})
}

//
// Application Reference Collection
//

// AddApplicationReference adds a new tenant
// Note: Can only be called from the management tenant
func (s *TenantService) AddApplicationReference(ctx context.Context, tenantID string, appSelfReference string) (*ApplicationReference, *Response, error) {
	data := new(ApplicationReference)
	body := &ApplicationReference{
		Application: &Application{
			Self: appSelfReference,
		},
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "tenant/tenants/" + tenantID + "/applications",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// GetApplicationReferences returns list of applications associated with the tenant
// Note: Can only be called from the management tenant
func (s *TenantService) GetApplicationReferences(ctx context.Context, tenantID string, opts *PaginationOptions) (*ApplicationReferenceCollection, *Response, error) {
	data := new(ApplicationReferenceCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants/" + tenantID + "/applications",
		Query:        opts,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteApplicationReference removes an application references from the tenant
// Note: Can only be called from the management tenant
func (s *TenantService) DeleteApplicationReference(ctx context.Context, tenantID string, applicationID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "tenant/tenants/" + tenantID + "/applications/" + applicationID,
	})
}

// GetLoginOptions returns the login options available for the tenant
func getAuthorizationRequest(ctx context.Context, client *http.Client, oauthUrl string) (*api.AuthorizationRequest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", oauthUrl, nil)
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = http.DefaultClient
	}

	// Disable redirects so we can capture the first redirect location
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location, err := resp.Location()
		if err != nil {
			return nil, fmt.Errorf("failed to get redirect location: %w", err)
		}

		endpoint := &api.AuthorizationRequest{
			URL: location,
		}

		for k, v := range location.Query() {
			switch k {
			case "client_id":
				if len(v) > 0 {
					endpoint.ClientID = v[0]
				}
			case "audience":
				if len(v) > 0 {
					endpoint.Audience = v[0]
				}
			case "scope":
				endpoint.Scopes = v
			}

		}

		return endpoint, nil
	}

	return &api.AuthorizationRequest{}, fmt.Errorf("not found")
}

// HasExternalAuthProvider checks if there is an external OAUTH2 provider is configured in the tenant
// Note: This does not require the client to be authenticated
func (s *TenantService) HasExternalAuthProvider(ctx context.Context) (loginOption *TenantLoginOption, found bool, err error) {
	loginOptions, _, err := s.client.Tenant.GetLoginOptions(ctx)
	if err != nil {
		return nil, found, err
	}

	for _, option := range loginOptions.LoginOptions {
		if option.Type == LoginTypeOAuth2 {
			loginOption = &option
			found = true
			break
		}
	}
	return
}

// AuthorizeWithDeviceFlow authorize the client using the OAuth2 Device Authorization Flow (the Auth provider must support it)
func (s *TenantService) AuthorizeWithDeviceFlow(ctx context.Context, initRequest string, auth_endpoints api.AuthEndpoints, displayFunc device.DeviceCodeFunc) (*api.AccessToken, error) {
	// Create a new client which uses the given certificate
	// Use similar setting as the main client for consistency
	skipVerify := false
	if s.client.client.Transport.(*http.Transport).TLSClientConfig != nil {
		skipVerify = s.client.client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify
	}

	httpClient := NewHTTPClient(
		WithInsecureSkipVerify(skipVerify),
		WithRequestDebugLogger(Logger),
	)
	endpoint, err := getAuthorizationRequest(ctx, httpClient, initRequest)
	if err != nil {
		return nil, err
	}

	scopes := make([]string, 0, len(auth_endpoints.Scopes))
	scopes = append(scopes, auth_endpoints.Scopes...)
	if len(scopes) == 0 {
		scopes = append(scopes, endpoint.Scopes...)
	}

	if auth_endpoints.TokenURL == "" || auth_endpoints.DeviceAuthorizationURL == "" {
		// Try detecting the endpoints via the open-id configuration endpoint
		openIDConfig := &api.OpenIDConfiguration{}

		if auth_endpoints.OpenIDConfigurationURL == "" {
			auth_endpoints.OpenIDConfigurationURL = api.GetOpenIDConnectConfigurationURL(endpoint.URL)
		}

		if err := api.GetOpenIDConfiguration(ctx, httpClient, endpoint.URL, auth_endpoints.OpenIDConfigurationURL, openIDConfig); err != nil {
			return nil, fmt.Errorf("%w. %w", ErrSSOInvalidConfiguration, err)
		} else {
			Logger.Infof("Found OpenID Connect configuration. url=%s, config=%#v", auth_endpoints.OpenIDConfigurationURL, openIDConfig)
			if auth_endpoints.TokenURL == "" {
				auth_endpoints.TokenURL = openIDConfig.TokenEndpoint
			}
			if auth_endpoints.DeviceAuthorizationURL == "" {
				auth_endpoints.DeviceAuthorizationURL = openIDConfig.DeviceAuthorizationEndpoint
			}
		}

		// Add default scope if none are defined, as microsoft generally requires at least one scope
		if len(scopes) == 0 && len(openIDConfig.ScopesSupported) > 0 {
			Logger.Infof("Adding default scope. value=%s", openIDConfig.ScopesSupported[0])
			scopes = append(scopes, openIDConfig.ScopesSupported[0])
		}
	}

	deviceCodeURL := api.GetEndpointUrl(endpoint.URL, auth_endpoints.DeviceAuthorizationURL)
	requestCodeOptions := append([]api.AuthRequestEditorFn{}, auth_endpoints.AuthRequestOptions...)
	requestCodeOptions = append(requestCodeOptions, api.WithAudience(endpoint.Audience))
	Logger.Infof("Requesting device code. url=%s, client_id=%s, scopes=%v", deviceCodeURL, endpoint.ClientID, scopes)
	code, err := device.RequestCode(httpClient, deviceCodeURL, endpoint.ClientID, scopes, requestCodeOptions...)
	if err != nil {
		return nil, err
	}

	if displayFunc == nil {
		displayFunc = device.DeviceCodeOnConsole(os.Stderr)
	}

	if displayErr := displayFunc(code); displayErr != nil {
		return nil, displayErr
	}

	accessToken, err := device.Wait(context.TODO(), httpClient, api.GetEndpointUrl(endpoint.URL, auth_endpoints.TokenURL), device.WaitOptions{
		ClientID:   endpoint.ClientID,
		DeviceCode: code,
	})
	if err != nil {
		return nil, err
	}

	// Update client auth
	Logger.Info("Using token from device flow")
	s.client.SetToken(accessToken.Token)

	return accessToken, nil
}
