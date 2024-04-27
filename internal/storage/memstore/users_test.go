package memstore_test

import (
	"context"
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/storage"
	"github.com/ivarprudnikov/secretshare/internal/storage/memstore"
)

func TestUserStore_GetUserWithPass(t *testing.T) {
	// Create a new UserStore instance
	store := memstore.NewMemUserStore("123")

	// Create a test user
	username := "testuser"
	password := "testpassword"

	store.AddUser(context.Background(), username, password, []string{storage.PERMISSION_READ_STATS})

	// Test case 1: Valid username and password
	foundUser, err := store.GetUserWithPass(context.Background(), username, password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundUser == nil {
		t.Fatalf("Expected user to be found, got nil")
	}
	if foundUser.PartitionKey != username {
		t.Fatalf("Expected username %s, got %s", username, foundUser.PartitionKey)
	}

	// Test case 2: Invalid password
	invalidPassword := "wrongpassword"
	foundUser, err = store.GetUserWithPass(context.Background(), username, invalidPassword)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundUser != nil {
		t.Fatalf("Expected user to be nil, got %v", foundUser)
	}

	// Test case 3: Invalid username
	invalidUsername := "invaliduser"
	foundUser, err = store.GetUserWithPass(context.Background(), invalidUsername, password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundUser != nil {
		t.Fatalf("Expected user to be nil, got %v", foundUser)
	}
}
