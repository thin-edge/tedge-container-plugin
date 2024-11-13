package c8y

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type ProxyReader struct {
	reader io.Reader
	value  interface{}
}

func (r ProxyReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r ProxyReader) Close() error {
	return nil
}

func (r ProxyReader) GetValue() string {
	if r.value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s", r.value)
}

func NewProxyReader(r io.Reader) *ProxyReader {
	return &ProxyReader{
		reader: r,
		value:  nil,
	}
}

func NewStringReader(v string) *ProxyReader {
	return &ProxyReader{
		reader: strings.NewReader(v),
		value:  v,
	}
}

func NewByteReader(v []byte) *ProxyReader {
	return &ProxyReader{
		reader: bytes.NewReader(v),
		value:  v,
	}
}
