package storage

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
)

const MAX_PIN_ATTEMPTS = 5

type MessageStore interface {
	ListMessages(username string) ([]Message, error)
	AddMessage(text string, username string) (Message, error)
	GetMessage(id string) (*Message, error)
	GetFullMessage(id string, pin string) (*Message, error)
	Encrypt(text string, pass string) (string, error)
	Decrypt(ciphertext string, pass string) (string, error)
}

type Message struct {
	aztables.Entity
	Content string `json:"content,omitempty"`
	Pin     string `json:"pin,omitempty"`
	Attempt int    `json:"Attempt,omitempty"`
}

func NewMessage(username string, ciphertext string, pin string) (Message, error) {
	pinHash, err := HashPass(pin)
	if err != nil {
		return Message{}, err
	}
	t := time.Now()
	return Message{
		Entity: aztables.Entity{
			PartitionKey: HashText(ciphertext),
			RowKey:       username,
			Timestamp:    aztables.EDMDateTime(t),
		},
		Content: ciphertext,
		Pin:     pinHash,
		Attempt: 0,
	}, nil
}
