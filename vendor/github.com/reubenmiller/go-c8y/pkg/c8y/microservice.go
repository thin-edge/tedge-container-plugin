package c8y

import "os"

// MicroserviceService api
type MicroserviceService service

// Application Bootstrap User Environment variables
const (
	EnvironmentBootstrapTenant   = "C8Y_BOOTSTRAP_TENANT"
	EnvironmentBootstrapUsername = "C8Y_BOOTSTRAP_USER"
	EnvironmentBootstrapPassword = "C8Y_BOOTSTRAP_PASSWORD"
)

// Application Environment variables
const (
	EnvironmentApplicationName = "APPLICATION_NAME"
	EnvironmentBaseURL         = "C8Y_BASEURL"
	EnvironmentTenant          = "C8Y_TENANT"
	EnvironmentUsername        = "C8Y_USER"
	EnvironmentPassword        = "C8Y_PASSWORD"
)

// Microservice environment variables
const (
	EnvironmentMicroserviceIsolation = "C8Y_MICROSERVICE_ISOLATION"
)

// GetBootstrapUserFromEnvironment returns the tenant, username and password set in environment variables (used by the microservice)
func GetBootstrapUserFromEnvironment() (tenant, username, password string) {
	return os.Getenv(EnvironmentBootstrapTenant), os.Getenv(EnvironmentBootstrapUsername), os.Getenv(EnvironmentBootstrapPassword)
}

// GetServiceUserFromEnvironment returns the service user information (tenant, username and password) from environment variables.
func GetServiceUserFromEnvironment() (tenant, username, password string) {
	return os.Getenv(EnvironmentTenant), os.Getenv(EnvironmentUsername), os.Getenv(EnvironmentPassword)
}

// SetServiceUsers sets the service users which can then be used later for following requests
// The service users are retrieved by using the bootstrap credentials stored in environment variables
func (s *MicroserviceService) SetServiceUsers() error {
	c := s.client
	serviceUsers, err := c.Microservice.GetServiceUsers()

	if err != nil {
		return err
	}

	c.clientMu.Lock()
	c.ServiceUsers = serviceUsers.Users
	c.clientMu.Unlock()
	return nil
}

// GetServiceUsers returns a list of the subscribed tenant where the application is running
// along with the service user subscriptions for each tenant
func (s *MicroserviceService) GetServiceUsers() (*ApplicationSubscriptions, error) {
	ctx := s.client.Context.BootstrapUserFromEnvironment()
	resp, _, err := s.client.Application.GetCurrentApplicationSubscriptions(ctx)
	return resp, err
}
