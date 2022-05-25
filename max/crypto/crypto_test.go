package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/saveio/dsp-go-sdk/types/suffix"
	"github.com/saveio/themis/crypto/encrypt"
	"github.com/saveio/themis/crypto/keypair"
	"strings"
	"testing"
)

func TestAESEncryptFile(t *testing.T) {
	file := "/Users/smallyu/work/test/file/aaa"
	err := AESEncryptFile(file, "pwd", file+".aes")
	if err != nil {
		t.Error(err)
	}
}

func TestAESDecryptFile(t *testing.T) {
	file := "/Users/smallyu/work/gogs/edge-deploy/node1/Chain-1/Downloads/AYKnc5VDkvpb5f68XSjTyQzVHU4ZaojGxq/SaveQmQjKfjVEgZdMhNMm2p9bNwqsd3P1zK8VQQrwmY6d9TaZf.ept/t2.ept"
	pwd := []byte{
		104, 40, 37, 132, 94, 53, 202, 206,
	}
	err := AESDecryptFile(file,
		"AAAAgw==AQGzMTQEidQl1OVsoZbzQLDr9luQQvRiLyx82BuJwRxmF9cw6h8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAANi9Vc2Vycy9zbWFsbHl1L3dvcmsvZ29ncy9lZGdlLWRlcGxveS9ub2RlMS90ZXN0MTI4L2FhYQAAAAAAAtg9eJo=",
		string(pwd),
		file+".aes.dec")
	if err != nil {
		t.Error(err)
	}
}

func TestECIESEncryptFile(t *testing.T) {
	input := "/Users/smallyu/work/test/file/test1/aaa"
	output := "/Users/smallyu/work/test/file/test1/aaa.ecies"
	_, pub, err := keypair.GenerateKeyPairWithSeed(
		keypair.PK_ECDSA,
		bytes.NewReader([]byte("f1472f1fc52a8674d361b7e6af23ada4522526aca304b9729c5a9518b909f1b6")),
		keypair.P256,
	)
	err = ECIESEncryptFile(input, output, pub)
	if err != nil {
		t.Error(err)
	}
}

func TestECIESDecryptFile(t *testing.T) {
	input := "/Users/smallyu/work/gogs/edge-deploy/node1/Chain-1/Downloads/AYKnc5VDkvpb5f68XSjTyQzVHU4ZaojGxq/SaveQmWeAW8UgeSrMY8BnfpBxCS8EA5WHoooTvxpfjvFxK1shs.ept/ttt"
	output := "/Users/smallyu/work/gogs/edge-deploy/node1/Chain-1/Downloads/AYKnc5VDkvpb5f68XSjTyQzVHU4ZaojGxq/SaveQmWeAW8UgeSrMY8BnfpBxCS8EA5WHoooTvxpfjvFxK1shs.ept/ttt.d"
	pri, _, err := keypair.GenerateKeyPairWithSeed(
		keypair.PK_ECDSA,
		bytes.NewReader([]byte("d341a64f05d3f46bd87220d29df183356aa5770ce7cb85cd063707e9fe163d27")),
		keypair.P256,
	)
	err = ECIESDecryptFile(input, "", output, pri)
	if err != nil {
		t.Error(err)
	}
}

func TestEciesLength(t *testing.T) {
	_, pub2, err := keypair.GenerateKeyPair(keypair.PK_ECIES, keypair.P256)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := encrypt.Encrypt(encrypt.AES128withSHA256, pub2, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(len(ct))

	message2 := []byte("Hello, world. Hello, world.")
	ct2, err := encrypt.Encrypt(encrypt.AES128withSHA256, pub2, message2, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(len(ct2))

	var salt [8]byte
	_, _ = rand.Read(salt[:])
	ct3, err := encrypt.Encrypt(encrypt.AES128withSHA256, pub2, salt[:], nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(len(ct3))
}

func TestGetCipherText(t *testing.T) {
	_, pub, err := keypair.GenerateKeyPairWithSeed(
		keypair.PK_ECDSA,
		bytes.NewReader([]byte("d341a64f05d3f46bd87220d29df183356aa5770ce7cb85cd063707e9fe163d27")),
		keypair.P256,
	)
	password, err := suffix.GenerateRandomPassword()
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(password)
	ct, err := GetCipherText(pub, password)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(ct)
	ctStr := hex.EncodeToString(ct)
	var cipherKey [suffix.SuffixLength]byte
	copy(cipherKey[:], ctStr)
	t.Log(ctStr)
	t.Log(string(cipherKey[:]))
}

func TestGetPwdFromCipherText(t *testing.T) {
	privateKey, err := hex.DecodeString(strings.TrimPrefix("d341a64f05d3f46bd87220d29df183356aa5770ce7cb85cd063707e9fe163d27", "0x"))
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	pri, _, err := keypair.GenerateKeyPairWithSeed(
		keypair.PK_ECDSA,
		bytes.NewReader(privateKey),
		keypair.P256,
	)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	ctStr := "00042e4a07c9df565257ca614dc337856f2de13e43083213157be0e8da9513ef9a432128d49770615bb71ec10a353574985ef624c76bf3112ac22f002fdb6804d64c6f4062387b5698b2509941ff4e827c0a052df5877000d03fe926c2c070e76d8dfd73b76025a689262fec51637836a4ff0ca53236b272ba0b"
	ct, err := hex.DecodeString(ctStr)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	password, err := encrypt.Decrypt(pri, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(password)
}
