package crypto

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/saveio/dsp-go-sdk/types/suffix"
	"github.com/saveio/themis/crypto/encrypt"
	"github.com/saveio/themis/crypto/keypair"
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
	file := "/Users/smallyu/work/test/file/aaa"
	err := AESDecryptFile(file+".aes", "", "pwd", file+".aes.dec")
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
	input := "/Users/smallyu/work/test/file/test1/aaa.ecies"
	output := "/Users/smallyu/work/test/file/test1/aaa.decies"
	pri, _, err := keypair.GenerateKeyPairWithSeed(
		keypair.PK_ECDSA,
		bytes.NewReader([]byte("f1472f1fc52a8674d361b7e6af23ada4522526aca304b9729c5a9518b909f1b6")),
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
	text, err := GetCipherText(pub, password)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(text)
}
