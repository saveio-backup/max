package cldr_test

import (
	"fmt"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ/go-text/unicode/cldr"
)

func ExampleSlice() {
	var dr *cldr.CLDR // assume this is initalized

	x, _ := dr.LDML("en")
	cs := x.Collations.Collation
	// remove all but the default
	cldr.MakeSlice(&cs).Filter(func(e cldr.Elem) bool {
		return e.GetCommon().Type != x.Collations.Default()
	})
	for i, c := range cs {
		fmt.Println(i, c.Type)
	}
}
