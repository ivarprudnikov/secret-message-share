package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

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

func (s *Store) AddMessage(text string) (Message, error) {
	textHash := sha256.Sum256([]byte(text))
	textHashHex := hex.EncodeToString(textHash[:])
	pin, err := makePin()
	if err != nil {
		return Message{}, err
	}
	msg := Message{
		Content: text,
		Digest:  textHashHex,
		Created: time.Now(),
		Pin:     fmt.Sprintf("%d", pin),
	}
	s.messages.Store(msg.Digest, msg)
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
			if msg.Pin == pin {
				return &Message{
					Digest:  msg.Digest,
					Created: msg.Created,
					Content: msg.Content,
					Pin:     msg.Pin,
				}, nil
			}
		} else {
			return nil, fmt.Errorf("unexpected message type")
		}
	}
	return nil, nil
}

type Message struct {
	Content string    `json:"content,omitempty"`
	Digest  string    `json:"digest"`
	Pin     string    `json:"pin,omitempty"`
	Created time.Time `json:"created"`
}

// simple pin generator
func makePin() (uint16, error) {
	c := 16
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}
