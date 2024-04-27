package storage_test

import (
	"encoding/json"
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/crypto"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

func TestUser(t *testing.T) {
	usr, err := storage.NewUser("foo", "bar", []string{"baz", "bau"})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	marshalled, err := json.Marshal(usr)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	var user *storage.User
	err = json.Unmarshal(marshalled, &user)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if user.PartitionKey != "foo" {
		t.Fatalf("Unexpected output %v", user.PartitionKey)
	}
	if user.RowKey != "foo" {
		t.Fatalf("Unexpected output %v", user.RowKey)
	}
	if user.Permissions != "baz,bau" {
		t.Fatalf("Unexpected output %v", user.Permissions)
	}
	err = crypto.CompareHashToPass(user.Password, "bar")
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if !user.HasPermission("baz") {
		t.Fatalf("user should have permission baz")
	}
	if !user.HasPermission("bau") {
		t.Fatalf("user should have permission bau")
	}
	if user.HasPermission("foo") {
		t.Fatalf("user should not have arbitrary permission")
	}
}
