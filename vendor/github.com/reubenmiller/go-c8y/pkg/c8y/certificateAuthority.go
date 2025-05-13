package c8y

import (
	"context"
	"errors"
	"net/http"
)

// CertificateAuthorityService certificate authority service. This features requires the certificate-authority feature toggle to be active.
type CertificateAuthorityService service

// CertificateAuthorityOptions options to control when creating a certificate authority
type CertificateAuthorityOptions struct {
	Status           string
	AutoRegistration bool
}

// ResourceCertificateAuthority endpoint
const ResourceCertificateAuthority = "certificate-authority"

// Create certificate authority
func (s *CertificateAuthorityService) Create(ctx context.Context, opts CertificateAuthorityOptions) (*Certificate, error) {
	cert := new(Certificate)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       http.MethodPost,
		Path:         ResourceCertificateAuthority,
		ResponseData: cert,
	})
	if err != nil {
		// Don't treat a conflict as an error
		if resp != nil && resp.StatusCode() == http.StatusConflict {
			// Get existing certificate
			existingCert, err := s.Get(WithDisabledDryRunContext(ctx))
			if err != nil {
				return nil, err
			}
			if existingCert == nil {
				return nil, nil
			}
			cert = existingCert
		} else {
			return nil, err
		}
	}

	if resp == nil {
		// Dry run
		return nil, nil
	}

	if opts.AutoRegistration != cert.IsAutoRegistrationEnabled() || (opts.Status != "" && opts.Status != cert.Status) {
		cert, _, err := s.client.DeviceCertificate.Update(ctx, s.client.GetTenantName(ctx), cert.Fingerprint, Certificate{
			AutoRegistrationEnabled: &opts.AutoRegistration,
			Status:                  opts.Status,
		})
		return cert, err
	}
	return cert, nil
}

// Delete certificate authority for the current tenant
func (s *CertificateAuthorityService) Delete(ctx context.Context, fingerprint string) (*Response, error) {
	if fingerprint == "" {
		cert, err := s.Get(WithDisabledDryRunContext(ctx))
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if cert == nil {
			return nil, nil
		}
		fingerprint = cert.Fingerprint
	}

	return s.client.DeviceCertificate.Delete(ctx, s.client.GetTenantName(ctx), fingerprint)
}

// Get certificate authority for the current tenant
func (s *CertificateAuthorityService) Get(ctx context.Context) (*Certificate, error) {
	paginationOptions := PaginationOptions{
		PageSize:          2000,
		WithTotalElements: true,
	}
	items, resp, err := s.client.DeviceCertificate.GetCertificates(ctx, s.client.GetTenantName(ctx), &DeviceCertificateCollectionOptions{
		PaginationOptions: paginationOptions,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	for _, item := range items.Certificates {
		if item.TenantCertificateAuthority {
			return &item, nil
		}
	}
	return nil, ErrNotFound
}

// Update certificate authority for the current tenant
// Leave the fingerprint blank if you want to automatically lookup the certificate authority
func (s *CertificateAuthorityService) Update(ctx context.Context, fingerprint string, opts *Certificate) (*Certificate, *Response, error) {
	if fingerprint == "" {
		cert, err := s.Get(WithDisabledDryRunContext(ctx))
		if err != nil {
			return nil, nil, err
		}
		if cert == nil {
			return nil, nil, nil
		}
		fingerprint = cert.Fingerprint
	}
	return s.client.DeviceCertificate.Update(ctx, s.client.GetTenantName(ctx), fingerprint, opts)
}
