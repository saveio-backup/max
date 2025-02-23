package record

import (
	"bytes"
	"errors"
	"fmt"

	u "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPdKqUcHGFdeSpvjVoaTRPPstGif9GBZb5Q56RVw9o69A/go-ipfs-util"
	mh "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
)

// PublicKeyValidator is a Validator that validates public keys.
type PublicKeyValidator struct{}

// Validate conforms to the Validator interface.
//
// It verifies that the passed in record value is the PublicKey that matches the
// passed in key.
func (pkv PublicKeyValidator) Validate(key string, value []byte) error {
	ns, key, err := SplitKey(key)
	if err != nil {
		return err
	}
	if ns != "pk" {
		return errors.New("namespace not 'pk'")
	}

	keyhash := []byte(key)
	if _, err := mh.Cast(keyhash); err != nil {
		return fmt.Errorf("key did not contain valid multihash: %s", err)
	}

	pkh := u.Hash(value)
	if !bytes.Equal(keyhash, pkh) {
		return errors.New("public key does not match storage key")
	}
	return nil
}

// Select conforms to the Validator interface.
//
// It always returns 0 as all public keys are equivalently valid.
func (pkv PublicKeyValidator) Select(k string, vals [][]byte) (int, error) {
	return 0, nil
}

var _ Validator = PublicKeyValidator{}
