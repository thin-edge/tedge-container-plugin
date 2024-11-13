package c8y

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tidwall/gjson"
)

// ApplicationService provides the service provider for the Cumulocity Application API
type ApplicationService service

// ApplicationOptions options that can be provided when using application api calls
type ApplicationOptions struct {
	PaginationOptions

	Name         string `url:"name,omitempty"`
	Owner        string `url:"owner,omitempty"`
	ProviderFor  string `url:"providerFor,omitempty"`
	Subscriber   string `url:"subscriber,omitempty"`
	Tenant       string `url:"tenant,omitempty"`
	Type         string `url:"type,omitempty"`
	User         string `url:"user,omitempty"`
	Availability string `url:"availability,omitempty"`
	HasVersions  *bool  `url:"hasVersions,omitempty"`
}

func (o *ApplicationOptions) WithHasVersions(v bool) *ApplicationOptions {
	o.HasVersions = &v
	return o
}

// Cumulocity Application Types
const (
	ApplicationTypeMicroservice = "MICROSERVICE"
	ApplicationTypeExternal     = "EXTERNAL"
	ApplicationTypeHosted       = "HOSTED"
	ApplicationTypeApamaCEPRule = "APAMA_CEP_RULE"
)

// Cumulocity Application Availability values
const (
	ApplicationAvailabilityMarket  = "MARKET"
	ApplicationAvailabilityPrivate = "PRIVATE"
	ApplicationAvailabilityShared  = "SHARED"
)

// Application todo
type Application struct {
	ID                string            `json:"id,omitempty"`
	Key               string            `json:"key,omitempty"`
	Name              string            `json:"name,omitempty"`
	Type              string            `json:"type,omitempty"`
	Availability      string            `json:"availability,omitempty"`
	Self              string            `json:"self,omitempty"`
	ContextPath       string            `json:"contextPath,omitempty"`
	ExternalURL       string            `json:"externalUrl,omitempty"`
	ResourcesURL      string            `json:"resourcesUrl,omitempty"`
	ResourcesUsername string            `json:"resourcesUsername,omitempty"`
	ResourcesPassword string            `json:"resourcesPassword,omitempty"`
	Owner             *ApplicationOwner `json:"owner,omitempty"`

	// Hosted application
	ActiveVersionID string `json:"activeVersionId,omitempty"`

	// Microservice roles
	RequiredRoles []string `json:"requiredRoles,omitempty"`
	Roles         []string `json:"roles,omitempty"`

	// Application versions
	ApplicationVersions []ApplicationVersion `json:"applicationVersions,omitempty"`

	Item gjson.Result `json:"-"`
}

// NewApplicationMicroservice returns a new microservice application representation
func NewApplicationMicroservice(name string) *Application {
	return &Application{
		Name: name,
		Key:  name + "-microservice-key",
		Type: ApplicationTypeMicroservice,
	}
}

// ApplicationOwner application owner
type ApplicationOwner struct {
	Self   string                      `json:"self,omitempty"`
	Tenant *ApplicationTenantReference `json:"tenant,omitempty"`
}

// ApplicationTenantReference tenant reference information about the application
type ApplicationTenantReference struct {
	ID string `json:"id,omitempty"`
}

// ApplicationCollection contains information about a list of applications
type ApplicationCollection struct {
	*BaseResponse

	Applications []Application `json:"applications"`

	Items []gjson.Result `json:"-"`
}

// ApplicationSubscriptions contains the list of service users for each application subscription
type ApplicationSubscriptions struct {
	Users []ServiceUser `json:"users"`

	Item gjson.Result `json:"-"`
}

