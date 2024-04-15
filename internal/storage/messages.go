package storage

import (
	"fmt"
	"sync"
	"time"
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

type memMessageStore struct {
	messages sync.Map
	salt     string
}

func NewMemMessageStore(salt string) MessageStore {
	return &memMessageStore{messages: sync.Map{}, salt: salt}
}

func (s *memMessageStore) Encrypt(text string, pass string) (string, error) {
	// derive a key from the pass
	key, err := StrongKey(pass, s.salt)
	if err != nil {
		return "", err
	}
	ciphertext, err := EncryptAES(key, text)
	if err != nil {
		return "", err
	}
	return ciphertext, nil
}

// Decrypt cipher text with a given PIN which will be used to derive a key
func (s *memMessageStore) Decrypt(ciphertext string, pass string) (string, error) {
	// derive a key from the pass
	key, err := StrongKey(pass, s.salt)
	if err != nil {
		return "", err
	}
	plaintext, err := DecryptAES(key, ciphertext)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

func (s *memMessageStore) ListMessages(username string) ([]Message, error) {
	var msgs []Message
	s.messages.Range(func(k, v any) bool {
		if msg, ok := v.(Message); ok && msg.Username == username {
			// do not expose sensitive info
			msgs = append(msgs, Message{
				Digest:  msg.Digest,
				Created: msg.Created,
			})
		}
		return true
	})
	return msgs, nil
}

// TODO: allow to reset the pin for the owner
func (s *memMessageStore) AddMessage(text string, username string) (Message, error) {
	// an easy to enter pin
	pin, err := MakePin()
	if err != nil {
		return Message{}, err
	}
	ciphertext, err := s.Encrypt(text, pin)
	if err != nil {
		return Message{}, err
	}
	msg, err := NewMessage(username, ciphertext, pin)
	if err != nil {
		return Message{}, err
	}
	// store unreadbale message, pin
	s.messages.Store(msg.Digest, msg)
	// temporarily show the pin to the creator
	msg.Pin = pin
	return msg, nil
}

func (s *memMessageStore) GetMessage(id string) (*Message, error) {
	if v, ok := s.messages.Load(id); ok {
		if msg, ok := v.(Message); ok {
			return &Message{
				Digest:  msg.Digest,
				Created: msg.Created,
			}, nil
		} else {
			return nil, fmt.Errorf("unexpected message type")
		}
	}
	return nil, nil
}

func (s *memMessageStore) GetFullMessage(id string, pin string) (*Message, error) {
	if v, ok := s.messages.Load(id); ok {
		if msg, ok := v.(Message); ok {

			if err := CompareHashToPass(msg.Pin, pin); err == nil {

				text, err := s.Decrypt(msg.Content, pin)
				if err != nil {
					return nil, err
				}

				// self destruct the message after successful retrieval
				s.messages.Delete(id)

				return &Message{
					Username: msg.Username,
					Digest:   msg.Digest,
					Created:  msg.Created,
					Content:  text,
					Pin:      msg.Pin,
					Attempt:  msg.Attempt,
				}, nil
			}

			msg.Attempt += 1
			s.messages.Store(id, msg)

			// If the pin was wrong then start tracking attempts
			if msg.Attempt >= MAX_PIN_ATTEMPTS {
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

type Message struct {
	Username string    `json:"user_displayname"`
	Digest   string    `json:"digest"`
	Created  time.Time `json:"created"`
	Content  string    `json:"content,omitempty"`
	Pin      string    `json:"pin,omitempty"`
	Attempt  int       `json:"Attempt,omitempty"`
}

func NewMessage(username string, content string, pin string) (Message, error) {
	pinHash, err := HashPass(pin)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Username: username,
		Content:  content,
		Digest:   HashText(content),
		Created:  time.Now(),
		Pin:      pinHash,
		Attempt:  0,
	}, nil
}
