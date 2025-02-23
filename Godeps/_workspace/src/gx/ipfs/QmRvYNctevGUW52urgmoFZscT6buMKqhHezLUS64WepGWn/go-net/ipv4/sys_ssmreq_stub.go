// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin,!freebsd,!linux,!solaris

package ipv4

import (
	"net"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/socket"
)

func (so *sockOpt) setGroupReq(c *socket.Conn, ifi *net.Interface, grp net.IP) error {
	return errOpNoSupport
}

func (so *sockOpt) setGroupSourceReq(c *socket.Conn, ifi *net.Interface, grp, src net.IP) error {
	return errOpNoSupport
}
