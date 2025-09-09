package c8y

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/reubenmiller/go-c8y/pkg/c8y/binary"
	"github.com/reubenmiller/go-c8y/pkg/c8y/contentType"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// DeviceFragmentName name of the c8yDevice Fragment property
const DeviceFragmentName = "c8y_IsDevice"

// InventoryService responsible for all inventory api calls
type InventoryService service

// ManagedObjectOptions managed object options which can be given with the managed object request
type ManagedObjectOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	WithParents bool `url:"withParents,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"query,omitempty"`

	PaginationOptions
}

// BinaryObjectOptions managed object options which can be given with the managed object request
type ManagedObjectDeleteOptions struct {
	// When set to true and the managed object is a device or group, all the hierarchy will be deleted
	Cascade *bool `url:"cascade,omitempty"`

	// When set to true all the hierarchy will be deleted without checking the type of managed object. It takes precedence over the parameter cascade
	ForceCascade *bool `url:"forceCascade,omitempty"`

	// When set to true and the managed object is a device, it deletes the associated device user (credentials)
	DeviceUser *bool `url:"withDeviceUser,omitempty"`
}

func (o *ManagedObjectDeleteOptions) WithCascade(v bool) *ManagedObjectDeleteOptions {
	o.Cascade = &v
	return o
}

func (o *ManagedObjectDeleteOptions) WithForceCascade(v bool) *ManagedObjectDeleteOptions {
	o.ForceCascade = &v
	return o
}

func (o *ManagedObjectDeleteOptions) WithDeviceUser(v bool) *ManagedObjectDeleteOptions {
	o.DeviceUser = &v
	return o
}

// EmptyFragment fragment used for special c8y fragments, i.e. c8y_IsDevice etc.
type EmptyFragment struct{}

// ConfigurationFragment fragment containing the agent's configuration information
type ConfigurationFragment struct {
	C8yConfiguration AgentConfiguration `json:"c8y_Configuration,omitempty"`
}

// SupportedOperationsFragment list of supported operations which can be sent to device/agent which has this fragment
type SupportedOperationsFragment struct {
	SupportedOperations []string `json:"c8y_SupportedOperations,omitempty"`
}

// AgentConfiguration agent configuration fragment
type AgentConfiguration struct {
	Configuration string `json:"config"`
}

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type AgentFragment struct {
	AgentFragment struct{} `json:"com_cumulocity_model_Agent"`
}

// DeviceFragment marks a managed object which are device representations
type DeviceFragment struct {
	DeviceFragment struct{} `json:"c8y_IsDevice"`
}

// BinaryFragment marks a managed object which are binary representations
// Note: the binary fragment is a string, not a struct!
type BinaryFragment struct {
	BinaryFragment string `json:"c8y_IsBinary"`
}

// ManagedObject is the general Inventory Managed Object data structure
type ManagedObject struct {
	ID               string              `json:"id,omitempty"`
	Name             string              `json:"name,omitempty"`
	Type             string              `json:"type,omitempty"`
	Self             string              `json:"self,omitempty"`
	Owner            string              `json:"owner,omitempty"`
	DeviceParents    *ParentDevices      `json:"deviceParents,omitempty"`
	ChildDevices     *ChildDevices       `json:"childDevices,omitempty"`
	AdditionParents  *AdditionParents    `json:"additionParents,omitempty"`
	AssetParents     *AssetParents       `json:"assetParents,omitempty"`
	ChildAdditions   *ChildAdditions     `json:"childAdditions,omitempty"`
	ChildAssets      *ChildAssets        `json:"childAssets,omitempty"`
	Kpi              *Kpi                `json:"c8y_Kpi,omitempty"`
	C8yConfiguration *AgentConfiguration `json:"c8y_Configuration,omitempty"`

	Item gjson.Result `json:"-"`
}

// Device is a subset of a managed object
type Device struct {
	DeviceFragment
	ManagedObject
}

// Agent is a subset of a managed object
type Agent struct {
	DeviceFragment
	AgentFragment
	ManagedObject
}

// NewDevice returns a simple device managed object
func NewDevice(name string) *Device {
	return &Device{
		ManagedObject: ManagedObject{
			Name: name,
		},
	}
}

// NewAgent returns a simple agent managed object
func NewAgent(name string) *Agent {
	return &Agent{
		ManagedObject: ManagedObject{
			Name: name,
		},
	}
}

