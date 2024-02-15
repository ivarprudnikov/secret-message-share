package storage

import (
	"fmt"
	"sync"
	"time"
)

const MAX_PIN_ATTEMPTS = 5

type Store struct {
	messages sync.Map
	salt     string
}

func NewStore(salt string) *Store {
	return &Store{messages: sync.Map{}, salt: salt}
}

func (s *Store) Encrypt(text string, pass string) (string, error) {
	// derive a key from the pass
	key, err := StrongKey(pass, s.salt)
	if err != nil {
		return "", err
	}
	cyphertext, err := EncryptAES(key, text)
	if err != nil {
		return "", err
	}
	return cyphertext, nil
}

func (s *Store) Decrypt(cyphertext string, pass string) (string, error) {
	// derive a key from the pass
	key, err := StrongKey(pass, s.salt)
	if err != nil {
		return "", err
	}
	plaintext, err := DecryptAES(key, cyphertext)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

func (s *Store) ListMessages() ([]Message, error) {
	var msgs []Message
	s.messages.Range(func(k, v any) bool {
		if msg, ok := v.(Message); ok {
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

func (s *Store) AddMessage(text string, username string) (Message, error) {
	// an easy to enter pin
	pin, err := MakePin()
	if err != nil {
		return Message{}, err
	}
	cyphertext, err := s.Encrypt(text, pin)
	msg := NewMessage(username, cyphertext, pin)
	if err != nil {
		return Message{}, err
	}
	// store unreadbale message, pin
	s.messages.Store(msg.Digest, msg)
	// temporarily show the pin to the creator
	msg.Pin = pin
	return msg, nil
}

func (s *Store) GetMessage(id string) (*Message, error) {
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

func (s *Store) GetFullMessage(id string, pin string) (*Message, error) {
	if v, ok := s.messages.Load(id); ok {
		if msg, ok := v.(Message); ok {
			// TODO use salted hash
			if msg.Pin == HashText(pin) {

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

			// If the pin was wrong then start tracking attempts
			if msg.Attempt > MAX_PIN_ATTEMPTS {
				s.messages.Delete(id)
			} else {
				msg.Attempt += 1
				s.messages.Store(id, msg)
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

func NewMessage(username string, content string, pin string) Message {
	return Message{
		Username: username,
		Content:  content,
		Digest:   HashText(content),
		Created:  time.Now(),
		// TODO use salt for hashing to mitigate rainbow attacks
		Pin:     HashText(pin),
		Attempt: 0,
	}
}
