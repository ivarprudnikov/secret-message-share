package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const keySizeBytes = 16 // 128-bit key

// simple text hashing
func HashText(text string) string {
	textHash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(textHash[:])
}

// simple pin generator
func MakePin() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	pin := binary.BigEndian.Uint16(b)
	return fmt.Sprintf("%d", pin), nil
}

func MakeToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// use hkdf to derive a strong key (RFC5869)
// maybe switch to bcrypt or scrypt
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

// use strong key to encrypt text
func EncryptAES(key []byte, plaintext string) (string, error) {
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return "", fmt.Errorf("failed to wrap cipher: %w", err)
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed create nonce: %w", err)
	}

	// ciphertext here is actually nonce+ciphertext
	// So that when we decrypt, just knowing the nonce size
	// is enough to separate it from the ciphertext.
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return hex.EncodeToString(ciphertext), nil
}

func DecryptAES(key []byte, ct string) (string, error) {
	ciphertext, _ := hex.DecodeString(ct)
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return "", fmt.Errorf("failed to wrap cipher: %w", err)
	}

	// Since we know the ciphertext is actually nonce+ciphertext
	// And len(nonce) == NonceSize(). We can separate the two.
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
