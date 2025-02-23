// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ianaindex_test

import (
	"fmt"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ/go-text/encoding/charmap"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ/go-text/encoding/ianaindex"
)

func ExampleIndex() {
	fmt.Println(ianaindex.MIME.Name(charmap.ISO8859_7))

	fmt.Println(ianaindex.IANA.Name(charmap.ISO8859_7))

	e, _ := ianaindex.IANA.Get("cp437")
	fmt.Println(ianaindex.IANA.Name(e))

	// TODO: Output:
	// ISO-8859-7
	// ISO8859_7:1987
	// IBM437
}
