package swarm

import (
	"fmt"
	"strings"

	transport "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUMTtHxeyVJPrpcpvEQppH3uTf3g1NnkRC6C36LpXy2no/go-libp2p-transport"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
)

// TransportForDialing retrieves the appropriate transport for dialing the given
// multiaddr.
func (s *Swarm) TransportForDialing(a ma.Multiaddr) transport.Transport {
	protocols := a.Protocols()
	if len(protocols) == 0 {
		return nil
	}

	s.transports.RLock()
	defer s.transports.RUnlock()
	for _, p := range protocols {
		transport, ok := s.transports.m[p.Code]
		if !ok {
			continue
		}
		if transport.Proxy() {
			return transport
		}
	}

	return s.transports.m[protocols[len(protocols)-1].Code]
}

// TransportForListening retrieves the appropriate transport for listening on
// the given multiaddr.
func (s *Swarm) TransportForListening(a ma.Multiaddr) transport.Transport {
	protocols := a.Protocols()
	if len(protocols) == 0 {
		return nil
	}

	s.transports.RLock()
	defer s.transports.RUnlock()
	selected := s.transports.m[protocols[len(protocols)-1].Code]
	for _, p := range protocols {
		transport, ok := s.transports.m[p.Code]
		if !ok {
			continue
		}
		if transport.Proxy() {
			selected = transport
		}
	}
	return selected
}

// AddTransport adds a transport to this swarm.
//
// Satisfies the Network interface from go-libp2p-transport.
func (s *Swarm) AddTransport(t transport.Transport) error {
	protocols := t.Protocols()

	s.transports.Lock()
	defer s.transports.Unlock()
	var registered []string
	for _, p := range protocols {
		if _, ok := s.transports.m[p]; ok {
			proto := ma.ProtocolWithCode(p)
			name := proto.Name
			if name == "" {
				name = fmt.Sprintf("unknown (%d)", p)
			}
			registered = append(registered, name)
		}
	}
	if len(registered) > 0 {
		return fmt.Errorf(
			"transports already registered for protocol(s): %s",
			strings.Join(registered, ", "),
		)
	}

	for _, p := range protocols {
		s.transports.m[p] = t
	}
	return nil
}
