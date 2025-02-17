package swarm

import (
	mamask "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	mafilter "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSW4uNHbvQia8iZDXzbwjiyHQtnyo9aFqfQAMasj3TJ6Y/go-maddr-filter"
)

// http://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml
var lowTimeoutFilters = mafilter.NewFilters()

func init() {
	for _, p := range []string{
		"/ip4/10.0.0.0/ipcidr/8",
		"/ip4/100.64.0.0/ipcidr/10",
		"/ip4/169.254.0.0/ipcidr/16",
		"/ip4/172.16.0.0/ipcidr/12",
		"/ip4/192.0.0.0/ipcidr/24",
		"/ip4/192.0.0.0/ipcidr/29",
		"/ip4/192.0.0.8/ipcidr/32",
		"/ip4/192.0.0.170/ipcidr/32",
		"/ip4/192.0.0.171/ipcidr/32",
		"/ip4/192.0.2.0/ipcidr/24",
		"/ip4/192.168.0.0/ipcidr/16",
		"/ip4/198.18.0.0/ipcidr/15",
		"/ip4/198.51.100.0/ipcidr/24",
		"/ip4/203.0.113.0/ipcidr/24",
		"/ip4/240.0.0.0/ipcidr/4",
	} {
		f, err := mamask.NewMask(p)
		if err != nil {
			panic("error in lowTimeoutFilters init: " + err.Error())
		}
		lowTimeoutFilters.AddDialFilter(f)
	}
}
