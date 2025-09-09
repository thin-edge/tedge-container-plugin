package c8y

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/reubenmiller/go-c8y/pkg/password"
	"github.com/tidwall/gjson"
	"go.mozilla.org/pkcs7"
)

// DeviceEnrollmentService provides enrollment function to enroll new devices and receive a device certificate
type DeviceEnrollmentService service

// IdentityOptions Identity parameters required when creating a new external id
type EnrollmentOptions struct {
	ExternalID string `json:"externalId"`
	Type       string `json:"type"`
}

// Identity Cumulocity Identity object holding the information about the external id and link to the managed object
type Enrollment struct {
	ExternalID    string            `json:"externalId"`
	Type          string            `json:"type"`
	Self          string            `json:"self"`
	ManagedObject IdentityReference `json:"managedObject"`

	Item gjson.Result `json:"-"`
}

// Create adds a new external id for the given managed object id
func (s *DeviceEnrollmentService) Enroll(ctx context.Context, externalID string, oneTimePassword string, csr *x509.CertificateRequest) (*x509.Certificate, *Response, error) {
	headers := http.Header{}
	headers.Add("Content-Transfer-Encoding", "base64")
	headers.Add("Authorization", NewBasicAuthString("", externalID, oneTimePassword))

	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:      "POST",
		Path:        ".well-known/est/simpleenroll",
		Header:      headers,
		ContentType: "application/pkcs10",
		Body:        base64.StdEncoding.EncodeToString(csr.Raw),
		AuthFunc:    WithNoAuthorization(),
	})

	if err != nil {
		return nil, resp, err
	}

	if resp.IsDryRun() {
		return nil, resp, nil
	}

	return s.parsePKCS7Response(resp)
}

// CreateCertificateSigningRequest creates a certificate signing request using the given external id and private key
// The subject of the certificate will be set to the external id
// Sensible defaults are used to set the Organization and OrganizationalUnit. If you want your own
// custom values, then use certutil.CreateCertificateSigningRequest directly
func (s *DeviceEnrollmentService) CreateCertificateSigningRequest(externalId string, key any) (*x509.CertificateRequest, error) {
	return certutil.CreateCertificateSigningRequest(pkix.Name{
		CommonName: externalId,
		Organization: []string{
			"Cumulocity",
		},
		OrganizationalUnit: []string{
			"Device",
		},
	}, key)
}

// Re enrollment options
type ReEnrollOptions struct {
	// Token to use for authorization
	Token string

	// Certificate Signing Request to request a new certificate
	CSR *x509.CertificateRequest
}

// ReEnroll an already enrolled device using an existing device certificate
// If the token is left empty, then the current user's credentials will be used, however the request will fail if the user does
// not have the following role: ROLE_DEVICE
func (s *DeviceEnrollmentService) ReEnroll(ctx context.Context, opts ReEnrollOptions) (*x509.Certificate, *Response, error) {
	if opts.CSR == nil {
		return nil, nil, fmt.Errorf("no certificate signing request was provided")
	}

	var reqContext context.Context
	if opts.Token != "" {
		reqContext = NewBearerAuthAuthorizationContext(ctx, opts.Token)
	} else {

		reqContext = ctx
	}

	headers := http.Header{}
	headers.Add("Content-Transfer-Encoding", "base64")

	resp, err := s.client.SendRequest(reqContext, RequestOptions{
		Method:      "POST",
		Path:        ".well-known/est/simplereenroll",
		Header:      headers,
		ContentType: "application/pkcs10",
		Body:        base64.StdEncoding.EncodeToString(opts.CSR.Raw),
	})

	if err != nil {
		return nil, resp, err
	}

	if resp.IsDryRun() {
		return nil, resp, nil
	}

	return s.parsePKCS7Response(resp)
}

// AccessToken device access token
type AccessToken struct {
	AccessToken string `json:"accessToken,omitempty"`
}

// RequestAccessToken using an x509 client certificate
// If the clientCert is to nil, then the current client will be used.
//
// If the uploaded trusted certificate is not an immediate issuer of the device
// certificate but belongs to the device’s chain of trust, then the device must
// send the entire certificate chain in the 'X-Ssl-Cert-Chain' to be authenticated
// successfully and retrieve the device access token via the headers argument
//
// See https://cumulocity.com/docs/device-integration/device-integration-rest/#device-authentication for more details
func (s *DeviceEnrollmentService) RequestAccessToken(ctx context.Context, clientCert *tls.Certificate, headers *http.Header) (*AccessToken, *Response, error) {
	deviceClient := s.client
	if clientCert != nil {
		// Create a new client which uses the given certificate
		// Use similar setting as the main client for consistency
		skipVerify := false
		if s.client.client.Transport.(*http.Transport).TLSClientConfig != nil {
			skipVerify = s.client.client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify
		}

		httpClient := NewHTTPClient(
			WithClientCertificate(*clientCert),
			WithInsecureSkipVerify(skipVerify),
		)
		deviceClient = NewClientFromOptions(httpClient, ClientOptions{
			BaseURL: s.client.BaseURL.String(),
		})
	}

	if headers == nil {
		headers = &http.Header{}
	}

	data := new(AccessToken)
	resp, err := deviceClient.SendRequest(context.Background(), RequestOptions{
		Method:       http.MethodPost,
		Path:         "devicecontrol/deviceAccessToken",
		Host:         mtlsEndpoint(s.client.BaseURL),
		Header:       *headers,
		ResponseData: data,

		// No auth is required as x509 certificates are being used
		AuthFunc: WithNoAuthorization(),
	})
	return data, resp, err
}

