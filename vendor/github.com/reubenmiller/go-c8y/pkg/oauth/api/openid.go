package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

type OpenIDConfiguration struct {
	Issuer                      string   `json:"issuer"`
	AuthorizationEndpoint       string   `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
	TokenEndpoint               string   `json:"token_endpoint"`
	UserInfoEndpoint            string   `json:"userinfo_endpoint"`
	JwksUri                     string   `json:"jwks_uri"`
	RegistrationEndpoint        string   `json:"registration_endpoint"`
	RevocationEndpoint          string   `json:"revocation_endpoint"`
	ResponseTypesSupported      []string `json:"response_types_supported"`
	ScopesSupported             []string `json:"scopes_supported"`
}
type OpenIDMatcher struct {
	Pattern string
	Path    func([]string) string
}

type OpenIDPath func([]string) string

func FixedPath(v string) OpenIDPath {
	return func(s []string) string {
		return v
	}
}

func FirstMatch(format string) OpenIDPath {
	return func(s []string) string {
		if len(s) > 1 {
			return fmt.Sprintf(format, s[1])
		}
		return fmt.Sprintf(format, "")
	}
}

func GetOpenIDConnectConfigurationURL(u *url.URL) string {
	path := "/"
	fullURL := u.String()
	definitions := []OpenIDMatcher{
		{
			// Microsoft
			Pattern: `.*login\.microsoftonline\.com/([^/]+)/.*`,
			Path:    FirstMatch("/%s/v2.0/"),
		},
		{
			// Keycloak
			Pattern: `.*(/realms/[^/]+/).*`,
			Path:    FirstMatch("%s"),
		},
	}
	for _, def := range definitions {
		re, err := regexp.Compile(def.Pattern)
		if err == nil {
			if re.MatchString(fullURL) {
				path = def.Path(re.FindStringSubmatch(fullURL))
				break
			}
		}
	}
	return path + ".well-known/openid-configuration"
}

func GetOpenIDConfiguration(ctx context.Context, client *http.Client, oauthUrl *url.URL, oidc_url string, data any) error {
	if oidc_url == "" {
		oidc_url = GetOpenIDConnectConfigurationURL(oauthUrl)
	}
	u, err := oauthUrl.Parse(oidc_url)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
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
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 399 {
		return fmt.Errorf("request failed. status_code=%s, url=%s", resp.Status, u.String())
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(data)
}
