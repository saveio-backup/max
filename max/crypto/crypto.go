package crypto

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/saveio/dsp-go-sdk/types/suffix"
	"github.com/saveio/themis/crypto/encrypt"
	"io"
	"os"
	"strings"

	"github.com/saveio/themis/crypto/keypair"
	"golang.org/x/crypto/scrypt"
)

type SymmetricScheme byte

const (
	AES SymmetricScheme = iota
)

var names = []string{
	"AES",
}

func GetScheme(name string) (SymmetricScheme, error) {
	for i, v := range names {
		if strings.ToUpper(v) == strings.ToUpper(name) {
			return SymmetricScheme(i), nil
		}
	}
	return 0, errors.New("unknown symmetric scheme " + name)
}

func kdf(pwd []byte, salt []byte) (dKey []byte, err error) {
	param := keypair.GetScryptParameters()
	if param.DKLen < 32 {
		err = errors.New("derived key length too short")
		return nil, err
	}
	// Derive the encryption key
	dKey, err = scrypt.Key([]byte(pwd), salt, param.N, param.R, param.P, param.DKLen)
	return dKey, err
}
func randomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// AESEncryptFile encrypt file and save file locally.
// The file include first 32 bytes of salt data, and the remains are encrypted data
func AESEncryptFile(file string, password string, out string) error {
	salt, err := randomBytes(16)
	if err != nil {
		return errors.New("salt generate error")
	}
	dKey, err := kdf([]byte(password), salt)
	if err != nil {
		return err
	}
	nonce := dKey[:16]
	eKey := dKey[len(dKey)-16:]

	inFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer inFile.Close()

	block, err := aes.NewCipher(eKey)
	if err != nil {
		return err
	}
	stream := cipher.NewOFB(block, nonce)
	outFile, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer outFile.Close()

	tag := []byte{}
	for _, v := range salt {
		tag = append(tag, v)
	}
	encAlg := make([]byte, 16)
	encAlg[15] = byte(AES)
	for _, v := range encAlg {
		tag = append(tag, v)
	}
	_, err = outFile.WriteAt(tag, 0)
	outFile.Seek(int64(len(tag)), io.SeekStart)

	writer := &cipher.StreamWriter{S: stream, W: outFile}
	if _, err := io.Copy(writer, inFile); err != nil {
		return err
	}
	return err
}

func AESEncryptFileReader(inFile *os.File, password string) (io.Reader, error) {
	salt, err := randomBytes(16)
	if err != nil {
		return nil, errors.New("salt generate error")
	}
	dKey, err := kdf([]byte(password), salt)
	if err != nil {
		return nil, err
	}
	nonce := dKey[:16]
	eKey := dKey[len(dKey)-16:]

	block, err := aes.NewCipher(eKey)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewOFB(block, nonce)
	reader := &cipher.StreamReader{S: stream, R: inFile}
	tag := []byte{}
	for _, v := range salt {
		tag = append(tag, v)
	}
	encAlg := make([]byte, 16)
	encAlg[15] = byte(AES)
	for _, v := range encAlg {
		tag = append(tag, v)
	}
	r := io.MultiReader(bytes.NewReader(tag), reader)
	return r, nil
}

// AESDecryptFile. use AES algorithm to decrypt a file
// The file is include first 32 bytes of salt data, the prefix data if exists, and remains are encrypted data
func AESDecryptFile(file, prefix, password, out string) error {
	inFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer inFile.Close()
	extData := make([]byte, 32)
	// read salt data after skip first N bytes of prefix
	inFile.ReadAt(extData, int64(len(prefix)))
	// skip extra data and prefix data from the begining
	_, err = inFile.Seek(int64(len(extData)+len(prefix)), io.SeekStart)
	if err != nil {
		return err
	}
	salt := extData[:16]
	dKey, err := kdf([]byte(password), salt)
	if err != nil {
		return err
	}
	nonce := dKey[:16]
	eKey := dKey[len(dKey)-16:]
	encAlg := extData[31]
	if encAlg != byte(AES) {
		return errors.New("unknown algorithm")
	}
	block, err := aes.NewCipher(eKey)
	if err != nil {
		return err
	}

	stream := cipher.NewOFB(block, nonce)
	outFile, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer outFile.Close()
	reader := &cipher.StreamReader{S: stream, R: inFile}
	// Copy the input file to the output file, decrypting as we go.
	if _, err := io.Copy(outFile, reader); err != nil {
		return err
	}
	return nil
}

func AESDecryptFileWriter(inFile *os.File, password string) (io.Writer, error) {
	extData := make([]byte, 32)
	inFile.ReadAt(extData, 0)
	_, err := inFile.Seek(int64(len(extData)), io.SeekStart)
	if err != nil {
		return nil, err
	}
	salt := extData[:16]
	dKey, err := kdf([]byte(password), salt)
	if err != nil {
		return nil, err
	}
	nonce := dKey[:16]
	eKey := dKey[len(dKey)-16:]
	encAlg := extData[31]
	if encAlg != byte(AES) {
		return nil, errors.New("unknown algorithm")
	}
	block, err := aes.NewCipher(eKey)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewOFB(block, nonce)
	writer := &cipher.StreamWriter{S: stream}
	return writer, nil
}

var eciesScheme = encrypt.AES128withSHA256

func ECIESEncryptFile(file string, out string, pubKey crypto.PublicKey) error {
	err := os.Remove(out)
	if err != nil {
		// ignore this error
	}
	password, err := suffix.GenerateRandomPassword()
	if err != nil {
		return err
	}
	err = AESEncryptFile(file, string(password), out)
	if err != nil {
		return err
	}
	ct, err := encrypt.Encrypt(eciesScheme, pubKey, password, nil, nil)
	if err != nil {
		return err
	}
	// encode convenient for debug
	ctStr := hex.EncodeToString(ct)
	err = suffix.AddCipherTextSuffixToFile([]byte(ctStr), out)
	if err != nil {
		return err
	}
	return nil
}

func ECIESDecryptFile(file, prefix, out string, priKey crypto.PrivateKey) error {
	// create a temp file because need cut suffix data
	tmp := file + ".tmp"
	tmpFile, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	openFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer openFile.Close()
	_, err = io.Copy(tmpFile, openFile)
	// read cipher text from file
	ctStr, err := suffix.ReadCipherTextFromFile(tmp)
	if err != nil {
		return err
	}
	err = suffix.CutCipherTextFromFile(tmp)
	if err != nil {
		return err
	}
	ct, err := hex.DecodeString(string(ctStr))
	if err != nil {
		return err
	}
	password, err := encrypt.Decrypt(priKey, ct, nil, nil)
	if err != nil {
		return err
	}
	err = AESDecryptFile(tmp, prefix, string(password), out)
	if err != nil {
		return err
	}
	err = os.Remove(tmp)
	if err != nil {
		return err
	}
	return nil
}
