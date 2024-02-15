package storage

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// simple text hashing
func HashText(text string) string {
	textHash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(textHash[:])
}

// simple pin generator
func MakePin() (string, error) {
	c := 16
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	pin := binary.BigEndian.Uint16(b)
	return fmt.Sprintf("%d", pin), nil
}

// use hkdf to derive a strong key (RFC5869)
func StrongKey(passText string, saltText string) ([]byte, error) {
	hashFn := sha256.New
	hashSize := hashFn().Size()
	passBytes := []byte(passText)
	if len(saltText) != hashSize {
		return nil, fmt.Errorf("invalid salt length, must be %d", hashSize)
	}
	saltBytes := []byte(saltText)
	hkdf := hkdf.New(hashFn, passBytes, saltBytes, nil)
	key := make([]byte, 16) // 128-bit key
	if _, err := io.ReadFull(hkdf, key); err != nil {
		return nil, err
	}
	return key, nil
}

// use strong key to encrypt text
func EncryptAES(key []byte, plaintext string) (string, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	out := make([]byte, len(plaintext))
	// FIXME pad the text to reach minimum block size
	c.Encrypt(out, []byte(plaintext))
	return hex.EncodeToString(out), nil
}

func DecryptAES(key []byte, ct string) (string, error) {
	ciphertext, _ := hex.DecodeString(ct)
	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	pt := make([]byte, len(ciphertext))
	c.Decrypt(pt, ciphertext)
	return string(pt[:]), nil
}
