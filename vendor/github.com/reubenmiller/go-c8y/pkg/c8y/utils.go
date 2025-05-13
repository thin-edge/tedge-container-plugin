package c8y

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Prepare multipart form-data request which uses io.Pipe to buffer reading the message to ensure files won't be read entirely into memory
func prepareMultipartRequest(method string, url string, values map[string]io.Reader) (*http.Request, error) {
	pr, pw := io.Pipe()

	// Prepare a form that you will submit to that URL.
	w := multipart.NewWriter(pw)

	// Sort form data keys
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	go func() {
		var err error
		for _, key := range keys {
			r := values[key]
			if key == "filename" {
				// Ignore filename as it is used to name the uploaded file
				continue
			}

			var fw io.Writer
			if x, ok := r.(io.Closer); ok {
				defer x.Close()
			}
			// Add an image file
			if x, ok := r.(*os.File); ok {

				// Check if manual filename field was provided, otherwise use the basename
				filename := filepath.Base(x.Name())
				if manual_filename, ok := values["filename"]; ok {
					if b, rErr := io.ReadAll(manual_filename); rErr == nil {
						filename = string(b)
					} else {
						pw.CloseWithError(rErr)
						return
					}
				}
				if fw, err = w.CreateFormFile(key, filename); err != nil {
					pw.CloseWithError(err)
					return
				}
			} else {
				// Add other fields
				if fw, err = w.CreateFormField(key); err != nil {
					pw.CloseWithError(err)
					return
				}
			}
			if _, err = io.Copy(fw, r); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		// Don't forget to close the multipart writer.
		// If you don't close it, your request will be missing the terminating boundary.
		pw.CloseWithError(w.Close())
	}()

	// Now that you have a form, you can submit it to your handler.
	req, rErr := http.NewRequest(method, url, pr)
	if rErr != nil {
		return req, rErr
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// Upload performs a http binary upload
func Upload(client *http.Client, url string, values map[string]io.Reader) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return err
		}

	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return
}

// IsID check if a string is most likely an id
func IsID(v string) bool {
	isNotDigit := func(c rune) bool { return c < '0' || c > '9' }
	value := strings.TrimSpace(v)
	return strings.IndexFunc(value, isNotDigit) <= -1
}