// Kpi is the Data Point Library fragment
type Kpi struct {
	Series   string `json:"series"`
	Fragment string `json:"fragment"`
}

// ChildDevices managed object references
type ChildDevices struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// ParentDevices managed object references
type ParentDevices struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// AdditionParents managed object references
type AdditionParents struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// AssetParents managed object references
type AssetParents struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// ChildAssets managed object references
type ChildAssets struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// ChildAdditions managed object references
type ChildAdditions struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// ManagedObjectCollection collection of managed objects
type ManagedObjectCollection struct {
	*BaseResponse

	ManagedObjects []ManagedObject `json:"managedObjects"`

	Items []gjson.Result `json:"-"`
}

// SupportedSeries is a list of the supported series in the format of <fragment>.<series>
type SupportedSeries struct {
	SupportedSeries []string `json:"c8y_SupportedSeries"`
}

// SupportedMeasurements is a list of measurement fragments for the given device
type SupportedMeasurements struct {
	SupportedMeasurements []string `json:"c8y_SupportedMeasurements"`
}

// ManagedObjectReferencesCollection Managed object references
type ManagedObjectReferencesCollection struct {
	*BaseResponse
	References []ManagedObjectReference `json:"references"`

	Items []gjson.Result `json:"-"`
}

// ManagedObjectReference Managed object reference
type ManagedObjectReference struct {
	Self          string        `json:"self,omitempty"`
	ManagedObject ManagedObject `json:"managedObject,omitempty"`
}

// GetDevicesByName returns managed object devices by filter by a name
func (s *InventoryService) GetDevicesByName(ctx context.Context, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("(name eq '%s') and has(%s)", name, DeviceFragmentName),
		PaginationOptions: *paging,
	}
	return s.GetManagedObjects(ctx, opt)
}

// GetDevices returns the c8y device managed objects. These are the objects with the fragment "c8y_IsDevice"
func (s *InventoryService) GetDevices(ctx context.Context, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		FragmentType:      "c8y_IsDevice",
		PaginationOptions: *paging,
	}

	data := new(ManagedObjectCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "inventory/managedObjects",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetManagedObject returns a managed object by its id
func (s *InventoryService) GetManagedObject(ctx context.Context, ID string, opt *ManagedObjectOptions) (*ManagedObject, *Response, error) {
	data := new(ManagedObject)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "inventory/managedObjects/" + ID,
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetManagedObjects returns a list of managed objects
func (s *InventoryService) GetManagedObjects(ctx context.Context, opt *ManagedObjectOptions) (*ManagedObjectCollection, *Response, error) {
	data := new(ManagedObjectCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "inventory/managedObjects",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetSupportedSeries returns the supported series for a give device
func (s *InventoryService) GetSupportedSeries(ctx context.Context, id string) (*SupportedSeries, *Response, error) {
	data := new(SupportedSeries)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         fmt.Sprintf("/inventory/managedObjects/%s/supportedSeries", id),
		ResponseData: data,
	})

	return data, resp, err
}

// GetSupportedMeasurements returns the supported measurements for a given device
func (s *InventoryService) GetSupportedMeasurements(ctx context.Context, id string) (*SupportedMeasurements, *Response, error) {
	data := new(SupportedMeasurements)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         fmt.Sprintf("/inventory/managedObjects/%s/supportedMeasurements", id),
		ResponseData: data,
	})

	return data, resp, err
}

// GetChildDevices Get the child devices of a given managed object
func (s *InventoryService) GetChildDevices(ctx context.Context, id string, opt *PaginationOptions) (*ManagedObjectReferencesCollection, *Response, error) {
	data := new(ManagedObjectReferencesCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         fmt.Sprintf("inventory/managedObjects/%s/childDevices", id),
		Query:        opt,
		ResponseData: data,
	})

	return data, resp, err
}

