package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

type EntityEncryptHelper struct{}

func (e EntityEncryptHelper) Encrypt(text, pass, salt string) (string, error) {
	// derive a key from the pass
	key, err := StrongKey(pass, salt)
	if err != nil {
		return "", err
	}
	ciphertext, err := EncryptAES(key, text)
	if err != nil {
		return "", err
	}
	return ciphertext, nil
}

// Decrypt cipher text with a given PIN which will be used to derive a key
func (e EntityEncryptHelper) Decrypt(ciphertext, pass, salt string) (string, error) {
	// derive a key from the pass
	key, err := StrongKey(pass, salt)
	if err != nil {
		return "", err
	}
	plaintext, err := DecryptAES(key, ciphertext)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

// Encrypts the plain text with a given key
// see StrongKey() to create one
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

// Decrypt the cipher text using the key provided
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
