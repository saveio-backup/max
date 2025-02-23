package crypto

import (
	"fmt"
	"io"

	pb "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto/pb"
	btcec "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWq5PJgAQKDWQerAijYUVKW8mN5MDatK5j7VMp8rizKQd/btcec"
	sha256 "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXTpwq2AkzQsPjKqFQDNY2bMdsAT53hUBETeyj8QRHTZU/sha256-simd"
)

type Secp256k1PrivateKey btcec.PrivateKey
type Secp256k1PublicKey btcec.PublicKey

func GenerateSecp256k1Key(src io.Reader) (PrivKey, PubKey, error) {
	privk, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}

	k := (*Secp256k1PrivateKey)(privk)
	return k, k.GetPublic(), nil
}

func UnmarshalSecp256k1PrivateKey(data []byte) (PrivKey, error) {
	if len(data) != btcec.PrivKeyBytesLen {
		return nil, fmt.Errorf("expected secp256k1 data size to be %d", btcec.PrivKeyBytesLen)
	}

	privk, _ := btcec.PrivKeyFromBytes(btcec.S256(), data)
	return (*Secp256k1PrivateKey)(privk), nil
}

func UnmarshalSecp256k1PublicKey(data []byte) (PubKey, error) {
	k, err := btcec.ParsePubKey(data, btcec.S256())
	if err != nil {
		return nil, err
	}

	return (*Secp256k1PublicKey)(k), nil
}

func (sk *Secp256k1PrivateKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

func (sk *Secp256k1PrivateKey) Bytes() ([]byte, error) {
	return MarshalPrivateKey(sk)
}

func (k *Secp256k1PrivateKey) Raw() ([]byte, error) {
	return (*btcec.PrivateKey)(k).Serialize(), nil
}

func (k *Secp256k1PrivateKey) Equals(o Key) bool {
	sk, ok := o.(*Secp256k1PrivateKey)
	if !ok {
		return false
	}

	return k.D.Cmp(sk.D) == 0
}

func (k *Secp256k1PrivateKey) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	sig, err := (*btcec.PrivateKey)(k).Sign(hash[:])
	if err != nil {
		return nil, err
	}

	return sig.Serialize(), nil
}

func (k *Secp256k1PrivateKey) GetPublic() PubKey {
	return (*Secp256k1PublicKey)((*btcec.PrivateKey)(k).PubKey())
}

func (sk *Secp256k1PublicKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

func (sk *Secp256k1PublicKey) Bytes() ([]byte, error) {
	return MarshalPublicKey(sk)
}

func (k *Secp256k1PublicKey) Raw() ([]byte, error) {
	return (*btcec.PublicKey)(k).SerializeCompressed(), nil
}

func (k *Secp256k1PublicKey) Equals(o Key) bool {
	sk, ok := o.(*Secp256k1PublicKey)
	if !ok {
		return false
	}

	return (*btcec.PublicKey)(k).IsEqual((*btcec.PublicKey)(sk))
}

func (k *Secp256k1PublicKey) Verify(data []byte, sigStr []byte) (bool, error) {
	sig, err := btcec.ParseDERSignature(sigStr, btcec.S256())
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(data)
	return sig.Verify(hash[:], (*btcec.PublicKey)(k)), nil
}