// ServiceUser has the service user credentials for a given application subscription
type ServiceUser struct {
	Username string `json:"name"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
}

// getApplicationData todo
func (s *ApplicationService) getApplicationData(ctx context.Context, partialURL string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	data := new(ApplicationCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         partialURL,
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetApplicationsByName returns a list of applications by name
func (s *ApplicationService) GetApplicationsByName(ctx context.Context, name string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByName/%s", name)
	data, resp, err := s.getApplicationData(ctx, u, opt)
	return data, resp, err
}

// GetApplicationsByOwner returns a list of applications by owner
func (s *ApplicationService) GetApplicationsByOwner(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByOwner/%s", tenant)
	return s.getApplicationData(ctx, u, opt)
}

// GetApplicationsByTenant returns a list of applications by tenant name
func (s *ApplicationService) GetApplicationsByTenant(ctx context.Context, tenant string, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	u := fmt.Sprintf("application/applicationsByTenant/%s", tenant)
	return s.getApplicationData(ctx, u, opt)
}

// GetApplication returns an application by its ID
func (s *ApplicationService) GetApplication(ctx context.Context, ID string) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/applications/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetApplications returns a list of applications with no filtering
func (s *ApplicationService) GetApplications(ctx context.Context, opt *ApplicationOptions) (*ApplicationCollection, *Response, error) {
	return s.getApplicationData(ctx, "/application/applications", opt)
}

// Create adds a new application to Cumulocity
func (s *ApplicationService) Create(ctx context.Context, body *Application) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "application/applications",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Copy creates a new application based on an already existing one
// The properties are copied to the newly created application. For name, key and context path a "clone" prefix is added in order to be unique.
// If the target application is hosted and has an active version, the new application will have the active version with the same content.
// The response contains a representation of the newly created application.
func (s *ApplicationService) Copy(ctx context.Context, ID string) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "application/applications/" + ID + "/clone",
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates an existing application
func (s *ApplicationService) Update(ctx context.Context, ID string, body *Application) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "application/applications/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes an existing application
func (s *ApplicationService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "application/applications/" + ID,
	})
}

//
// Current Application (Microservice) API
//

// GetCurrentApplication returns the current application. Note: Required authentication with bootstrap user
func (s *ApplicationService) GetCurrentApplication(ctx context.Context) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/currentApplication",
		ResponseData: data,
	})
	return data, resp, err
}

// ApplicationUser is the representation of the bootstrap user for microservices
type ApplicationUser struct {
	Username string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
	Tenant   string `json:"tenant,omitempty"`
}

// GetApplicationUser returns the application user for a given microservice application
func (s *ApplicationService) GetApplicationUser(ctx context.Context, ID string) (*ApplicationUser, *Response, error) {
	data := new(ApplicationUser)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/applications/" + ID + "/bootstrapUser",
		ResponseData: data,
	})
	return data, resp, err
}

// UpdateCurrentApplication updates the current application from a microservice using bootstrap credentials
func (s *ApplicationService) UpdateCurrentApplication(ctx context.Context, ID string, body *Application) (*Application, *Response, error) {
	data := new(Application)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "application/currentApplication",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// GetCurrentApplicationSubscriptions returns the list of application subscriptions per tenant along with the service user credentials
// This function can only be called using Application Bootstrap credentials, otherwise a 403 (forbidden) response will be returned
func (s *ApplicationService) GetCurrentApplicationSubscriptions(ctx context.Context) (*ApplicationSubscriptions, *Response, error) {
	data := new(ApplicationSubscriptions)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/currentApplication/subscriptions",
		ResponseData: data,
	})
	return data, resp, err
}

/* Application binaries */

// CreateBinary uploads a binary
// For the applications of type "microservice", "web application" and "custom Apama rule" to be available for Cumulocity platform users, a binary zip file must be uploaded.
// For the microservice application, the zip file must consist of:
//   - cumulocity.json - file describing the deployment
//   - image.tar - executable docker image
//
// For the web application, the zip file must include index.html in the root directory.
// For the custom Apama rule application, the zip file must consist of a single .mon file.
func (s *ApplicationService) CreateBinary(ctx context.Context, filename string, ID string) (*Response, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	values := map[string]io.Reader{
		"file": file,
	}

	return s.client.SendRequest(ctx, RequestOptions{
		Method:   "POST",
		Accept:   "application/json",
		Path:     "/application/applications/" + ID + "/binaries",
		FormData: values,
	})
}
