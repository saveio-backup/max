package ipns

import (
	"bytes"
	"errors"

	pb "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVHij7PuWUFeLcmRbD1ykDwB1WZMYP8yixo9bprUb3QHG/go-ipns/pb"

	ic "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto"
	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	record "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUTQSGgjs8CHm9yBcUHicpRs7C9abhyZiBwjzCUp1pNgX/go-libp2p-record"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	proto "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
)

var log = logging.Logger("ipns")

var _ record.Validator = Validator{}

// RecordKey returns the libp2p record key for a given peer ID.
func RecordKey(pid peer.ID) string {
	return "/ipns/" + string(pid)
}

// Validator is an IPNS record validator that satisfies the libp2p record
// validator interface.
type Validator struct {
	// KeyBook, if non-nil, will be used to lookup keys for validating IPNS
	// records.
	KeyBook pstore.KeyBook
}

// Validate validates an IPNS record.
func (v Validator) Validate(key string, value []byte) error {
	ns, pidString, err := record.SplitKey(key)
	if err != nil || ns != "ipns" {
		return ErrInvalidPath
	}

	// Parse the value into an IpnsEntry
	entry := new(pb.IpnsEntry)
	err = proto.Unmarshal(value, entry)
	if err != nil {
		return ErrBadRecord
	}

	// Get the public key defined by the ipns path
	pid, err := peer.IDFromString(pidString)
	if err != nil {
		log.Debugf("failed to parse ipns record key %s into peer ID", pidString)
		return ErrKeyFormat
	}

	pubk, err := v.getPublicKey(pid, entry)
	if err != nil {
		return err
	}

	return Validate(pubk, entry)
}

func (v Validator) getPublicKey(pid peer.ID, entry *pb.IpnsEntry) (ic.PubKey, error) {
	pk, err := ExtractPublicKey(pid, entry)
	if err != nil {
		return nil, err
	}
	if pk != nil {
		return pk, nil
	}

	if v.KeyBook == nil {
		log.Debugf("public key with hash %s not found in IPNS record and no peer store provided", pid)
		return nil, ErrPublicKeyNotFound
	}

	pubk := v.KeyBook.PubKey(pid)
	if pubk == nil {
		log.Debugf("public key with hash %s not found in peer store", pid)
		return nil, ErrPublicKeyNotFound
	}
	return pubk, nil
}

// Select selects the best record by checking which has the highest sequence
// number and latest EOL.
//
// This function returns an error if any of the records fail to parse. Validate
// your records first!
func (v Validator) Select(k string, vals [][]byte) (int, error) {
	var recs []*pb.IpnsEntry
	for _, v := range vals {
		e := new(pb.IpnsEntry)
		if err := proto.Unmarshal(v, e); err != nil {
			return -1, err
		}
		recs = append(recs, e)
	}

	return selectRecord(recs, vals)
}

func selectRecord(recs []*pb.IpnsEntry, vals [][]byte) (int, error) {
	switch len(recs) {
	case 0:
		return -1, errors.New("no usable records in given set")
	case 1:
		return 0, nil
	}

	var i int
	for j := 1; j < len(recs); j++ {
		cmp, err := Compare(recs[i], recs[j])
		if err != nil {
			return -1, err
		}
		if cmp == 0 {
			cmp = bytes.Compare(vals[i], vals[j])
		}
		if cmp < 0 {
			i = j
		}
	}

	return i, nil
}
