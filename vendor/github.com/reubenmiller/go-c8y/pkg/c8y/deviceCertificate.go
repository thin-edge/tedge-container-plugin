package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

const (
	CertificateStatusEnabled  = "ENABLED"
	CertificateStatusDisabled = "DISABLED"
)

// DeviceCertificateService interacts with the trusted device certificates in the platform
type DeviceCertificateService service

// DeviceCertificateCollectionOptions query options
type DeviceCertificateCollectionOptions struct {
	// Pagination options
	PaginationOptions
}

// DeviceCertificateCollection a list of the trusted device certificates
type DeviceCertificateCollection struct {
	*BaseResponse

	Certificates []Certificate `json:"certificates"`

	Items []gjson.Result `json:"-"`
}

func NewCertificate() *Certificate {
	return &Certificate{}
}

// Certificate properties
type Certificate struct {
	AlgorithmName              string `json:"algorithmName,omitempty"`
	CertInPemFormat            string `json:"certInPemFormat,omitempty"`
	Fingerprint                string `json:"fingerprint,omitempty"`
	Issuer                     string `json:"issuer,omitempty"`
	Name                       string `json:"name,omitempty"`
	NotAfter                   string `json:"notAfter,omitempty"`
	NotBefore                  string `json:"notBefore,omitempty"`
	Self                       string `json:"self,omitempty"`
	SerialNumber               string `json:"serialNumber,omitempty"`
	Status                     string `json:"status,omitempty"`
	Subject                    string `json:"subject,omitempty"`
	AutoRegistrationEnabled    *bool  `json:"autoRegistrationEnabled,omitempty"`
	TenantCertificateAuthority bool   `json:"tenantCertificateAuthority,omitempty"`
	Version                    int    `json:"version,omitempty"`
}

// Check if auto registration is enabled or not
func (c *Certificate) IsAutoRegistrationEnabled() bool {
	if c.AutoRegistrationEnabled == nil {
		return false
	}
	return *c.AutoRegistrationEnabled
}

// Set the certificate status, ENABLED or DISABLED
func (c *Certificate) WithStatus(v string) *Certificate {
	c.Status = v
	return c
}

// Set the auto registration status
func (c *Certificate) WithAutoRegistration(enabled bool) *Certificate {
	c.AutoRegistrationEnabled = &enabled
	return c
}

// GetCertificates returns collection of certificates
func (s *DeviceCertificateService) GetCertificates(ctx context.Context, tenant string, opt *DeviceCertificateCollectionOptions) (*DeviceCertificateCollection, *Response, error) {
	if tenant == "" {
		tenant = s.client.TenantName
	}
	data := new(DeviceCertificateCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants/" + tenant + "/trusted-certificates",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetCertificate returns a single certificate
func (s *DeviceCertificateService) GetCertificate(ctx context.Context, tenant string, fingerprint string) (*Certificate, *Response, error) {
	if tenant == "" {
		tenant = s.client.TenantName
	}
	data := new(Certificate)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants/" + tenant + "/trusted-certificates/" + fingerprint,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removed a measurement by ID
func (s *DeviceCertificateService) Delete(ctx context.Context, tenant string, fingerprint string) (*Response, error) {
	if tenant == "" {
		tenant = s.client.TenantName
	}
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "tenant/tenants/" + tenant + "/trusted-certificates/" + fingerprint,
	})
}

// Create will upload a new trusted certificate
func (s *DeviceCertificateService) Create(ctx context.Context, tenant string, body interface{}) (*Certificate, *Response, error) {
	if tenant == "" {
		tenant = s.client.TenantName
	}
	data := new(Certificate)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "tenant/tenants/" + tenant + "/trusted-certificates",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update an existing trusted certificate
func (s *DeviceCertificateService) Update(ctx context.Context, tenant string, fingerprint string, body interface{}) (*Certificate, *Response, error) {
	if tenant == "" {
		tenant = s.client.TenantName
	}
	data := new(Certificate)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "tenant/tenants/" + tenant + "/trusted-certificates/" + fingerprint,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}
