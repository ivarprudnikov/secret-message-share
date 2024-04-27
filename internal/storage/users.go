package storage

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/ivarprudnikov/secretshare/internal/crypto"
)

const PERMISSION_READ_STATS = "read:stats"

type UserStore interface {
	CountUsers(ctx context.Context) (int64, error)
	AddUser(ctx context.Context, username string, password string, permissions []string) (*User, error)
	GetUser(ctx context.Context, username string) (*User, error)
	GetUserWithPass(ctx context.Context, username string, password string) (*User, error)
}

type User struct {
	aztables.Entity
	Password    string
	Permissions string
}

func (u *User) FormattedDate() string {
	t := time.Time(u.Timestamp)
	return t.Format(time.RFC822)
}

func (u *User) HasPermission(permission string) bool {
	for _, v := range strings.Split(u.Permissions, ",") {
		if permission == v {
			return true
		}
	}
	return false
}

func NewUser(username string, password string, permissions []string) (User, error) {
	hashedPass, err := crypto.HashPass(password)
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
		Password:    hashedPass,
		Permissions: strings.Join(permissions, ","),
	}, nil
}
