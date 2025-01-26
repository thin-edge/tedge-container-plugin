package c8y

import (
	"encoding/json"
)

// OperationBuilder is a generic operation representation that can be used to build custom operations with a free format
type OperationBuilder struct {
	data map[string]interface{}
}

// NewOperationBuilder returns a new Custom Operation with the specified device id
// The operation requires at least one custom fragment before it is sent to Cumulocity
// i.e. b.Set("my_CustomOperation", map[string]string{"myprop": "one"})
func NewOperationBuilder(deviceID string) *OperationBuilder {
	b := &OperationBuilder{
		data: map[string]interface{}{},
	}
	b.SetDeviceID(deviceID)
	return b
}

// NewOperationAgentUpdateConfiguration returns a operation which can be used to update the agent's configuration
func NewOperationAgentUpdateConfiguration(deviceID string, configuration string) *OperationBuilder {
	b := NewOperationBuilder(deviceID)
	b.Set("c8y_Configuration", map[string]string{
		"config": configuration,
	})
	return b
}

// MarshalJSON returns the given operation in json format
func (b OperationBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.data)
}

// DeviceID returns the device id of the operation
func (b OperationBuilder) DeviceID() string {
	return b.data["deviceId"].(string)
}

// SetDeviceID sets the device id for the custom operation
func (b *OperationBuilder) SetDeviceID(ID string) *OperationBuilder {
	b.data["deviceId"] = ID
	return b
}

// Set sets the name property with the given value
func (b *OperationBuilder) Set(name string, value interface{}) *OperationBuilder {
	b.data[name] = value
	return b
}

// Get returns the given property value. If the property does not exist, then the second return parameter will be set to false
func (b OperationBuilder) Get(name string) (interface{}, bool) {
	if v, ok := b.data[name]; ok {
		return v, true
	}
	return nil, false
}
