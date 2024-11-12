package c8y

import (
	"encoding/json"
)

// AlarmBuilder represents a alarm where the mandatory properties are set via its constructor NewAlarmBuilder
type AlarmBuilder struct {
	data map[string]interface{}
}

// NewAlarmBuilder returns a new alarm builder with the required fields set.
// The alarm will have a timestamp set to Now(). The timestamp can be set to another timestamp by using SetTimestamp()
// The alarm will have a default severity of MAJOR, but it can be changed by using
// .SetSeverity functions
func NewAlarmBuilder(deviceID string, typeName string, text string) *AlarmBuilder {
	b := &AlarmBuilder{
		data: map[string]interface{}{},
	}
	b.SetTimestamp(nil)
	b.SetType(typeName)
	b.SetText(text)
	b.SetDeviceID(deviceID)
	b.SetSeverityMajor() // Set default severity to MAJOR
	return b
}

// MarshalJSON returns the given event in json format
func (b AlarmBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.data)
}

// Severity returns the alarm severity
func (b AlarmBuilder) Severity() string {
	if v, ok := b.data["severity"].(string); ok {
		return v
	}
	return ""
}

// SetSeverityMajor sets the alarm severity to Major
func (b *AlarmBuilder) SetSeverityMajor() *AlarmBuilder {
	return b.Set("severity", AlarmSeverityMajor)
}

// SetSeverityMinor sets the alarm severity to Minor
func (b *AlarmBuilder) SetSeverityMinor() *AlarmBuilder {
	return b.Set("severity", AlarmSeverityMinor)
}

// SetSeverityCritical sets the alarm severity to Critical
func (b *AlarmBuilder) SetSeverityCritical() *AlarmBuilder {
	return b.Set("severity", AlarmSeverityCritical)
}

// SetSeverityWarning sets the alarm severity to Warning
func (b *AlarmBuilder) SetSeverityWarning() *AlarmBuilder {
	return b.Set("severity", AlarmSeverityWarning)
}

// DeviceID returns the device id of the alarm
func (b AlarmBuilder) DeviceID() string {
	if v, ok := b.data["source"].(Source); ok {
		return v.ID
	}
	return ""
}

// SetDeviceID sets the device id for the alarm
func (b *AlarmBuilder) SetDeviceID(ID string) *AlarmBuilder {
	return b.Set("source", Source{
		ID: ID,
	})
}

// Text returns the device id of the alarm
func (b AlarmBuilder) Text() string {
	return b.data["text"].(string)
}

// SetText sets the alarm text
func (b *AlarmBuilder) SetText(ID string) *AlarmBuilder {
	return b.Set("text", ID)
}

// Type returns the alarm type
func (b AlarmBuilder) Type() string {
	return b.data["type"].(string)
}

// SetType sets the alarm type
func (b *AlarmBuilder) SetType(ID string) *AlarmBuilder {
	return b.Set("type", ID)
}

// Timestamp returns the timestamp of the alarm
func (b AlarmBuilder) Timestamp() Timestamp {
	return *b.data["time"].(*Timestamp)
}

// SetTimestamp sets the timestamp when the event was created. If the value is nil, then the current timestamp will be used
func (b *AlarmBuilder) SetTimestamp(value *Timestamp) *AlarmBuilder {
	if value == nil {
		value = NewTimestamp()
	}
	return b.Set("time", value)
}

// Set sets the name property with the given value
func (b *AlarmBuilder) Set(name string, value interface{}) *AlarmBuilder {
	b.data[name] = value
	return b
}

// Get returns the given property value. If the property does not exist, then the second parameter will be set to false
func (b AlarmBuilder) Get(name string) (interface{}, bool) {
	if v, ok := b.data[name]; ok {
		return v, true
	}
	return nil, false
}
