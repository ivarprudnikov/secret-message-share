package aztablestore

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/ivarprudnikov/secretshare/internal/crypto"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

type azMessageStore struct {
	crypto.EntityEncryptHelper
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

func (s *azMessageStore) CountMessages(ctx context.Context) (int64, error) {
	var count int64 = 0
	client, err := s.getClient()
	if err != nil {
		return count, fmt.Errorf("failed to get aztable client: %w", err)
	}
	keySelector := "PartitionKey"
	metadataFormat := aztables.MetadataFormatNone
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Select: &keySelector,
		Format: &metadataFormat,
	})
	for listPager.More() {
		response, err := listPager.NextPage(ctx)
		if err != nil {
			return count, fmt.Errorf("failed to get page of results: %w", err)
		}
		count += int64(len(response.Entities))
	}
	return count, nil
}

func (s *azMessageStore) ListMessages(ctx context.Context, username string) ([]*storage.Message, error) {
	var msgs []*storage.Message
	client, err := s.getClient()
	if err != nil {
		return msgs, fmt.Errorf("failed to get aztable client: %w", err)
	}
	userFilter := fmt.Sprintf("RowKey eq '%s'", username)
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: &userFilter,
	})
	for listPager.More() {
		response, err := listPager.NextPage(ctx)
		if err != nil {
			return msgs, fmt.Errorf("failed to get page of results: %w", err)
		}
		for _, v := range response.Entities {
			var msg *storage.Message
			err = json.Unmarshal(v, &msg)
			if err != nil {
				return msgs, fmt.Errorf("failed to unmarshal message in list of results: %w", err)
			}
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

// TODO: allow to reset the pin for the owner
func (s *azMessageStore) AddMessage(ctx context.Context, text string, username string) (*storage.Message, error) {
	// an easy to enter pin
	pin, err := crypto.MakePin()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pin: %w", err)
	}
	ciphertext, err := s.Encrypt(text, pin, s.salt)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt text: %w", err)
	}
	msg, err := storage.NewMessage(username, ciphertext, pin)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new message: %w", err)
	}
	err = s.saveMessage(ctx, &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}
	msg.Pin = pin
	return &msg, nil
}

func (s *azMessageStore) GetMessage(ctx context.Context, id string) (*storage.Message, error) {
	msg, err := s.getMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	// clear the pin to let the view know it needs decryption
	msg.Pin = ""
	return msg, nil
}

func (s *azMessageStore) GetFullMessage(ctx context.Context, id string, pin string) (*storage.Message, error) {
	msg, err := s.getMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}

	if err := crypto.CompareHashToPass(msg.Pin, pin); err == nil {
		text, err := s.Decrypt(msg.Content, pin, s.salt)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt message content: %w", err)
		}
		msg.Content = text
		err = s.deleteMessage(ctx, msg)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "failed to delete message", slog.String("id", msg.PartitionKey), slog.String("username", msg.RowKey), slog.Any("error", err))
		}
		return msg, nil
	}

	msg.AttemptsRemaining -= 1
	// If the pin was wrong then track attempts
	if msg.AttemptsRemaining <= 0 {
		err = s.deleteMessage(ctx, msg)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "failed to delete message", slog.String("id", msg.PartitionKey), slog.String("username", msg.RowKey), slog.Any("error", err))
		}
	} else {
		err = s.saveMessage(ctx, msg)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "failed to update message", slog.String("id", msg.PartitionKey), slog.String("username", msg.RowKey), slog.Any("error", err))
		}
	}

	return nil, nil
}

func (s *azMessageStore) getMessage(ctx context.Context, id string) (*storage.Message, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get aztable client: %w", err)
	}
	var msgs []*storage.Message
	idFilter := fmt.Sprintf("PartitionKey eq '%s'", id)
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: &idFilter,
	})
	for listPager.More() {
		response, err := listPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get page of results: %w", err)
		}
		for _, v := range response.Entities {
			var msg *storage.Message
			err = json.Unmarshal(v, &msg)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal message: %w", err)
			}
			msgs = append(msgs, msg)
		}
	}
	if len(msgs) > 1 {
		slog.LogAttrs(ctx, slog.LevelError, "more than one message with the same id", slog.String("id", id), slog.Int("total", len(msgs)))
		return msgs[0], nil
	} else if len(msgs) == 1 {
		return msgs[0], nil
	}
	return nil, nil
}

func (s *azMessageStore) saveMessage(ctx context.Context, msg *storage.Message) error {
	marshalled, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	client, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to get aztable client: %w", err)
	}
	_, err = client.UpsertEntity(ctx, marshalled, &aztables.UpsertEntityOptions{
		UpdateMode: aztables.UpdateModeReplace,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert message entity: %w", err)
	}
	return nil
}

func (s *azMessageStore) deleteMessage(ctx context.Context, msg *storage.Message) error {
	client, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to get aztable client: %w", err)
	}
	_, err = client.DeleteEntity(ctx, msg.PartitionKey, msg.RowKey, nil)
	if err != nil {
		return fmt.Errorf("failed to delete message entity: %w", err)
	}
	return nil
}
