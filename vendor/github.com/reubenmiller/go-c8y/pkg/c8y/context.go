package c8y

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// ContextService api
type ContextService service

// ClientUser returns a context with the client username (if set)
func (s *ContextService) ClientUser() context.Context {
	auth := NewBasicAuthString(s.client.TenantName, s.client.Username, s.client.Password)
	return context.WithValue(context.Background(), GetContextAuthTokenKey(), auth)
}

// BootstrapUserFromEnvironment returns a context with the Microservice Bootstrap user (set via environment variables)
func (s *ContextService) BootstrapUserFromEnvironment() context.Context {
	tenant, username, password := GetBootstrapUserFromEnvironment()
	auth := NewBasicAuthString(tenant, username, password)
	return context.WithValue(context.Background(), GetContextAuthTokenKey(), auth)
}

// ServiceUserFromEnvironment returns a context with the Service User authorization (set via environment variables)
func (s *ContextService) ServiceUserFromEnvironment() context.Context {
	tenant, username, password := GetServiceUserFromEnvironment()
	auth := NewBasicAuthString(tenant, username, password)
	return context.WithValue(context.Background(), GetContextAuthTokenKey(), auth)
}

// ServiceUserContext returns authorization context for a Microservice Service user based on the given tenant.
// If tenant is empty, then the first service user will be used.
// If no service users are found then it will panic.
func (s *ContextService) ServiceUserContext(tenant string, skipUpdateServiceUsers bool) context.Context {
	client := s.client
	if !skipUpdateServiceUsers {
		client.Microservice.SetServiceUsers()
	}

	if len(client.ServiceUsers) == 0 {
		panic("No service users found")
	}
	var auth string

	for _, user := range client.ServiceUsers {
		if tenant == user.Tenant || tenant == "" {
			auth = NewBasicAuthString(user.Tenant, user.Username, user.Password)
			break
		}
	}
	return context.WithValue(context.Background(), GetContextAuthTokenKey(), auth)
}

// ServiceUserFromRequest returns a new context with the Authorization token set which will override the Basic Auth in subsequent
// REST requests. The service user will be seletected based on the tenant credentials provided in the request.
// If the request's Authorization header does not use the tenant/username format, then the request's URL
// will be used to determine which tenant to use.
// Should only be used for MULTI_TENANT microservices
func (s *ContextService) ServiceUserFromRequest(req *http.Request) context.Context {
	if req == nil {
		return context.Background()
	}
	auth := req.Header.Get("Authorization")
	data, err := base64.StdEncoding.DecodeString(auth)

	if err != nil {
		panic(err)
	}

	var tenant string

	parts := strings.SplitN(string(data), ":", 2)

	if len(parts) != 2 {
		panic("Invalid basic 64 encoding in Authorization header")
	}

	username := parts[0]

	if strings.Contains(username, "/") {
		usernameParts := strings.SplitN(username, "/", 2)
		if len(usernameParts) != 2 {
			panic("Username does not contain the tenant name")
		}
		tenant = usernameParts[0]
	} else {
		// Get tenant name from the url
		if parts := strings.Split(req.Host, "."); len(parts) > 0 {
			tenant = parts[0]
		} else {
			panic(fmt.Sprintf("Could not detect tenant name from host url %s", req.Host))
		}
	}

	return s.ServiceUserContext(tenant, false)
}

// CommonOptions create common options for a single request
func (s *ContextService) CommonOptions(opts CommonOptions) context.Context {
	return context.WithValue(context.Background(), GetContextCommonOptionsKey(), opts)
}

// Create a context where dry run is disabled
func WithDisabledDryRunContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, GetContextCommonOptionsKey(), CommonOptions{
		DryRun: false,
	})
}

// Create a context with common options
func WithCommonOptionsContext(ctx context.Context, opts CommonOptions) context.Context {
	return context.WithValue(ctx, GetContextCommonOptionsKey(), opts)
}
