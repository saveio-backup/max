package crypto

import (
	"fmt"
	"github.com/pborman/uuid"
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

func TestCEIESEncryptFile(t *testing.T) {
	file := "/Users/smallyu/work/test/file/aaa"
	s, err := CEIESEncryptFile(file, "pwd", file+".ceies", nil)
	if err != nil {
		t.Error(err)
	}
	t.Log(s)
}

func TestCEIESDecryptFile(t *testing.T) {
	file := "/Users/smallyu/work/test/file/aaa"
	err := CEIESDecryptFile(file+".ceies", "", "pwd", file+".ceies.dec", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestUUID(t *testing.T) {
	u := uuid.New()
	t.Log(u)
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

	message3 := []byte("Hello, world. Hello, world. Hello, world.")
	ct3, err := encrypt.Encrypt(encrypt.AES128withSHA256, pub2, message3, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	t.Log(len(ct3))
}
