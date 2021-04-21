package meterstream

import (
	inet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXfkENeeBvh3zYA51MaSdGUdBjhQ99cP5WQe8zgr6wchG/go-libp2p-net"
	protocol "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	metrics "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdeBtQGXjSt7cb97nx9JyLHHv5va2LyEAue7Q5tDFzpLy/go-libp2p-metrics"
)

type meteredStream struct {
	// keys for accessing metrics data
	protoKey protocol.ID
	peerKey  peer.ID

	inet.Stream

	// callbacks for reporting bandwidth usage
	mesSent metrics.StreamMeterCallback
	mesRecv metrics.StreamMeterCallback
}

func newMeteredStream(base inet.Stream, p peer.ID, recvCB, sentCB metrics.StreamMeterCallback) inet.Stream {
	return &meteredStream{
		Stream:   base,
		mesSent:  sentCB,
		mesRecv:  recvCB,
		protoKey: base.Protocol(),
		peerKey:  p,
	}
}

func WrapStream(base inet.Stream, bwc metrics.Reporter) inet.Stream {
	return newMeteredStream(base, base.Conn().RemotePeer(), bwc.LogRecvMessageStream, bwc.LogSentMessageStream)
}

func (s *meteredStream) Read(b []byte) (int, error) {
	n, err := s.Stream.Read(b)

	// Log bytes read
	s.mesRecv(int64(n), s.protoKey, s.peerKey)

	return n, err
}

func (s *meteredStream) Write(b []byte) (int, error) {
	n, err := s.Stream.Write(b)

	// Log bytes written
	s.mesSent(int64(n), s.protoKey, s.peerKey)

	return n, err
}
