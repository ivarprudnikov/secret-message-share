package storage

import (
	"errors"
	"sync"
	"time"
)

type UserStore interface {
	AddUser(username string, password string) (User, error)
	GetUser(username string) (*User, error)
	GetUserWithPass(username string, password string) (*User, error)
}

type memUserStore struct {
	users sync.Map
	salt  string
}

func NewMemUserStore(salt string) UserStore {
	return &memUserStore{users: sync.Map{}, salt: salt}
}

func (u *memUserStore) AddUser(username string, password string) (User, error) {
	if _, ok := u.users.Load(username); ok {
		return User{}, errors.New("username is not available")
	}
	usr := NewUser(username, password)
	u.users.Store(usr.Username, usr)
	return usr, nil
}

func (u *memUserStore) GetUser(username string) (*User, error) {
	if v, ok := u.users.Load(username); ok {
		if usr, ok := v.(User); ok {
			return &User{
				Username: usr.Username,
				Created:  usr.Created,
			}, nil
		}
	}
	return nil, nil
}

func (u *memUserStore) GetUserWithPass(username string, password string) (*User, error) {
	if v, ok := u.users.Load(username); ok {
		if usr, ok := v.(User); ok {
			// TODO use salted hash
			if usr.Password == HashText(password) {
				return &User{
					Username: usr.Username,
					Created:  usr.Created,
				}, nil
			}
		}
	}
	return nil, nil
}

type User struct {
	Id       string    `json:"id"`
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
