package c8y

// PaginationOptions is the cumulocity pagination options
type PaginationOptions struct {
	// Pagesize of results to return in one request
	PageSize int `url:"pageSize,omitempty"`

	// Include total pages included in the pagination at the given page size
	WithTotalPages bool `url:"withTotalPages,omitempty"`

	// Include count of elements in the statistics response. Only supported >= 10.13
	WithTotalElements bool `url:"withTotalElements,omitempty"`

	// Defines the slice of data to be returned, starting with 1. By default, the first page is returned.
	CurrentPage *int `url:"currentPage,omitempty"`
}

// Set the current page to return
func (o *PaginationOptions) SetCurrentPage(v int) *PaginationOptions {
	o.CurrentPage = &v
	return o
}

// NewPaginationOptions returns a pagination options object with a specified pagesize and WithTotalPages set to false
func NewPaginationOptions(pageSize int) *PaginationOptions {
	return &PaginationOptions{
		PageSize: pageSize,
	}
}
