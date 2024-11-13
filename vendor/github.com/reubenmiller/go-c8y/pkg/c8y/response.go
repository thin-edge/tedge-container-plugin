package c8y

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// newResponse creates a new Response for the provided http.Response.
// r must not be nil.
func newResponse(r *http.Response, duration time.Duration) *Response {
	response := &Response{Response: r, duration: duration}
	return response
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response struct and methods
//_______________________________________________________________________

// Response struct holds response values of executed request.
type Response struct {
	Response *http.Response

	body       []byte
	size       int64
	receivedAt time.Time
	duration   time.Duration
}

func (r *Response) Duration() time.Duration {
	return r.duration
}

func (r *Response) SetBody(v []byte) {
	r.body = v
	r.size = int64(len(r.body))
}

func (r *Response) JSON(path ...string) gjson.Result {
	if len(path) > 0 {
		return gjson.GetBytes(r.body, path[0])
	}
	return gjson.ParseBytes(r.body)
}

// DecodeJSON returns the json response decoded into the given interface
func (r *Response) DecodeJSON(v interface{}) error {
	if r.body == nil {
		return fmt.Errorf("JSON object does not exist (i.e. is nil)")
	}
	err := DecodeJSONBytes(r.body, v)

	if err != nil {
		return err
	}
	return nil
}

// Body method returns HTTP response as []byte array for the executed request.
//
// Note: `Response.Body` might be nil, if `Request.SetOutput` is used.
func (r *Response) Body() []byte {
	if r.Response == nil {
		return []byte{}
	}
	return r.body
}

// Status method returns the HTTP status string for the executed request.
//
//	Example: 200 OK
func (r *Response) Status() string {
	if r.Response == nil {
		return ""
	}
	return r.Response.Status
}

// StatusCode method returns the HTTP status code for the executed request.
//
//	Example: 200
func (r *Response) StatusCode() int {
	if r.Response == nil {
		return 0
	}
	return r.Response.StatusCode
}

// Proto method returns the HTTP response protocol used for the request.
func (r *Response) Proto() string {
	if r.Response == nil {
		return ""
	}
	return r.Response.Proto
}

// Result method returns the response value as an object if it has one
// func (r *Response) Result() interface{} {
// 	return r.Request.Result
// }

// Error method returns the error object if it has one
// func (r *Response) Error() interface{} {
// 	return r.Request.Error
// }

// Header method returns the response headers
func (r *Response) Header() http.Header {
	if r.Response == nil {
		return http.Header{}
	}
	return r.Response.Header
}

// Cookies method to access all the response cookies
func (r *Response) Cookies() []*http.Cookie {
	if r.Response == nil {
		return make([]*http.Cookie, 0)
	}
	return r.Response.Cookies()
}

// String method returns the body of the server response as String.
func (r *Response) String() string {
	if r.body == nil {
		return ""
	}
	return strings.TrimSpace(string(r.body))
}

// Time method returns the time of HTTP response time that from request we sent and received a request.
//
// See `Response.ReceivedAt` to know when client received response and see `Response.Request.Time` to know
// when client sent a request.
// func (r *Response) Time() time.Duration {
// 	if r.Request.clientTrace != nil {
// 		return r.Request.TraceInfo().TotalTime
// 	}
// 	return r.receivedAt.Sub(r.Request.Time)
// }

// ReceivedAt method returns when response got received from server for the request.
func (r *Response) ReceivedAt() time.Time {
	return r.receivedAt
}

// Size method returns the HTTP response size in bytes. Ya, you can relay on HTTP `Content-Length` header,
// however it won't be good for chucked transfer/compressed response. Since Resty calculates response size
// at the client end. You will get actual size of the http response.
func (r *Response) Size() int64 {
	return r.size
}

// RawBody method exposes the HTTP raw response body. Use this method in-conjunction with `SetDoNotParseResponse`
// option otherwise you get an error as `read err: http: read on closed response body`.
//
// Do not forget to close the body, otherwise you might get into connection leaks, no connection reuse.
// Basically you have taken over the control of response parsing from `Resty`.
func (r *Response) RawBody() io.ReadCloser {
	if r.Response == nil {
		return nil
	}
	return r.Response.Body
}

// IsSuccess method returns true if HTTP status `code >= 200 and <= 299` otherwise false.
func (r *Response) IsSuccess() bool {
	return r.StatusCode() > 199 && r.StatusCode() < 300
}

// IsError method returns true if HTTP status `code >= 400` otherwise false.
func (r *Response) IsError() bool {
	return r.StatusCode() > 399
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response Unexported methods
//_______________________________________________________________________

// func (r *Response) setReceivedAt() {
// 	r.receivedAt = time.Now()
// 	if r.Request.clientTrace != nil {
// 		r.Request.clientTrace.endTime = r.receivedAt
// 	}
// }
