// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 The Go Authors.  All rights reserved.
// https://github.com/gxed/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package ptypes

// This file implements operations on google.protobuf.Timestamp.

import (
	"errors"
	"fmt"
	"time"

	tspb "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXSs8cccbT4zDR95c1iRpYKDqVMzqeF1J6iZcavgE6eNw/protobuf/ptypes/timestamp"
)

const (
	// Seconds field of the earliest valid Timestamp.
	// This is time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Unix().
	minValidSeconds = -62135596800
	// Seconds field just after the latest valid Timestamp.
	// This is time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC).Unix().
	maxValidSeconds = 253402300800
)

// validateTimestamp determines whether a Timestamp is valid.
// A valid timestamp represents a time in the range
// [0001-01-01, 10000-01-01) and has a Nanos field
// in the range [0, 1e9).
//
// If the Timestamp is valid, validateTimestamp returns nil.
// Otherwise, it returns an error that describes
// the problem.
//
// Every valid Timestamp can be represented by a time.Time, but the converse is not true.
func validateTimestamp(ts *tspb.Timestamp) error {
	if ts == nil {
		return errors.New("timestamp: nil Timestamp")
	}
	if ts.Seconds < minValidSeconds {
		return fmt.Errorf("timestamp: %v before 0001-01-01", ts)
	}
	if ts.Seconds >= maxValidSeconds {
		return fmt.Errorf("timestamp: %v after 10000-01-01", ts)
	}
	if ts.Nanos < 0 || ts.Nanos >= 1e9 {
		return fmt.Errorf("timestamp: %v: nanos not in range [0, 1e9)", ts)
	}
	return nil
}

// Timestamp converts a google.protobuf.Timestamp proto to a time.Time.
// It returns an error if the argument is invalid.
//
// Unlike most Go functions, if Timestamp returns an error, the first return value
// is not the zero time.Time. Instead, it is the value obtained from the
// time.Unix function when passed the contents of the Timestamp, in the UTC
// locale. This may or may not be a meaningful time; many invalid Timestamps
// do map to valid time.Times.
//
// A nil Timestamp returns an error. The first return value in that case is
// undefined.
func Timestamp(ts *tspb.Timestamp) (time.Time, error) {
	// Don't return the zero value on error, because corresponds to a valid
	// timestamp. Instead return whatever time.Unix gives us.
	var t time.Time
	if ts == nil {
		t = time.Unix(0, 0).UTC() // treat nil like the empty Timestamp
	} else {
		t = time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
	}
	return t, validateTimestamp(ts)
}

// TimestampNow returns a google.protobuf.Timestamp for the current time.
func TimestampNow() *tspb.Timestamp {
	ts, err := TimestampProto(time.Now())
	if err != nil {
		panic("ptypes: time.Now() out of Timestamp range")
	}
	return ts
}

// TimestampProto converts the time.Time to a google.protobuf.Timestamp proto.
// It returns an error if the resulting Timestamp is invalid.
func TimestampProto(t time.Time) (*tspb.Timestamp, error) {
	seconds := t.Unix()
	nanos := int32(t.Sub(time.Unix(seconds, 0)))
	ts := &tspb.Timestamp{
		Seconds: seconds,
		Nanos:   nanos,
	}
	if err := validateTimestamp(ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// TimestampString returns the RFC 3339 string for valid Timestamps. For invalid
// Timestamps, it returns an error message in parentheses.
func TimestampString(ts *tspb.Timestamp) string {
	t, err := Timestamp(ts)
	if err != nil {
		return fmt.Sprintf("(%v)", err)
	}
	return t.Format(time.RFC3339Nano)
}
