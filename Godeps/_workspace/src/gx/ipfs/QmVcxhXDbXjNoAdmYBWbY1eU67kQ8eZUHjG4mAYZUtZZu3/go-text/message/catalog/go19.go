// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.9

package catalog

import "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/internal/catmsg"

// A Message holds a collection of translations for the same phrase that may
// vary based on the values of substitution arguments.
type Message = catmsg.Message

type firstInSequence = catmsg.FirstOf
