package memstore

import (
	"fmt"
	"sync"

	"github.com/ivarprudnikov/secretshare/internal/crypto"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

type memMessageStore struct {
	crypto.EntityEncryptHelper
	messages sync.Map
	salt     string
}

func NewMemMessageStore(salt string) storage.MessageStore {
	return &memMessageStore{messages: sync.Map{}, salt: salt}
}

func (s *memMessageStore) CountMessages() (int64, error) {
	var count int64
	s.messages.Range(func(k, v any) bool {
		count++
		return true
	})
	return count, nil
}

func (s *memMessageStore) ListMessages(username string) ([]*storage.Message, error) {
	var msgs []*storage.Message
	s.messages.Range(func(k, v any) bool {
		if msg, ok := v.(storage.Message); ok && msg.RowKey == username {
			msgs = append(msgs, &msg)
		}
		return true
	})
	return msgs, nil
}

// TODO: allow to reset the pin for the owner
func (s *memMessageStore) AddMessage(text string, username string) (*storage.Message, error) {
	// an easy to enter pin
	pin, err := crypto.MakePin()
	if err != nil {
		return nil, err
	}
	ciphertext, err := s.Encrypt(text, pin, s.salt)
	if err != nil {
		return nil, err
	}
	msg, err := storage.NewMessage(username, ciphertext, pin)
	if err != nil {
		return nil, err
	}
	// store unreadbale message, pin
	s.messages.Store(msg.Entity.PartitionKey, msg)
	// temporarily show the pin to the creator
	msg.Pin = pin
	return &msg, nil
}

func (s *memMessageStore) GetMessage(id string) (*storage.Message, error) {
	if v, ok := s.messages.Load(id); ok {
		if msg, ok := v.(storage.Message); ok {
			// clear the pin to let the view know it needs decryption
			msg.Pin = ""
			return &msg, nil
		} else {
			return nil, fmt.Errorf("unexpected message type")
		}
	}
	return nil, nil
}

func (s *memMessageStore) GetFullMessage(id string, pin string) (*storage.Message, error) {
	if v, ok := s.messages.Load(id); ok {
		if msg, ok := v.(storage.Message); ok {

			if err := crypto.CompareHashToPass(msg.Pin, pin); err == nil {

				text, err := s.Decrypt(msg.Content, pin, s.salt)
				if err != nil {
					return nil, err
				}

				msg.Content = text

				// self destruct the message after successful retrieval
				s.messages.Delete(id)

				return &msg, nil
			}

			msg.AttemptsRemaining -= 1
			s.messages.Store(id, msg)

			// If the pin was wrong then start tracking attempts
			if msg.AttemptsRemaining <= 0 {
				s.messages.Delete(id)
			}
		} else {
			// do not keep broken messages
			s.messages.Delete(id)
			return nil, fmt.Errorf("unexpected message type")
		}
	}
	return nil, nil
}
