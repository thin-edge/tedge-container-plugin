package c8y

import (
	"io"
	"net/http"
)

// CommonOptions provides options on how the request is processed by the client
type CommonOptions struct {
	// DryRun command will not be sent
	DryRun bool

	// OnResponse called on the response before the body is processed
	OnResponse func(response *http.Response) io.Reader
}