// mtlsEndpoint returns the host address for the mtls endpoint that can be used for x509 client based authentication
func mtlsEndpoint(u *url.URL) string {
	out := fmt.Sprintf("%s://%s:%s", u.Scheme, u.Hostname(), "8443")
	if u.Path != "" {
		out = out + "/" + u.Path
	}
	return out
}

func (s *DeviceEnrollmentService) parsePKCS7Response(resp *Response) (*x509.Certificate, *Response, error) {
	var err error
	// Decode response
	var contents []byte

	if transferEncoding := resp.Response.Header.Get("Content-Transfer-Encoding"); transferEncoding == "base64" {
		v, decodeErr := certutil.Base64Decode(resp.Body())
		if decodeErr != nil {
			return nil, resp, fmt.Errorf("failed to decode response using base64. %w", decodeErr)
		}
		contents = v
	} else {
		contents = resp.Body()
	}

	// Parse certificate
	var cert *x509.Certificate
	contentType := resp.Response.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/pkcs7-mime") {
		p7, p7Err := pkcs7.Parse(contents)
		if p7Err != nil {
			return nil, resp, p7Err
		}

		if len(p7.Certificates) == 0 {
			return nil, resp, fmt.Errorf("response did not contain any x509 certificates")
		}

		cert = p7.Certificates[0]
	} else if strings.HasPrefix(contentType, "application/pkcs10") {
		cert, err = certutil.ParseCertificatePEM(contents)
	}

	return cert, resp, err
}

// GenerateOneTimePassword generates a one-time password with the recommended password length by default
// and uses symbols which are compatible with the Bulk Registration API.
func (s *DeviceEnrollmentService) GenerateOneTimePassword(opts ...password.PasswordOption) (string, error) {
	defaults := []password.PasswordOption{
		// enforce min/max that the api supports
		password.WithLengthConstraints(8, 32),

		// note: increase to 32 once c8y-ca is in general release
		password.WithLength(31),

		// use reduced set of symbols so that it is more compatible
		// for different usages, e.g. on the shell, within a url without encoding
		password.WithUrlCompatibleSymbols(2),
	}
	defaults = append(defaults, opts...)
	return password.NewRandomPassword(defaults...)
}

// DeviceEnrollmentOption device enrollment options when using the poller
type DeviceEnrollmentOption struct {
	// External ID of the device
	ExternalID string

	// Initial delay before first trying to download the certificate
	InitDelay time.Duration

	// Retry interval when attempting to download the certificate
	Interval time.Duration

	// Overall timeout
	Timeout time.Duration

	// Device one-time password. If left blank a randomly generated password will be used
	OneTimePassword string

	// Certificate Signing Request
	CertificateSigningRequest *x509.CertificateRequest

	// OnProgressBefore callback which is executed just before attempting to download the certificate
	OnProgressBefore func()

	// OnProgressError callback which is executed after a failed download attempt
	OnProgressError func(*Response, error)

	// Banner options
	Banner *DeviceEnrollmentBannerOptions
}

// DeviceEnrollmentDefaultTemplate default enrollment template
var DeviceEnrollmentDefaultTemplate = `
{{.Title}}

{{- if .ShowQRCode }}
Scan the QR Code
{{ .QRCode }}
{{- end}}

{{- if .ShowURL }}
Use the following URL

{{.Url}}
{{- end}}

`

// NewDeviceEnrollmentBannerOptions create default enrollment banner options
func NewDeviceEnrollmentBannerOptions(showQRCode bool, showURL bool) *DeviceEnrollmentBannerOptions {
	return &DeviceEnrollmentBannerOptions{
		Enable:     true,
		ShowQRCode: showQRCode,
		ShowURL:    showURL,
		Template:   DeviceEnrollmentDefaultTemplate,
	}
}

// DeviceEnrollmentBannerOptions banner options to control the visualization of the enrollment information
type DeviceEnrollmentBannerOptions struct {
	Enable bool

	// BannerTemplate is the template which is used to display when Banner is set to True
	//
	// The following variables are supported
	//
	// {{ .Title }} - Banner title
	// {{ .BaseURL }} - BaseURL of the Cumulocity instance
	// {{ .Url }} - Registration URL
	// {{ .ExternalID }} - Device External ID
	// {{ .OneTimePassword }} - One-time password to used to download the certificate
	// {{ .QRCode }} - QR Code which displays the encoded registration URL
	// {{ .Divider }} - Title divider (e.g. `------`)
	Template string

	// ShowQRCode show the QR Code
	ShowQRCode bool

	// ShowURL display the URL
	ShowURL bool
}

