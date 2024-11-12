package c8y

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/binary"
)

const FragmentFirmware = "c8y_Firmware"
const FragmentFirmwareBinary = "c8y_FirmwareBinary"

// InventoryFirmwareService responsible for all inventory api calls
type InventoryFirmwareService service

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type FirmwareFragment struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Firmware is the general Inventory Managed Object data structure
type Firmware struct {
	ManagedObject
}

// FirmwareVersion firmware version details
type FirmwareVersion struct {
	ManagedObject

	Firmware *FirmwareFragment `json:"c8y_Firmware,omitempty"`
}

// NewFirmware returns a simple firmware managed object
func NewFirmware(name string) *Firmware {
	return &Firmware{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentFirmware,
		},
	}
}

// NewFirmwareVersion returns a firmware version
func NewFirmwareVersion(name string) *FirmwareVersion {
	return &FirmwareVersion{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentFirmwareBinary,
		},
	}
}

// CreateVersion upload a binary and creates a firmware version referencing it
// THe URL can be left blank in the firmware version as it will be automatically set if a filename is provided
func (s *InventoryFirmwareService) CreateVersion(ctx context.Context, firmwareID string, binaryFile binary.MultiPartReader, version FirmwareVersion) (*ManagedObject, *Response, error) {
	return s.client.Inventory.CreateChildAdditionWithBinary(ctx, firmwareID, binaryFile, func(binaryURL string) interface{} {
		version.Firmware.URL = binaryURL
		return version
	})
}

// GetFirmwareByName returns firmware packages by name
func (s *InventoryFirmwareService) GetFirmwareByName(ctx context.Context, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	if paging == nil {
		paging = NewPaginationOptions(100)
	}

	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("$filter=(name eq '%s') and type eq '%s' $orderby=name,creationTime", name, FragmentFirmware),
		PaginationOptions: *paging,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}

// GetFirmwareVersionsByName returns firmware package versions by name
// firmware: can also be referenced by name
func (s *InventoryFirmwareService) GetFirmwareVersionsByName(ctx context.Context, firmware string, name string, withParents bool, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	if paging == nil {
		paging = NewPaginationOptions(100)
	}

	if !IsID(firmware) {
		// Lookup via name
		collection, resp, err := s.GetFirmwareByName(ctx, firmware, NewPaginationOptions(2))

		if err != nil {
			return nil, resp, err
		}
		if len(collection.ManagedObjects) == 0 {
			return nil, resp, ErrNotFound
		}
		if len(collection.ManagedObjects) > 0 {
			firmware = collection.ManagedObjects[0].ID
		}
	}

	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("$filter=(c8y_Firmware.version eq '%s') and bygroupid(%s) $orderby=c8y_Firmware.version,creationTime", name, firmware),
		PaginationOptions: *paging,
		WithParents:       withParents,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}
