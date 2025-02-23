// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.9

package ipv6_test

import (
	"bytes"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/iana"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/internal/nettest"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/ipv6"
)

func BenchmarkPacketConnReadWriteUnicast(b *testing.B) {
	switch runtime.GOOS {
	case "nacl", "plan9", "windows":
		b.Skipf("not supported on %s", runtime.GOOS)
	}

	payload := []byte("HELLO-R-U-THERE")
	iph := []byte{
		0x69, 0x8b, 0xee, 0xf1, 0xca, 0xfe, 0xff, 0x01,
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x02, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	greh := []byte{0x00, 0x00, 0x86, 0xdd, 0x00, 0x00, 0x00, 0x00}
	datagram := append(greh, append(iph, payload...)...)
	bb := make([]byte, 128)
	cm := ipv6.ControlMessage{
		TrafficClass: iana.DiffServAF11 | iana.CongestionExperienced,
		HopLimit:     1,
		Src:          net.IPv6loopback,
	}
	if ifi := nettest.RoutedInterface("ip6", net.FlagUp|net.FlagLoopback); ifi != nil {
		cm.IfIndex = ifi.Index
	}

	b.Run("UDP", func(b *testing.B) {
		c, err := nettest.NewLocalPacketListener("udp6")
		if err != nil {
			b.Skipf("not supported on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
		}
		defer c.Close()
		p := ipv6.NewPacketConn(c)
		dst := c.LocalAddr()
		cf := ipv6.FlagHopLimit | ipv6.FlagInterface
		if err := p.SetControlMessage(cf, true); err != nil {
			b.Fatal(err)
		}
		wms := []ipv6.Message{
			{
				Buffers: [][]byte{payload},
				Addr:    dst,
				OOB:     cm.Marshal(),
			},
		}
		rms := []ipv6.Message{
			{
				Buffers: [][]byte{bb},
				OOB:     ipv6.NewControlMessage(cf),
			},
		}
		b.Run("Net", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := c.WriteTo(payload, dst); err != nil {
					b.Fatal(err)
				}
				if _, _, err := c.ReadFrom(bb); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("ToFrom", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := p.WriteTo(payload, &cm, dst); err != nil {
					b.Fatal(err)
				}
				if _, _, _, err := p.ReadFrom(bb); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("Batch", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := p.WriteBatch(wms, 0); err != nil {
					b.Fatal(err)
				}
				if _, err := p.ReadBatch(rms, 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
	b.Run("IP", func(b *testing.B) {
		switch runtime.GOOS {
		case "netbsd":
			b.Skip("need to configure gre on netbsd")
		case "openbsd":
			b.Skip("net.inet.gre.allow=0 by default on openbsd")
		}

		c, err := net.ListenPacket(fmt.Sprintf("ip6:%d", iana.ProtocolGRE), "::1")
		if err != nil {
			b.Skipf("not supported on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
		}
		defer c.Close()
		p := ipv6.NewPacketConn(c)
		dst := c.LocalAddr()
		cf := ipv6.FlagTrafficClass | ipv6.FlagHopLimit | ipv6.FlagSrc | ipv6.FlagDst | ipv6.FlagInterface | ipv6.FlagPathMTU
		if err := p.SetControlMessage(cf, true); err != nil {
			b.Fatal(err)
		}
		wms := []ipv6.Message{
			{
				Buffers: [][]byte{datagram},
				Addr:    dst,
				OOB:     cm.Marshal(),
			},
		}
		rms := []ipv6.Message{
			{
				Buffers: [][]byte{bb},
				OOB:     ipv6.NewControlMessage(cf),
			},
		}
		b.Run("Net", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := c.WriteTo(datagram, dst); err != nil {
					b.Fatal(err)
				}
				if _, _, err := c.ReadFrom(bb); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("ToFrom", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := p.WriteTo(datagram, &cm, dst); err != nil {
					b.Fatal(err)
				}
				if _, _, _, err := p.ReadFrom(bb); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("Batch", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := p.WriteBatch(wms, 0); err != nil {
					b.Fatal(err)
				}
				if _, err := p.ReadBatch(rms, 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

func TestPacketConnConcurrentReadWriteUnicast(t *testing.T) {
	switch runtime.GOOS {
	case "nacl", "plan9", "windows":
		t.Skipf("not supported on %s", runtime.GOOS)
	}

	payload := []byte("HELLO-R-U-THERE")
	iph := []byte{
		0x69, 0x8b, 0xee, 0xf1, 0xca, 0xfe, 0xff, 0x01,
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x02, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	greh := []byte{0x00, 0x00, 0x86, 0xdd, 0x00, 0x00, 0x00, 0x00}
	datagram := append(greh, append(iph, payload...)...)

	t.Run("UDP", func(t *testing.T) {
		c, err := nettest.NewLocalPacketListener("udp6")
		if err != nil {
			t.Skipf("not supported on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
		}
		defer c.Close()
		p := ipv6.NewPacketConn(c)
		t.Run("ToFrom", func(t *testing.T) {
			testPacketConnConcurrentReadWriteUnicast(t, p, payload, c.LocalAddr(), false)
		})
		t.Run("Batch", func(t *testing.T) {
			testPacketConnConcurrentReadWriteUnicast(t, p, payload, c.LocalAddr(), true)
		})
	})
	t.Run("IP", func(t *testing.T) {
		switch runtime.GOOS {
		case "netbsd":
			t.Skip("need to configure gre on netbsd")
		case "openbsd":
			t.Skip("net.inet.gre.allow=0 by default on openbsd")
		}

		c, err := net.ListenPacket(fmt.Sprintf("ip6:%d", iana.ProtocolGRE), "::1")
		if err != nil {
			t.Skipf("not supported on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
		}
		defer c.Close()
		p := ipv6.NewPacketConn(c)
		t.Run("ToFrom", func(t *testing.T) {
			testPacketConnConcurrentReadWriteUnicast(t, p, datagram, c.LocalAddr(), false)
		})
		t.Run("Batch", func(t *testing.T) {
			testPacketConnConcurrentReadWriteUnicast(t, p, datagram, c.LocalAddr(), true)
		})
	})
}

func testPacketConnConcurrentReadWriteUnicast(t *testing.T, p *ipv6.PacketConn, data []byte, dst net.Addr, batch bool) {
	ifi := nettest.RoutedInterface("ip6", net.FlagUp|net.FlagLoopback)
	cf := ipv6.FlagTrafficClass | ipv6.FlagHopLimit | ipv6.FlagSrc | ipv6.FlagDst | ipv6.FlagInterface | ipv6.FlagPathMTU

	if err := p.SetControlMessage(cf, true); err != nil { // probe before test
		if nettest.ProtocolNotSupported(err) {
			t.Skipf("not supported on %s", runtime.GOOS)
		}
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	reader := func() {
		defer wg.Done()
		b := make([]byte, 128)
		n, cm, _, err := p.ReadFrom(b)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(b[:n], data) {
			t.Errorf("got %#v; want %#v", b[:n], data)
			return
		}
		s := cm.String()
		if strings.Contains(s, ",") {
			t.Errorf("should be space-separated values: %s", s)
			return
		}
	}
	batchReader := func() {
		defer wg.Done()
		ms := []ipv6.Message{
			{
				Buffers: [][]byte{make([]byte, 128)},
				OOB:     ipv6.NewControlMessage(cf),
			},
		}
		n, err := p.ReadBatch(ms, 0)
		if err != nil {
			t.Error(err)
			return
		}
		if n != len(ms) {
			t.Errorf("got %d; want %d", n, len(ms))
			return
		}
		var cm ipv6.ControlMessage
		if err := cm.Parse(ms[0].OOB[:ms[0].NN]); err != nil {
			t.Error(err)
			return
		}
		b := ms[0].Buffers[0][:ms[0].N]
		if !bytes.Equal(b, data) {
			t.Errorf("got %#v; want %#v", b, data)
			return
		}
		s := cm.String()
		if strings.Contains(s, ",") {
			t.Errorf("should be space-separated values: %s", s)
			return
		}
	}
	writer := func(toggle bool) {
		defer wg.Done()
		cm := ipv6.ControlMessage{
			TrafficClass: iana.DiffServAF11 | iana.CongestionExperienced,
			HopLimit:     1,
			Src:          net.IPv6loopback,
		}
		if ifi != nil {
			cm.IfIndex = ifi.Index
		}
		if err := p.SetControlMessage(cf, toggle); err != nil {
			t.Error(err)
			return
		}
		n, err := p.WriteTo(data, &cm, dst)
		if err != nil {
			t.Error(err)
			return
		}
		if n != len(data) {
			t.Errorf("got %d; want %d", n, len(data))
			return
		}
	}
	batchWriter := func(toggle bool) {
		defer wg.Done()
		cm := ipv6.ControlMessage{
			TrafficClass: iana.DiffServAF11 | iana.CongestionExperienced,
			HopLimit:     1,
			Src:          net.IPv6loopback,
		}
		if ifi != nil {
			cm.IfIndex = ifi.Index
		}
		if err := p.SetControlMessage(cf, toggle); err != nil {
			t.Error(err)
			return
		}
		ms := []ipv6.Message{
			{
				Buffers: [][]byte{data},
				OOB:     cm.Marshal(),
				Addr:    dst,
			},
		}
		n, err := p.WriteBatch(ms, 0)
		if err != nil {
			t.Error(err)
			return
		}
		if n != len(ms) {
			t.Errorf("got %d; want %d", n, len(ms))
			return
		}
		if ms[0].N != len(data) {
			t.Errorf("got %d; want %d", ms[0].N, len(data))
			return
		}
	}

	const N = 10
	wg.Add(N)
	for i := 0; i < N; i++ {
		if batch {
			go batchReader()
		} else {
			go reader()
		}
	}
	wg.Add(2 * N)
	for i := 0; i < 2*N; i++ {
		if batch {
			go batchWriter(i%2 != 0)
		} else {
			go writer(i%2 != 0)
		}
	}
	wg.Add(N)
	for i := 0; i < N; i++ {
		if batch {
			go batchReader()
		} else {
			go reader()
		}
	}
	wg.Wait()
}
