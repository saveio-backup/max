// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runenames

import (
	"strings"
	"testing"
	"unicode"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/gen"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/testtext"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/ucd"
)

func TestName(t *testing.T) {
	testtext.SkipIfNotLong(t)

	wants := make([]string, 1+unicode.MaxRune)
	ucd.Parse(gen.OpenUCDFile("UnicodeData.txt"), func(p *ucd.Parser) {
		wants[p.Rune(0)] = getName(p)
	})

	nErrors := 0
	for r, want := range wants {
		got := Name(rune(r))
		if got != want {
			t.Errorf("r=%#08x: got %q, want %q", r, got, want)
			nErrors++
			if nErrors == 100 {
				t.Fatal("too many errors")
			}
		}
	}
}

// Copied from gen.go.
func getName(p *ucd.Parser) string {
	s := p.String(ucd.Name)
	if s == "" {
		return ""
	}
	if s[0] == '<' {
		const first = ", First>"
		if i := strings.Index(s, first); i >= 0 {
			s = s[:i] + ">"
		}

	}
	return s
}
