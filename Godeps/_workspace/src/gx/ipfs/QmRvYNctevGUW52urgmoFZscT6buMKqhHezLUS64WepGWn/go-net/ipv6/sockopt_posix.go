// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris windows

package ipv6

import (
	"net"
	"unsafe"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/bpf"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/socket"
)

func (so *sockOpt) getMulticastInterface(c *socket.Conn) (*net.Interface, error) {
	n, err := so.GetInt(c)
	if err != nil {
		return nil, err
	}
	return net.InterfaceByIndex(n)
}

func (so *sockOpt) setMulticastInterface(c *socket.Conn, ifi *net.Interface) error {
	var n int
	if ifi != nil {
		n = ifi.Index
	}
	return so.SetInt(c, n)
}

func (so *sockOpt) getICMPFilter(c *socket.Conn) (*ICMPFilter, error) {
	b := make([]byte, so.Len)
	n, err := so.Get(c, b)
	if err != nil {
		return nil, err
	}
	if n != sizeofICMPv6Filter {
		return nil, errOpNoSupport
	}
	return (*ICMPFilter)(unsafe.Pointer(&b[0])), nil
}

func (so *sockOpt) setICMPFilter(c *socket.Conn, f *ICMPFilter) error {
	b := (*[sizeofICMPv6Filter]byte)(unsafe.Pointer(f))[:sizeofICMPv6Filter]
	return so.Set(c, b)
}

func (so *sockOpt) getMTUInfo(c *socket.Conn) (*net.Interface, int, error) {
	b := make([]byte, so.Len)
	n, err := so.Get(c, b)
	if err != nil {
		return nil, 0, err
	}
	if n != sizeofIPv6Mtuinfo {
		return nil, 0, errOpNoSupport
	}
	mi := (*ipv6Mtuinfo)(unsafe.Pointer(&b[0]))
	if mi.Addr.Scope_id == 0 {
		return nil, int(mi.Mtu), nil
	}
	ifi, err := net.InterfaceByIndex(int(mi.Addr.Scope_id))
	if err != nil {
		return nil, 0, err
	}
	return ifi, int(mi.Mtu), nil
}

func (so *sockOpt) setGroup(c *socket.Conn, ifi *net.Interface, grp net.IP) error {
	switch so.typ {
	case ssoTypeIPMreq:
		return so.setIPMreq(c, ifi, grp)
	case ssoTypeGroupReq:
		return so.setGroupReq(c, ifi, grp)
	default:
		return errOpNoSupport
	}
}

func (so *sockOpt) setSourceGroup(c *socket.Conn, ifi *net.Interface, grp, src net.IP) error {
	return so.setGroupSourceReq(c, ifi, grp, src)
}

func (so *sockOpt) setBPF(c *socket.Conn, f []bpf.RawInstruction) error {
	return so.setAttachFilter(c, f)
}
