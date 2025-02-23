// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compact

import (
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/language"
)

func TestParents(t *testing.T) {
	testCases := []struct {
		tag, parent string
	}{
		{"af", "und"},
		{"en", "und"},
		{"en-001", "en"},
		{"en-AU", "en-001"},
		{"en-US", "en"},
		{"en-US-u-va-posix", "en-US"},
		{"ca-ES-valencia", "ca-ES"},
	}
	for _, tc := range testCases {
		tag, ok := LanguageID(Make(language.MustParse(tc.tag)))
		if !ok {
			t.Fatalf("Could not get index of flag %s", tc.tag)
		}
		want, ok := LanguageID(Make(language.MustParse(tc.parent)))
		if !ok {
			t.Fatalf("Could not get index of parent %s of tag %s", tc.parent, tc.tag)
		}
		if got := parents[tag]; got != want {
			t.Errorf("Parent[%s] = %d; want %d (%s)", tc.tag, got, want, tc.parent)
		}
	}
}