// DeviceEnrollmentPollResult result of the device enrollment polling
type DeviceEnrollmentPollResult struct {
	// Err the last error encountered. If it is not nil, then it indicates that a certificate was not downloaded successfully
	Err error

	// ExternalID. The device's external ID
	ExternalID string

	// Certificate x509 is the downloaded certificate as a result of the enrollment action
	Certificate *x509.Certificate

	// Duration is how long the polling took from start to finish
	Duration time.Duration
}

// Ok. Whether the enrollment was successful or not
func (r *DeviceEnrollmentPollResult) Ok() bool {
	return r.Err == nil
}

//go:embed device_registration.txt
var DeviceRegistrationHeader string

func (s *DeviceEnrollmentService) printEnrollmentLog(externalID string, oneTimePassword string, opts DeviceEnrollmentBannerOptions) error {

	if opts.Template == "" {
		opts.Template = DeviceEnrollmentDefaultTemplate
	}

	bannerTemplate, err := template.New("registration").Parse(opts.Template)
	if err != nil {
		return err
	}

	fullURL := fmt.Sprintf(
		"%s/apps/devicemanagement/index.html#/deviceregistration?externalId=%s&one-time-password=%s",
		strings.TrimRight(s.client.BaseURL.String(), "/"),
		externalID,
		oneTimePassword,
	)

	qrcode := bytes.NewBufferString("")
	qrterminal.GenerateWithConfig(fullURL, qrterminal.Config{
		Level:      qrterminal.M,
		Writer:     qrcode,
		HalfBlocks: true,
		QuietZone:  1,
	})

	bannerText := "Device Registration Banner"
	b := bytes.NewBufferString("")
	bannerTemplate.Execute(b, struct {
		Title           string
		BaseURL         string
		Url             string
		ShowQRCode      bool
		ShowURL         bool
		QRCode          string
		ExternalID      string
		OneTimePassword string
		Divider         string
	}{
		Title:           DeviceRegistrationHeader,
		BaseURL:         s.client.BaseURL.String(),
		Url:             fullURL,
		ExternalID:      externalID,
		OneTimePassword: oneTimePassword,
		ShowQRCode:      opts.ShowQRCode,
		ShowURL:         opts.ShowURL,
		QRCode:          qrcode.String(),
		Divider:         strings.Repeat("-", len(bannerText)),
	})
	_, err = fmt.Fprintf(os.Stderr, "%s\n", b.String())
	return err
}

// PollEnroll continuously tries to download the x509 certificate for the given device.
// The polling will give up when
// * The certificate was successful downloaded
// * Timeout is exceeded
func (s *DeviceEnrollmentService) PollEnroll(ctx context.Context, opts DeviceEnrollmentOption) <-chan DeviceEnrollmentPollResult {
	if opts.Interval == 0 {
		opts.Interval = 5 * time.Second
	}

	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Minute
	}

	if opts.OneTimePassword == "" {
		opts.OneTimePassword, _ = s.GenerateOneTimePassword()
	}

	done := make(chan DeviceEnrollmentPollResult)

	if opts.Banner != nil && opts.Banner.Enable {
		if err := s.printEnrollmentLog(opts.ExternalID, opts.OneTimePassword, *opts.Banner); err != nil {
			Logger.Warnf("Failed to print enrollment banner .err=%s", err)
		}
	}

	go func() {
		startedAt := time.Now()

		if opts.InitDelay > 0 {
			time.Sleep(opts.InitDelay)
		}

		ticker := time.NewTicker(opts.Interval)
		timeoutTimer := time.NewTimer(opts.Timeout)

		defer func() {
			ticker.Stop()
			timeoutTimer.Stop()
		}()

		for {
			// try to download certificate
			tick := time.Now()
			if opts.OnProgressBefore != nil {
				opts.OnProgressBefore()
			}

			deviceCert, resp, err := s.Enroll(ctx, opts.ExternalID, opts.OneTimePassword, opts.CertificateSigningRequest)

			if err != nil || resp.IsError() {
				if opts.OnProgressError != nil {
					opts.OnProgressError(resp, err)
				}
			} else {
				done <- DeviceEnrollmentPollResult{
					ExternalID:  opts.ExternalID,
					Certificate: deviceCert,
					Duration:    tick.Sub(startedAt),
				}
				return
			}

			// block waiting for next tick, or polling event
			select {
			case <-ctx.Done():
				done <- DeviceEnrollmentPollResult{
					Err: ctx.Err(),

					ExternalID:  opts.ExternalID,
					Certificate: nil,
					Duration:    time.Since(startedAt),
				}
				return

			case <-ticker.C:
				// continue to next action
				continue

			case tick := <-timeoutTimer.C:
				done <- DeviceEnrollmentPollResult{
					Err:        errors.New("timeout trying to download certificate"),
					ExternalID: opts.ExternalID,
					Duration:   tick.Sub(startedAt),
				}
				return
			}
		}
	}()

	return done
}
