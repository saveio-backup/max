// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package unix_test

import (
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTq8ag5pgTCqtGDtmpm1F5TPE2i1H8bcU6295WFKTc5ie/sys/unix"
)

func TestMmap(t *testing.T) {
	b, err := unix.Mmap(-1, 0, unix.Getpagesize(), unix.PROT_NONE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		t.Fatalf("Mmap: %v", err)
	}
	if err := unix.Munmap(b); err != nil {
		t.Fatalf("Munmap: %v", err)
	}
}
