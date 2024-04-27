package storage

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/ivarprudnikov/secretshare/internal/crypto"
)

const MAX_PIN_ATTEMPTS = 5

type MessageStore interface {
	CountMessages(ctx context.Context) (int64, error)
	ListMessages(ctx context.Context, username string) ([]*Message, error)
	AddMessage(ctx context.Context, text string, username string) (*Message, error)
	GetMessage(ctx context.Context, id string) (*Message, error)
	GetFullMessage(ctx context.Context, id string, pin string) (*Message, error)
	Encrypt(text, pass, salt string) (string, error)
	Decrypt(ciphertext, pass, salt string) (string, error)
}

type Message struct {
	aztables.Entity
	Content           string
	Pin               string
	AttemptsRemaining int
}

func (m *Message) FormattedDate() string {
	t := time.Time(m.Timestamp)
	return t.Format(time.RFC822)
}

func NewMessage(username string, ciphertext string, pin string) (Message, error) {
	pinHash, err := crypto.HashPass(pin)
	if err != nil {
		return Message{}, err
	}
	t := time.Now()
	return Message{
		Entity: aztables.Entity{
			PartitionKey: crypto.HashText(ciphertext),
			RowKey:       username,
			Timestamp:    aztables.EDMDateTime(t),
		},
		Content:           ciphertext,
		Pin:               pinHash,
		AttemptsRemaining: MAX_PIN_ATTEMPTS,
	}, nil
}