// GetChildAdditions returns a list of child additions related to a given managed object
func (s *InventoryService) GetChildAdditions(ctx context.Context, id string, opt *ManagedObjectOptions) (*ManagedObjectReferencesCollection, *Response, error) {
	data := new(ManagedObjectReferencesCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         fmt.Sprintf("inventory/managedObjects/%s/childAdditions", id),
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetChildAssets returns a list of child assets related to a given managed object
func (s *InventoryService) GetChildAssets(ctx context.Context, id string, opt *ManagedObjectOptions) (*ManagedObjectReferencesCollection, *Response, error) {
	data := new(ManagedObjectReferencesCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         fmt.Sprintf("inventory/managedObjects/%s/childAssets", id),
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates a managed object
// Link: http://cumulocity.com/guides/reference/inventory
func (s *InventoryService) Update(ctx context.Context, ID string, body interface{}) (*ManagedObject, *Response, error) {
	data := new(ManagedObject)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "inventory/managedObjects/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// CreateDevice creates a device in the Cumulocity platform with the required Device Fragment
func (s *InventoryService) CreateDevice(ctx context.Context, name string) (*ManagedObject, *Response, error) {
	return s.Create(ctx, NewDevice(name))
}

// Create create a new managed object
func (s *InventoryService) Create(ctx context.Context, body interface{}) (*ManagedObject, *Response, error) {
	data := new(ManagedObject)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "inventory/managedObjects",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes a managed object by ID
func (s *InventoryService) Delete(ctx context.Context, ID string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "inventory/managedObjects/" + ID,
	})

	return resp, err
}

// Delete a managed object with additional options
func (s *InventoryService) DeleteWithOptions(ctx context.Context, ID string, options *ManagedObjectDeleteOptions) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "inventory/managedObjects/" + ID,
		Query:  options,
	})

	return resp, err
}

// DownloadBinary downloads a binary by its ID
func (s *InventoryService) DownloadBinary(ctx context.Context, ID string) (filepath string, err error) {
	// set binary api
	client := s.client

	mo, _, err := client.Inventory.GetManagedObject(ctx, ID, nil)

	if err != nil {
		zap.S().Errorf("Could not retrieve managed object. %s", err)
		return
	}

	u := "inventory/binaries/" + ID

	// req, err := http.NewRequest("GET", u.String(), nil)
	req, err := s.client.NewRequest("GET", u, "", nil)
	if err != nil {
		zap.S().Errorf("Could not create request. %s", err)
		return
	}

	req.Header.Set("Accept", "*/*")

	// Create the file
	tempDir, err := os.MkdirTemp("", "go-c8y_")
	if err != nil {
		err = fmt.Errorf("could not create temp folder. %s", err)
		return
	}

	filepath = path.Join(tempDir, mo.Name)
	out, err := os.Create(filepath)
	if err != nil {
		filepath = ""
		return
	}
	defer out.Close()

	// Get the data
	resp, err := client.Do(ctx, req, out)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Check server response
	if resp.StatusCode() != http.StatusOK {
		err = fmt.Errorf("bad status: %s", resp.Response.Status)
		return
	}

	return
}

// CreateBinary uploads a given binary to Cumulocity under the inventory managed objects
func (s *InventoryService) CreateBinary(ctx context.Context, binaryFile binary.MultiPartReader, middleware ...RequestMiddleware) (*ManagedObject, *Response, error) {
	client := s.client

	values, err := binaryFile.GetMultiPartBody()

	if err != nil {
		return nil, nil, err
	}

	// set binary api
	u, _ := url.Parse(client.BaseURL.String())
	u.Path = path.Join(u.Path, "/inventory/binaries")

	req, err := prepareMultipartRequest("POST", u.String(), values)
	if err != nil {
		err = errors.Wrap(err, "Could not create binary upload request object")
		zap.S().Error(err)
		return nil, nil, err
	}
	s.client.SetAuthorization(req)

	req.Header.Set("Accept", "application/json")

	data := new(ManagedObject)
	resp, err := client.Do(ctx, req, data, middleware...)

	if err != nil {
		return nil, resp, err
	}

	data.Item = resp.JSON()

	return data, resp, nil
}

// UpdateBinary updates an existing binary under the inventory managed objects
func (s *InventoryService) UpdateBinary(ctx context.Context, ID string, file io.Reader) (*ManagedObject, *Response, error) {
	data := new(ManagedObject)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "inventory/binaries/" + ID,
		Body:         file,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteBinary removes a managed object Binary by ID
func (s *InventoryService) DeleteBinary(ctx context.Context, ID string) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "inventory/binaries/" + ID,
	})
	return resp, err
}

