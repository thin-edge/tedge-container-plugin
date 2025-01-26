package c8y

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/binary"
)

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

// InventorySoftwareService responsible for all inventory api calls
type InventorySoftwareService service

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type SoftwareFragment struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Software is the general Inventory Managed Object data structure
type Software struct {
	ManagedObject
}

type SoftwareVersion struct {
	ManagedObject

	Software *SoftwareFragment `json:"c8y_Software,omitempty"`
}

// NewSoftware returns a simple software managed object
func NewSoftware(name string) *Software {
	return &Software{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentSoftware,
		},
	}
}

func NewSoftwareVersion(name string) *SoftwareVersion {
	return &SoftwareVersion{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentSoftwareBinary,
		},
	}
}

// CreateVersion upload a binary and creates a software version referencing it
// THe URL can be left blank in the software version as it will be automatically set if a filename is provided
func (s *InventorySoftwareService) CreateVersion(ctx context.Context, softwareID string, binaryFile binary.MultiPartReader, version SoftwareVersion) (*ManagedObject, *Response, error) {
	return s.client.Inventory.CreateChildAdditionWithBinary(ctx, softwareID, binaryFile, func(binaryURL string) interface{} {
		version.Software.URL = binaryURL
		return version
	})
}

// GetSoftwareByName returns software packages by name
func (s *InventorySoftwareService) GetSoftwareByName(ctx context.Context, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	if paging == nil {
		paging = NewPaginationOptions(100)
	}

	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("$filter=(name eq '%s') and type eq '%s' $orderby=name,creationTime", name, FragmentSoftware),
		PaginationOptions: *paging,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}

// GetSoftwareVersionsByName returns software package versions by name
// software: can also be referenced by name
func (s *InventorySoftwareService) GetSoftwareVersionsByName(ctx context.Context, software string, name string, withParents bool, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	if paging == nil {
		paging = NewPaginationOptions(100)
	}
	if !IsID(software) {
		// Lookup software via name
		softwareMO, resp, err := s.GetSoftwareByName(ctx, software, NewPaginationOptions(2))

		if err != nil {
			return nil, resp, err
		}
		if len(softwareMO.ManagedObjects) == 0 {
			return nil, resp, ErrNotFound
		}
		if len(softwareMO.ManagedObjects) > 0 {
			software = softwareMO.ManagedObjects[0].ID
		}
	}

	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("$filter=(c8y_Software.version eq '%s') and bygroupid(%s) $orderby=c8y_Software.version,creationTime", name, software),
		PaginationOptions: *paging,
		WithParents:       withParents,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}
