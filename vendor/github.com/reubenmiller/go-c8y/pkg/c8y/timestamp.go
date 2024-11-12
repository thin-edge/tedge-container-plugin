// Copyright 2013 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package c8y

import (
	"strconv"
	"time"
)

// NewTimestamp returns a new timestamp set to Now() or using the specified timestamp
// If the function is called with multiple time.Time values, then only the first will be used to generate the Timestamp
func NewTimestamp(value ...time.Time) *Timestamp {
	if len(value) == 0 {
		return &Timestamp{time.Now()}
	}
	return &Timestamp{value[0]}
}

// Timestamp represents a time that can be unmarshalled from a JSON string
// formatted as either an RFC3339 or Unix timestamp. This is necessary for some
// fields since the GitHub API is inconsistent in how it represents times. All
// exported methods of time.Time can be called on Timestamp.
type Timestamp struct {
	time.Time
}

func (t *Timestamp) String() string {
	return t.Time.Format(time.RFC3339Nano)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Time is expected in RFC3339 or Unix format.
func (t *Timestamp) UnmarshalJSON(data []byte) (err error) {
	str := string(data)
	// Try parsing unix timestamp
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		t.Time = time.Unix(i, 0)
		return
	}

	// Try ISO8806 (without nano seconds)
	t.Time, err = time.Parse(`"`+time.RFC3339+`"`, str)
	if err == nil {
		return
	}

	// Try parsing ISO8806 (with nano seconds)
	t.Time, err = time.Parse(`"`+time.RFC3339Nano+`"`, str)
	if err == nil {
		return
	}
	return err
}

// Equal reports whether t and u are equal based on time.Equal
func (t *Timestamp) Equal(u Timestamp) bool {
	return t.Time.Equal(u.Time)
}
