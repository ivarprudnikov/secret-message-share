package storage

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
)

type UserStore interface {
	AddUser(username string, password string) (*User, error)
	GetUser(username string) (*User, error)
	GetUserWithPass(username string, password string) (*User, error)
}

type User struct {
	aztables.Entity
	Password string `json:"password"`
}

func (u *User) FormattedDate() string {
	t := time.Time(u.Timestamp)
	return t.Format(time.RFC822)
}

func NewUser(username string, password string) (User, error) {
	hashedPass, err := HashPass(password)
	if err != nil {
		return User{}, err
	}
	t := time.Now()
	return User{
		Entity: aztables.Entity{
			PartitionKey: username,
			RowKey:       username,
			Timestamp:    aztables.EDMDateTime(t),
		},
		Password: hashedPass,
	}, nil
}
