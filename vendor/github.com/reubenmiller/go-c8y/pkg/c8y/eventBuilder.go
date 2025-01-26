package c8y

import (
	"encoding/json"
)

// EventBuilder represents a custom event where the mandatory properties are set via its constructor NewEventBuilder
type EventBuilder struct {
	data map[string]interface{}
}

// NewEventBuilder returns a new custom event with the required fields set.
// The event will have a timestamp set to Now(). The timestamp can be set to another timestamp by using SetTimestamp()
func NewEventBuilder(deviceID string, typeName string, text string) *EventBuilder {
	b := &EventBuilder{
		data: map[string]interface{}{},
	}
	b.SetTimestamp(nil)
	b.SetDeviceID(deviceID)
	b.SetText(text)
	b.SetType(typeName)
	return b
}

// MarshalJSON returns the given event in json format
func (b EventBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.data)
}

// DeviceID returns the device id of the custom event
func (b EventBuilder) DeviceID() string {
	if v, ok := b.data["source"].(Source); ok {
		return v.ID
	}
	return ""
}

// SetDeviceID sets the device id for the custom event
func (b *EventBuilder) SetDeviceID(ID string) *EventBuilder {
	return b.Set("source", Source{
		ID: ID,
	})
}

// Type returns the event type
func (b EventBuilder) Type() string {
	return b.data["type"].(string)
}

// SetType sets the event type
func (b *EventBuilder) SetType(name string) *EventBuilder {
	return b.Set("type", name)
}

// Text returns the device id of the custom event
func (b EventBuilder) Text() string {
	return b.data["text"].(string)
}

// SetText sets the event text for the custom event
func (b *EventBuilder) SetText(text string) *EventBuilder {
	return b.Set("text", text)
}

// Timestamp returns the timestamp of the custom event
func (b EventBuilder) Timestamp() Timestamp {
	return *b.data["time"].(*Timestamp)
}

// SetTimestamp sets the timestamp when the event was created. If the value is nil, then the current timestamp will be used
func (b *EventBuilder) SetTimestamp(value *Timestamp) *EventBuilder {
	if value == nil {
		value = NewTimestamp()
	}
	return b.Set("time", value)
}

// Set sets the name property with the given value
func (b *EventBuilder) Set(name string, value interface{}) *EventBuilder {
	b.data[name] = value
	return b
}

// Get returns the given property value. If the property does not exist, then the second return parameter will be set to false
func (b EventBuilder) Get(name string) (interface{}, bool) {
	if v, ok := b.data[name]; ok {
		return v, true
	}
	return nil, false
}
