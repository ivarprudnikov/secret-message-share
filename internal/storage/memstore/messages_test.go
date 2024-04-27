package memstore_test

import (
	"context"
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/storage"
	"github.com/ivarprudnikov/secretshare/internal/storage/memstore"
)

func TestMessageStore_GetMessage(t *testing.T) {
	// Create a new MessageStore instance
	store := memstore.NewMemMessageStore("12345678123456781234567812345678")

	// Create a test message
	content := "testcontent testcontent testcontent testcontent testcontent testcontent"
	msg, err := store.AddMessage(context.Background(), content, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Invalid id
	foundMsg, err := store.GetMessage(context.Background(), "foobar")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg != nil {
		t.Fatalf("Unexpected message")
	}

	// Get encrypted message
	foundMsg, err = store.GetMessage(context.Background(), msg.PartitionKey)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg == nil {
		t.Fatalf("Expected message to be found, got nil")
	}
	if foundMsg.Content == content {
		t.Fatalf("Expected encrypted content, got %s", foundMsg.Content)
	}
	if foundMsg.Pin == msg.Pin {
		t.Fatalf("Expected encrypted pin, got %s", foundMsg.Pin)
	}
}

func TestMessageStore_GetFullMessage(t *testing.T) {
	// Create a new MessageStore instance
	store := memstore.NewMemMessageStore("12345678123456781234567812345678")

	// Create a test message
	content := "testcontent testcontent testcontent testcontent testcontent testcontent"
	msg, err := store.AddMessage(context.Background(), content, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// invalid Pin
	foundMsg, err := store.GetFullMessage(context.Background(), msg.PartitionKey, "foobar")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg != nil {
		t.Fatalf("Expected no message")
	}

	// Get full message
	foundMsg, err = store.GetFullMessage(context.Background(), msg.PartitionKey, msg.Pin)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundMsg == nil {
		t.Fatalf("Expected message to be found, got nil")
	}
	if foundMsg.Content != content {
		t.Fatalf("Expected content %s, got %s", content, foundMsg.Content)
	}

	// Message was deleted after access
	msgs, err := store.ListMessages(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("Expected no messages, got %d", len(msgs))
	}
}

func TestMessageStore_DeletedAfterFailedAttempts(t *testing.T) {
	// Create a new MessageStore instance
	store := memstore.NewMemMessageStore("12345678123456781234567812345678")

	// Create a test message
	content := "testcontent testcontent testcontent testcontent testcontent testcontent"
	msg, err := store.AddMessage(context.Background(), content, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// before, the message exists
	foundMsg, _ := store.GetMessage(context.Background(), msg.PartitionKey)
	if foundMsg == nil {
		t.Fatalf("Expected message to be found, got nil")
	}

	// invalid Pin
	for range storage.MAX_PIN_ATTEMPTS {
		store.GetFullMessage(context.Background(), msg.PartitionKey, "invalidpin")
	}

	goneMessage, _ := store.GetMessage(context.Background(), msg.PartitionKey)
	if goneMessage != nil {
		t.Fatalf("Expected the message to be deleted")
	}
}

func TestMessageStore_EncryptDecrypt(t *testing.T) {
	// Create a new MessageStore instance
	salt := "12345678123456781234567812345678"
	store := memstore.NewMemMessageStore(salt)

	message := "abc"
	key := "pass"
	ciphertext, err := store.Encrypt(message, key, salt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	plaintext, err := store.Decrypt(ciphertext, key, salt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if plaintext != message {
		t.Fatalf("Decrypted content does not match original %s != %s", message, plaintext)
	}
}
