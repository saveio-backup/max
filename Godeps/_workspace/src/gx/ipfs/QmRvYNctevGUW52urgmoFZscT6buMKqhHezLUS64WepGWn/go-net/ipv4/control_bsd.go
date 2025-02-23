// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin dragonfly freebsd netbsd openbsd

package ipv4

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/iana"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/socket"
)

func marshalDst(b []byte, cm *ControlMessage) []byte {
	m := socket.ControlMessage(b)
	m.MarshalHeader(iana.ProtocolIP, sysIP_RECVDSTADDR, net.IPv4len)
	return m.Next(net.IPv4len)
}

func parseDst(cm *ControlMessage, b []byte) {
	if len(cm.Dst) < net.IPv4len {
		cm.Dst = make(net.IP, net.IPv4len)
	}
	copy(cm.Dst, b[:net.IPv4len])
}

func marshalInterface(b []byte, cm *ControlMessage) []byte {
	m := socket.ControlMessage(b)
	m.MarshalHeader(iana.ProtocolIP, sysIP_RECVIF, syscall.SizeofSockaddrDatalink)
	return m.Next(syscall.SizeofSockaddrDatalink)
}

func parseInterface(cm *ControlMessage, b []byte) {
	sadl := (*syscall.SockaddrDatalink)(unsafe.Pointer(&b[0]))
	cm.IfIndex = int(sadl.Index)
}
