package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// simple pin number generator
// returns 4-5 digits
func MakePin() (string, error) {
	b, err := generateRandomBytes(16)
	if err != nil {
		return "", err
	}
	pin := binary.BigEndian.Uint16(b)
	return fmt.Sprintf("%d", pin), nil
}

// simple random token generator (url encoded)
func MakeToken() (string, error) {
	b, err := generateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// simple text hashing
func HashText(text string) string {
	textHash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(textHash[:])
}

const keySizeBytes = 16 // 128-bit key

// Use HMAC Key Derivation Function (HKDF) to derive a strong key (RFC5869)
// This function is useful to derive a key from some small password text the user knows
func StrongKey(passText string, saltText string) ([]byte, error) {
	hashFn := sha256.New
	hashSize := hashFn().Size()
	passBytes := []byte(passText)
	if len(saltText) != hashSize {
		return nil, fmt.Errorf("invalid salt length, must be %d", hashSize)
	}
	saltBytes := []byte(saltText)
	hkdf := hkdf.New(hashFn, passBytes, saltBytes, nil)
	key := make([]byte, keySizeBytes)
	if _, err := io.ReadFull(hkdf, key); err != nil {
		return nil, err
	}
	return key, nil
}

// Generate cryptographically secure random of a given length
func generateRandomBytes(size uint32) ([]byte, error) {
	bucket := make([]byte, size)
	if _, err := rand.Read(bucket); err != nil {
		return nil, err
	}
	return bucket, nil
}
