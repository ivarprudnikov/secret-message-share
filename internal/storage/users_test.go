package storage_test

import (
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/storage"
)

func TestUserStore_GetUserWithPass(t *testing.T) {
	// Create a new UserStore instance
	store := storage.NewUserStore("123", storage.NewLocalClient())

	// Create a test user
	username := "testuser"
	password := "testpassword"

	store.AddUser(username, password)

	// Test case 1: Valid username and password
	foundUser, err := store.GetUserWithPass(username, password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundUser == nil {
		t.Fatalf("Expected user to be found, got nil")
	}
	if foundUser.Username != username {
		t.Fatalf("Expected username %s, got %s", username, foundUser.Username)
	}

	// Test case 2: Invalid password
	invalidPassword := "wrongpassword"
	foundUser, err = store.GetUserWithPass(username, invalidPassword)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundUser != nil {
		t.Fatalf("Expected user to be nil, got %v", foundUser)
	}

	// Test case 3: Invalid username
	invalidUsername := "invaliduser"
	foundUser, err = store.GetUserWithPass(invalidUsername, password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if foundUser != nil {
		t.Fatalf("Expected user to be nil, got %v", foundUser)
	}
}
