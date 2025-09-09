package wsurl

import (
	"net/url"
	"strings"
)

func GetWebsocketURL(host string, path string) (*url.URL, error) {
	if !strings.HasSuffix(host, "/") {
		host += "/"
	}
	if !strings.Contains(host, "://") {
		host = "wss://" + host
	}
	tempUrl, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	// Check if the url has already been converted
	if tempUrl.Scheme == "ws" || tempUrl.Scheme == "wss" {
		return tempUrl, nil
	}

	if tempUrl.Scheme == "http" {
		tempUrl.Scheme = "ws"
	} else {
		tempUrl.Scheme = "wss"
	}

	return tempUrl.ResolveReference(&url.URL{Path: path}), nil
}
