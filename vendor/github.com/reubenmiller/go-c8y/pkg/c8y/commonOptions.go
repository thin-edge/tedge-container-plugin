package c8y

import (
	"io"
	"net/http"
)

// CommonOptions provides options on how the request is processed by the client
type CommonOptions struct {
	// DryRun command will not be sent
	DryRun bool

	// Include the body in the response on HTTP errors (>=400)
	// otherwise the body will only be included in the returned err
	WithError bool

	// OnResponse called on the response before the body is processed
	OnResponse func(response *http.Response) io.Reader
}
