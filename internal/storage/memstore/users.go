package memstore

import (
	"context"
	"errors"
	"sync"

	"github.com/ivarprudnikov/secretshare/internal/crypto"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

type memUserStore struct {
	users sync.Map
	salt  string
}

func NewMemUserStore(salt string) storage.UserStore {
	return &memUserStore{users: sync.Map{}, salt: salt}
}

func (u *memUserStore) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	u.users.Range(func(k, v any) bool {
		count++
		return true
	})
	return count, nil
}

func (u *memUserStore) AddUser(ctx context.Context, username string, password string, permissions []string) (*storage.User, error) {
	if _, ok := u.users.Load(username); ok {
		return nil, errors.New("username is not available")
	}
	usr, err := storage.NewUser(username, password, permissions)
	if err != nil {
		return nil, err
	}
	u.users.Store(usr.PartitionKey, usr)
	return &usr, nil
}

func (u *memUserStore) GetUser(ctx context.Context, username string) (*storage.User, error) {
	if v, ok := u.users.Load(username); ok {
		if usr, ok := v.(storage.User); ok {
			return &usr, nil
		}
	}
	return nil, nil
}

func (u *memUserStore) GetUserWithPass(ctx context.Context, username string, password string) (*storage.User, error) {
	if v, ok := u.users.Load(username); ok {
		if usr, ok := v.(storage.User); ok {
			if err := crypto.CompareHashToPass(usr.Password, password); err == nil {
				return &usr, nil
			}
		}
	}
	return nil, nil
}
