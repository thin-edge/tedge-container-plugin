package c8y

// Statistics statistics about the returned rest response
type Statistics struct {
	CurrentPage   *int `json:"currentPage"`
	PageSize      *int `json:"pageSize"`
	TotalPages    *int `json:"totalPages"`
	TotalElements *int `json:"totalElements"`
}

// BaseResponse common response from all c8y api calls
type BaseResponse struct {
	Next       *string     `json:"next"`
	Self       *string     `json:"self"`
	Statistics *Statistics `json:"statistics"`
}
