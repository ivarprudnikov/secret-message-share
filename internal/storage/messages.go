package storage

import (
	"fmt"
	"sync"
	"time"
)

const MAX_PIN_ATTEMPTS = 5

type Store struct {
	messages sync.Map
}

func NewStore() *Store {
	return &Store{messages: sync.Map{}}
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
	pin, err := MakePin()
	if err != nil {
		return Message{}, err
	}
	msg := Message{
		Username: username,
		Content:  text,
		Digest:   HashText(text),
		Created:  time.Now(),
		Pin:      HashText(pin),
		Attempt:  0,
	}
	s.messages.Store(msg.Digest, msg)
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
			if msg.Pin == HashText(pin) {
				// self destruct the message after successful retrieval
				s.messages.Delete(id)
				return &Message{
					Username: msg.Username,
					Digest:   msg.Digest,
					Created:  msg.Created,
					Content:  msg.Content,
					Pin:      msg.Pin,
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
