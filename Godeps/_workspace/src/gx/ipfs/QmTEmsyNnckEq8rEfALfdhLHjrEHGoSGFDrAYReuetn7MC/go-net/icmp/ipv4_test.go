// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icmp

import (
	"net"
	"reflect"
	"runtime"
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTEmsyNnckEq8rEfALfdhLHjrEHGoSGFDrAYReuetn7MC/go-net/ipv4"
)

var (
	wireHeaderFromKernel = [ipv4.HeaderLen]byte{
		0x45, 0x01, 0xbe, 0xef,
		0xca, 0xfe, 0x45, 0xdc,
		0xff, 0x01, 0xde, 0xad,
		172, 16, 254, 254,
		192, 168, 0, 1,
	}
	wireHeaderFromTradBSDKernel = [ipv4.HeaderLen]byte{
		0x45, 0x01, 0xef, 0xbe,
		0xca, 0xfe, 0x45, 0xdc,
		0xff, 0x01, 0xde, 0xad,
		172, 16, 254, 254,
		192, 168, 0, 1,
	}
	// TODO(mikio): Add platform dependent wire header formats when
	// we support new platforms.

	testHeader = &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      1,
		TotalLen: 0xbeef,
		ID:       0xcafe,
		Flags:    ipv4.DontFragment,
		FragOff:  1500,
		TTL:      255,
		Protocol: 1,
		Checksum: 0xdead,
		Src:      net.IPv4(172, 16, 254, 254),
		Dst:      net.IPv4(192, 168, 0, 1),
	}
)

func TestParseIPv4Header(t *testing.T) {
	var wh []byte
	switch runtime.GOOS {
	case "darwin":
		wh = wireHeaderFromTradBSDKernel[:]
	case "freebsd":
		if freebsdVersion >= 1000000 {
			wh = wireHeaderFromKernel[:]
		} else {
			wh = wireHeaderFromTradBSDKernel[:]
		}
	default:
		wh = wireHeaderFromKernel[:]
	}
	h, err := ParseIPv4Header(wh)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h, testHeader) {
		t.Fatalf("got %#v; want %#v", h, testHeader)
	}
}
