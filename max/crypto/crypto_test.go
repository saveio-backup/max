package crypto

import (
	"github.com/pborman/uuid"
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
	err := CEIESEncryptFile(file, "pwd", file+".ceies")
	if err != nil {
		t.Error(err)
	}
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
