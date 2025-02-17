// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin

package ipv6

import (
	"syscall"
	"unsafe"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTEmsyNnckEq8rEfALfdhLHjrEHGoSGFDrAYReuetn7MC/go-net/internal/iana"
)

func marshal2292HopLimit(b []byte, cm *ControlMessage) []byte {
	m := (*syscall.Cmsghdr)(unsafe.Pointer(&b[0]))
	m.Level = iana.ProtocolIPv6
	m.Type = sysIPV6_2292HOPLIMIT
	m.SetLen(syscall.CmsgLen(4))
	if cm != nil {
		data := b[syscall.CmsgLen(0):]
		// TODO(mikio): fix potential misaligned memory access
		*(*int32)(unsafe.Pointer(&data[:4][0])) = int32(cm.HopLimit)
	}
	return b[syscall.CmsgSpace(4):]
}

func marshal2292PacketInfo(b []byte, cm *ControlMessage) []byte {
	m := (*syscall.Cmsghdr)(unsafe.Pointer(&b[0]))
	m.Level = iana.ProtocolIPv6
	m.Type = sysIPV6_2292PKTINFO
	m.SetLen(syscall.CmsgLen(sysSizeofInet6Pktinfo))
	if cm != nil {
		pi := (*sysInet6Pktinfo)(unsafe.Pointer(&b[syscall.CmsgLen(0)]))
		if ip := cm.Src.To16(); ip != nil && ip.To4() == nil {
			copy(pi.Addr[:], ip)
		}
		if cm.IfIndex > 0 {
			pi.setIfindex(cm.IfIndex)
		}
	}
	return b[syscall.CmsgSpace(sysSizeofInet6Pktinfo):]
}

func marshal2292NextHop(b []byte, cm *ControlMessage) []byte {
	m := (*syscall.Cmsghdr)(unsafe.Pointer(&b[0]))
	m.Level = iana.ProtocolIPv6
	m.Type = sysIPV6_2292NEXTHOP
	m.SetLen(syscall.CmsgLen(sysSizeofSockaddrInet6))
	if cm != nil {
		sa := (*sysSockaddrInet6)(unsafe.Pointer(&b[syscall.CmsgLen(0)]))
		sa.setSockaddr(cm.NextHop, cm.IfIndex)
	}
	return b[syscall.CmsgSpace(sysSizeofSockaddrInet6):]
}
