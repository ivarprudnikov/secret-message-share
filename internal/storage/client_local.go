package storage

import (
	"strings"
	"sync"

	"golang.org/x/net/context"
)

type memoryClient struct {
	things sync.Map
}

func NewLocalClient() StorageClient {
	return &memoryClient{things: sync.Map{}}
}

func (c *memoryClient) ListAll(ctx context.Context, partitionKey string) ([][]byte, error) {
	var things [][]byte
	c.things.Range(func(k, v any) bool {
		if key, ok := k.(string); ok && strings.HasPrefix(key, partitionKey) {
			things = append(things, v.([]byte))
		}
		return true
	})
	return things, nil
}

func (c *memoryClient) GetOne(ctx context.Context, partitionKey string, rowKey string) ([]byte, error) {
	if val, ok := c.things.Load(partitionKey + "_" + rowKey); ok {
		return val.([]byte), nil
	}
	return nil, nil
}

func (c *memoryClient) SaveOne(ctx context.Context, partitionKey string, rowKey string, item []byte) error {
	key := partitionKey + "_" + rowKey
	c.things.Store(key, item)
	return nil
}
