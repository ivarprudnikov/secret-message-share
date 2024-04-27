package crypto_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/crypto"
)

func TestHash_MakePin(t *testing.T) {
	pin, err := crypto.MakePin()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(pin) < 4 || len(pin) > 5 {
		t.Fatalf("unexpected pin lenght %s", pin)
	}
	re := regexp.MustCompile(`\d+`)
	if !re.Match([]byte(pin)) {
		t.Fatalf("unexpected pin %s", pin)
	}
}

func TestHash_StrongKey(t *testing.T) {
	salt := "12345678901234567890123456789012"
	key1, err := crypto.StrongKey("1234", salt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	key2, err := crypto.StrongKey("1234", salt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if slices.Compare(key1, key2) != 0 {
		t.Fatalf("Keys must match %s %s", key1, key2)
	}
}

func TestHash_EncryptDecrypt(t *testing.T) {
	key := []byte("1234567890123456")
	cipher, err := crypto.EncryptAES(key, "abc")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cipher == "abc" {
		t.Fatal("ciphertext should not be the same as input")
	}
	plaintext, err := crypto.DecryptAES(key, cipher)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if plaintext != "abc" {
		t.Fatal("failed to decrypt the content to the same state")
	}
}

func TestHash_HashText(t *testing.T) {
	digest := crypto.HashText("foobar")
	if digest != "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2" {
		t.Fatalf("unexpected digest %s", digest)
	}
}

func TestHash_HashPass_ThenCompare(t *testing.T) {
	hashed, err := crypto.HashPass("foobar")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.HasPrefix(hashed, "$argon2id$v=") {
		t.Fatalf("unexpected value %s", hashed)
	}

	err = crypto.CompareHashToPass(hashed, "foobar")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