// GetBinaries returns a list of managed object binaries
func (s *InventoryService) GetBinaries(ctx context.Context, opt *ManagedObjectOptions) (*ManagedObjectCollection, *Response, error) {
	data := new(ManagedObjectCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "inventory/binaries",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// ExpandCollection fetches all of the results by iterating through the .next links in the Cumulocity responses
func (s *InventoryService) ExpandCollection(ctx context.Context, col *ManagedObjectCollection, maxPages int) (out *ManagedObjectCollection) {
	i := 0
	out = col

	for i < maxPages {
		if out.Next == nil {
			return
		}
		Logger.Infof("Requesting page (index=%d, max=%d): %s", i, maxPages, *out.Next)

		urlObj, err := url.Parse(*out.Next)

		if err != nil {
			zap.S().Errorf("")
			return
		}

		data := new(ManagedObjectCollection)

		_, err = s.client.SendRequest(ctx, RequestOptions{
			Path:         urlObj.Path,
			Query:        urlObj.RawQuery,
			ResponseData: data,
		})

		if err != nil {
			return
		}

		Logger.Infof("Adding pagination results - %d managed objects found", len(data.ManagedObjects))

		out.Items = append(out.Items, data.Items...)
		out.ManagedObjects = append(out.ManagedObjects, data.ManagedObjects...)
		out.Next = data.Next
		out.Statistics = data.Statistics

		if len(data.ManagedObjects) < *data.Statistics.PageSize {
			Logger.Infof("No more results in pagination result. total=%d, pages=%d", len(data.ManagedObjects), *data.Statistics.PageSize)
			return
		}

		i++
	}
	return
}

// CreateChildAddition create a new managed object as a child addition to an existing managed object
func (s *InventoryService) CreateChildAddition(ctx context.Context, ID string, body interface{}) (*ManagedObject, *Response, error) {
	data := new(ManagedObject)

	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "inventory/managedObjects/" + ID + "/childAdditions",
		ContentType:  contentType.ContentTypeManagedObject,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// AddChildAddition add a managed object as a child addition to an existing managed object
func (s *InventoryService) AddChildAddition(ctx context.Context, ID, childID string) (*ManagedObject, *Response, error) {
	data := new(ManagedObject)

	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "POST",
		Accept: contentType.ContentTypeJSON,
		Path:   "inventory/managedObjects/" + ID + "/childAdditions",
		Body: &ManagedObjectReference{
			ManagedObject: ManagedObject{
				ID: childID,
			},
		},
		ResponseData: data,
	})
	return data, resp, err
}

// CreateChildAdditionWithBinary create a child addition with a child binary upload a binary and creates a software version referencing it
func (s *InventoryService) CreateChildAdditionWithBinary(ctx context.Context, parentID string, binaryFile binary.MultiPartReader, bodyFunc func(binaryURL string) interface{}, middleware ...RequestMiddleware) (*ManagedObject, *Response, error) {
	// Upload file
	binary, resp, err := s.client.Inventory.CreateBinary(ctx, binaryFile, middleware...)
	if err != nil {
		return binary, resp, err
	}

	// Create managed object (as child addition of software)
	var body interface{}
	if binary != nil {
		body = bodyFunc(binary.Self)
	}
	mo, resp, err := s.client.Inventory.CreateChildAddition(ctx, parentID, body)

	if err != nil {
		return mo, resp, err
	}

	// Add binary as child addition to managed object
	if childMO, childResp, childErr := s.client.Inventory.AddChildAddition(ctx, mo.ID, binary.ID); childErr != nil {
		return childMO, childResp, childErr
	}
	return mo, resp, err
}

// CreateWithBinary create managed object which also has a binary linked as a child addition so that the binary is deleted when the parent managed object is deleted
func (s *InventoryService) CreateWithBinary(ctx context.Context, binaryFile binary.MultiPartReader, bodyFunc func(binaryURL string) interface{}, middleware ...RequestMiddleware) (*ManagedObject, *Response, error) {
	// Upload file
	binary, resp, err := s.client.Inventory.CreateBinary(ctx, binaryFile, middleware...)
	if err != nil {
		return binary, resp, err
	}

	// Create managed object
	var body interface{}
	if binary != nil {
		body = bodyFunc(binary.Self)
	}
	mo, resp, err := s.client.Inventory.Create(ctx, body)

	if err != nil {
		return mo, resp, err
	}

	// Add binary as child addition to managed object
	if childMO, childResp, childErr := s.client.Inventory.AddChildAddition(ctx, mo.ID, binary.ID); childErr != nil {
		return childMO, childResp, childErr
	}
	return mo, resp, err
}
