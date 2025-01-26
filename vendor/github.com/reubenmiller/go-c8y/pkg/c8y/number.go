package c8y

import (
	"encoding/json"
	"fmt"
)

// Number is a nullable representation of a JSON Number which can either be null, an integer or a float(64)
// The number is actually stored as a string internally and then converted to a float64 or an integer on demand.
type Number struct {
	json.Number
}

// NewNumber returns a new number
func NewNumber(value string) *Number {
	return &Number{
		Number: json.Number(value),
	}
}

// IsNull checks if the number is null/valid or not
func (s *Number) IsNull() bool {
	if _, err := s.Float64(); err != nil {
		return true
	}
	return false
}

// SimpleFloat64 returns the value as a float64. If the value is currently null, then 0 will be returned.
// This is simpler to use rather than .Float64() as the user does not have to worry about error checking.
// However the user should call IsNull() first in order to determine if the number is valid or not.
func (s *Number) SimpleFloat64() float64 {
	v, err := s.Float64()
	if err != nil {
		return 0.0
	}
	return v
}

// MarshalJSON converts the Number to its json representation (allowing for null values)
func (s *Number) MarshalJSON() ([]byte, error) {

	if s.IsNull() {
		return []byte("null"), nil
	}

	v := fmt.Sprintf("%v", s.SimpleFloat64())
	return []byte(v), nil
}
