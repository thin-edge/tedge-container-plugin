package c8y

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// UIExtensionService to managed UI extensions
// WARNING: THE UI Extension Service API is not yet finalized so expect changes in the future!
type UIExtensionService service

var ApplicationTagLatest = "latest"

type UIExtension struct {
	Application
	Manifest     *UIManifest     `json:"manifest,omitempty"`
	ManifestFile *UIManifestFile `json:"-"`
}

type UIManifest struct {
	Package   string `json:"package,omitempty"`
	IsPackage *bool  `json:"isPackage,omitempty"`
}

func (m *UIManifest) WithIsPackage(v bool) *UIManifest {
	m.IsPackage = &v
	return m
}
func (m *UIManifest) WithPackage(v string) *UIManifest {
	m.Package = v
	return m
}

type UIManifestFile struct {
	Name        string `json:"name,omitempty"`
	Key         string `json:"key,omitempty"`
	ContextPath string `json:"contextPath,omitempty"`
	Package     string `json:"package,omitempty"`
	IsPackage   bool   `json:"isPackage,omitempty"`
	Version     string `json:"version,omitempty"`

	Author                  string              `json:"author"`
	Description             string              `json:"description,omitempty"`
	License                 string              `json:"license"`
	Remotes                 map[string][]string `json:"remotes"`
	RequiredPlatformVersion string              `json:"requiredPlatformVersion"`
}

const CumulocityUIManifestFile = "cumulocity.json"

func NewUIExtension(name string) *UIExtension {
	ext := &UIExtension{
		Application: Application{
			Name:        name,
			Key:         name + "-key",
			ContextPath: name,
			Type:        "HOSTED",
		},
		Manifest: &UIManifest{},
	}
	ext.Manifest.
		WithIsPackage(true).
		WithPackage("plugin")
	return ext
}

func GetUIExtensionManifestContents(zipFilename string, contents interface{}) error {
	reader, err := zip.OpenReader(zipFilename)
	if err != nil {
		return err
	}

	defer reader.Close()

	for _, file := range reader.File {
		// check if the file matches the name for application portfolio xml
		if strings.EqualFold(file.Name, CumulocityUIManifestFile) {
			rc, err := file.Open()
			if err != nil {
				return err
			}

			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return err
			}

			defer rc.Close()

			// Unmarshal bytes
			if err := json.Unmarshal(buf.Bytes(), &contents); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *UIExtensionService) NewUIExtensionFromFile(filename string) (*UIExtension, error) {
	app := &UIExtension{
		Manifest:     &UIManifest{},
		ManifestFile: &UIManifestFile{},
	}
	err := GetUIExtensionManifestContents(filename, app.ManifestFile)

	app.Name = app.ManifestFile.Name
	app.Key = app.ManifestFile.Key
	app.Type = ApplicationTypeHosted
	app.ContextPath = app.ManifestFile.ContextPath
	app.Manifest.WithIsPackage(app.ManifestFile.IsPackage)
	app.Manifest.WithPackage(app.ManifestFile.Package)
	return app, err
}

func NewApplicationExtension(name string) *UIExtension {
	app := &UIExtension{
		Application: Application{
			Name:        name,
			Key:         name + "-key",
			ContextPath: name,
			Type:        ApplicationTypeHosted,
		},
		Manifest: &UIManifest{},
	}
	app.Manifest.WithIsPackage(true)
	app.Manifest.WithPackage("package")

	return app
}

type UpsertOptions struct {
	SkipActivation bool
	Version        *ApplicationVersion
}

func HasTag(tags []string, tag string) bool {
	for _, v := range tags {
		if strings.EqualFold(v, tag) {
			return true
		}
	}
	return false
}

// CreateVersion creates a new version of an application from a given file
// The filename can either be a string or a io.Reader
func (s *UIExtensionService) CreateExtension(ctx context.Context, application *Application, filename any, opt UpsertOptions) (*ApplicationVersion, *Response, error) {
	var app *Application
	var resp *Response
	var err error

	// Check if application already exits
	if application.ID != "" {
		// No need to look it up
		app = &Application{
			ID: application.ID,
		}
	} else if application.Name != "" {
		// Lookup via name
		opts := &ApplicationOptions{}
		matches, listResp, listErr := s.client.Application.GetApplicationsByName(ctx, application.Name, opts.WithHasVersions(true))
		if listErr != nil {
			return nil, listResp, listErr
		}
		if len(matches.Applications) > 0 {
			app = &matches.Applications[0]
		}
	} else {
		return nil, nil, fmt.Errorf("application must have either the .ID or .Name set")
	}

	if app == nil {
		// Create the new application
		app, resp, err = s.client.Application.Create(ctx, application)

		// New applications must have the first binary be activated, so ignore the existing SkipActivation option
		opt.SkipActivation = false

		// Append latest tag if not already defined
		if opt.Version != nil {
			if !HasTag(opt.Version.Tags, ApplicationTagLatest) {
				opt.Version.Tags = append(opt.Version.Tags, ApplicationTagLatest)
			}
		}
	} else {
		// Update the existing application
		if application.Availability != "" {
			props := &Application{}
			props.Availability = application.Availability
			app, resp, err = s.client.Application.Update(ctx, app.ID, props)
		}
	}
	if err != nil {
		return nil, resp, err
	}

	var binaryVersion *ApplicationVersion
	var binaryVersionResponse *Response

	// Upload binary
	switch v := filename.(type) {
	case string:
		binaryVersion, binaryVersionResponse, err = s.client.ApplicationVersions.CreateVersion(ctx, app.ID, v, *opt.Version)
	case io.Reader:
		binaryVersion, binaryVersionResponse, err = s.client.ApplicationVersions.CreateVersionFromReader(ctx, app.ID, v, *opt.Version)
	default:
		return nil, nil, fmt.Errorf("invalid file type. Only string or reader is accepted")
	}

	if err != nil {
		return binaryVersion, binaryVersionResponse, err
	}

	if binaryVersion != nil {
		// Store a reference to the related application
		binaryVersion.Application = app
	}

	// Activate the version
	if !opt.SkipActivation {
		_, resp, err = s.client.Application.Update(ctx, app.ID, &Application{
			ActiveVersionID: binaryVersion.BinaryID,
		})
		if err != nil {
			return binaryVersion, resp, err
		}
	}

	return binaryVersion, binaryVersionResponse, err
}

func (s *UIExtensionService) SetActive(ctx context.Context, appID string, binaryID string) (*Application, *Response, error) {
	return s.client.Application.Update(ctx, appID, &Application{
		ActiveVersionID: binaryID,
	})
}

type ExtensionOptions struct {
	PaginationOptions

	Name         string `url:"name,omitempty"`
	Owner        string `url:"owner,omitempty"`
	Availability string `url:"availability,omitempty"`
	ProviderFor  string `url:"providerFor,omitempty"`
	Subscriber   string `url:"subscriber,omitempty"`
	Tenant       string `url:"tenant,omitempty"`
	Type         string `url:"type,omitempty"`
	User         string `url:"user,omitempty"`
	HasVersions  bool   `url:"hasVersions,omitempty"`
}

// GetVersions returns a list of versions for a given application
func (s *UIExtensionService) GetExtensions(ctx context.Context, opt *ExtensionOptions) (*ApplicationCollection, *Response, error) {
	data := new(ApplicationCollection)
	if opt == nil {
		opt = &ExtensionOptions{}
	}
	opt.HasVersions = true
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "application/applications",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}
