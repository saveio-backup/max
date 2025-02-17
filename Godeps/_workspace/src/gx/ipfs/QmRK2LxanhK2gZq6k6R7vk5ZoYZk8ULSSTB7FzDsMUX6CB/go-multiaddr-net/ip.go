package manet

import (
	"bytes"

	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
)

// Loopback Addresses
var (
	// IP4Loopback is the ip4 loopback multiaddr
	IP4Loopback = ma.StringCast("/ip4/127.0.0.1")

	// IP6Loopback is the ip6 loopback multiaddr
	IP6Loopback = ma.StringCast("/ip6/::1")

	// IP6LinkLocalLoopback is the ip6 link-local loopback multiaddr
	IP6LinkLocalLoopback = ma.StringCast("/ip6/fe80::1")
)

// Unspecified Addresses (used for )
var (
	IP4Unspecified = ma.StringCast("/ip4/0.0.0.0")
	IP6Unspecified = ma.StringCast("/ip6/::")
)

// IsThinWaist returns whether a Multiaddr starts with "Thin Waist" Protocols.
// This means: /{IP4, IP6}[/{TCP, UDP}]
func IsThinWaist(m ma.Multiaddr) bool {
	p := m.Protocols()

	// nothing? not even a waist.
	if len(p) == 0 {
		return false
	}

	if p[0].Code != ma.P_IP4 && p[0].Code != ma.P_IP6 {
		return false
	}

	// only IP? still counts.
	if len(p) == 1 {
		return true
	}

	switch p[1].Code {
	case ma.P_TCP, ma.P_UDP, ma.P_IP4, ma.P_IP6:
		return true
	default:
		return false
	}
}

// IsIPLoopback returns whether a Multiaddr is a "Loopback" IP address
// This means either /ip4/127.0.0.1 or /ip6/::1
// TODO: differentiate IsIPLoopback and OverIPLoopback
func IsIPLoopback(m ma.Multiaddr) bool {
	b := m.Bytes()

	// /ip4/127 prefix (_entire_ /8 is loopback...)
	if bytes.HasPrefix(b, []byte{ma.P_IP4, 127}) {
		return true
	}

	// /ip6/::1
	if !m.Decapsulate(IP6Loopback).Equal(m) || !m.Decapsulate(IP6LinkLocalLoopback).Equal(m) {
		return true
	}

	return false
}

// IsIP6LinkLocal returns if a multiaddress is an IPv6 local link. These
// addresses are non routable. The prefix is technically
// fe80::/10, but we test fe80::/16 for simplicity (no need to mask).
// So far, no hardware interfaces exist long enough to use those 2 bits.
// Send a PR if there is.
func IsIP6LinkLocal(m ma.Multiaddr) bool {
	return bytes.HasPrefix(m.Bytes(), []byte{ma.P_IP6, 0xfe, 0x80})
}

// IsIPUnspecified returns whether a Multiaddr is am Unspecified IP address
// This means either /ip4/0.0.0.0 or /ip6/::
func IsIPUnspecified(m ma.Multiaddr) bool {
	return IP4Unspecified.Equal(m) || IP6Unspecified.Equal(m)
}
