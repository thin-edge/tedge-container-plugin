package c8y

import (
	"context"

	"github.com/tidwall/gjson"
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

// Certificate properties
type Certificate struct {
	AlgorithmName           string `json:"algorithmName"`
	CertInPemFormat         string `json:"certInPemFormat"`
	Fingerprint             string `json:"fingerprint"`
	Issuer                  string `json:"issuer"`
	Name                    string `json:"name"`
	NotAfter                string `json:"notAfter"`
	NotBefore               string `json:"notBefore"`
	Self                    string `json:"self"`
	SerialNumber            string `json:"serialNumber"`
	Status                  string `json:"status"`
	Subject                 string `json:"subject"`
	AutoRegistrationEnabled bool   `json:"autoRegistrationEnabled"`
	Version                 int    `json:"version"`
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
