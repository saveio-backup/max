package dshelp

import (
	"testing"

	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

func TestKey(t *testing.T) {
	c, _ := cid.Decode("QmP63DkAFEnDYNjDYBpyNDfttu1fvUw99x1brscPzpqmmq")
	dsKey := CidToDsKey(c)
	c2, err := DsKeyToCid(dsKey)
	if err != nil {
		t.Fatal(err)
	}
	if c.String() != c2.String() {
		t.Fatal("should have parsed the same key")
	}
}
