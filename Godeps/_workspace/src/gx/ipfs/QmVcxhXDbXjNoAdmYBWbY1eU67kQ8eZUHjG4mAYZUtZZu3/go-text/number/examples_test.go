// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package number_test

import (
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/language"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/message"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/number"
)

func ExampleMaxIntegerDigits() {
	const year = 1999
	p := message.NewPrinter(language.English)
	p.Println("Year:", number.Decimal(year, number.MaxIntegerDigits(2)))

	// Output:
	// Year: 99
}

func ExampleIncrementString() {
	p := message.NewPrinter(language.English)

	p.Println(number.Decimal(1.33, number.IncrementString("0.50")))

	// Output: 1.50
}
