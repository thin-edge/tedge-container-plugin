package c8y

// Source represents a source reference
type Source struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Self string `json:"self,omitempty"`
}

// NewSource returns a new source object
func NewSource(id string) *Source {
	return &Source{
		ID: id,
	}
}
