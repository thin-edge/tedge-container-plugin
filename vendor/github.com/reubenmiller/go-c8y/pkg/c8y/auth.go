package c8y

import (
	"encoding/base64"
	"errors"
	"strings"
)

// DecodeBasicAuth returns Service User Credentials object from a given a Basic Auth
func DecodeBasicAuth(auth string, host string) (*ServiceUser, error) {
	data, err := base64.StdEncoding.DecodeString(auth)

	if err != nil {
		return nil, err
	}

	// Remove "Basic " prefix
	dataStr := string(data)
	dataStr = strings.Replace(dataStr, "Basic ", "", -1)
	dataStr = strings.TrimSpace(dataStr)

	parts := strings.SplitN(dataStr, ":", 2)

	if len(parts) != 2 {
		panic("Invalid basic 64 encoding in Authorization header")
	}

	tenant := ""
	username := parts[0]
	password := parts[1]

	if strings.Contains(username, "/") {
		usernameParts := strings.SplitN(username, "/", 2)
		if len(usernameParts) != 2 {
			return nil, errors.New("username does not contain the tenant name")
		}
		tenant = usernameParts[0]
		username = usernameParts[1]
	} else {
		// Get tenant name from the url
		if parts := strings.Split(host, "."); len(parts) > 0 {
			tenant = parts[0]
		} else {
			return nil, errors.New("could not detect tenant name from host url")
		}
	}
	user := &ServiceUser{
		Tenant:   tenant,
		Username: username,
		Password: password,
	}
	return user, nil
}
