// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpu_test

import (
	"runtime"
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVGjyM9i2msKvLXwh9VosCTgP4mL91kC7hDmqnwTTx6Hu/sys/cpu"
)

func TestAMD64minimalFeatures(t *testing.T) {
	if runtime.GOARCH == "amd64" {
		if !cpu.X86.HasSSE2 {
			t.Fatal("HasSSE2 expected true, got false")
		}
	}
}

func TestAVX2hasAVX(t *testing.T) {
	if runtime.GOARCH == "amd64" {
		if cpu.X86.HasAVX2 && !cpu.X86.HasAVX {
			t.Fatal("HasAVX expected true, got false")
		}
	}
}
