package crypto

import (
	"bytes"
	"fmt"
	"io"

	pb "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto/pb"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmQ51pHe6u7CWodkUGDLqaCEMchkbMt7VEZnECF5mp6tVb/ed25519"
	extra "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmQ51pHe6u7CWodkUGDLqaCEMchkbMt7VEZnECF5mp6tVb/ed25519/extra25519"
)

type Ed25519PrivateKey struct {
	sk *[64]byte
	pk *[32]byte
}

type Ed25519PublicKey struct {
	k *[32]byte
}

func GenerateEd25519Key(src io.Reader) (PrivKey, PubKey, error) {
	pub, priv, err := ed25519.GenerateKey(src)
	if err != nil {
		return nil, nil, err
	}

	return &Ed25519PrivateKey{
			sk: priv,
			pk: pub,
		},
		&Ed25519PublicKey{
			k: pub,
		},
		nil
}

func (k *Ed25519PrivateKey) Type() pb.KeyType {
	return pb.KeyType_Ed25519
}

func (k *Ed25519PrivateKey) Bytes() ([]byte, error) {
	return MarshalPrivateKey(k)
}

func (k *Ed25519PrivateKey) Raw() ([]byte, error) {
	buf := make([]byte, 96)
	copy(buf, k.sk[:])
	copy(buf[64:], k.pk[:])

	return buf, nil
}

func (k *Ed25519PrivateKey) Equals(o Key) bool {
	edk, ok := o.(*Ed25519PrivateKey)
	if !ok {
		return false
	}

	return bytes.Equal((*k.sk)[:], (*edk.sk)[:]) && bytes.Equal((*k.pk)[:], (*edk.pk)[:])
}

func (k *Ed25519PrivateKey) GetPublic() PubKey {
	return &Ed25519PublicKey{k.pk}
}

func (k *Ed25519PrivateKey) Sign(msg []byte) ([]byte, error) {
	out := ed25519.Sign(k.sk, msg)
	return (*out)[:], nil
}

func (k *Ed25519PrivateKey) ToCurve25519() *[32]byte {
	var sk [32]byte
	extra.PrivateKeyToCurve25519(&sk, k.sk)
	return &sk
}

func (k *Ed25519PublicKey) Type() pb.KeyType {
	return pb.KeyType_Ed25519
}

func (k *Ed25519PublicKey) Bytes() ([]byte, error) {
	return MarshalPublicKey(k)
}

func (k *Ed25519PublicKey) Raw() ([]byte, error) {
	return (*k.k)[:], nil
}

func (k *Ed25519PublicKey) Equals(o Key) bool {
	edk, ok := o.(*Ed25519PublicKey)
	if !ok {
		return false
	}

	return bytes.Equal((*k.k)[:], (*edk.k)[:])
}

func (k *Ed25519PublicKey) Verify(data []byte, sig []byte) (bool, error) {
	var asig [64]byte
	copy(asig[:], sig)
	return ed25519.Verify(k.k, data, &asig), nil
}

func (k *Ed25519PublicKey) ToCurve25519() (*[32]byte, error) {
	var pk [32]byte
	success := extra.PublicKeyToCurve25519(&pk, k.k)
	if !success {
		return nil, fmt.Errorf("Error converting ed25519 pubkey to curve25519 pubkey")
	}
	return &pk, nil
}

func UnmarshalEd25519PublicKey(data []byte) (PubKey, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("expect ed25519 public key data size to be 32")
	}

	var pub [32]byte
	copy(pub[:], data)

	return &Ed25519PublicKey{
		k: &pub,
	}, nil
}

func UnmarshalEd25519PrivateKey(data []byte) (PrivKey, error) {
	if len(data) != 96 {
		return nil, fmt.Errorf("expected ed25519 data size to be 96")
	}
	var priv [64]byte
	var pub [32]byte
	copy(priv[:], data)
	copy(pub[:], data[64:])

	return &Ed25519PrivateKey{
		sk: &priv,
		pk: &pub,
	}, nil
}
