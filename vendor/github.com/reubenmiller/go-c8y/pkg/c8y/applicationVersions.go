package c8y

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tidwall/gjson"
)

var ContentTypeApplicationVersion = "application/vnd.com.nsn.cumulocity.applicationVersion+json"
var ContentTypeApplicationVersionCollection = "application/vnd.com.nsn.cumulocity.applicationVersionCollection+json"

// ApplicationService provides the service provider for the Cumulocity Application API
// WARNING: THE UI Extension Service API is not yet finalized so expect changes in the future!
type ApplicationVersionsService service

// ApplicationVersionsOptions options that can be provided when using application api calls
type ApplicationVersionsOptions struct {
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

func (o *ApplicationVersionsOptions) WithHasVersions(v bool) *ApplicationVersionsOptions {
	o.HasVersions = &v
	return o
}

// Application version
type ApplicationVersion struct {
	Version  string   `json:"version,omitempty"`
	BinaryID string   `json:"binaryId,omitempty"`
	Tags     []string `json:"tags,omitempty"`

	Application *Application `json:"-,omitempty"`

	Item gjson.Result `json:"-"`
}

// ApplicationVersionsCollection a list of versions related to an application
type ApplicationVersionsCollection struct {
	*BaseResponse

	Versions []ApplicationVersion `json:"applicationVersions"`

	Items []gjson.Result `json:"-"`
}

// Retrieve the selected version of an application in your tenant using the tag
func (s *ApplicationVersionsService) GetVersionByTag(ctx context.Context, ID string, tag string) (*ApplicationVersion, *Response, error) {
	data := new(ApplicationVersion)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "GET",
		Path:   "application/applications/" + ID + "/versions",
		Accept: ContentTypeApplicationVersion,
		Query: &versionOption{
			Tag: tag,
		},
		ResponseData: data,
	})
	return data, resp, err
}

// Retrieve the selected version of an application in your tenant using the version name
func (s *ApplicationVersionsService) GetVersionByName(ctx context.Context, ID string, version string) (*ApplicationVersion, *Response, error) {
	data := new(ApplicationVersion)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "GET",
		Path:   "application/applications/" + ID + "/versions",
		Accept: ContentTypeApplicationVersion,
		Query: &versionOption{
			Version: version,
		},
		ResponseData: data,
	})
	return data, resp, err
}

// Retrieve all versions of an application in your tenant
func (s *ApplicationVersionsService) GetVersions(ctx context.Context, ID string, opt *ApplicationVersionsOptions) (*ApplicationVersionsCollection, *Response, error) {
	data := new(ApplicationVersionsCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/applications/" + ID + "/versions",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Replaces the tags of a given application version in your tenant
func (s *ApplicationVersionsService) ReplaceTags(ctx context.Context, ID string, version string, tags []string) (*ApplicationVersion, *Response, error) {
	data := new(ApplicationVersion)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "PUT",
		Path:   "application/applications/" + ID + "/versions/" + version,
		Accept: ContentTypeApplicationVersion,
		Body: &ApplicationVersion{
			Tags: tags,
		},
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes an application version by the tag
func (s *ApplicationVersionsService) DeleteVersionByTag(ctx context.Context, ID string, tag string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "application/applications/" + ID + "/versions",
		Query: &versionOption{
			Tag: tag,
		},
	})
}

type versionOption struct {
	Tag     string `url:"tag,omitempty"`
	Version string `url:"version,omitempty"`
}

// Delete removes an application version by the version name
func (s *ApplicationVersionsService) DeleteVersionByName(ctx context.Context, ID string, version string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "application/applications/" + ID + "/versions",
		Query: &versionOption{
			Version: version,
		},
	})
}

func (s *ApplicationVersionsService) IsUrl(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// CreateVersion creates a new version of an application from a given file or url
func (s *ApplicationVersionsService) CreateVersion(ctx context.Context, ID string, filenameOrURL string, version ApplicationVersion) (*ApplicationVersion, *Response, error) {
	var file *os.File
	var err error
	if s.IsUrl(filenameOrURL) {
		urlLink := filenameOrURL
		file, err = os.CreateTemp("", "extension.zip")
		if err != nil {
			return nil, nil, fmt.Errorf("could not create temp file. %w", err)
		}
		resp, downloadErr := http.Get(urlLink)
		if downloadErr != nil {
			return nil, nil, fmt.Errorf("failed to download extension from url. %w", downloadErr)
		}
		defer func() {
			resp.Body.Close()
			file.Close()
			_ = os.Remove(file.Name())
		}()
		if _, writeErr := io.Copy(file, resp.Body); writeErr != nil {
			return nil, nil, fmt.Errorf("failed to write extension to file. %w", writeErr)
		}
	} else {
		file, err = os.Open(filenameOrURL)
	}
	if err != nil {
		return nil, nil, err
	}

	return s.CreateVersionFromReader(ctx, ID, file, version)
}

// CreateVersion creates a new version of an application from a given file or url
func (s *ApplicationVersionsService) CreateVersionFromReader(ctx context.Context, ID string, file io.Reader, version ApplicationVersion) (*ApplicationVersion, *Response, error) {
	applicationVersion, err := json.Marshal(version)
	if err != nil {
		return nil, nil, err
	}

	values := map[string]io.Reader{
		"applicationBinary":  file,
		"applicationVersion": bytes.NewBuffer(applicationVersion),
	}
	data := new(ApplicationVersion)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Accept:       ContentTypeApplicationVersion,
		Path:         "/application/applications/" + ID + "/versions",
		FormData:     values,
		ResponseData: data,
	})
	return data, resp, err
}
