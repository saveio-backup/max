// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"log"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/gen"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/language"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/language/compact"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/unicode/cldr"
)

func main() {
	r := gen.OpenCLDRCoreZip()
	defer r.Close()

	d := &cldr.Decoder{}
	data, err := d.DecodeZip(r)
	if err != nil {
		log.Fatalf("DecodeZip: %v", err)
	}

	w := gen.NewCodeWriter()
	defer w.WriteGoFile("parents.go", "compact")

	// Create parents table.
	type ID uint16
	parents := make([]ID, compact.NumCompactTags)
	for _, loc := range data.Locales() {
		tag := language.MustParse(loc)
		index, ok := compact.FromTag(tag)
		if !ok {
			continue
		}
		parentIndex := compact.ID(0) // und
		for p := tag.Parent(); p != language.Und; p = p.Parent() {
			if x, ok := compact.FromTag(p); ok {
				parentIndex = x
				break
			}
		}
		parents[index] = ID(parentIndex)
	}

	w.WriteComment(`
	parents maps a compact index of a tag to the compact index of the parent of
	this tag.`)
	w.WriteVar("parents", parents)
}
