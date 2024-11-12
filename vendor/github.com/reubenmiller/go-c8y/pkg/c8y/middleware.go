package c8y

import "net/http"

type RequestMiddleware func(r *http.Request) (*http.Request, error)
