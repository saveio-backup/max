package filter

import (
	"net"
	"sync"

	manet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmV6FjemM1K8oXjrvuq3wuVWWoU2TLDPmNnKrxHzY3v6Ai/go-multiaddr-net"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
)

type Filters struct {
	mu      sync.RWMutex
	filters map[string]*net.IPNet
}

func NewFilters() *Filters {
	return &Filters{
		filters: make(map[string]*net.IPNet),
	}
}

func (fs *Filters) AddDialFilter(f *net.IPNet) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.filters[f.String()] = f
}

func (f *Filters) AddrBlocked(a ma.Multiaddr) bool {
	maddr := ma.Split(a)
	if len(maddr) == 0 {
		return false
	}
	netaddr, err := manet.ToNetAddr(maddr[0])
	if err != nil {
		// if we cant parse it, its probably not blocked
		return false
	}
	netip := net.ParseIP(netaddr.String())
	if netip == nil {
		return false
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, ft := range f.filters {
		if ft.Contains(netip) {
			return true
		}
	}
	return false
}

func (f *Filters) Filters() []*net.IPNet {
	var out []*net.IPNet
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, ff := range f.filters {
		out = append(out, ff)
	}
	return out
}

func (f *Filters) Remove(ff *net.IPNet) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.filters, ff.String())
}
