// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package unix_test

import (
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTsTrxKNXu8sZLv7FP6p884CHoDbT9upKA1j4FkCy5V7T/sys/unix"
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
