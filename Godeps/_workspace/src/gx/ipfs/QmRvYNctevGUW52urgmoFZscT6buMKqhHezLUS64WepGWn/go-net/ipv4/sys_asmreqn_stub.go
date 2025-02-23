// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin,!freebsd,!linux

package ipv4

import (
	"net"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/socket"
)

func (so *sockOpt) getIPMreqn(c *socket.Conn) (*net.Interface, error) {
	return nil, errOpNoSupport
}

func (so *sockOpt) setIPMreqn(c *socket.Conn, ifi *net.Interface, grp net.IP) error {
	return errOpNoSupport
}
