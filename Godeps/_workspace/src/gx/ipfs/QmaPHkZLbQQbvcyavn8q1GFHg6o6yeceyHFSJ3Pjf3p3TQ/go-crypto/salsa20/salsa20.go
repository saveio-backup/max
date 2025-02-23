// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package salsa20 implements the Salsa20 stream cipher as specified in http://cr.yp.to/snuffle/spec.pdf.

Salsa20 differs from many other stream ciphers in that it is message orientated
rather than byte orientated. Keystream blocks are not preserved between calls,
therefore each side must encrypt/decrypt data with the same segmentation.

Another aspect of this difference is that part of the counter is exposed as
an nonce in each call. Encrypting two different messages with the same (key,
nonce) pair leads to trivial plaintext recovery. This is analogous to
encrypting two different messages with the same key with a traditional stream
cipher.

This package also implements XSalsa20: a version of Salsa20 with a 24-byte
nonce as specified in http://cr.yp.to/snuffle/xsalsa-20081128.pdf. Simply
passing a 24-byte slice as the nonce triggers XSalsa20.
*/
package salsa20

// TODO(agl): implement XORKeyStream12 and XORKeyStream8 - the reduced round variants of Salsa20.

import (
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaPHkZLbQQbvcyavn8q1GFHg6o6yeceyHFSJ3Pjf3p3TQ/go-crypto/salsa20/salsa"
)

// XORKeyStream crypts bytes from in to out using the given key and nonce. In
// and out may be the same slice but otherwise should not overlap. Nonce must
// be either 8 or 24 bytes long.
func XORKeyStream(out, in []byte, nonce []byte, key *[32]byte) {
	if len(out) < len(in) {
		in = in[:len(out)]
	}

	var subNonce [16]byte

	if len(nonce) == 24 {
		var subKey [32]byte
		var hNonce [16]byte
		copy(hNonce[:], nonce[:16])
		salsa.HSalsa20(&subKey, &hNonce, key, &salsa.Sigma)
		copy(subNonce[:], nonce[16:])
		key = &subKey
	} else if len(nonce) == 8 {
		copy(subNonce[:], nonce[:])
	} else {
		panic("salsa20: nonce must be 8 or 24 bytes")
	}

	salsa.XORKeyStream(out, in, &subNonce, key)
}
