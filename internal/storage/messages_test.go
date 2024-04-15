package storage_test

import (
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/storage"
)

func TestMessageStore_GetMessage(t *testing.T) {
	// Create a new MessageStore instance
	store := storage.NewMemMessageStore("12345678123456781234567812345678")

	// Create a test message
	content := "testcontent testcontent testcontent testcontent testcontent testcontent"
	msg, err := store.AddMessage(content, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Invalid id
	foundMsg, err := store.GetMessage("foobar")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg != nil {
		t.Fatalf("Unexpected message")
	}

	// Get shallow message
	foundMsg, err = store.GetMessage(msg.Digest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg == nil {
		t.Fatalf("Expected message to be found, got nil")
	}
	if foundMsg.Content != "" {
		t.Fatalf("Expected no content, got %s", foundMsg.Content)
	}
	if foundMsg.Pin != "" {
		t.Fatalf("Expected no pin, got %s", foundMsg.Pin)
	}
}

func TestMessageStore_GetFullMessage(t *testing.T) {
	// Create a new MessageStore instance
	store := storage.NewMemMessageStore("12345678123456781234567812345678")

	// Create a test message
	content := "testcontent testcontent testcontent testcontent testcontent testcontent"
	msg, err := store.AddMessage(content, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// invalid Pin
	foundMsg, err := store.GetFullMessage(msg.Digest, "foobar")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg != nil {
		t.Fatalf("Expected no message")
	}

	// Get full message
	foundMsg, err = store.GetFullMessage(msg.Digest, msg.Pin)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg == nil {
		t.Fatalf("Expected message to be found, got nil")
	}
	if foundMsg.Content != content {
		t.Fatalf("Expected content %s, got %s", content, foundMsg.Content)
	}

	// Mesage was deleted after access
	msgs, err := store.ListMessages("testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("Expected no messages, got %d", len(msgs))
	}
}

func TestMessageStore_DeletedAfterFailedAttempts(t *testing.T) {
	// Create a new MessageStore instance
	store := storage.NewMemMessageStore("12345678123456781234567812345678")

	// Create a test message
	content := "testcontent testcontent testcontent testcontent testcontent testcontent"
	msg, err := store.AddMessage(content, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// before, the message exists
	foundMsg, _ := store.GetMessage(msg.Digest)
	if foundMsg == nil {
		t.Fatalf("Expected message to be found, got nil")
	}

	// invalid Pin
	for range storage.MAX_PIN_ATTEMPTS {
		store.GetFullMessage(msg.Digest, "invalidpin")
	}

	goneMessage, _ := store.GetMessage(msg.Digest)
	if goneMessage != nil {
		t.Fatalf("Expected the message to be deleted")
	}
}

func TestMessageStore_EncryptDecrypt(t *testing.T) {
	// Create a new MessageStore instance
	store := storage.NewMemMessageStore("12345678123456781234567812345678")

	message := "abc"
	key := "pass"
	ciphertext, err := store.Encrypt(message, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	plaintext, err := store.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if plaintext != message {
		t.Fatalf("Decrypted content does not match original %s != %s", message, plaintext)
	}
}
