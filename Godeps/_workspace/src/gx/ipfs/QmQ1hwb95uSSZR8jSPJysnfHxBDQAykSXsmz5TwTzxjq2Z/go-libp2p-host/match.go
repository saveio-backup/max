package host

import (
	"strings"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	semver "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcrrEpx3VMUbrbgVroH3YiYyUS5c4YAykzyPJWKspUYLa/go-semver/semver"
)

// MultistreamSemverMatcher returns a matcher function for a given base protocol.
// The matcher function will return a boolean indicating whether a protocol ID
// matches the base protocol. A given protocol ID matches the base protocol if
// the IDs are the same and if the semantic version of the base protocol is the
// same or higher than that of the protocol ID provided.
func MultistreamSemverMatcher(base protocol.ID) (func(string) bool, error) {
	parts := strings.Split(string(base), "/")
	vers, err := semver.NewVersion(parts[len(parts)-1])
	if err != nil {
		return nil, err
	}

	return func(check string) bool {
		chparts := strings.Split(check, "/")
		if len(chparts) != len(parts) {
			return false
		}

		for i, v := range chparts[:len(chparts)-1] {
			if parts[i] != v {
				return false
			}
		}

		chvers, err := semver.NewVersion(chparts[len(chparts)-1])
		if err != nil {
			return false
		}

		return vers.Major == chvers.Major && vers.Minor >= chvers.Minor
	}, nil
}
