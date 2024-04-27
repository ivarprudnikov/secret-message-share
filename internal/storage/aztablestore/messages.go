package aztablestore

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

type azMessageStore struct {
	accountName string
	tableName   string
	salt        string
}

func NewAzMessageStore(accountName, tableName, salt string) storage.MessageStore {
	return &azMessageStore{accountName: accountName, tableName: tableName, salt: salt}
}

func (s *azMessageStore) getClient() (*aztables.Client, error) {
	return getTableClient(s.accountName, s.tableName)
}

func (s *azMessageStore) CountMessages() (int64, error) {
	var count int64 = 0
	client, err := s.getClient()
	if err != nil {
		return count, err
	}
	keySelector := "PartitionKey"
	metadataFormat := aztables.MetadataFormatNone
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Select: &keySelector,
		Format: &metadataFormat,
	})
	for listPager.More() {
		response, err := listPager.NextPage(context.TODO())
		if err != nil {
			return count, err
		}
		count += int64(len(response.Entities))
	}
	return count, nil
}

// TODO move to storage
func (s *azMessageStore) Encrypt(text string, pass string) (string, error) {
	// derive a key from the pass
	key, err := storage.StrongKey(pass, s.salt)
	if err != nil {
		return "", err
	}
	ciphertext, err := storage.EncryptAES(key, text)
	if err != nil {
		return "", err
	}
	return ciphertext, nil
}

// Decrypt cipher text with a given PIN which will be used to derive a key
func (s *azMessageStore) Decrypt(ciphertext string, pass string) (string, error) {
	// derive a key from the pass
	key, err := storage.StrongKey(pass, s.salt)
	if err != nil {
		return "", err
	}
	plaintext, err := storage.DecryptAES(key, ciphertext)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

func (s *azMessageStore) ListMessages(username string) ([]*storage.Message, error) {
	var msgs []*storage.Message
	client, err := s.getClient()
	if err != nil {
		return msgs, err
	}
	userFilter := fmt.Sprintf("RowKey eq '%s'", username)
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: &userFilter,
	})
	for listPager.More() {
		response, err := listPager.NextPage(context.TODO())
		if err != nil {
			return msgs, err
		}
		for _, v := range response.Entities {
			var msg *storage.Message
			err = json.Unmarshal(v, &msg)
			if err != nil {
				return msgs, err
			}
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

// TODO: allow to reset the pin for the owner
func (s *azMessageStore) AddMessage(text string, username string) (*storage.Message, error) {
	// an easy to enter pin
	pin, err := storage.MakePin()
	if err != nil {
		return nil, err
	}
	ciphertext, err := s.Encrypt(text, pin)
	if err != nil {
		return nil, err
	}
	msg, err := storage.NewMessage(username, ciphertext, pin)
	if err != nil {
		return nil, err
	}
	err = s.saveMessage(&msg)
	msg.Pin = pin
	return &msg, nil
}

func (s *azMessageStore) GetMessage(id string) (*storage.Message, error) {
	msg, err := s.getMessage(id)
	if err != nil {
		return nil, err
	}
	// clear the pin to let the view know it needs decryption
	msg.Pin = ""
	return msg, nil
}

func (s *azMessageStore) GetFullMessage(id string, pin string) (*storage.Message, error) {
	msg, err := s.getMessage(id)
	if err != nil {
		return nil, err
	}

	if err := storage.CompareHashToPass(msg.Pin, pin); err == nil {
		text, err := s.Decrypt(msg.Content, pin)
		if err != nil {
			return nil, err
		}
		msg.Content = text
		err = s.deleteMessage(msg)
		if err != nil {
			slog.LogAttrs(context.TODO(), slog.LevelError, "failed to delete message", slog.String("id", msg.PartitionKey), slog.String("username", msg.RowKey))
		}
		return msg, nil
	}

	msg.AttemptsRemaining -= 1
	// If the pin was wrong then track attempts
	if msg.AttemptsRemaining <= 0 {
		err = s.deleteMessage(msg)
		if err != nil {
			slog.LogAttrs(context.TODO(), slog.LevelError, "failed to delete message", slog.String("id", msg.PartitionKey), slog.String("username", msg.RowKey))
		}
	} else {
		err = s.saveMessage(msg)
		if err != nil {
			slog.LogAttrs(context.TODO(), slog.LevelError, "failed to update message", slog.String("id", msg.PartitionKey), slog.String("username", msg.RowKey))
		}
	}

	return nil, nil
}

func (s *azMessageStore) getMessage(id string) (*storage.Message, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}
	var msgs []*storage.Message
	idFilter := fmt.Sprintf("PartitionKey eq '%s'", id)
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: &idFilter,
	})
	for listPager.More() {
		response, err := listPager.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, v := range response.Entities {
			var msg *storage.Message
			err = json.Unmarshal(v, &msg)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, msg)
		}
	}
	if len(msgs) > 1 {
		slog.LogAttrs(context.TODO(), slog.LevelError, "more than one message with the same id", slog.String("id", id), slog.Int("total", len(msgs)))
	}
	return msgs[0], nil
}

func (s *azMessageStore) saveMessage(msg *storage.Message) error {
	marshalled, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	client, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to get aztable client: %w", err)
	}
	_, err = client.UpsertEntity(context.TODO(), marshalled, &aztables.UpsertEntityOptions{
		UpdateMode: aztables.UpdateModeReplace,
	})
	if err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	return nil
}

func (s *azMessageStore) deleteMessage(msg *storage.Message) error {
	client, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to get aztable client: %w", err)
	}
	_, err = client.DeleteEntity(context.TODO(), msg.PartitionKey, msg.RowKey, nil)
	if err != nil {
		return err
	}
	return nil
}
