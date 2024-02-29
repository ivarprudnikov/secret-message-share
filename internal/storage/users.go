package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type UserStore struct {
	client StorageClient
	salt   string
}

func NewUserStore(salt string, client StorageClient) *UserStore {
	return &UserStore{client: client, salt: salt}
}

func (u *UserStore) AddUser(username string, password string) (User, error) {
	existing, err := u.client.GetOne(context.Background(), username, username)
	if err != nil {
		return User{}, fmt.Errorf("failed to add user: %w", err)
	}
	if existing != nil {
		return User{}, errors.New("username is not available")
	}
	usr := NewUser(username, password)
	marshalled, err := json.Marshal(usr)
	if err != nil {
		return User{}, fmt.Errorf("failed to marshal user: %w", err)
	}
	u.client.SaveOne(context.Background(), usr.Username, usr.Username, marshalled)
	return usr, nil
}

func (u *UserStore) GetUser(username string) (*User, error) {
	usr, err := u.getUser(context.Background(), username)
	if err != nil {
		return nil, err
	}
	if usr != nil {
		return &User{
			Username: usr.Username,
			Created:  usr.Created,
		}, nil
	}
	return nil, nil
}

func (u *UserStore) GetUserWithPass(username string, password string) (*User, error) {
	usr, err := u.getUser(context.Background(), username)
	if err != nil {
		return nil, err
	}
	if usr != nil && usr.Password == HashText(password) {
		return &User{
			Username: usr.Username,
			Created:  usr.Created,
		}, nil
	}
	return nil, nil
}

func (u *UserStore) getUser(ctx context.Context, username string) (*User, error) {
	usr, err := u.client.GetOne(context.Background(), username, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if usr != nil {
		var u User
		err = json.Unmarshal(usr, &u)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal user: %w", err)
		}
		return &u, nil
	}
	return nil, nil
}

type User struct {
	Username string    `json:"username"`
	Password string    `json:"password"`
	Created  time.Time `json:"created"`
}

func NewUser(username string, password string) User {
	return User{
		Username: username,
		// TODO use salt for hashing to mitigate rainbow attacks
		Password: HashText(password),
		Created:  time.Now(),
	}
}
