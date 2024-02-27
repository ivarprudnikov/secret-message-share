package storage_test

import (
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/storage"
)

func TestHash_EncryptDecrypt(t *testing.T) {
	key := []byte("1234567890123456")
	cipher, err := storage.EncryptAES(key, "abc")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cipher == "abc" {
		t.Fatal("ciphertext should not be the same as input")
	}
	plaintext, err := storage.DecryptAES(key, cipher)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if plaintext != "abc" {
		t.Fatal("failed to decrypt the content to the same state")
	}
}

func TestHash_HashText(t *testing.T) {
	digest := storage.HashText("foobar")
	if digest != "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2" {
		t.Fatalf("unexpected digest %s", digest)
	}
}
