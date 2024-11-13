package binary

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

type BinaryFileOption func(bf *BinaryFile) error

type MultiPartReader interface {
	GetMultiPartBody() (map[string]io.Reader, error)
}

type BinaryFile struct {
	Reader     io.Reader
	Properties map[string]interface{}
}

func (f *BinaryFile) GetMultiPartBody() (map[string]io.Reader, error) {

	var err error
	var metadataBytes []byte
	if f.Properties != nil {
		metadataBytes, err = json.Marshal(f.Properties)
	} else {
		metadataBytes = []byte("{}")
	}

	if err != nil {
		return nil, err
	}

	return map[string]io.Reader{
		"file":   f.Reader,
		"object": bytes.NewReader(metadataBytes),
	}, nil
}

func (f *BinaryFile) WithName2(v string) *BinaryFile {
	f.Properties["name"] = v
	return f
}

func WithName(v string) BinaryFileOption {
	return func(bf *BinaryFile) error {
		bf.Properties["name"] = v
		return nil
	}
}

func WithType(v string) BinaryFileOption {
	return func(bf *BinaryFile) error {
		bf.Properties["type"] = v
		return nil
	}
}

func WithReader(v io.Reader) BinaryFileOption {
	return func(bf *BinaryFile) error {
		bf.Reader = v
		return nil
	}
}

func WithProperty(key string, value interface{}) BinaryFileOption {
	return func(bf *BinaryFile) error {
		bf.Properties[key] = value
		return nil
	}
}

func WithFileProperties(filename string) BinaryFileOption {
	return func(bf *BinaryFile) error {
		for k, v := range GetProperties(filename, false) {
			bf.Properties[k] = v
		}
		return nil
	}
}

func WithProperties(props map[string]interface{}) BinaryFileOption {
	return func(bf *BinaryFile) error {
		for k, v := range props {
			bf.Properties[k] = v
		}
		return nil
	}
}

func WithGlobal() BinaryFileOption {
	return func(bf *BinaryFile) error {
		bf.Properties["c8y_Global"] = map[string]interface{}{}
		return nil
	}
}

func NewBinaryFile(options ...BinaryFileOption) (*BinaryFile, error) {
	bf := &BinaryFile{
		Properties: map[string]interface{}{},
	}
	// ... (write initializations with default values)...
	for _, op := range options {
		err := op(bf)
		if err != nil {
			return nil, err
		}
	}
	return bf, nil
}

func GetProperties(filename string, global bool) map[string]interface{} {
	props := make(map[string]interface{})
	if global {
		props["c8y_Global"] = map[string]interface{}{}
	}

	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	props["name"] = filepath.Base(filename)
	props["type"] = strings.Split(mimeType, ";")[0]
	return props
}
